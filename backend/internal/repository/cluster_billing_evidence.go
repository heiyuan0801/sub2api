package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *usageBillingRepository) StageClusterBillingEvidence(ctx context.Context, cmd *service.UsageBillingCommand, usage *service.UsageLog, attemptID string) error {
	// Explicit scalar projection: never serialize User/APIKey/Account relations,
	// provider credentials, request bodies, output text, IPs or user agents.
	v := map[string]any{"model": cmd.Model, "account_id": cmd.AccountID, "input_tokens": cmd.InputTokens, "output_tokens": cmd.OutputTokens, "cache_creation_tokens": cmd.CacheCreationTokens, "cache_read_tokens": cmd.CacheReadTokens, "image_count": cmd.ImageCount, "balance_cost": cmd.BalanceCost, "subscription_cost": cmd.SubscriptionCost}
	cost := cmd.BalanceCost + cmd.SubscriptionCost
	if usage != nil {
		v["actual_cost"] = usage.ActualCost
		v["total_cost"] = usage.TotalCost
		cost = usage.ActualCost
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var attempt any
	if attemptID != "" {
		attempt = attemptID
	}
	// Normal traffic persists and returns its fingerprint in one WAN round trip.
	// Only a duplicate needs a second statement, which sees the winning row
	// after ON CONFLICT has waited for its transaction to commit.
	var fingerprint string
	err = r.db.QueryRowContext(ctx, `INSERT INTO cluster_billing_evidence(request_id,api_key_id,attempt_id,user_id,fingerprint,expected_cost,usage_snapshot) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(request_id,api_key_id) DO NOTHING RETURNING fingerprint`, cmd.RequestID, cmd.APIKeyID, attempt, cmd.UserID, cmd.RequestFingerprint, cost, raw).Scan(&fingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		err = r.db.QueryRowContext(ctx, `SELECT fingerprint FROM cluster_billing_evidence WHERE request_id=$1 AND api_key_id=$2`, cmd.RequestID, cmd.APIKeyID).Scan(&fingerprint)
	}
	if err != nil {
		return err
	}
	if fingerprint != cmd.RequestFingerprint {
		return service.ErrUsageBillingRequestConflict
	}
	return nil
}
