package clusterops

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Opt-in synthetic scale data lives only in a fresh, dropped schema.
func TestClusterScalePostgres(t *testing.T) {
	count, _ := strconv.Atoi(os.Getenv("SUB2API_CLUSTER_SCALE_ROWS"))
	dsn := os.Getenv("SUB2API_CLUSTER_TEST_DSN")
	if count == 0 || dsn == "" {
		t.Skip("set test DSN and scale rows for isolated query-plan acceptance")
	}
	require.GreaterOrEqual(t, count, 100000)
	require.LessOrEqual(t, count, 2000000)
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	schema := "cluster_scale_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = db.Exec(`CREATE SCHEMA ` + schema)
	require.NoError(t, err)
	defer func() { _, _ = db.Exec(`DROP SCHEMA ` + schema + ` CASCADE`) }()
	_, err = db.Exec(`SET search_path TO ` + schema)
	require.NoError(t, err)
	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "247_cluster_operations.sql"))
	require.NoError(t, err)
	_, err = db.Exec(string(migration))
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE usage_logs(id bigserial,request_id text,api_key_id bigint,actual_cost numeric(20,10),UNIQUE(request_id,api_key_id)); CREATE TABLE usage_billing_dedup(request_id text,api_key_id bigint,PRIMARY KEY(request_id,api_key_id)); CREATE TABLE usage_billing_dedup_archive(LIKE usage_billing_dedup INCLUDING ALL)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO cluster_request_journal(attempt_id,request_id,node_id,instance_id,user_id,api_key_id,route,state,started_at,http_status) SELECT md5(i::text)::uuid,'scale-'||i,'a','fixture',1,1,'/v1/responses',CASE WHEN i%1000=0 THEN 'needs_review' ELSE 'finished' END,NOW()-i*INTERVAL '1 second',200 FROM generate_series(1,$1) i`, count)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO cluster_billing_evidence(request_id,api_key_id,attempt_id,user_id,fingerprint,expected_cost,usage_snapshot) SELECT request_id,1,attempt_id,1,'fixture',0.01,'{}' FROM cluster_request_journal;INSERT INTO usage_logs(request_id,api_key_id,actual_cost) SELECT request_id,1,0.01 FROM cluster_request_journal; INSERT INTO usage_billing_dedup SELECT request_id,1 FROM cluster_request_journal; ANALYZE cluster_request_journal; ANALYZE cluster_billing_evidence; ANALYZE usage_logs; ANALYZE usage_billing_dedup; ANALYZE usage_billing_dedup_archive`)
	require.NoError(t, err)
	cfg := &config.Config{}
	cfg.Cluster.Enabled = true
	m := NewManager(db, nil, cfg, "scale")
	for _, review := range []bool{false, true} {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		rows, e := m.Requests(ctx, review, time.Time{})
		cancel()
		require.NoError(t, e)
		require.Len(t, rows, 100)
		for _, r := range rows {
			require.Equal(t, "settled", r.BillingState)
		}
		var plan string
		err = db.QueryRow(`EXPLAIN(ANALYZE,BUFFERS,FORMAT JSON) `+requestProjection+` WHERE ($1::boolean=false OR (j.reviewed_at IS NULL AND j.state='needs_review')) AND j.started_at<$2 ORDER BY j.started_at DESC,j.attempt_id LIMIT 100`, review, time.Now().Add(time.Second)).Scan(&plan)
		require.NoError(t, err)
		var parsed []map[string]any
		require.NoError(t, json.Unmarshal([]byte(plan), &parsed))
		t.Logf("rows=%d review=%v execution_ms=%v plan=%s", count, review, parsed[0]["Execution Time"], plan)
	}
}
