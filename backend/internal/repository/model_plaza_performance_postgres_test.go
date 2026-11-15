package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestModelPlazaPerformancePostgres(t *testing.T) {
	dsn := os.Getenv("SUB2API_MODEL_PERFORMANCE_TEST_DSN")
	if dsn == "" {
		t.Skip("set SUB2API_MODEL_PERFORMANCE_TEST_DSN for isolated PostgreSQL validation")
	}
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	schema := "plaza_perf_" + uuid.NewString()[:8]
	_, err = db.ExecContext(ctx, "CREATE SCHEMA "+schema)
	require.NoError(t, err)
	defer func() { _, _ = db.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE") }()
	_, err = db.ExecContext(ctx, "SET search_path TO "+schema)
	require.NoError(t, err)
	for _, migration := range []string{"194_channel_monitor_v2.sql", "199_channel_monitor_v2_fixed_rollups.sql"} {
		content, err := migrations.FS.ReadFile(migration)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, string(content))
		require.NoError(t, err)
	}
	_, err = db.ExecContext(ctx, `
		CREATE TABLE groups (id bigint PRIMARY KEY, name text, platform text);
		CREATE TABLE accounts (id bigint PRIMARY KEY, platform text);
		CREATE TABLE usage_logs (
			group_id bigint, account_id bigint, model text, requested_model text, created_at timestamptz,
			output_tokens int, duration_ms int, first_token_ms int, actual_cost numeric, request_type int
		);
		INSERT INTO groups VALUES (1,'One','openai'), (2,'Two','openai'), (3,'Three','composite');
		INSERT INTO accounts VALUES (1,'openai'), (2,'anthropic');
	`)
	require.NoError(t, err)
	start := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	through := start.Add(25 * time.Minute)
	_, err = db.ExecContext(ctx, `
		UPDATE channel_monitor_v2_watermarks
		SET usage_coverage_start=$1, error_coverage_start=$1, data_through=$2, last_successful_at=$2, backfill_cursor=$1
		WHERE id=1`, start, through)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO usage_logs VALUES
		(1,1,'mapped','Foo',$1,150,2000,1000,1,1),
		(1,1,'foo',NULL,$1,300,3000,1000,1,1),
		(1,1,'foo',NULL,$1,500,4000,NULL,1,1),
		(1,1,'foo',NULL,$1,500,4000,1000,0,1),
		(1,1,'foo',NULL,$1,500,4000,1000,1,4),
		(1,1,'foo',NULL,$2,500,4000,1000,1,1),
		(2,1,'foo',NULL,$1,500,4000,1000,1,1),
		(3,1,'foo',NULL,$1,100,2000,1000,1,1),
		(3,2,'foo',NULL,$1,500,4000,1000,1,1)
	`, start.Add(15*time.Minute), through)
	require.NoError(t, err)
	cfg := service.ChannelMonitorV2Config{
		GroupIDs:               []int64{1, 2, 3},
		Platforms:              []service.ChannelMonitorV2PlatformConfig{{Platform: "openai", Enabled: true}},
		IgnoredErrorCategories: []string{"content_policy"},
	}
	repo := &channelMonitorV2Repository{db: db}
	for _, bucket := range []time.Duration{time.Hour, 12 * time.Hour, 24 * time.Hour} {
		seconds := int(bucket.Seconds())
		_, err = db.ExecContext(ctx, `
			INSERT INTO channel_monitor_v2_metrics_rollup
			(bucket_start,bucket_seconds,platform,group_id,model,success_requests,error_requests,output_tokens,ttft_sum_ms,ttft_count,duration_sum_ms,duration_count)
			VALUES
			 ($1,$2,'openai',1,'Foo',1,2,150,1000,1,2000,1),
			 ($1,$2,'openai',1,'foo',1,2,300,1000,1,3000,1),
			 ($1,$2,'openai',2,'foo',9,0,500,1000,1,4000,1),
			 ($1,$2,'openai',3,'foo',1,1,100,1000,1,2000,1),
			 ($1,$2,'anthropic',3,'foo',1,9,500,1000,1,4000,1),
			 ($1,$2,'openai',1,'bar',6,6,500,1000,1,4000,1)`, start, seconds)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `
			INSERT INTO channel_monitor_v2_error_metrics_rollup
			(bucket_start,bucket_seconds,platform,group_id,model,error_category,taxonomy_version,error_requests)
			VALUES ($1,$2,'openai',1,'foo','content_policy',1,2)`, start, seconds)
		require.NoError(t, err)
		filter := service.ChannelMonitorV2Filter{
			Start: start, End: start.Add(bucket), Bucket: bucket, ExactModel: "foo",
			Platforms: []string{"openai"}, Models: []string{"foo"},
			RestrictGroups: true, AllowedGroupIDs: []int64{1, 3},
		}
		facts, err := repo.GetModelPerformance(ctx, filter, cfg)
		require.NoError(t, err)
		require.ElementsMatch(t, []service.PlazaPerformanceSample{
			{GroupID: 1, BucketStart: start, Successes: 2, Failures: 2, TTFTSumMs: 2000, TTFTCount: 2, LatencySumMs: 5000, LatencyCount: 2},
			{GroupID: 3, BucketStart: start, Successes: 1, Failures: 1, TTFTSumMs: 1000, TTFTCount: 1, LatencySumMs: 2000, LatencyCount: 1},
		}, facts.Buckets)
		require.Equal(t, map[int64]service.PlazaPerformanceGeneration{
			1: {OutputTokens: 450, DurationMs: 3000},
			3: {OutputTokens: 100, DurationMs: 1000},
		}, facts.Generation)
	}
}
