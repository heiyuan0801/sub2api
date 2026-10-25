package clusterops

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"math/big"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/google/uuid"
)

func BillingRequestID(ctx context.Context) string {
	if v, _ := ctx.Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(v) != "" {
		return "client:" + strings.TrimSpace(v)
	}
	if v, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(v) != "" {
		return "local:" + strings.TrimSpace(v)
	}
	return "local:" + uuid.NewString()
}

func (m *Manager) StartRequest(ctx context.Context, userID, keyID int64, route string) (context.Context, string, error) {
	if !m.HAEnabled() {
		return ctx, "", nil
	}
	id := uuid.NewString()
	requestID := BillingRequestID(ctx)
	if len(requestID) > 256 || len(route) > 256 {
		return ctx, "", fmt.Errorf("request metadata exceeds journal limits")
	}
	qctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, err := m.db.ExecContext(qctx, `INSERT INTO cluster_request_journal(attempt_id,request_id,node_id,instance_id,user_id,api_key_id,route) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, requestID, m.cfg.Cluster.NodeID, m.instance, userID, keyID, route)
	if err != nil {
		return ctx, "", err
	}
	return WithAttempt(ctx, id), id, nil
}

// Review eligibility is materialized when the request finishes. The dashboard
// can query the small pending-state index instead of joining every historic request.
const billingMismatchSQL = `EXISTS(SELECT 1 FROM cluster_billing_evidence b LEFT JOIN usage_logs u ON u.request_id=b.request_id AND u.api_key_id=b.api_key_id WHERE (b.attempt_id=j.attempt_id OR (b.request_id=j.request_id AND b.api_key_id=j.api_key_id)) AND (u.id IS NULL OR ABS(u.actual_cost-b.expected_cost)>0.00000001 OR (NOT EXISTS(SELECT 1 FROM usage_billing_dedup d WHERE d.request_id=b.request_id AND d.api_key_id=b.api_key_id) AND NOT EXISTS(SELECT 1 FROM usage_billing_dedup_archive d WHERE d.request_id=b.request_id AND d.api_key_id=b.api_key_id))))`

func (m *Manager) FinishRequest(id string, status int) {
	if id == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := m.db.ExecContext(ctx, `UPDATE cluster_request_journal j SET state=CASE WHEN (`+billingMismatchSQL+`) OR ($2 BETWEEN 200 AND 299 AND $2<>202 AND route ~ '(/responses|/chat/completions|/messages|/embeddings|/images/(generations|edits))$|:(streamGenerateContent|generateContent)$' AND NOT EXISTS(SELECT 1 FROM cluster_billing_evidence b WHERE b.attempt_id=j.attempt_id OR (b.request_id=j.request_id AND b.api_key_id=j.api_key_id))) THEN 'needs_review' ELSE 'finished' END,finished_at=NOW(),http_status=$2 WHERE attempt_id=$1 AND state='running'`, id, status)
	if err != nil {
		m.errors.Add(1)
	}
}

const requestProjection = `SELECT j.attempt_id::text,j.request_id,j.node_id,j.user_id,j.api_key_id,j.route,j.state,COALESCE(j.http_status,0),j.started_at,j.finished_at,j.review_note,j.reviewed_at,
COALESCE(e.evidence_count,0),COALESCE(e.settled_count,0),COALESCE(e.usage_count,0),COALESCE(e.expected_cost,0)::text,COALESCE(e.recorded_cost,0)::text
FROM cluster_request_journal j LEFT JOIN LATERAL (
 SELECT count(*) evidence_count,
 count(*) FILTER (WHERE EXISTS(SELECT 1 FROM usage_billing_dedup d WHERE d.request_id=b.request_id AND d.api_key_id=b.api_key_id) OR EXISTS(SELECT 1 FROM usage_billing_dedup_archive d WHERE d.request_id=b.request_id AND d.api_key_id=b.api_key_id)) settled_count,
 count(u.id) usage_count, sum(b.expected_cost) expected_cost, sum(u.actual_cost) recorded_cost
 FROM cluster_billing_evidence b LEFT JOIN usage_logs u ON u.request_id=b.request_id AND u.api_key_id=b.api_key_id
 WHERE b.attempt_id=j.attempt_id OR (b.request_id=j.request_id AND b.api_key_id=j.api_key_id)
) e ON true `

func (m *Manager) Requests(ctx context.Context, onlyReview bool, before time.Time) ([]Request, error) {
	if !m.Enabled() {
		return []Request{}, nil
	}
	q := requestProjection + ` WHERE ($1::boolean=false OR (j.reviewed_at IS NULL AND j.state='needs_review')) AND j.started_at<$2 ORDER BY j.started_at DESC,j.attempt_id LIMIT 100`
	if before.IsZero() {
		before = time.Now().Add(time.Second)
	}
	rows, err := m.db.QueryContext(ctx, q, onlyReview, before)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := []Request{}
	for rows.Next() {
		var v Request
		if err = rows.Scan(&v.ID, &v.RequestID, &v.NodeID, &v.UserID, &v.APIKeyID, &v.Route, &v.State, &v.HTTPStatus, &v.StartedAt, &v.FinishedAt, &v.ReviewNote, &v.ReviewedAt, &v.EvidenceCount, &v.SettledCount, &v.UsageCount, &v.ExpectedCost, &v.RecordedCost); err != nil {
			return nil, err
		}
		v.BillingState = ClassifyBilling(v)
		items = append(items, v)
	}
	return items, rows.Err()
}

func ClassifyBilling(r Request) string {
	if r.EvidenceCount == 0 {
		if r.State == "needs_review" {
			return "unknown"
		}
		return "no_evidence"
	}
	if r.SettledCount < r.EvidenceCount {
		return "settlement_pending"
	}
	if r.UsageCount < r.EvidenceCount {
		return "usage_missing"
	}
	expected, ok := new(big.Rat).SetString(r.ExpectedCost)
	recorded, valid := new(big.Rat).SetString(r.RecordedCost)
	if !ok || !valid {
		return "unknown"
	}
	delta := new(big.Rat).Abs(new(big.Rat).Sub(expected, recorded))
	if delta.Cmp(big.NewRat(1, 100000000)) > 0 {
		return "cost_mismatch"
	}
	return "settled"
}

func (m *Manager) ReviewRequest(ctx context.Context, id, note string, actor int64) error {
	if !m.Enabled() {
		return ErrUnavailable
	}
	if _, err := uuid.Parse(id); err != nil {
		return err
	}
	note = strings.TrimSpace(note)
	if len(note) < 3 || len(note) > 1000 {
		return fmt.Errorf("review note must contain 3-1000 bytes")
	}
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	r, err := tx.ExecContext(ctx, `UPDATE cluster_request_journal SET reviewed_at=NOW(),reviewed_by=$2,review_note=$3 WHERE attempt_id=$1 AND state<>'running'`, id, actor, note)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrConflict
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO cluster_events(kind,message,actor_id) VALUES('billing_review',$1,$2)`, id+": review recorded", actor)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (m *Manager) BalanceStatus(ctx context.Context, userID int64) (map[string]any, error) {
	var balance string
	if err := m.db.QueryRowContext(ctx, `SELECT balance::text FROM users WHERE id=$1 AND deleted_at IS NULL`, userID).Scan(&balance); err != nil {
		return nil, err
	}
	cache, err := m.rdb.Get(ctx, fmt.Sprintf("billing:balance:%d", userID)).Result()
	result := map[string]any{"user_id": userID, "database_balance": balance, "cache_balance": nil, "cache_status": "missing", "inflight": nil}
	if err == nil {
		result["cache_balance"] = cache
		result["cache_status"] = "present"
	} else if !errors.Is(err, redis.Nil) {
		return nil, err
	}
	count, err := m.rdb.ZCount(ctx, fmt.Sprintf("billing:inflight:{%d}", userID), fmt.Sprintf("%d", time.Now().UnixMilli()), "+inf").Result()
	if err != nil {
		return nil, err
	}
	result["inflight"] = count
	return result, nil
}

func (m *Manager) InvalidateBalance(ctx context.Context, userID, actor int64) error {
	if !m.Enabled() {
		return ErrUnavailable
	}
	if _, err := m.BalanceStatus(ctx, userID); err != nil {
		return err
	}
	// Invalidation never writes a guessed balance or removes active reservations.
	if err := m.rdb.Del(ctx, fmt.Sprintf("billing:balance:%d", userID)).Err(); err != nil {
		return err
	}
	_, err := m.db.ExecContext(ctx, `INSERT INTO cluster_events(kind,message,actor_id) VALUES('balance_cache_invalidated',$1,$2)`, fmt.Sprintf("user %d; rebuild from database on next use", userID), actor)
	return err
}
