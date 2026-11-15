package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// A managed announcement always has a no-match legacy rule. Older binaries
// may see the content row during a rolling deploy or rollback, but cannot
// treat it as a public announcement.
const unpublishedLegacyTargeting = `{"any_of":[{}]}`

const managedAnnouncementColumns = `a.id, a.title, a.content, a.status, a.notify_mode,
	a.starts_at, a.ends_at, a.created_by, a.updated_by, a.created_at, a.updated_at`

// Equivalent to User.CanBindGroup for a standard group. A live key is not
// itself an entitlement to keep seeing an exclusive/restricted group.
const standardGroupExplicitMember = `EXISTS (
	SELECT 1 FROM user_allowed_groups ug WHERE ug.user_id = u.id AND ug.group_id = g.id
)`

const standardGroupVisibleToUser = `((NOT g.is_exclusive AND NOT u.restrict_public_groups) OR ` + standardGroupExplicitMember + `)`

const managedAnnouncementTargetColumns = `r.read_at, p.kind, p.plan_id, p.standard_group_id,
	COALESCE(g.name, ''), COALESCE(g.status = 'active', false)`

const visibleManagedAnnouncements = `FROM announcement_deliveries d
	JOIN announcement_publications p USING (announcement_id)
	JOIN announcements a ON a.id = d.announcement_id
	JOIN users u ON u.id = d.user_id AND u.deleted_at IS NULL
	LEFT JOIN groups g ON g.id = p.standard_group_id AND g.deleted_at IS NULL
		AND g.subscription_type = 'standard' AND ` + standardGroupVisibleToUser + `
	LEFT JOIN announcement_reads r ON r.announcement_id = d.announcement_id AND r.user_id = d.user_id
	WHERE d.user_id = $1 AND p.delivery_status = 'published'
	  AND a.status = 'active' AND p.published_at <= $2`

type announcementDeliveryRepository struct {
	db *sql.DB
}

func NewAnnouncementDeliveryRepository(db *sql.DB) service.AnnouncementDeliveryRepository {
	return &announcementDeliveryRepository{db: db}
}

func (r *announcementDeliveryRepository) Create(ctx context.Context, a *service.Announcement, publication service.AnnouncementPublication) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := validateAnnouncementAudienceReferences(ctx, tx, publication); err != nil {
		return err
	}
	audience, err := json.Marshal(publication.Audience)
	if err != nil {
		return err
	}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO announcements(title, content, status, notify_mode, targeting, starts_at, ends_at, created_by, updated_by)
		VALUES ($1, $2, 'draft', $3, $4::jsonb, $5, $6, $7, $7)
		RETURNING id, created_at, updated_at
	`, a.Title, a.Content, a.NotifyMode, unpublishedLegacyTargeting, a.StartsAt, a.EndsAt, a.CreatedBy).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO announcement_publications(announcement_id, kind, plan_id, group_id, audience, standard_group_id)
		VALUES ($1, $2, $3, (SELECT group_id FROM subscription_plans WHERE id = $3), $4::jsonb, $5)
	`, a.ID, publication.Kind, publication.PlanID, string(audience), publication.GroupID); err != nil {
		return err
	}
	return tx.Commit()
}

func validateAnnouncementAudienceReferences(ctx context.Context, tx *sql.Tx, publication service.AnnouncementPublication) error {
	var count int
	if publication.PlanID != nil {
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM subscription_plans WHERE id = $1`, *publication.PlanID).Scan(&count); err != nil {
			return err
		}
		if count != 1 {
			return service.ErrAnnouncementInvalidPublication
		}
	}
	if publication.GroupID != nil {
		exists, err := standardAnnouncementGroupExists(ctx, tx, *publication.GroupID)
		if err != nil {
			return err
		}
		if !exists {
			return service.ErrAnnouncementInvalidPublication
		}
	}
	if ids := publication.Audience.GroupIDs; len(ids) > 0 {
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM groups WHERE id = ANY($1) AND deleted_at IS NULL`, pq.Array(ids)).Scan(&count); err != nil {
			return err
		}
		if count != len(ids) {
			return service.ErrAnnouncementInvalidPublication
		}
	}
	if ids := publication.Audience.UserIDs; len(ids) > 0 {
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM users WHERE id = ANY($1) AND deleted_at IS NULL`, pq.Array(ids)).Scan(&count); err != nil {
			return err
		}
		if count != len(ids) {
			return service.ErrAnnouncementInvalidPublication
		}
	}
	return nil
}

func standardAnnouncementGroupExists(ctx context.Context, tx *sql.Tx, id int64) (bool, error) {
	var exists bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM groups WHERE id = $1 AND deleted_at IS NULL AND subscription_type = 'standard'
	)`, id).Scan(&exists)
	return exists, err
}

func (r *announcementDeliveryRepository) Update(ctx context.Context, a *service.Announcement, publication service.AnnouncementPublication) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	current, updatedAt, _, err := lockAnnouncementPublication(ctx, tx, a.ID)
	if err != nil {
		return err
	}
	if !a.UpdatedAt.Equal(updatedAt) || current.DeliveryStatus == service.AnnouncementDeliveryWithdrawn {
		return service.ErrAnnouncementPublicationState
	}
	if current.DeliveryStatus == service.AnnouncementDeliveryPublished {
		// Existing recipients and destination remain stable after publication.
		if current.Kind != publication.Kind || !reflect.DeepEqual(current.PlanID, publication.PlanID) ||
			!reflect.DeepEqual(current.GroupID, publication.GroupID) || !reflect.DeepEqual(current.Audience, publication.Audience) {
			return service.ErrAnnouncementPublicationState
		}
	} else if err := validateAnnouncementAudienceReferences(ctx, tx, publication); err != nil {
		return err
	}
	audience, err := json.Marshal(publication.Audience)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE announcement_publications SET kind = $2, plan_id = $3, audience = $4::jsonb, standard_group_id = $5,
		    group_id = CASE WHEN plan_id IS DISTINCT FROM $3 OR $5::bigint IS NOT NULL
		        THEN (SELECT group_id FROM subscription_plans WHERE id = $3) ELSE group_id END
		WHERE announcement_id = $1
	`, a.ID, publication.Kind, publication.PlanID, string(audience), publication.GroupID); err != nil {
		return err
	}
	// Queued standard group notices stay hidden from old workers. Handle an
	// edit that changes the destination type after scheduling as well.
	if current.DeliveryStatus == service.AnnouncementDeliveryQueued && (current.GroupID == nil) != (publication.GroupID == nil) {
		status := service.AnnouncementStatusActive
		if publication.GroupID != nil {
			status = service.AnnouncementStatusDraft
		}
		if _, err := tx.ExecContext(ctx, `UPDATE announcements SET status = $2 WHERE id = $1`, a.ID, status); err != nil {
			return err
		}
		a.Status = status
	}
	err = tx.QueryRowContext(ctx, `
		UPDATE announcements
		SET title = $2, content = $3, notify_mode = $4, ends_at = $5, updated_by = $6, updated_at = clock_timestamp()
		WHERE id = $1 RETURNING updated_at
	`, a.ID, a.Title, a.Content, a.NotifyMode, a.EndsAt, a.UpdatedBy).Scan(&a.UpdatedAt)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func lockAnnouncementPublication(ctx context.Context, tx *sql.Tx, id int64) (service.AnnouncementPublication, time.Time, *time.Time, error) {
	var publication service.AnnouncementPublication
	var updatedAt time.Time
	var endsAt *time.Time
	var audience []byte
	err := tx.QueryRowContext(ctx, `
		SELECT p.kind, p.plan_id, p.standard_group_id, p.audience, p.delivery_status, a.updated_at, a.ends_at
		FROM announcements a JOIN announcement_publications p ON p.announcement_id = a.id
		WHERE a.id = $1 FOR UPDATE OF a, p
	`, id).Scan(&publication.Kind, &publication.PlanID, &publication.GroupID, &audience, &publication.DeliveryStatus, &updatedAt, &endsAt)
	if errors.Is(err, sql.ErrNoRows) {
		err = service.ErrAnnouncementNotFound
	}
	if err == nil {
		err = json.Unmarshal(audience, &publication.Audience)
	}
	return publication, updatedAt, endsAt, err
}

func (r *announcementDeliveryRepository) Publications(ctx context.Context, ids []int64) (map[int64]service.AnnouncementPublication, error) {
	out := make(map[int64]service.AnnouncementPublication)
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.announcement_id, p.kind, p.plan_id, p.standard_group_id, COALESCE(g.name, ''), p.audience, p.delivery_status
		FROM announcement_publications p LEFT JOIN groups g ON g.id = p.standard_group_id
		WHERE p.announcement_id = ANY($1)
	`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id int64
		var p service.AnnouncementPublication
		var audience []byte
		if err := rows.Scan(&id, &p.Kind, &p.PlanID, &p.GroupID, &p.GroupName, &audience, &p.DeliveryStatus); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(audience, &p.Audience); err != nil {
			return nil, err
		}
		out[id] = p
	}
	return out, rows.Err()
}

func (r *announcementDeliveryRepository) Publish(ctx context.Context, id, actorID int64, at time.Time) error {
	return r.changeState(ctx, id, actorID, service.AnnouncementDeliveryQueued, at)
}

func (r *announcementDeliveryRepository) Retry(ctx context.Context, id, actorID int64) error {
	return r.changeState(ctx, id, actorID, service.AnnouncementDeliveryQueued, time.Time{})
}

func (r *announcementDeliveryRepository) Withdraw(ctx context.Context, id, actorID int64) error {
	return r.changeState(ctx, id, actorID, service.AnnouncementDeliveryWithdrawn, time.Time{})
}

func (r *announcementDeliveryRepository) changeState(ctx context.Context, id, actorID int64, state string, at time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	current, _, endsAt, err := lockAnnouncementPublication(ctx, tx, id)
	if err != nil {
		return err
	}
	retrying := at.IsZero()
	if state == service.AnnouncementDeliveryQueued {
		if current.DeliveryStatus == service.AnnouncementDeliveryPublished || current.DeliveryStatus == service.AnnouncementDeliveryQueued {
			return tx.Commit()
		}
		if retrying {
			if current.DeliveryStatus != service.AnnouncementDeliveryFailed {
				return service.ErrAnnouncementPublicationState
			}
			at = time.Now()
		} else if current.DeliveryStatus != service.AnnouncementDeliveryDraft {
			return service.ErrAnnouncementPublicationState
		}
		if endsAt != nil && !at.Before(*endsAt) {
			return service.ErrAnnouncementInvalidSchedule
		}
		status := service.AnnouncementStatusActive
		if current.GroupID != nil {
			exists, err := standardAnnouncementGroupExists(ctx, tx, *current.GroupID)
			if err != nil {
				return err
			}
			if !exists {
				return service.ErrAnnouncementInvalidPublication
			}
			status = service.AnnouncementStatusDraft
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE announcements SET status = $4, starts_at = $2, updated_by = $3, updated_at = clock_timestamp()
			WHERE id = $1
		`, id, at, actorID, status); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE announcement_publications
			SET delivery_status = 'queued', consecutive_failures = 0, next_attempt_at = $2, published_by = $3
			WHERE announcement_id = $1
		`, id, at, actorID); err != nil {
			return err
		}
	} else {
		if current.DeliveryStatus == service.AnnouncementDeliveryWithdrawn {
			return tx.Commit()
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE announcements SET status = 'archived', updated_by = $2, updated_at = clock_timestamp()
			WHERE id = $1
		`, id, actorID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE announcement_publications SET delivery_status = 'withdrawn', next_attempt_at = NULL
			WHERE announcement_id = $1
		`, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *announcementDeliveryRepository) Dispatch(ctx context.Context, id int64) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	var audienceJSON []byte
	var attempts, failures int
	var announcementID int64
	var now time.Time
	var endsAt *time.Time
	var preferenceGroupID, standardGroupID *int64
	// Guard every dispatcher with the same row lock. No external calls are
	// made inside this transaction, and a crash rolls back publication.
	err = tx.QueryRowContext(ctx, `
		SELECT p.announcement_id, p.audience, p.attempts, p.consecutive_failures, a.ends_at, now(),
		    CASE WHEN p.kind IN ('price_increase', 'price_decrease', 'new_plan', 'plan_unlisted', 'new_group', 'group_unlisted')
		        THEN COALESCE(p.standard_group_id, p.group_id) END, p.standard_group_id
		FROM announcement_publications p JOIN announcements a ON a.id = p.announcement_id
		WHERE p.delivery_status = 'queued' AND a.starts_at <= now()
			AND ((p.standard_group_id IS NULL AND a.status = 'active') OR (p.standard_group_id IS NOT NULL AND a.status = 'draft'))
			AND p.next_attempt_at <= now() AND ($1::bigint = 0 OR p.announcement_id = $1)
		ORDER BY p.next_attempt_at, p.announcement_id LIMIT 1 FOR UPDATE OF a, p SKIP LOCKED
	`, id).Scan(&announcementID, &audienceJSON, &attempts, &failures, &endsAt, &now, &preferenceGroupID, &standardGroupID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `SAVEPOINT announcement_delivery`); err != nil {
		return false, err
	}
	attempts++
	reason := "DELIVERY_FAILED"
	var audience service.AnnouncementAudience
	forceTerminal := false
	switch {
	case endsAt != nil && !now.Before(*endsAt):
		reason, forceTerminal = "SCHEDULE_EXPIRED", true
		err = service.ErrAnnouncementInvalidSchedule
	default:
		err = json.Unmarshal(audienceJSON, &audience)
		if err == nil && standardGroupID != nil {
			var exists bool
			exists, err = standardAnnouncementGroupExists(ctx, tx, *standardGroupID)
			if err == nil && !exists {
				reason, forceTerminal = "STANDARD_GROUP_UNAVAILABLE", true
				err = service.ErrAnnouncementInvalidPublication
			}
		}
		if err == nil {
			err = snapshotAnnouncementDeliveries(ctx, tx, announcementID, audience, preferenceGroupID, standardGroupID, now)
		}
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, `
			UPDATE announcement_publications
			SET delivery_status = 'published', published_at = $2, next_attempt_at = NULL, attempts = $3,
			    consecutive_failures = 0, last_error = ''
			WHERE announcement_id = $1
		`, announcementID, now, attempts)
		if err == nil {
			_, err = tx.ExecContext(ctx, `UPDATE announcements SET status = 'active', updated_at = clock_timestamp() WHERE id = $1`, announcementID)
		}
	}
	if err != nil {
		if _, rollbackErr := tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT announcement_delivery`); rollbackErr != nil {
			return false, fmt.Errorf("announcement delivery rollback: %w", rollbackErr)
		}
		failures++
		state := service.AnnouncementDeliveryQueued
		delay := now.Add(30 * time.Second * time.Duration(1<<uint(min(failures-1, 4))))
		next := &delay
		if forceTerminal || failures >= 5 {
			state = service.AnnouncementDeliveryFailed
			next = nil
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE announcement_publications
			SET delivery_status = $2, next_attempt_at = $3, attempts = $4, consecutive_failures = $5, last_error = $6
			WHERE announcement_id = $1
		`, announcementID, state, next, attempts, failures, reason); err != nil {
			return false, err
		}
	}
	return true, tx.Commit()
}

func snapshotAnnouncementDeliveries(ctx context.Context, tx *sql.Tx, id int64, audience service.AnnouncementAudience, preferenceGroupID, standardGroupID *int64, now time.Time) error {
	query := `
		INSERT INTO announcement_deliveries(announcement_id, user_id, delivered_at)
		SELECT $1, u.id, $2 FROM users u WHERE u.deleted_at IS NULL
		  AND ($3::bigint IS NULL OR NOT EXISTS (
		      SELECT 1 FROM announcement_group_mutes m WHERE m.user_id = u.id AND m.group_id = $3
		  ))
		  AND ($4::bigint IS NULL OR EXISTS (
		      SELECT 1 FROM groups g WHERE g.id = $4 AND g.deleted_at IS NULL
		      AND g.subscription_type = 'standard' AND ` + standardGroupVisibleToUser + `
		  ))`
	args := []any{id, now, preferenceGroupID, standardGroupID}
	switch audience.Mode {
	case "all":
	case "users":
		query += ` AND u.id = ANY($5)`
		args = append(args, pq.Array(audience.UserIDs))
	case "groups":
		// Public visibility is not membership. Include explicit assignments
		// and usable keys, including directly configured aggregate routes.
		query += ` AND (EXISTS (
			SELECT 1 FROM user_subscriptions s
			JOIN groups sg ON sg.id = s.group_id AND sg.subscription_type = 'subscription'
			WHERE s.user_id = u.id AND s.group_id = ANY($5)
			AND s.deleted_at IS NULL AND s.status = 'active' AND s.starts_at <= $2 AND s.expires_at > $2
		) OR EXISTS (
			SELECT 1 FROM groups g WHERE g.id = ANY($5) AND g.deleted_at IS NULL
			AND g.subscription_type = 'standard' AND ` + standardGroupVisibleToUser + `
			AND (` + standardGroupExplicitMember + ` OR (u.status = 'active' AND EXISTS (
			    SELECT 1 FROM api_keys k WHERE k.user_id = u.id AND k.deleted_at IS NULL AND k.status = 'active'
			    AND (k.expires_at IS NULL OR k.expires_at > $2) AND (k.quota = 0 OR k.quota_used < k.quota)
			    AND (k.group_id = g.id OR (k.is_aggregate AND k.aggregate_group_ids @> jsonb_build_array(g.id)))
			)))
		))`
		args = append(args, pq.Array(audience.GroupIDs))
	default:
		return service.ErrAnnouncementInvalidPublication
	}
	_, err := tx.ExecContext(ctx, query+" ON CONFLICT (announcement_id, user_id) DO NOTHING", args...)
	return err
}

func (r *announcementDeliveryRepository) Stats(ctx context.Context, id int64) (service.AnnouncementDeliveryStats, error) {
	var stats service.AnnouncementDeliveryStats
	err := r.db.QueryRowContext(ctx, `
		SELECT p.delivery_status, p.attempts, p.last_error, p.published_at, p.next_attempt_at,
		       counts.targets, counts.reads, counts.targets-counts.reads, counts.clicks
		FROM announcement_publications p
		CROSS JOIN LATERAL (
			SELECT count(*) AS targets, count(r.read_at) AS reads, coalesce(sum(d.click_count), 0)::bigint AS clicks
			FROM announcement_deliveries d
			LEFT JOIN announcement_reads r USING (announcement_id, user_id)
			WHERE d.announcement_id = p.announcement_id
		) counts
		WHERE p.announcement_id = $1
	`, id).Scan(&stats.DeliveryStatus, &stats.Attempts, &stats.LastError, &stats.PublishedAt, &stats.NextAttemptAt,
		&stats.TargetCount, &stats.ReadCount, &stats.UnreadCount, &stats.ClickCount)
	if errors.Is(err, sql.ErrNoRows) {
		err = service.ErrAnnouncementNotFound
	}
	return stats, err
}

func (r *announcementDeliveryRepository) Deliveries(ctx context.Context, id int64, params pagination.PaginationParams) ([]service.AnnouncementDelivery, int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx, `
		SELECT (SELECT count(*) FROM announcement_deliveries WHERE announcement_id = $1)
		FROM announcement_publications WHERE announcement_id = $1
	`, id).Scan(&total)
	if errors.Is(err, sql.ErrNoRows) {
		err = service.ErrAnnouncementNotFound
	}
	if err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.user_id, u.email, u.username, d.delivered_at, r.read_at, d.click_count
		FROM announcement_deliveries d JOIN users u ON u.id = d.user_id
		LEFT JOIN announcement_reads r USING (announcement_id, user_id)
		WHERE d.announcement_id = $1 ORDER BY d.user_id
		LIMIT $2 OFFSET $3
	`, id, params.Limit(), params.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.AnnouncementDelivery, 0, params.Limit())
	for rows.Next() {
		var item service.AnnouncementDelivery
		if err := rows.Scan(&item.UserID, &item.Email, &item.Username, &item.DeliveredAt, &item.ReadAt, &item.ClickCount); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *announcementDeliveryRepository) ListForUser(ctx context.Context, userID int64, now time.Time, activeOnly, unreadOnly bool, offset, limit int) ([]service.UserAnnouncement, int64, int64, error) {
	where := visibleManagedAnnouncements
	if activeOnly {
		where += ` AND (a.ends_at IS NULL OR a.ends_at > $2)`
	}
	var total, unread int64
	if err := r.db.QueryRowContext(ctx, `SELECT count(*), count(*) FILTER (WHERE r.read_at IS NULL) `+where, userID, now).Scan(&total, &unread); err != nil {
		return nil, 0, 0, err
	}
	if unreadOnly {
		where += ` AND r.read_at IS NULL`
		total = unread
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+managedAnnouncementColumns+`, `+managedAnnouncementTargetColumns+` `+where+
		` ORDER BY (r.read_at IS NULL) DESC, a.id DESC LIMIT $3 OFFSET $4`, userID, now, limit, offset)
	if err != nil {
		return nil, 0, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.UserAnnouncement, 0, limit)
	for rows.Next() {
		item, err := scanManagedUserAnnouncement(rows)
		if err != nil {
			return nil, 0, 0, err
		}
		items = append(items, *item)
	}
	return items, total, unread, rows.Err()
}

func (r *announcementDeliveryRepository) GetForUser(ctx context.Context, userID, id int64, now time.Time) (*service.UserAnnouncement, error) {
	item, err := scanManagedUserAnnouncement(r.db.QueryRowContext(ctx,
		`SELECT `+managedAnnouncementColumns+`, `+managedAnnouncementTargetColumns+` `+visibleManagedAnnouncements+` AND a.id = $3`,
		userID, now, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrAnnouncementNotFound
	}
	return item, err
}

func scanManagedUserAnnouncement(row interface{ Scan(...any) error }) (*service.UserAnnouncement, error) {
	var item service.UserAnnouncement
	a := &item.Announcement
	err := row.Scan(&a.ID, &a.Title, &a.Content, &a.Status, &a.NotifyMode, &a.StartsAt, &a.EndsAt, &a.CreatedBy, &a.UpdatedBy,
		&a.CreatedAt, &a.UpdatedAt, &item.ReadAt, &item.Kind, &item.PlanID, &item.GroupID, &item.GroupName, &item.GroupActionAvailable)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *announcementDeliveryRepository) MarkAllRead(ctx context.Context, userID int64, legacyIDs []int64, now time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO announcement_reads(announcement_id, user_id, read_at)
		SELECT a.id, $1, $2 `+visibleManagedAnnouncements+` AND r.read_at IS NULL
		UNION
		SELECT a.id, $1, $2 FROM announcements a
		WHERE a.id = ANY($3) AND a.status = 'active'
		  AND (a.starts_at IS NULL OR a.starts_at <= $2) AND (a.ends_at IS NULL OR a.ends_at > $2)
		  AND NOT EXISTS (SELECT 1 FROM announcement_publications p WHERE p.announcement_id = a.id)
		ON CONFLICT (announcement_id, user_id) DO NOTHING`, userID, now, pq.Array(legacyIDs))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *announcementDeliveryRepository) Click(ctx context.Context, userID, id int64, now time.Time) (service.AnnouncementActionTarget, error) {
	var target service.AnnouncementActionTarget
	err := r.db.QueryRowContext(ctx, `
		UPDATE announcement_deliveries d SET click_count = d.click_count + 1
		FROM announcement_publications p JOIN announcements a ON a.id = p.announcement_id
		JOIN users u ON u.id = $1 AND u.deleted_at IS NULL
		LEFT JOIN groups g ON g.id = p.standard_group_id AND g.deleted_at IS NULL AND g.status = 'active'
		    AND g.subscription_type = 'standard' AND `+standardGroupVisibleToUser+`
		WHERE d.announcement_id = p.announcement_id AND d.user_id = $1 AND a.id = $2
		  AND a.status = 'active' AND p.delivery_status = 'published' AND p.published_at <= $3
		  AND (p.plan_id IS NOT NULL OR g.id IS NOT NULL) RETURNING p.plan_id, g.id
	`, userID, id, now).Scan(&target.PlanID, &target.GroupID)
	if errors.Is(err, sql.ErrNoRows) {
		err = service.ErrAnnouncementNotFound
	}
	return target, err
}

func (r *announcementDeliveryRepository) MutedGroups(ctx context.Context, userID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT group_id FROM announcement_group_mutes WHERE user_id = $1 ORDER BY group_id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	result := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

func (r *announcementDeliveryRepository) SetGroupMute(ctx context.Context, userID, groupID int64, mute bool) error {
	if !mute {
		_, err := r.db.ExecContext(ctx, `DELETE FROM announcement_group_mutes WHERE user_id = $1 AND group_id = $2`, userID, groupID)
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO announcement_group_mutes(user_id, group_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, userID, groupID)
	return err
}

func (r *announcementDeliveryRepository) MuteStandardGroupIfAvailable(ctx context.Context, userID, groupID int64) (bool, error) {
	var standard, allowed bool
	err := r.db.QueryRowContext(ctx, `
		WITH target AS (
		    SELECT id, deleted_at, status, is_exclusive FROM groups WHERE id = $2 AND subscription_type = 'standard'
		), allowed AS (
		    SELECT g.id FROM target g JOIN users u ON u.id = $1 AND u.deleted_at IS NULL
		    WHERE g.deleted_at IS NULL AND g.status = 'active'
		      AND `+standardGroupVisibleToUser+`
		), added AS (
		    INSERT INTO announcement_group_mutes(user_id, group_id)
		    SELECT $1, id FROM allowed ON CONFLICT DO NOTHING
		)
		SELECT EXISTS (SELECT 1 FROM target), EXISTS (SELECT 1 FROM allowed)
	`, userID, groupID).Scan(&standard, &allowed)
	if err == nil && standard && !allowed {
		err = service.ErrAnnouncementNotFound
	}
	return standard, err
}
