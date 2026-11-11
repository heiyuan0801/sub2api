//go:build unit

package repository

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRewardActivitiesWithWalletBillingPostgres(t *testing.T) {
	dsn := os.Getenv("SUB2API_REWARD_TEST_DSN")
	if dsn == "" {
		t.Skip("set SUB2API_REWARD_TEST_DSN to an isolated sub2api_reward_test database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()
	var database string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT current_database()").Scan(&database))
	require.Equal(t, "sub2api_reward_test", database)
	require.NoError(t, ApplyMigrations(ctx, db))
	peerDB, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer peerDB.Close()

	repo := NewRewardActivityRepository(db)
	peer := NewRewardActivityRepository(peerDB)
	billing := NewUsageBillingRepository(nil, db)
	var userID, keyID int64
	require.NoError(t, db.QueryRowContext(ctx,
		"INSERT INTO users (email,password_hash,balance) VALUES ($1,'test-fixture',10) RETURNING id",
		uuid.NewString()+"@example.test").Scan(&userID))
	require.NoError(t, db.QueryRowContext(ctx,
		"INSERT INTO api_keys (user_id,key,name) VALUES ($1,$2,'reward test') RETURNING id",
		userID, uuid.NewString()).Scan(&keyID))
	t.Cleanup(func() {
		// Fixtures live in an explicitly isolated database.
		cleanup, err := sql.Open("postgres", dsn)
		if err == nil {
			defer cleanup.Close()
			_, _ = cleanup.Exec("DELETE FROM users WHERE id = $1", userID)
		}
	})

	charge := func(amount float64) *service.UsageBillingCommand {
		cmd := &service.UsageBillingCommand{RequestID: uuid.NewString(), UserID: userID, APIKeyID: keyID, BalanceCost: amount}
		result, err := billing.Apply(ctx, cmd)
		require.NoError(t, err)
		require.True(t, result.Applied)
		return cmd
	}
	cfg := service.DefaultRewardActivityConfig()
	require.NoError(t, repo.SaveConfig(ctx, cfg))
	charge(0.20)
	disabled, err := repo.Status(ctx, userID)
	require.NoError(t, err)
	require.Zero(t, disabled.SpentCents)

	cfg.CheckIn.Enabled, cfg.Wheel.Enabled = true, true
	cfg.CheckIn.MaxRewardCents = 10
	cfg.Wheel.DailyLimit = 2
	cfg.Wheel.Prizes = []service.RewardPrize{{Name: "Credit A", RewardCents: 35, Weight: 1}, {Name: "Credit B", RewardCents: 35, Weight: 1}}
	require.NoError(t, repo.SaveConfig(ctx, cfg))
	charge(0.99)
	_, err = repo.Claim(ctx, userID, service.RewardCheckIn, uuid.New())
	require.ErrorIs(t, err, service.ErrRewardSpendRequired)
	cmd := charge(0.01)
	duplicate, err := billing.Apply(ctx, cmd)
	require.NoError(t, err)
	require.False(t, duplicate.Applied)
	charge(1)

	batchID := uuid.NewString()
	hold := &service.BatchImageBalanceHoldCommand{BatchID: batchID, RequestID: service.BatchImageHoldRequestID(batchID), APIKeyID: keyID, UserID: userID, HoldAmount: 0.75}
	_, err = billing.ReserveBatchImageBalance(ctx, hold)
	require.NoError(t, err)
	beforeCapture, err := repo.Status(ctx, userID)
	require.NoError(t, err)
	require.EqualValues(t, 200, beforeCapture.SpentCents)
	settlement := *hold
	settlement.RequestID = uuid.NewString()
	settlement.ActualAmount = 0.5
	_, err = billing.CaptureBatchImageBalance(ctx, &settlement)
	require.NoError(t, err)
	_, err = billing.CaptureBatchImageBalance(ctx, &settlement)
	require.NoError(t, err)
	afterCapture, err := repo.Status(ctx, userID)
	require.NoError(t, err)
	require.EqualValues(t, 250, afterCapture.SpentCents)
	require.Equal(t, 2, afterCapture.DrawsAvailable)

	// Different instances must share the daily check-in constraint.
	checkIns := make(chan error, 12)
	var wg sync.WaitGroup
	for i := 0; i < cap(checkIns); i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			instance := repo
			if index%2 == 1 {
				instance = peer
			}
			_, err := instance.Claim(ctx, userID, service.RewardCheckIn, uuid.New())
			checkIns <- err
		}(i)
	}
	wg.Wait()
	close(checkIns)
	var successes int
	for err := range checkIns {
		if err == nil {
			successes++
		} else {
			require.ErrorIs(t, err, service.ErrRewardAlreadyCheckedIn)
		}
	}
	require.Equal(t, 1, successes)

	// Concurrent retries of one draw have one record and one wallet credit.
	drawID := uuid.New()
	results := make(chan *service.RewardActivityClaim, 8)
	drawErrors := make(chan error, 8)
	for i := 0; i < cap(results); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := peer.Claim(ctx, userID, service.RewardWheel, drawID)
			results <- result
			drawErrors <- err
		}()
	}
	wg.Wait()
	close(results)
	close(drawErrors)
	for err := range drawErrors {
		require.NoError(t, err)
	}
	var claimID int64
	for result := range results {
		require.EqualValues(t, 35, result.RewardCents)
		if claimID == 0 {
			claimID = result.ID
		}
		require.Equal(t, claimID, result.ID)
	}
	_, err = repo.Claim(ctx, userID, service.RewardWheel, uuid.New())
	require.NoError(t, err)
	_, err = repo.Claim(ctx, userID, service.RewardWheel, uuid.New())
	require.ErrorIs(t, err, service.ErrRewardDrawLimit)
	var balance, totalRecharged float64
	require.NoError(t, db.QueryRowContext(ctx, "SELECT balance, total_recharged FROM users WHERE id = $1", userID).Scan(&balance, &totalRecharged))
	require.InDelta(t, 8.10, balance, 0.00000001)
	require.Zero(t, totalRecharged)

	final, err := repo.Status(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, 2, final.DrawsUsed)
	require.EqualValues(t, 200, final.DrawSpendUsedCents)
	require.Len(t, final.History, 3)
	require.True(t, final.CheckedIn)
	require.EqualValues(t, 250, final.SpentCents)

	_, err = db.ExecContext(ctx, "UPDATE reward_activity_spend SET spend_day = spend_day - 1 WHERE user_id = $1", userID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "UPDATE reward_activity_claims SET activity_day = activity_day - 1 WHERE user_id = $1", userID)
	require.NoError(t, err)
	charge(0.50)
	today, err := repo.Status(ctx, userID)
	require.NoError(t, err)
	require.EqualValues(t, 50, today.SpentCents)
	require.Zero(t, today.DrawsUsed)
	require.False(t, today.CheckedIn)
	require.Len(t, today.History, 3)
}
