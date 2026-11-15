package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/require"
)

func TestRegistrationSourcePostgres(t *testing.T) {
	dsn := os.Getenv("SUB2API_REGISTRATION_ATTRIBUTION_TEST_DSN")
	if dsn == "" {
		t.Skip("set SUB2API_REGISTRATION_ATTRIBUTION_TEST_DSN to an empty, disposable PostgreSQL database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	var usersTable sql.NullString
	require.NoError(t, db.QueryRowContext(ctx, "SELECT to_regclass('users')::text").Scan(&usersTable))
	require.False(t, usersTable.Valid, "test requires an empty, disposable database")
	require.NoError(t, ApplyMigrations(ctx, db))

	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	repo := newUserRepositoryWithSQL(client, db)
	create := func(ctx context.Context, email string, when time.Time) *service.User {
		user := &service.User{Email: email, PasswordHash: "test-hash", Role: service.RoleUser, Status: service.StatusActive, Concurrency: 1}
		require.NoError(t, repo.CreateWithEmailAliasGuard(ctx, user))
		_, err := db.ExecContext(ctx, "UPDATE users SET created_at = $1 WHERE id = $2", when, user.ID)
		require.NoError(t, err)
		return user
	}

	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	today := time.Date(2026, 11, 16, 0, 0, 0, 0, location)
	source := service.RegistrationAttribution{SelectedSource: "friend", UTMSource: "github", UTMCampaign: "launch"}
	first := create(service.WithRegistrationAttribution(ctx, source), "identified@example.com", today.Add(time.Minute))
	older := create(ctx, "older@example.com", today.AddDate(0, 0, -1))
	deleted := create(ctx, "deleted@example.com", today.Add(time.Minute))
	require.NoError(t, repo.Delete(ctx, deleted.ID))

	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	rollbackCandidate := &service.User{
		Email: "rolled-back@example.com", PasswordHash: "test-hash", Role: service.RoleUser, Status: service.StatusActive,
	}
	txCtx := dbent.NewTxContext(service.WithRegistrationAttribution(ctx, source), tx)
	require.NoError(t, repo.CreateWithEmailAliasGuard(txCtx, rollbackCandidate))
	require.NoError(t, tx.Rollback())
	var found int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_registration_attributions WHERE user_id = $1", rollbackCandidate.ID).Scan(&found))
	require.Zero(t, found)

	duplicate := &service.User{
		Email: first.Email, PasswordHash: "test-hash", Role: service.RoleUser, Status: service.StatusActive,
	}
	require.ErrorIs(t, repo.CreateWithEmailAliasGuard(service.WithRegistrationAttribution(ctx,
		service.RegistrationAttribution{SelectedSource: "bing"}), duplicate), service.ErrEmailExists)

	promo := service.NewPromoService(NewPromoCodeRepository(client), repo, nil, client, nil)
	future := time.Now().Add(time.Hour)
	code, err := promo.Create(ctx, &service.CreatePromoCodeInput{Code: "TRACKINGONLY", MaxUses: 1, ExpiresAt: &future})
	require.NoError(t, err)
	require.NoError(t, promo.ApplyPromoCode(ctx, first.ID, code.Code))
	require.ErrorIs(t, promo.ApplyPromoCode(ctx, older.ID, code.Code), service.ErrPromoCodeMaxUsed)
	code, err = promo.GetByID(ctx, code.ID)
	require.NoError(t, err)
	require.Equal(t, 1, code.UsedCount)
	first, err = repo.GetByID(ctx, first.ID)
	require.NoError(t, err)
	require.Zero(t, first.Balance)

	affiliates := NewAffiliateRepository(client, db)
	_, err = affiliates.BindInviter(ctx, first.ID, first.ID)
	require.ErrorIs(t, err, service.ErrAffiliateCodeInvalid)
	bound, err := affiliates.BindInviter(ctx, first.ID, older.ID)
	require.NoError(t, err)
	require.True(t, bound)
	bound, err = affiliates.BindInviter(ctx, first.ID, older.ID)
	require.NoError(t, err)
	require.False(t, bound)

	filter := service.RegistrationSourceFilter{
		Start: today.AddDate(0, 0, -1), End: today.AddDate(0, 0, 1),
		TodayStart: today, TodayEnd: today.AddDate(0, 0, 1),
		Timezone: location.String(), Dimension: service.RegistrationDimensionSelected,
	}
	report, err := repo.RegistrationSourceSummary(ctx, filter)
	require.NoError(t, err)
	require.EqualValues(t, 2, report.TotalRegistrations)
	require.EqualValues(t, 1, report.TodayRegistrations)
	require.EqualValues(t, 1, report.UnreportedCount)
	require.Equal(t, []service.RegistrationSourceChannel{
		{Source: "friend", Count: 1, Percentage: 50}, {Source: "unknown", Count: 1, Percentage: 50},
	}, report.Channels)
	require.Equal(t, today.Format("2006-01-02"), report.Daily[1].Date)

	filter.Dimension, filter.Source = service.RegistrationDimensionUTM, "github"
	users, page, err := repo.ListRegistrationSourceUsers(ctx, filter, pagination.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.EqualValues(t, 1, page.Total)
	require.Len(t, users, 1)
	require.Equal(t, source, users[0].RegistrationAttribution)
	require.Equal(t, "email", users[0].SignupSource)
	require.NotNil(t, users[0].InviterID)
	require.Equal(t, older.ID, *users[0].InviterID)
	require.Equal(t, code.Code, users[0].PromoCode)
	require.NoError(t, repo.ExportRegistrationSourceUsers(ctx, filter, func(user service.RegistrationSourceUser) error {
		require.Equal(t, first.ID, user.ID)
		return nil
	}))

	rollback, err := os.ReadFile("../../scripts/rollback/255_user_registration_attribution.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(rollback))
	require.NoError(t, err)
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL").Scan(&found))
	require.Equal(t, 2, found)
	require.NoError(t, ApplyMigrations(ctx, db))
}
