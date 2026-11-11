package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

// Wallet billing and reward claims both acquire the user row first. The spend
// update is part of the already deduplicated wallet debit, never a usage-log sum.
const recordRewardActivitySpend = `
	), reward_spend AS (
		INSERT INTO reward_activity_spend (user_id, spend_day, amount)
		SELECT user_id, (clock_timestamp() AT TIME ZONE 'Asia/Shanghai')::date, amount
		FROM charged
		WHERE amount > 0 AND EXISTS (
			SELECT 1 FROM reward_activity_config WHERE id = 1
			  AND (config #>> '{check_in,enabled}' = 'true' OR config #>> '{wheel,enabled}' = 'true')
		)
		ON CONFLICT (user_id) DO UPDATE
		SET spend_day = EXCLUDED.spend_day,
		    amount = CASE WHEN reward_activity_spend.spend_day = EXCLUDED.spend_day
		                  THEN reward_activity_spend.amount + EXCLUDED.amount ELSE EXCLUDED.amount END
		WHERE reward_activity_spend.spend_day <= EXCLUDED.spend_day
	)
`

const rewardActivityClaimFields = `id, user_id, to_char(activity_day, 'YYYY-MM-DD'), kind, reward_cents, prize_index, prize_name, created_at`

type rewardActivityRepository struct {
	db *sql.DB
}

func NewRewardActivityRepository(db *sql.DB) service.RewardActivityRepository {
	return &rewardActivityRepository{db: db}
}

type rewardActivityQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (r *rewardActivityRepository) GetConfig(ctx context.Context) (service.RewardActivityConfig, error) {
	return readRewardActivityConfig(r.db.QueryRowContext(ctx, "SELECT config FROM reward_activity_config WHERE id = 1"))
}

func readRewardActivityConfig(row *sql.Row) (service.RewardActivityConfig, error) {
	cfg := service.DefaultRewardActivityConfig()
	var data []byte
	if err := row.Scan(&data); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, cfg.Validate()
}

func (r *rewardActivityRepository) SaveConfig(ctx context.Context, cfg service.RewardActivityConfig) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO reward_activity_config (id, config) VALUES (1, $1::jsonb)
		ON CONFLICT (id) DO UPDATE SET config = EXCLUDED.config
	`, string(data))
	return err
}

func rewardActivityProgress(ctx context.Context, tx *sql.Tx, userID int64, cfg service.RewardActivityConfig) (*service.RewardActivityStatus, error) {
	status := &service.RewardActivityStatus{Config: cfg}
	err := tx.QueryRowContext(ctx, `
		WITH day AS (SELECT (statement_timestamp() AT TIME ZONE 'Asia/Shanghai')::date AS value)
		SELECT to_char(day.value, 'YYYY-MM-DD'),
		       COALESCE((SELECT LEAST(floor(amount * 100), 9007199254740991)::bigint
		                 FROM reward_activity_spend WHERE user_id = $1 AND spend_day = day.value), 0),
		       COUNT(claim.id) FILTER (WHERE claim.kind = 'check_in') > 0,
		       COUNT(claim.id) FILTER (WHERE claim.kind = 'wheel'),
		       COALESCE(SUM(claim.consumed_cents) FILTER (WHERE claim.kind = 'wheel'), 0)
		FROM day
		LEFT JOIN reward_activity_claims claim ON claim.user_id = $1 AND claim.activity_day = day.value
		GROUP BY day.value
	`, userID).Scan(&status.Day, &status.SpentCents, &status.CheckedIn, &status.DrawsUsed, &status.DrawSpendUsedCents)
	if err != nil {
		return nil, err
	}
	status.SetDrawsAvailable()
	return status, nil
}

func (r *rewardActivityRepository) Status(ctx context.Context, userID int64) (*service.RewardActivityStatus, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	cfg, err := readRewardActivityConfig(tx.QueryRowContext(ctx, "SELECT config FROM reward_activity_config WHERE id = 1"))
	if err != nil {
		return nil, err
	}
	status, err := rewardActivityProgress(ctx, tx, userID, cfg)
	if err != nil {
		return nil, err
	}
	status.History, err = listRewardActivityClaims(ctx, tx, "WHERE user_id = $1 ORDER BY id DESC LIMIT 20", userID)
	if err != nil {
		return nil, err
	}
	return status, tx.Commit()
}

func (r *rewardActivityRepository) Claim(ctx context.Context, userID int64, kind string, requestID uuid.UUID) (*service.RewardActivityClaim, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	// This lock serializes wallet rewards across both application nodes.
	var id int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE id = $1 AND deleted_at IS NULL AND status = 'active' FOR UPDATE`, userID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	var claim service.RewardActivityClaim
	err = scanRewardActivityClaim(tx.QueryRowContext(ctx,
		"SELECT "+rewardActivityClaimFields+" FROM reward_activity_claims WHERE user_id = $1 AND request_id = $2",
		userID, requestID), &claim)
	if err == nil {
		if claim.Kind != kind {
			return nil, service.ErrRewardRequestConflict
		}
		return &claim, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	cfg, err := readRewardActivityConfig(tx.QueryRowContext(ctx, "SELECT config FROM reward_activity_config WHERE id = 1 FOR SHARE"))
	if err != nil {
		return nil, err
	}
	status, err := rewardActivityProgress(ctx, tx, userID, cfg)
	if err != nil {
		return nil, err
	}
	claim, consumed, err := status.PickReward(kind)
	if err != nil {
		return nil, err
	}

	err = scanRewardActivityClaim(tx.QueryRowContext(ctx, `
		INSERT INTO reward_activity_claims (user_id, request_id, activity_day, kind, reward_cents, consumed_cents, prize_index, prize_name)
		VALUES ($1, $2, $3::date, $4, $5, $6, $7, $8)
		RETURNING `+rewardActivityClaimFields,
		userID, requestID, claim.Day, kind, claim.RewardCents, consumed, claim.PrizeIndex, claim.PrizeName), &claim)
	if err != nil {
		return nil, err
	}
	if claim.RewardCents > 0 {
		// Activity credit is spendable balance, separate from recharge totals.
		if _, err := tx.ExecContext(ctx, `UPDATE users SET balance = balance + $2::numeric / 100, updated_at = NOW() WHERE id = $1`, userID, claim.RewardCents); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &claim, nil
}

func (r *rewardActivityRepository) RecentClaims(ctx context.Context) ([]service.RewardActivityClaim, error) {
	return listRewardActivityClaims(ctx, r.db, "ORDER BY id DESC LIMIT 50")
}

func listRewardActivityClaims(ctx context.Context, db rewardActivityQuerier, clause string, args ...any) ([]service.RewardActivityClaim, error) {
	rows, err := db.QueryContext(ctx, "SELECT "+rewardActivityClaimFields+" FROM reward_activity_claims "+clause, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	claims := make([]service.RewardActivityClaim, 0)
	for rows.Next() {
		var claim service.RewardActivityClaim
		if err := scanRewardActivityClaim(rows, &claim); err != nil {
			return nil, err
		}
		claims = append(claims, claim)
	}
	return claims, rows.Err()
}

func scanRewardActivityClaim(row interface{ Scan(...any) error }, claim *service.RewardActivityClaim) error {
	return row.Scan(&claim.ID, &claim.UserID, &claim.Day, &claim.Kind, &claim.RewardCents, &claim.PrizeIndex, &claim.PrizeName, &claim.CreatedAt)
}
