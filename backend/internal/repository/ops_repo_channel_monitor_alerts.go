package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// SaveChannelMonitorAlertEvent checkpoints an existing Ops incident. Locking
// its rule serializes even the first pending insert, when no event exists yet.
// The revision check rejects stale evaluations and duplicate notification state.
func (r *opsRepository) SaveChannelMonitorAlertEvent(ctx context.Context, event *service.OpsAlertEvent, expectedEventID, expectedRevision int64) (*service.OpsAlertEvent, error) {
	if r == nil || r.db == nil || event == nil || event.RuleID <= 0 {
		return nil, fmt.Errorf("invalid channel monitor checkpoint")
	}
	dimensions, err := opsNullJSONMap(event.Dimensions)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var ruleID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM ops_alert_rules WHERE id = $1 FOR UPDATE`, event.RuleID).Scan(&ruleID); err != nil {
		return nil, err
	}
	var latestID int64
	var raw []byte
	var latestStatus string
	err = tx.QueryRowContext(ctx, `SELECT id, dimensions, status FROM ops_alert_events WHERE rule_id = $1 ORDER BY fired_at DESC, id DESC LIMIT 1 FOR UPDATE`, event.RuleID).Scan(&latestID, &raw, &latestStatus)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	var envelope struct {
		State struct {
			Revision int64 `json:"revision"`
		} `json:"channel_monitor_state"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return nil, err
		}
	}
	if latestID != expectedEventID || envelope.State.Revision != expectedRevision {
		return nil, service.ErrOpsAlertCheckpointConflict
	}
	// Manual status updates predate monitor revisions. Their row lock still
	// serializes here, and a closed incident must never be reopened in place.
	if event.ID != 0 && (latestStatus == service.OpsAlertStatusResolved || latestStatus == service.OpsAlertStatusManualResolved) && (event.Status == "pending" || event.Status == service.OpsAlertStatusFiring) {
		return nil, service.ErrOpsAlertCheckpointConflict
	}
	out := *event
	if event.ID == 0 {
		err = tx.QueryRowContext(ctx, `INSERT INTO ops_alert_events
			(rule_id, severity, status, title, description, metric_value, threshold_value, dimensions, fired_at, resolved_at, email_sent, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW()) RETURNING id, created_at`,
			event.RuleID, event.Severity, event.Status, event.Title, event.Description,
			opsNullFloat64(event.MetricValue), opsNullFloat64(event.ThresholdValue), dimensions,
			event.FiredAt, opsNullTime(event.ResolvedAt), event.EmailSent).Scan(&out.ID, &out.CreatedAt)
	} else {
		if event.ID != latestID {
			return nil, service.ErrOpsAlertCheckpointConflict
		}
		_, err = tx.ExecContext(ctx, `UPDATE ops_alert_events SET status=$2, title=$3, description=$4,
			metric_value=$5, threshold_value=$6, dimensions=$7, fired_at=$8, resolved_at=$9, email_sent=$10 WHERE id=$1`,
			event.ID, event.Status, event.Title, event.Description,
			opsNullFloat64(event.MetricValue), opsNullFloat64(event.ThresholdValue), dimensions,
			event.FiredAt, opsNullTime(event.ResolvedAt), event.EmailSent)
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &out, nil
}
