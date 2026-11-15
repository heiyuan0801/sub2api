package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAnnouncementDeliveryPostgres(t *testing.T) {
	dsn := os.Getenv("SUB2API_ANNOUNCEMENT_TEST_DSN")
	if dsn == "" {
		t.Skip("set SUB2API_ANNOUNCEMENT_TEST_DSN to a disposable sub2api_notification_test database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	var database string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT current_database()").Scan(&database))
	require.Equal(t, "sub2api_notification_test", database)
	require.NoError(t, ApplyMigrations(ctx, db))
	clients := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	repo := NewAnnouncementDeliveryRepository(db)
	svc := service.ProvideAnnouncementService(NewAnnouncementRepository(clients), NewAnnouncementReadRepository(clients),
		NewUserRepository(clients, db), NewUserSubscriptionRepository(clients), repo, nil)

	createUser := func(role string) int64 {
		t.Helper()
		var id int64
		require.NoError(t, db.QueryRowContext(ctx, `
			INSERT INTO users(email, password_hash, role) VALUES ($1, 'test-only', $2) RETURNING id
		`, uuid.NewString()+"@example.test", role).Scan(&id))
		return id
	}
	admin, first, muted, outsider := createUser("admin"), createUser("user"), createUser("user"), createUser("user")
	var groupID, planID int64
	require.NoError(t, db.QueryRowContext(ctx, `
		INSERT INTO groups(name, subscription_type) VALUES ($1, 'subscription') RETURNING id
	`, uuid.NewString()).Scan(&groupID))
	require.NoError(t, db.QueryRowContext(ctx, `
		INSERT INTO subscription_plans(group_id, name, price) VALUES ($1, 'Test plan', 1.25) RETURNING id
	`, groupID).Scan(&planID))
	_, err = db.ExecContext(ctx, `
		INSERT INTO user_subscriptions(user_id, group_id, starts_at, expires_at) VALUES
		    ($1, $3, now() - interval '1 day', now() + interval '1 day'),
		    ($2, $3, now() - interval '1 day', now() + interval '1 day')
	`, first, muted, groupID)
	require.NoError(t, err)

	createNotice := func(kind string, audience service.AnnouncementAudience) *service.Announcement {
		t.Helper()
		notice, err := svc.Create(ctx, &service.CreateAnnouncementInput{
			Title: uuid.NewString(), Content: "Only entitled users should see this", ActorID: &admin,
			Publication: &service.AnnouncementPublication{Kind: kind, PlanID: &planID, Audience: audience},
		})
		require.NoError(t, err)
		return notice
	}
	require.ErrorIs(t, svc.SetGroupSubscription(ctx, outsider, groupID, false), service.ErrAnnouncementNotFound)
	require.NoError(t, svc.SetGroupSubscription(ctx, muted, groupID, false))
	require.NoError(t, svc.SetGroupSubscription(ctx, muted, groupID, false))
	groups, err := svc.MutedGroups(ctx, muted)
	require.NoError(t, err)
	require.Equal(t, []int64{groupID}, groups)
	otherGroups, err := svc.MutedGroups(ctx, first)
	require.NoError(t, err)
	require.Empty(t, otherGroups)

	scheduled := createNotice("price_increase", service.AnnouncementAudience{Mode: "groups", GroupIDs: []int64{groupID, groupID}})
	_, err = svc.GetForUser(ctx, first, scheduled.ID)
	require.ErrorIs(t, err, service.ErrAnnouncementNotFound)
	future := time.Now().Add(time.Hour)
	require.NoError(t, svc.Publish(ctx, scheduled.ID, admin, &future))
	processed, err := repo.Dispatch(ctx, scheduled.ID)
	require.NoError(t, err)
	require.False(t, processed)
	list, err := svc.ListForUser(ctx, first, false)
	require.NoError(t, err)
	require.Empty(t, list)
	_, err = svc.GetForUser(ctx, first, scheduled.ID)
	require.ErrorIs(t, err, service.ErrAnnouncementNotFound)

	// Simulate an offline worker coming back after the stored deadline.
	_, err = db.ExecContext(ctx, `UPDATE announcements SET starts_at = now() - interval '1 minute' WHERE id = $1`, scheduled.ID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `UPDATE announcement_publications SET next_attempt_at = now() - interval '1 minute' WHERE announcement_id = $1`, scheduled.ID)
	require.NoError(t, err)

	var deliveries sync.WaitGroup
	const workers = 4
	results := make(chan bool, workers)
	dispatchErrors := make(chan error, workers)
	for range workers {
		deliveries.Go(func() {
			delivered, err := repo.Dispatch(ctx, scheduled.ID)
			results <- delivered
			dispatchErrors <- err
		})
	}
	deliveries.Wait()
	close(results)
	close(dispatchErrors)
	activated := 0
	for delivered := range results {
		if delivered {
			activated++
		}
	}
	for err := range dispatchErrors {
		require.NoError(t, err)
	}
	require.Equal(t, 1, activated)
	for _, denied := range []int64{muted, outsider} {
		_, err := svc.GetForUser(ctx, denied, scheduled.ID)
		require.ErrorIs(t, err, service.ErrAnnouncementNotFound)
		require.ErrorIs(t, svc.MarkRead(ctx, denied, scheduled.ID), service.ErrAnnouncementNotFound)
		_, err = svc.RecordClick(ctx, denied, scheduled.ID)
		require.ErrorIs(t, err, service.ErrAnnouncementNotFound)
	}
	item, err := svc.GetForUser(ctx, first, scheduled.ID)
	require.NoError(t, err)
	require.Equal(t, "price_increase", item.Kind)
	require.Equal(t, &planID, item.PlanID)
	_, err = db.ExecContext(ctx, "UPDATE user_subscriptions SET expires_at = now() WHERE user_id = $1", first)
	require.NoError(t, err)
	_, err = svc.GetForUser(ctx, first, scheduled.ID)
	require.NoError(t, err, "the recipient snapshot survives subscription expiration")
	url, err := svc.RecordClick(ctx, first, scheduled.ID)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("/purchase?plan_id=%d", planID), url)
	require.NoError(t, svc.MarkRead(ctx, first, scheduled.ID))
	require.NoError(t, svc.MarkRead(ctx, first, scheduled.ID))
	stats, err := svc.DeliveryStats(ctx, scheduled.ID)
	require.NoError(t, err)
	require.Equal(t, service.AnnouncementDeliveryPublished, stats.DeliveryStatus)
	require.Equal(t, 1, stats.Attempts)
	require.EqualValues(t, 1, stats.TargetCount)
	require.EqualValues(t, 1, stats.ReadCount)
	require.Zero(t, stats.UnreadCount)
	require.EqualValues(t, 1, stats.ClickCount)

	// System notices ignore subscription opt-out even if a plan is linked.
	system := createNotice("system", service.AnnouncementAudience{Mode: "users", UserIDs: []int64{muted}})
	require.NoError(t, svc.Publish(ctx, system.ID, admin, nil))
	_, err = svc.GetForUser(ctx, muted, system.ID)
	require.NoError(t, err)
	require.NoError(t, svc.SetGroupSubscription(ctx, muted, groupID, true))
	legacy, err := svc.Create(ctx, &service.CreateAnnouncementInput{
		Title: "Legacy group notice", Content: "Keep the original rules and inbox", Status: service.AnnouncementStatusActive,
		Targeting: service.AnnouncementTargeting{AnyOf: []service.AnnouncementConditionGroup{
			{AllOf: []service.AnnouncementCondition{{
				Type: service.AnnouncementConditionTypeSubscription, Operator: service.AnnouncementOperatorIn, GroupIDs: []int64{groupID},
			}}},
		}},
	})
	require.NoError(t, err)
	replacement := createNotice("new_plan", service.AnnouncementAudience{Mode: "groups", GroupIDs: []int64{groupID}})
	require.NoError(t, svc.Publish(ctx, replacement.ID, admin, nil))
	history, total, unread, err := svc.HistoryForUser(ctx, muted, false, pagination.PaginationParams{Page: 1, PageSize: 1})
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.EqualValues(t, 3, unread)
	require.Equal(t, replacement.ID, history[0].Announcement.ID)
	history, _, _, err = svc.HistoryForUser(ctx, muted, false, pagination.PaginationParams{Page: 2, PageSize: 1})
	require.NoError(t, err)
	require.Equal(t, legacy.ID, history[0].Announcement.ID)
	history, _, _, err = svc.HistoryForUser(ctx, muted, false, pagination.PaginationParams{Page: 3, PageSize: 1})
	require.NoError(t, err)
	require.Equal(t, system.ID, history[0].Announcement.ID)
	updated, err := svc.MarkAllRead(ctx, muted)
	require.NoError(t, err)
	require.EqualValues(t, 3, updated)
	updated, err = svc.MarkAllRead(ctx, muted)
	require.NoError(t, err)
	require.Zero(t, updated)
	require.NoError(t, svc.Withdraw(ctx, replacement.ID, admin))
	_, err = svc.GetForUser(ctx, muted, replacement.ID)
	require.ErrorIs(t, err, service.ErrAnnouncementNotFound)

	// A real SQL failure must roll back the recipient write, persist retry
	// diagnostics and let a later retry complete once without duplicates.
	failing := createNotice("activity", service.AnnouncementAudience{Mode: "users", UserIDs: []int64{first}})
	_, err = db.ExecContext(ctx, fmt.Sprintf(`
		ALTER TABLE announcement_deliveries ADD CONSTRAINT test_delivery_rejection CHECK (announcement_id <> %d)
	`, failing.ID))
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupDB, openErr := sql.Open("postgres", dsn)
		if openErr == nil {
			defer func() { _ = cleanupDB.Close() }()
			_, _ = cleanupDB.Exec("ALTER TABLE IF EXISTS announcement_deliveries DROP CONSTRAINT IF EXISTS test_delivery_rejection")
		}
	})
	require.NoError(t, svc.Publish(ctx, failing.ID, admin, nil))
	for i := 1; i < 5; i++ {
		_, err = db.ExecContext(ctx, `UPDATE announcement_publications SET next_attempt_at = now() WHERE announcement_id = $1`, failing.ID)
		require.NoError(t, err)
		processed, err = repo.Dispatch(ctx, failing.ID)
		require.NoError(t, err)
		require.True(t, processed)
	}
	stats, err = svc.DeliveryStats(ctx, failing.ID)
	require.NoError(t, err)
	require.Equal(t, service.AnnouncementDeliveryFailed, stats.DeliveryStatus)
	require.Equal(t, 5, stats.Attempts)
	require.Zero(t, stats.TargetCount)
	require.Equal(t, "DELIVERY_FAILED", stats.LastError)
	_, err = db.ExecContext(ctx, "ALTER TABLE announcement_deliveries DROP CONSTRAINT test_delivery_rejection")
	require.NoError(t, err)
	require.NoError(t, svc.RetryDelivery(ctx, failing.ID, admin))
	require.NoError(t, svc.RetryDelivery(ctx, failing.ID, admin))
	stats, err = svc.DeliveryStats(ctx, failing.ID)
	require.NoError(t, err)
	require.Equal(t, service.AnnouncementDeliveryPublished, stats.DeliveryStatus)
	require.Equal(t, 6, stats.Attempts)
	require.EqualValues(t, 1, stats.TargetCount)

	t.Run("standard groups", func(t *testing.T) {
		createGroup := func(exclusive bool) (int64, string) {
			t.Helper()
			var id int64
			name := "Standard " + uuid.NewString()
			require.NoError(t, db.QueryRowContext(ctx, `
				INSERT INTO groups(name, subscription_type, is_exclusive) VALUES ($1, 'standard', $2) RETURNING id
			`, name, exclusive).Scan(&id))
			return id, name
		}
		standardID, standardName := createGroup(false)
		privateID, _ := createGroup(true)
		member, direct, aggregate, expired, disabled, exhausted, restricted := createUser("user"), createUser("user"), createUser("user"), createUser("user"), createUser("user"), createUser("user"), createUser("user")
		_, err := db.ExecContext(ctx, `
			INSERT INTO user_allowed_groups(user_id, group_id) VALUES ($1, $2), ($3, $2), ($1, $4)
		`, member, standardID, muted, privateID)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `UPDATE users SET restrict_public_groups = true WHERE id = $1`, restricted)
		require.NoError(t, err)
		// A stale subscription on a group now configured for standard billing
		// must not count as standard membership.
		_, err = db.ExecContext(ctx, `
			INSERT INTO user_subscriptions (user_id, group_id, starts_at, expires_at)
			VALUES ($1, $3, now() - interval '1 day', now() + interval '1 day'),
			       ($2, $3, now() - interval '1 day', now() + interval '1 day')
		`, outsider, restricted, standardID)
		require.NoError(t, err)
		createKey := func(userID int64, groupID *int64, routes string, status string, expiresAt *time.Time, quota, used float64) {
			t.Helper()
			_, err := db.ExecContext(ctx, `
				INSERT INTO api_keys(user_id, key, name, group_id, is_aggregate, aggregate_group_ids, status, expires_at, quota, quota_used)
				VALUES ($1,$2,'test key',$3,$3::bigint IS NULL,$4::jsonb,$5,$6,$7,$8)
			`, userID, uuid.NewString(), groupID, routes, status, expiresAt, quota, used)
			require.NoError(t, err)
		}
		past := time.Now().Add(-time.Hour)
		createKey(direct, &standardID, "null", "active", nil, 0, 0)
		createKey(aggregate, nil, fmt.Sprintf("[%d,%d]", standardID, privateID), "active", nil, 0, 0)
		createKey(expired, &standardID, "null", "active", &past, 0, 0)
		createKey(disabled, &standardID, "null", "disabled", nil, 0, 0)
		createKey(exhausted, &standardID, "null", "active", nil, 1, 1)
		createKey(restricted, &standardID, "null", "active", nil, 0, 0)

		require.NoError(t, svc.SetGroupSubscription(ctx, outsider, standardID, false), "eligible users may proactively mute public group notifications")
		require.NoError(t, svc.SetGroupSubscription(ctx, outsider, standardID, true))
		require.NoError(t, svc.SetGroupSubscription(ctx, muted, standardID, false))
		require.NoError(t, svc.SetGroupSubscription(ctx, muted, standardID, false))
		require.ErrorIs(t, svc.SetGroupSubscription(ctx, restricted, standardID, false), service.ErrAnnouncementNotFound)
		require.ErrorIs(t, svc.SetGroupSubscription(ctx, aggregate, privateID, false), service.ErrAnnouncementNotFound)

		createStandardNotice := func(publication service.AnnouncementPublication) (*service.Announcement, error) {
			return svc.Create(ctx, &service.CreateAnnouncementInput{
				Title: uuid.NewString(), Content: "A standard group update", ActorID: &admin, Publication: &publication,
			})
		}
		for _, inaccessibleGroup := range []int64{groupID, 999999999} {
			_, err := createStandardNotice(service.AnnouncementPublication{
				Kind: "price_increase", GroupID: &inaccessibleGroup, Audience: service.AnnouncementAudience{Mode: "all"},
			})
			require.ErrorIs(t, err, service.ErrAnnouncementInvalidPublication)
		}
		_, err = createStandardNotice(service.AnnouncementPublication{
			Kind: "price_increase", GroupID: &standardID, PlanID: &planID, Audience: service.AnnouncementAudience{Mode: "all"},
		})
		require.ErrorIs(t, err, service.ErrAnnouncementInvalidPublication)

		increase, err := createStandardNotice(service.AnnouncementPublication{
			Kind: "price_increase", GroupID: &standardID, Audience: service.AnnouncementAudience{Mode: "groups", GroupIDs: []int64{standardID}},
		})
		require.NoError(t, err)
		metadata, err := svc.Publications(ctx, []int64{increase.ID, scheduled.ID})
		require.NoError(t, err)
		require.Equal(t, &standardID, metadata[increase.ID].GroupID)
		require.Equal(t, standardName, metadata[increase.ID].GroupName)
		require.Nil(t, metadata[scheduled.ID].GroupID)
		require.Equal(t, &planID, metadata[scheduled.ID].PlanID)

		require.NoError(t, svc.Publish(ctx, increase.ID, admin, &future))
		var legacyCandidates int
		require.NoError(t, db.QueryRowContext(ctx, `
			SELECT count(*) FROM announcements a JOIN announcement_publications p ON a.id = p.announcement_id
			WHERE a.id = $1 AND a.status = 'active' AND p.delivery_status = 'queued'
		`, increase.ID).Scan(&legacyCandidates))
		require.Zero(t, legacyCandidates, "the old dispatcher cannot publish a standard group notice")
		_, err = svc.GetForUser(ctx, direct, increase.ID)
		require.ErrorIs(t, err, service.ErrAnnouncementNotFound)
		_, err = db.ExecContext(ctx, `UPDATE announcements SET starts_at = now() WHERE id = $1`, increase.ID)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `UPDATE announcement_publications SET next_attempt_at = now() WHERE announcement_id = $1`, increase.ID)
		require.NoError(t, err)
		processed, err := repo.Dispatch(ctx, increase.ID)
		require.NoError(t, err)
		require.True(t, processed)
		stats, err := svc.DeliveryStats(ctx, increase.ID)
		require.NoError(t, err)
		require.Equal(t, service.AnnouncementDeliveryPublished, stats.DeliveryStatus)
		require.EqualValues(t, 3, stats.TargetCount)
		for _, userID := range []int64{member, direct, aggregate} {
			item, err := svc.GetForUser(ctx, userID, increase.ID)
			require.NoError(t, err)
			require.Equal(t, &standardID, item.GroupID)
			require.Equal(t, standardName, item.GroupName)
			require.True(t, item.GroupActionAvailable)
		}
		for _, userID := range []int64{outsider, muted, expired, disabled, exhausted, restricted} {
			_, err := svc.GetForUser(ctx, userID, increase.ID)
			require.ErrorIs(t, err, service.ErrAnnouncementNotFound)
			_, err = svc.RecordClick(ctx, userID, increase.ID)
			require.ErrorIs(t, err, service.ErrAnnouncementNotFound)
		}
		action, err := svc.RecordClick(ctx, direct, increase.ID)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("/available-channels?group_id=%d", standardID), action)
		otherGroup := &privateID
		_, err = svc.Update(ctx, increase.ID, &service.UpdateAnnouncementInput{GroupID: &otherGroup})
		require.ErrorIs(t, err, service.ErrAnnouncementPublicationState)

		// Even manual user targeting cannot disclose an exclusive group to
		// someone holding a stale, no-longer-authorized key route.
		decrease, err := createStandardNotice(service.AnnouncementPublication{
			Kind: "price_decrease", GroupID: &privateID, Audience: service.AnnouncementAudience{Mode: "users", UserIDs: []int64{member, aggregate}},
		})
		require.NoError(t, err)
		require.NoError(t, svc.Publish(ctx, decrease.ID, admin, nil))
		item, err := svc.GetForUser(ctx, member, decrease.ID)
		require.NoError(t, err)
		require.True(t, item.GroupActionAvailable)
		_, err = svc.GetForUser(ctx, aggregate, decrease.ID)
		require.ErrorIs(t, err, service.ErrAnnouncementNotFound)
		_, err = db.ExecContext(ctx, `DELETE FROM user_allowed_groups WHERE group_id = $1 AND user_id = $2`, privateID, member)
		require.NoError(t, err)
		item, err = svc.GetForUser(ctx, member, decrease.ID)
		require.NoError(t, err, "historical recipient keeps their published message")
		require.Empty(t, item.GroupName)
		require.False(t, item.GroupActionAvailable)
		_, err = svc.RecordClick(ctx, member, decrease.ID)
		require.ErrorIs(t, err, service.ErrAnnouncementNotFound)

		newGroup, err := createStandardNotice(service.AnnouncementPublication{
			Kind: "new_group", GroupID: &standardID, Audience: service.AnnouncementAudience{Mode: "all"},
		})
		require.NoError(t, err)
		require.NoError(t, svc.Publish(ctx, newGroup.ID, admin, nil))
		_, err = svc.GetForUser(ctx, outsider, newGroup.ID)
		require.NoError(t, err, "an eligible user without a key can discover a new public group")
		for _, userID := range []int64{muted, restricted} {
			_, err := svc.GetForUser(ctx, userID, newGroup.ID)
			require.ErrorIs(t, err, service.ErrAnnouncementNotFound)
		}

		_, err = db.ExecContext(ctx, `UPDATE groups SET status = 'disabled' WHERE id = $1`, standardID)
		require.NoError(t, err)
		delisted, err := createStandardNotice(service.AnnouncementPublication{
			Kind: "group_unlisted", GroupID: &standardID, Audience: service.AnnouncementAudience{Mode: "groups", GroupIDs: []int64{standardID}},
		})
		require.NoError(t, err)
		require.NoError(t, svc.Publish(ctx, delisted.ID, admin, nil))
		item, err = svc.GetForUser(ctx, direct, delisted.ID)
		require.NoError(t, err, "existing members get the notice after the group is disabled")
		require.False(t, item.GroupActionAvailable)
		require.ErrorIs(t, svc.SetGroupSubscription(ctx, outsider, standardID, false), service.ErrAnnouncementNotFound)

		removed, err := createStandardNotice(service.AnnouncementPublication{
			Kind: "price_decrease", GroupID: &standardID, Audience: service.AnnouncementAudience{Mode: "all"},
		})
		require.NoError(t, err)
		require.NoError(t, svc.Publish(ctx, removed.ID, admin, &future))
		_, err = db.ExecContext(ctx, `UPDATE groups SET deleted_at = now() WHERE id = $1`, standardID)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `UPDATE announcements SET starts_at = now() WHERE id = $1`, removed.ID)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `UPDATE announcement_publications SET next_attempt_at = now() WHERE announcement_id = $1`, removed.ID)
		require.NoError(t, err)
		processed, err = repo.Dispatch(ctx, removed.ID)
		require.NoError(t, err)
		require.True(t, processed)
		stats, err = svc.DeliveryStats(ctx, removed.ID)
		require.NoError(t, err)
		require.Equal(t, service.AnnouncementDeliveryFailed, stats.DeliveryStatus)
		require.Equal(t, "STANDARD_GROUP_UNAVAILABLE", stats.LastError)
		require.Zero(t, stats.TargetCount)

		rollback, err := os.ReadFile("../../scripts/rollback/256_announcement_standard_group.sql")
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, string(rollback))
		require.NoError(t, err)
		var status string
		require.NoError(t, db.QueryRowContext(ctx, "SELECT status FROM announcements WHERE id = $1", increase.ID).Scan(&status))
		require.Equal(t, service.AnnouncementStatusArchived, status)
		require.NoError(t, db.QueryRowContext(ctx, "SELECT delivery_status FROM announcement_publications WHERE announcement_id = $1", scheduled.ID).Scan(&status))
		require.Equal(t, service.AnnouncementDeliveryPublished, status, "rolling back standard groups must preserve previous plan notices")
	})

	rollback, err := os.ReadFile("../../scripts/rollback/254_announcement_delivery.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(rollback))
	require.NoError(t, err)
	var restoredStatus string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT status FROM announcements WHERE id=$1", legacy.ID).Scan(&restoredStatus))
	require.Equal(t, service.AnnouncementStatusActive, restoredStatus)
	require.NoError(t, db.QueryRowContext(ctx, "SELECT status FROM announcements WHERE id=$1", failing.ID).Scan(&restoredStatus))
	require.Equal(t, service.AnnouncementStatusArchived, restoredStatus)
	require.NoError(t, ApplyMigrations(ctx, db))
	var publications int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM announcement_publications").Scan(&publications))
	require.Zero(t, publications)
}
