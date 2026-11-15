package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionPlanOperationsPostgres(t *testing.T) {
	dsn := os.Getenv("SUB2API_PLAN_OPERATIONS_TEST_DSN")
	if dsn == "" {
		t.Skip("set SUB2API_PLAN_OPERATIONS_TEST_DSN to a disposable empty database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	var table sql.NullString
	require.NoError(t, db.QueryRowContext(ctx, "SELECT to_regclass('subscription_plans')::text").Scan(&table))
	require.False(t, table.Valid, "test requires an empty database")
	require.NoError(t, ApplyMigrations(ctx, db))
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	config := service.NewPaymentConfigService(client, nil, nil)

	group, err := client.Group.Create().SetName("Ops subscription").SetPlatform("openai").
		SetSubscriptionType("subscription").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	eligible, err := client.User.Create().SetEmail("eligible@example.test").SetPasswordHash("test").AddAllowedGroupIDs(group.ID).Save(ctx)
	require.NoError(t, err)
	other, err := client.User.Create().SetEmail("other@example.test").SetPasswordHash("test").Save(ctx)
	require.NoError(t, err)
	create := func(name, scope string, userIDs, groupIDs []int64) *dbent.SubscriptionPlan {
		plan, err := config.CreatePlan(ctx, service.CreatePlanRequest{
			Name: name, GroupID: group.ID, Price: 10, Currency: "USD", ValidityDays: 30, ValidityUnit: "days", ForSale: true,
			PlanPresentation: service.PlanPresentation{VisibilityScope: scope, VisibilityUserIDs: userIDs, VisibilityGroupIDs: groupIDs,
				DisplayTags: []string{"recommended"}},
		})
		require.NoError(t, err)
		return plan
	}
	public := create("Public", "all", nil, nil)
	private := create("Selected users", "users", []int64{eligible.ID}, nil)
	groupPlan := create("Selected groups", "groups", nil, []int64{group.ID})
	del := create("Disposable", "all", nil, nil)
	require.NoError(t, config.ReorderPlans(ctx, []int64{del.ID, groupPlan.ID, private.ID, public.ID}))
	plans, err := config.ListPlansForSale(ctx, eligible.ID)
	require.NoError(t, err)
	require.Len(t, plans, 4)
	require.Equal(t, del.ID, plans[0].ID)
	require.Equal(t, []string{"recommended"}, plans[0].DisplayTags)
	plans, err = config.ListPlansForSale(ctx, other.ID)
	require.NoError(t, err)
	require.Len(t, plans, 2)
	require.Equal(t, public.ID, plans[1].ID)
	require.NoError(t, config.DeletePlan(ctx, del.ID, eligible.ID))

	order, err := client.PaymentOrder.Create().SetUserID(eligible.ID).SetUserEmail(eligible.Email).
		SetUserName("Test").SetAmount(public.Price).SetPayAmount(public.Price).SetFeeRate(0).
		SetRechargeCode("OPS-TEST-ORDER").SetOutTradeNo("sub2_ops_price_test").
		SetPaymentType("alipay").SetPaymentTradeNo("OPS-TEST-TRADE").SetOrderType("subscription").SetPlanID(public.ID).
		SetSubscriptionGroupID(group.ID).SetSubscriptionDays(30).SetStatus("completed").SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").SetSrcHost("ops.example.test").Save(ctx)
	require.NoError(t, err)

	old := public.Price
	change, err := config.ChangePlanPrice(ctx, public.ID, eligible.ID, service.PlanPriceChangeRequest{
		ExpectedPrice: &old, NewPrice: 15, Reason: "Updated quota costs",
	})
	require.NoError(t, err)
	require.Equal(t, "applied", change.Status)
	require.Equal(t, 50.0, *change.Percent)
	_, err = config.ChangePlanPrice(ctx, public.ID, eligible.ID, service.PlanPriceChangeRequest{ExpectedPrice: &old, NewPrice: 12})
	require.ErrorContains(t, err, "reload the plan")

	old = change.NewPrice
	scheduledAt := time.Now().Add(time.Hour)
	change, err = config.ChangePlanPrice(ctx, public.ID, eligible.ID, service.PlanPriceChangeRequest{
		ExpectedPrice: &old, NewPrice: 9.50, Reason: "Planned lower price", EffectiveAt: &scheduledAt,
	})
	require.NoError(t, err)
	require.Equal(t, "pending", change.Status)
	published, err := config.GetPlan(ctx, public.ID)
	require.NoError(t, err)
	require.Equal(t, 15.0, published.Price)
	_, err = db.ExecContext(ctx, "UPDATE subscription_plan_price_changes SET effective_at=now()-interval '1 second' WHERE id=$1", change.ID)
	require.NoError(t, err)
	func() {
		workerA := service.ProvidePlanPriceWorker(config)
		defer workerA.Stop()
		workerB := service.ProvidePlanPriceWorker(config)
		defer workerB.Stop()
		require.Eventually(t, func() bool {
			snapshot, err := config.GetPlan(ctx, public.ID)
			return err == nil && snapshot.Price == 9.50
		}, 8*time.Second, 25*time.Millisecond)
	}()
	events, err := config.ListPlanPriceChanges(ctx, public.ID, 0)
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "applied", events[0].Status)
	require.NotNil(t, events[0].ResolvedAt)
	saved, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, 10.0, saved.Amount)
	require.Equal(t, 10.0, saved.PayAmount)
	require.Equal(t, group.ID, *saved.SubscriptionGroupID)
	require.Equal(t, 30, *saved.SubscriptionDays)

	rollback, err := os.ReadFile("../../scripts/rollback/253_subscription_plan_operations.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(rollback))
	require.NoError(t, err)
	require.NoError(t, ApplyMigrations(ctx, db))
	published, err = config.GetPlan(ctx, private.ID)
	require.NoError(t, err)
	require.False(t, published.ForSale)
	published, err = config.GetPlan(ctx, public.ID)
	require.NoError(t, err)
	require.Equal(t, 9.50, published.Price)
}
