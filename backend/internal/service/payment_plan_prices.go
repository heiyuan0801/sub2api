package service

import (
	"context"
	"database/sql"
	"log/slog"
	"math"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/subscriptionplan"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type PlanPriceChangeRequest struct {
	ExpectedPrice *float64   `json:"expected_price"`
	NewPrice      float64    `json:"new_price"`
	Reason        string     `json:"reason"`
	EffectiveAt   *time.Time `json:"effective_at"`
}

type PlanPriceChange struct {
	ID          int64      `json:"id"`
	PlanID      int64      `json:"plan_id"`
	PlanName    string     `json:"plan_name"`
	GroupID     int64      `json:"group_id"`
	Currency    string     `json:"currency"`
	OldPrice    float64    `json:"old_price"`
	NewPrice    float64    `json:"new_price"`
	Delta       float64    `json:"delta"`
	Percent     *float64   `json:"percent"`
	Reason      string     `json:"reason"`
	ActorID     int64      `json:"actor_id"`
	CancelledBy *int64     `json:"cancelled_by,omitempty"`
	Status      string     `json:"status"`
	EffectiveAt time.Time  `json:"effective_at"`
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

const planPriceColumns = "id, plan_id, plan_name, group_id, currency, old_price, new_price, reason, actor_id, cancelled_by, status, effective_at, created_at, resolved_at"

func lockPlan(ctx context.Context, client *dbent.Client, id int64) (*dbent.SubscriptionPlan, error) {
	plan, err := client.SubscriptionPlan.Query().Where(subscriptionplan.IDEQ(id)).ForUpdate().Only(ctx)
	if dbent.IsNotFound(err) {
		return nil, infraerrors.NotFound("PLAN_NOT_FOUND", "subscription plan not found")
	}
	return plan, err
}

func scanPriceChanges(rows *sql.Rows) ([]PlanPriceChange, error) {
	defer func() { _ = rows.Close() }()
	result := []PlanPriceChange{}
	for rows.Next() {
		var c PlanPriceChange
		if err := rows.Scan(&c.ID, &c.PlanID, &c.PlanName, &c.GroupID, &c.Currency, &c.OldPrice, &c.NewPrice,
			&c.Reason, &c.ActorID, &c.CancelledBy, &c.Status, &c.EffectiveAt, &c.CreatedAt, &c.ResolvedAt); err != nil {
			return nil, err
		}
		c.Delta = math.Round((c.NewPrice-c.OldPrice)*100) / 100
		if c.OldPrice > 0 {
			percent := c.Delta / c.OldPrice * 100
			c.Percent = &percent
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func ensureNoPendingPlanPrice(ctx context.Context, client *dbent.Client, id int64) error {
	rows, err := client.QueryContext(ctx, "SELECT 1 FROM subscription_plan_price_changes WHERE plan_id=$1 AND status='pending'", id)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	if rows.Next() {
		return infraerrors.Conflict("PLAN_PRICE_PENDING", "cancel the pending price change first")
	}
	return rows.Err()
}

func insertPlanPriceChange(ctx context.Context, client *dbent.Client, plan *dbent.SubscriptionPlan, price float64, reason string, actorID int64, at *time.Time) (PlanPriceChange, error) {
	if utf8.RuneCountInString(reason) > 1000 {
		return PlanPriceChange{}, infraerrors.BadRequest("PLAN_PRICE_REASON_LENGTH", "price change reason is too long")
	}
	rows, err := client.QueryContext(ctx, `
INSERT INTO subscription_plan_price_changes
(plan_id, plan_name, group_id, currency, old_price, new_price, reason, actor_id, status, effective_at, resolved_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,
        CASE WHEN $9::timestamptz IS NULL THEN 'applied' ELSE 'pending' END,
        COALESCE($9::timestamptz, now()), CASE WHEN $9::timestamptz IS NULL THEN now() END)
RETURNING `+planPriceColumns, plan.ID, plan.Name, plan.GroupID, plan.Currency, plan.Price, price, reason, actorID, at)
	if err != nil {
		return PlanPriceChange{}, err
	}
	result, err := scanPriceChanges(rows)
	if err != nil {
		return PlanPriceChange{}, err
	}
	if len(result) == 0 {
		return PlanPriceChange{}, sql.ErrNoRows
	}
	return result[0], nil
}

func (s *PaymentConfigService) ChangePlanPrice(ctx context.Context, planID, actorID int64, request PlanPriceChangeRequest) (PlanPriceChange, error) {
	reason := strings.TrimSpace(request.Reason)
	if request.ExpectedPrice == nil || !validPlanPrice(request.NewPrice) {
		return PlanPriceChange{}, infraerrors.BadRequest("PLAN_PRICE_INVALID", "current price and new price are required")
	}
	if request.EffectiveAt != nil && (!isJSONTimeInRange(request.EffectiveAt) || !request.EffectiveAt.After(time.Now())) {
		return PlanPriceChange{}, infraerrors.BadRequest("PLAN_PRICE_SCHEDULE_INVALID", "effective time must be in the future")
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return PlanPriceChange{}, err
	}
	defer func() { _ = tx.Rollback() }()
	client := tx.Client()
	plan, err := lockPlan(ctx, client, planID)
	if err != nil {
		return PlanPriceChange{}, err
	}
	if plan.Price != *request.ExpectedPrice {
		return PlanPriceChange{}, infraerrors.Conflict("PLAN_PRICE_CHANGED", "reload the plan before confirming the price change")
	}
	if plan.Price == request.NewPrice {
		return PlanPriceChange{}, infraerrors.BadRequest("PLAN_PRICE_UNCHANGED", "the new price matches the current price")
	}
	if err := ensureNoPendingPlanPrice(ctx, client, planID); err != nil {
		return PlanPriceChange{}, err
	}
	if request.EffectiveAt == nil {
		if err := client.SubscriptionPlan.UpdateOneID(planID).SetPrice(request.NewPrice).Exec(ctx); err != nil {
			return PlanPriceChange{}, err
		}
	}
	change, err := insertPlanPriceChange(ctx, client, plan, request.NewPrice, reason, actorID, request.EffectiveAt)
	if err != nil {
		return PlanPriceChange{}, err
	}
	return change, tx.Commit()
}

// Price history retains the plan snapshot after the administrator deletes the plan.
func (s *PaymentConfigService) ListPlanPriceChanges(ctx context.Context, planID, beforeID int64) ([]PlanPriceChange, error) {
	rows, err := s.entClient.QueryContext(ctx, "SELECT "+planPriceColumns+`
FROM subscription_plan_price_changes WHERE ($1=0 OR plan_id=$1) AND ($2=0 OR id<$2)
ORDER BY id DESC LIMIT 50`, planID, beforeID)
	if err != nil {
		return nil, err
	}
	return scanPriceChanges(rows)
}

func (s *PaymentConfigService) CancelPlanPriceChange(ctx context.Context, id, actorID int64) error {
	result, err := s.entClient.ExecContext(ctx, `
UPDATE subscription_plan_price_changes SET status='cancelled', cancelled_by=$2, resolved_at=now()
WHERE id=$1 AND status='pending'`, id, actorID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return infraerrors.Conflict("PLAN_PRICE_NOT_PENDING", "this price change is no longer pending")
	}
	return err
}

func (s *PaymentConfigService) ReorderPlans(ctx context.Context, order []int64) error {
	ids := normalizedAudienceIDs(order)
	if len(ids) == 0 || len(order) != len(ids) {
		return infraerrors.BadRequest("PLAN_ORDER_INVALID", "provide distinct subscription plan IDs")
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	client := tx.Client()
	locked, err := client.SubscriptionPlan.Query().Order(subscriptionplan.ByID()).ForUpdate().IDs(ctx)
	if err != nil {
		return err
	}
	if !slices.Equal(ids, locked) {
		return infraerrors.Conflict("PLAN_ORDER_STALE", "reload the plan list before reordering")
	}
	for index, id := range order {
		if err := client.SubscriptionPlan.UpdateOneID(id).SetSortOrder(index).Exec(ctx); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Lock the plan before touching its schedule, in the same order as edits/deletes.
func (s *PaymentConfigService) applyNextPlanPriceChange(ctx context.Context) (bool, error) {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	client := tx.Client()
	rows, err := client.QueryContext(ctx, `
SELECT p.id FROM subscription_plans p JOIN subscription_plan_price_changes c ON p.id=c.plan_id
WHERE c.status='pending' AND c.effective_at<=now()
ORDER BY c.effective_at, c.id LIMIT 1 FOR UPDATE OF p SKIP LOCKED`)
	if err != nil {
		return false, err
	}
	var planID int64
	if rows.Next() {
		err = rows.Scan(&planID)
	} else {
		err = rows.Err()
	}
	_ = rows.Close()
	if err != nil || planID == 0 {
		return false, err
	}
	rows, err = client.QueryContext(ctx, "SELECT "+planPriceColumns+`
FROM subscription_plan_price_changes WHERE plan_id=$1 AND status='pending' AND effective_at<=now() FOR UPDATE`, planID)
	if err != nil {
		return false, err
	}
	changes, err := scanPriceChanges(rows)
	if err != nil || len(changes) == 0 {
		return false, err
	}
	change := changes[0]
	plan, err := client.SubscriptionPlan.Get(ctx, planID)
	if err != nil {
		return false, err
	}
	status := "conflict"
	if plan.Price == change.OldPrice && plan.Currency == change.Currency && plan.GroupID == change.GroupID {
		if err := client.SubscriptionPlan.UpdateOneID(planID).SetPrice(change.NewPrice).Exec(ctx); err != nil {
			return false, err
		}
		status = "applied"
	}
	if _, err := client.ExecContext(ctx, "UPDATE subscription_plan_price_changes SET status=$2, resolved_at=now() WHERE id=$1", change.ID, status); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

type PlanPriceWorker struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func ProvidePlanPriceWorker(svc *PaymentConfigService) *PlanPriceWorker {
	ctx, cancel := context.WithCancel(context.Background())
	worker := &PlanPriceWorker{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(worker.done)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			runCtx, cancelRun := context.WithTimeout(ctx, 4*time.Second)
			for range 30 {
				applied, err := svc.applyNextPlanPriceChange(runCtx)
				if err != nil && ctx.Err() == nil {
					slog.Warn("subscription plan price job failed", "error", err)
				}
				if !applied || err != nil {
					break
				}
			}
			cancelRun()
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return worker
}

func (w *PlanPriceWorker) Stop() {
	w.cancel()
	<-w.done
}
