package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func insertRegistrationAttribution(ctx context.Context, client *dbent.Client, userID int64, source service.RegistrationAttribution) error {
	_, err := client.ExecContext(ctx, `
		INSERT INTO user_registration_attributions
		    (user_id, selected_source, source_note, utm_source, utm_medium, utm_campaign, utm_content, utm_term)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		userID, source.SelectedSource, source.SourceNote, source.UTMSource, source.UTMMedium, source.UTMCampaign, source.UTMContent, source.UTMTerm)
	if err != nil {
		return fmt.Errorf("save registration attribution: %w", err)
	}
	return nil
}

// RegistrationSourceSummary computes the counters and trend from one statement
// so that newly committed registrations cannot produce conflicting percentages.
func (r *userRepository) RegistrationSourceSummary(ctx context.Context, f service.RegistrationSourceFilter) (*service.RegistrationSourceSummary, error) {
	q, err := r.userProfileIdentitySQL(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.QueryContext(ctx, `
		WITH totals AS (
		    SELECT COUNT(*) AS all_count,
		           COUNT(*) FILTER (WHERE created_at >= $3 AND created_at < $4) AS today_count
		    FROM users WHERE deleted_at IS NULL
		), selected_range AS (
		    SELECT to_char(u.created_at AT TIME ZONE $5, 'YYYY-MM-DD') AS local_day,
		           COALESCE(NULLIF(CASE WHEN $6 = 'utm' THEN a.utm_source ELSE a.selected_source END, ''), 'unknown') AS source,
		           COALESCE(a.selected_source, '') = '' AS unreported
		    FROM users u
		    LEFT JOIN user_registration_attributions a ON a.user_id = u.id
		    WHERE u.deleted_at IS NULL AND u.created_at >= $1 AND u.created_at < $2
		), range_totals AS (
		    SELECT COUNT(*) AS range_count, COUNT(*) FILTER (WHERE unreported) AS unreported_count
		    FROM selected_range
		), daily AS (
		    SELECT local_day, source, COUNT(*) AS count
		    FROM selected_range GROUP BY local_day, source
		)
		SELECT totals.all_count, totals.today_count, range_totals.range_count,
		       range_totals.unreported_count, daily.local_day, daily.source, daily.count
		FROM totals CROSS JOIN range_totals LEFT JOIN daily ON true
		ORDER BY daily.local_day, daily.source`,
		f.Start, f.End, f.TodayStart, f.TodayEnd, f.Timezone, f.Dimension)
	if err != nil {
		return nil, fmt.Errorf("query registration source summary: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := &service.RegistrationSourceSummary{
		Channels: []service.RegistrationSourceChannel{},
		Daily:    []service.RegistrationSourceDaily{},
	}
	sources := map[string]int64{}
	for rows.Next() {
		var day, source sql.NullString
		var count sql.NullInt64
		if err := rows.Scan(&out.TotalRegistrations, &out.TodayRegistrations, &out.RangeRegistrations, &out.UnreportedCount, &day, &source, &count); err != nil {
			return nil, err
		}
		if day.Valid {
			sources[source.String] += count.Int64
			out.Daily = append(out.Daily, service.RegistrationSourceDaily{Date: day.String, Source: source.String, Count: count.Int64})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for source, count := range sources {
		out.Channels = append(out.Channels, service.RegistrationSourceChannel{
			Source: source, Count: count, Percentage: float64(count) / float64(out.RangeRegistrations) * 100,
		})
	}
	sort.Slice(out.Channels, func(i, j int) bool {
		if out.Channels[i].Count != out.Channels[j].Count {
			return out.Channels[i].Count > out.Channels[j].Count
		}
		return out.Channels[i].Source < out.Channels[j].Source
	})
	return out, nil
}

const registrationSourceUserWhere = `
	WHERE u.deleted_at IS NULL AND u.created_at >= $1 AND u.created_at < $2
	  AND ($4 = '' OR $4 = COALESCE(NULLIF(CASE WHEN $3 = 'utm' THEN a.utm_source ELSE a.selected_source END, ''), 'unknown'))`

const registrationSourceUserQuery = `
	SELECT u.id, u.email, u.username, u.created_at, u.signup_source,
	       COALESCE(a.selected_source, ''), COALESCE(a.source_note, ''),
	       COALESCE(a.utm_source, ''), COALESCE(a.utm_medium, ''),
	       COALESCE(a.utm_campaign, ''), COALESCE(a.utm_content, ''),
	       COALESCE(a.utm_term, ''), ua.inviter_id, COALESCE(promo.code, '')
	FROM users u
	LEFT JOIN user_registration_attributions a ON a.user_id = u.id
	LEFT JOIN user_affiliates ua ON ua.user_id = u.id
	LEFT JOIN LATERAL (
	    SELECT p.code
	    FROM promo_code_usages pu
	    JOIN promo_codes p ON p.id = pu.promo_code_id
	    WHERE pu.user_id = u.id
	    ORDER BY pu.used_at, pu.id
	    LIMIT 1
	) promo ON true` + registrationSourceUserWhere + `
	ORDER BY u.created_at DESC, u.id DESC`

func (r *userRepository) ListRegistrationSourceUsers(ctx context.Context, f service.RegistrationSourceFilter, params pagination.PaginationParams) ([]service.RegistrationSourceUser, *pagination.PaginationResult, error) {
	q, err := r.userProfileIdentitySQL(ctx)
	if err != nil {
		return nil, nil, err
	}
	var total int64
	if err := scanSingleRow(ctx, q, `
		SELECT COUNT(*) FROM users u
		LEFT JOIN user_registration_attributions a ON a.user_id = u.id`+registrationSourceUserWhere,
		[]any{f.Start, f.End, f.Dimension, f.Source}, &total); err != nil {
		return nil, nil, fmt.Errorf("count registration source users: %w", err)
	}
	items := make([]service.RegistrationSourceUser, 0)
	rows, err := q.QueryContext(ctx, registrationSourceUserQuery+" LIMIT $5 OFFSET $6", f.Start, f.End, f.Dimension, f.Source, params.Limit(), params.Offset())
	if err != nil {
		return nil, nil, fmt.Errorf("list registration source users: %w", err)
	}
	err = forEachRegistrationSourceUser(rows, func(row service.RegistrationSourceUser) error {
		items = append(items, row)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return items, paginationResultFromTotal(total, params), nil
}

func (r *userRepository) ExportRegistrationSourceUsers(ctx context.Context, f service.RegistrationSourceFilter, write func(service.RegistrationSourceUser) error) error {
	q, err := r.userProfileIdentitySQL(ctx)
	if err != nil {
		return err
	}
	rows, err := q.QueryContext(ctx, registrationSourceUserQuery, f.Start, f.End, f.Dimension, f.Source)
	if err != nil {
		return fmt.Errorf("export registration source users: %w", err)
	}
	return forEachRegistrationSourceUser(rows, write)
}

func forEachRegistrationSourceUser(rows *sql.Rows, write func(service.RegistrationSourceUser) error) error {
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var u service.RegistrationSourceUser
		var inviter sql.NullInt64
		if err := rows.Scan(
			&u.ID, &u.Email, &u.Username, &u.CreatedAt, &u.SignupSource,
			&u.SelectedSource, &u.SourceNote, &u.UTMSource, &u.UTMMedium, &u.UTMCampaign, &u.UTMContent, &u.UTMTerm,
			&inviter, &u.PromoCode,
		); err != nil {
			return err
		}
		if inviter.Valid {
			u.InviterID = &inviter.Int64
		}
		if err := write(u); err != nil {
			return err
		}
	}
	return rows.Err()
}
