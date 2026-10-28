package repository

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// Account attribution comes from failed attempts, not the eventual successful
// account. Top-level account_id is a fallback for older logs without events.
// No error percentage: usage_logs lacks successful intermediate account attempts.
const accountErrorStatsSQL = `
WITH logs AS MATERIALIZED (
 SELECT id, account_id, error_owner, error_phase, error_type, is_business_limited, status_code,
   COALESCE(NULLIF(BTRIM(request_id), ''), 'row:' || id::text) AS request_key,
   COALESCE(api_key_id, 0) AS key_id,
   CASE WHEN jsonb_typeof(upstream_errors) = 'array'
        THEN upstream_errors ELSE '[]'::jsonb END AS events
 FROM ops_error_logs e WHERE created_at >= $2 AND created_at < $3
), attributed AS (
 SELECT e.request_key, e.key_id, a.account_id
 FROM logs e
 CROSS JOIN LATERAL (
   SELECT requested.account_id
   FROM unnest($1::bigint[]) AS requested(account_id)
   WHERE EXISTS (SELECT 1 FROM jsonb_array_elements(e.events) AS event
                 WHERE event->>'account_id' = requested.account_id::text)
     OR (e.account_id = requested.account_id
         AND NOT e.is_business_limited
         AND (e.error_owner = 'provider' OR e.error_phase IN ('upstream', 'account_auth', 'network'))
         AND (e.status_code >= 400 OR NOT EXISTS (
           SELECT 1 FROM jsonb_array_elements(e.events) AS event
           WHERE COALESCE(event->>'account_id','') ~ '^[1-9][0-9]*$')))
 ) a
 GROUP BY e.request_key, e.key_id, a.account_id
), outcomes AS (
 SELECT e.request_key, e.key_id,
   BOOL_OR(COALESCE(e.status_code, 0) >= 400 OR e.error_type = 'cyber_policy') AS failed,
   BOOL_OR(e.status_code >= 200 AND e.status_code < 300) AS recovered
 FROM logs e
 WHERE EXISTS (SELECT 1 FROM attributed a WHERE a.request_key = e.request_key AND a.key_id = e.key_id)
 GROUP BY e.request_key, e.key_id
)
SELECT a.account_id, COUNT(*),
 COUNT(*) FILTER (WHERE o.recovered AND NOT o.failed),
 COUNT(*) FILTER (WHERE o.failed)
FROM attributed a JOIN outcomes o USING (request_key, key_id)
GROUP BY a.account_id`

func (r *usageLogRepository) GetAccountErrorStatsBatch(ctx context.Context, ids []int64, start, end time.Time) (map[int64]*service.AccountErrorStats, error) {
	result := make(map[int64]*service.AccountErrorStats, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := r.sql.QueryContext(ctx, accountErrorStatsSQL, pq.Array(ids), start, end)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id int64
		stats := &service.AccountErrorStats{}
		if err := rows.Scan(&id, &stats.AbnormalRequests, &stats.RecoveredRequests, &stats.FailedRequests); err != nil {
			return nil, err
		}
		result[id] = stats
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range ids {
		if result[id] == nil {
			result[id] = &service.AccountErrorStats{}
		}
	}
	return result, nil
}
