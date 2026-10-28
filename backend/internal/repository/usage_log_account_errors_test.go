package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// All fixtures are transaction-local temporary tables; application data is untouched.
func TestAccountMetricsPostgres(t *testing.T) {
	dsn := os.Getenv("ACCOUNT_METRICS_TEST_DSN")
	if dsn == "" {
		t.Skip("set ACCOUNT_METRICS_TEST_DSN for real SQL verification")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	tx, err := db.Begin()
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	_, err = tx.Exec(`CREATE TEMP TABLE ops_error_logs (
 id bigserial, created_at timestamptz DEFAULT now(), request_id text,
 api_key_id bigint DEFAULT 1, account_id bigint, error_owner text DEFAULT 'provider',
 error_phase text DEFAULT 'upstream', error_type text DEFAULT 'upstream_error',
 is_business_limited boolean DEFAULT false, status_code int, upstream_errors jsonb
 ) ON COMMIT DROP;
 INSERT INTO ops_error_logs(request_id, account_id, status_code, upstream_errors) VALUES
 ('recovered',2,200,'[{"account_id":1},{"account_id":1}]'),
 ('recovered',2,200,'[{"account_id":1}]'),
 ('failed',2,502,'[{"account_id":1},{"account_id":2}]'),
 ('legacy',1,500,NULL),
 ('conflict',1,200,'[{"account_id":1}]'),
 ('conflict',1,500,'[{"account_id":1}]'),
 ('malformed',1,500,'{}'),
 ('bad-event',1,500,'[{"account_id":"9999999999999999999999999"}]'),
 ('no-id',1,200,'[{"account_id":0}]');
 INSERT INTO ops_error_logs(request_id,account_id,status_code,error_owner,error_phase,is_business_limited)
 VALUES ('business',1,429,'client','request',true),('client',1,400,'client','request',false);
 INSERT INTO ops_error_logs(request_id,account_id,status_code,created_at) VALUES ('old',1,500,now()-interval '2 day');
 `)
	require.NoError(t, err)
	repo := &usageLogRepository{sql: tx}
	stats, err := repo.GetAccountErrorStatsBatch(context.Background(), []int64{1, 2, 3}, time.Now().Add(-time.Hour), time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.EqualValues(t, 7, stats[1].AbnormalRequests)
	require.EqualValues(t, 2, stats[1].RecoveredRequests)
	require.EqualValues(t, 5, stats[1].FailedRequests)
	require.EqualValues(t, 1, stats[2].AbnormalRequests, "successful destination is not blamed for another account's errors")
	require.EqualValues(t, 1, stats[2].FailedRequests)
	require.Zero(t, stats[3].AbnormalRequests)
	_, err = tx.Exec(`CREATE TEMP TABLE usage_logs (
 account_id bigint, created_at timestamptz DEFAULT now(), input_tokens bigint DEFAULT 0,
 output_tokens bigint DEFAULT 0, cache_read_tokens bigint DEFAULT 0, cache_creation_tokens bigint DEFAULT 0,
 account_stats_cost numeric, total_cost numeric DEFAULT 0, actual_cost numeric DEFAULT 0,
 account_rate_multiplier numeric DEFAULT 1, billing_mode text, image_count int DEFAULT 0, first_token_ms int
 ) ON COMMIT DROP;
 INSERT INTO usage_logs(account_id,input_tokens,cache_read_tokens,cache_creation_tokens,billing_mode,image_count,first_token_ms)
 VALUES (1,60,924,16,'token',0,7400),(1,500,500,0,'token',0,2600),
 (1,1,99999,0,NULL,1,NULL),(1,1,99999,0,'image',1,0),
 (1,0,99999,0,'per_request',0,NULL),(1,0,99999,0,'video',0,NULL);
 `)
	require.NoError(t, err)
	usage, err := repo.GetAccountWindowStatsBatch(context.Background(), []int64{1}, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.EqualValues(t, 1424, usage[1].CacheReadTokens)
	require.EqualValues(t, 2000, usage[1].CacheInputTokens)
	require.Equal(t, 5000.0, *usage[1].AverageFirstTokenMs)
}
