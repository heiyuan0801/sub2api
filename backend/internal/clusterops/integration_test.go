package clusterops

import (
	"context"
	"database/sql"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Explicit local test database only. Tests create their own schema and never
// mutate a deployed schema or depend on production credentials.
func TestClusterOperationsPostgres(t *testing.T) {
	dsn := os.Getenv("SUB2API_CLUSTER_TEST_DSN")
	if dsn == "" {
		t.Skip("set SUB2API_CLUSTER_TEST_DSN for local PostgreSQL integration")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	schema := "cluster_test_" + uuid.NewString()[:8]
	_, err = db.Exec(`CREATE SCHEMA ` + schema)
	require.NoError(t, err)
	defer func() { _, _ = db.Exec(`DROP SCHEMA ` + schema + ` CASCADE`) }()
	_, err = db.Exec(`SET search_path TO ` + schema)
	require.NoError(t, err)
	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "247_cluster_operations.sql"))
	require.NoError(t, err)
	_, err = db.Exec(string(migration))
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE settings(key text PRIMARY KEY,value text,updated_at timestamptz);CREATE TABLE users(id bigint primary key,balance numeric(20,10),deleted_at timestamptz);CREATE TABLE usage_billing_dedup(request_id text,api_key_id bigint);CREATE TABLE usage_billing_dedup_archive(request_id text,api_key_id bigint);CREATE TABLE usage_logs(id bigserial,request_id text,api_key_id bigint,actual_cost numeric(20,10));`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO users VALUES(1,100,NULL)`)
	require.NoError(t, err)
	cfg := &config.Config{}
	cfg.JWT.Secret = "test-shared-jwt"
	cfg.Totp.EncryptionKey = "test-shared-totp"
	cfg.Cluster.Enabled = true
	cfg.Cluster.NodeID = "a"
	cfg.Cluster.Role = "primary"
	m := NewManager(db, nil, cfg, "test")
	ctx := context.Background()
	_, err = db.Exec(`INSERT INTO cluster_nodes(node_id,instance_id,role,snapshot) VALUES('a','a','primary','{"database_ok":true,"redis_ok":true,"lease_ok":true,"runtime_config_ok":true}'),('b','b','worker','{"database_ok":true,"redis_ok":true,"lease_ok":true,"runtime_config_ok":true}');INSERT INTO cluster_node_controls(node_id) VALUES('a'),('b')`)
	require.NoError(t, err)
	c := Control{Maintenance: true, MaxConcurrency: 50, Weight: 20, Revision: 1, Reason: "test drain"}
	require.ErrorIs(t, m.UpdateControl(ctx, "a", c, 1), ErrConflict)
	cfg.Cluster.HAEnabled = true
	require.NoError(t, m.UpdateControl(ctx, "a", c, 1))
	require.ErrorIs(t, m.UpdateControl(ctx, "b", c, 1), ErrLastNode)
	c.Maintenance = false
	require.ErrorIs(t, m.UpdateControl(ctx, "a", c, 1), ErrConflict)
	c.Revision = 2
	require.NoError(t, m.UpdateControl(ctx, "a", c, 1))
	cfg.Cluster.HAEnabled = false
	attempt := uuid.NewString()
	_, err = db.Exec(`INSERT INTO cluster_request_journal(attempt_id,request_id,node_id,instance_id,user_id,api_key_id,route,started_at) VALUES($1,'request','a','dead',1,2,'/v1/responses',NOW()-INTERVAL '2 minutes')`, attempt)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO cluster_image_tasks(task_id,node_id,instance_id,status,payload,expires_at,updated_at) VALUES('image','a','dead','processing','{"status":"processing"}',NOW()+INTERVAL '1 day',NOW()-INTERVAL '2 minutes')`)
	require.NoError(t, err)
	require.NoError(t, m.Reconcile(ctx))
	// A recovered dependency must clear its active alert and persist a recovery
	// event in the same transaction; otherwise all subsequent reconciliation
	// work rolls back and the dashboard keeps stale warnings indefinitely.
	var alerts int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM cluster_alerts WHERE active`).Scan(&alerts))
	require.Equal(t, 2, alerts)
	_, err = db.Exec(`UPDATE cluster_nodes SET snapshot=snapshot || '{"redis_link_ok":true}'::jsonb`)
	require.NoError(t, err)
	require.NoError(t, m.Reconcile(ctx))
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM cluster_alerts WHERE active`).Scan(&alerts))
	require.Zero(t, alerts)
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM cluster_events WHERE kind='alert_resolved' AND message IN ('a:redis','b:redis')`).Scan(&alerts))
	require.Equal(t, 2, alerts)
	requests, err := m.Requests(ctx, true, time.Time{})
	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.Equal(t, "unknown", requests[0].BillingState)
	var status string
	require.NoError(t, db.QueryRow(`SELECT payload->>'status' FROM cluster_image_tasks WHERE task_id='image'`).Scan(&status))
	require.Equal(t, "failed", status)
	_, err = db.Exec(`INSERT INTO cluster_billing_evidence(request_id,api_key_id,attempt_id,user_id,fingerprint,expected_cost,usage_snapshot) VALUES('request',2,$1,1,'fingerprint',0.5,'{}')`, attempt)
	require.NoError(t, err)
	requests, err = m.Requests(ctx, true, time.Time{})
	require.NoError(t, err)
	require.Equal(t, "settlement_pending", requests[0].BillingState)
	_, err = db.Exec(`INSERT INTO usage_billing_dedup VALUES('request',2);INSERT INTO usage_logs(request_id,api_key_id,actual_cost) VALUES('request',2,0)`)
	require.NoError(t, err)
	requests, err = m.Requests(ctx, true, time.Time{})
	require.NoError(t, err)
	require.Equal(t, "cost_mismatch", requests[0].BillingState)
	require.NoError(t, m.ReviewRequest(ctx, attempt, strings.Repeat("x", 1000), 1))
	requests, err = m.Requests(ctx, true, time.Time{})
	require.NoError(t, err)
	require.Empty(t, requests)
	var balance string
	require.NoError(t, db.QueryRow(`SELECT balance::text FROM users WHERE id=1`).Scan(&balance))
	require.Equal(t, "100.0000000000", balance)
	require.NoError(t, m.Reconcile(ctx))
	asyncID := uuid.NewString()
	_, err = db.Exec(`INSERT INTO cluster_request_journal(attempt_id,request_id,node_id,instance_id,user_id,api_key_id,route,state,http_status) VALUES($1,'async','a','dead',1,2,'/v1/images/tasks','finished',202)`, asyncID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO cluster_image_tasks(task_id,node_id,instance_id,status,payload,expires_at,updated_at) VALUES('async_image','a','dead','processing',jsonb_build_object('status','processing','_cluster_attempt_id',$1::text),NOW()+INTERVAL '1 day',NOW()-INTERVAL '2 minutes')`, asyncID)
	require.NoError(t, err)
	require.NoError(t, m.Reconcile(ctx))
	require.NoError(t, db.QueryRow(`SELECT state FROM cluster_request_journal WHERE attempt_id=$1`, asyncID).Scan(&status))
	require.Equal(t, "needs_review", status)
	missingID := uuid.NewString()
	_, err = db.Exec(`INSERT INTO cluster_request_journal(attempt_id,request_id,node_id,instance_id,user_id,api_key_id,route) VALUES($1,'missing-evidence','a','a',1,2,'/v1/responses')`, missingID)
	require.NoError(t, err)
	m.FinishRequest(missingID, 200)
	require.NoError(t, db.QueryRow(`SELECT state FROM cluster_request_journal WHERE attempt_id=$1`, missingID).Scan(&status))
	require.Equal(t, "needs_review", status)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()
	m.rdb = rdb
	require.NoError(t, rdb.Set(ctx, "billing:balance:1", "99", time.Minute).Err())
	require.NoError(t, rdb.ZAdd(ctx, "billing:inflight:{1}", redis.Z{Score: float64(time.Now().Add(time.Minute).UnixMilli()), Member: "active"}).Err())
	cache, err := m.BalanceStatus(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "99", cache["cache_balance"])
	require.Equal(t, int64(1), cache["inflight"])
	require.NoError(t, m.InvalidateBalance(ctx, 1, 1))
	require.ErrorIs(t, rdb.Get(ctx, "billing:balance:1").Err(), redis.Nil)
	require.Equal(t, int64(1), rdb.ZCard(ctx, "billing:inflight:{1}").Val())

	// Resource telemetry travels through the existing JSON snapshot, without a
	// schema migration. Missing resources remain valid during mixed versions.
	sampledAt := time.Now().UTC().Truncate(time.Second)
	usage := 25.0
	m.resources.Store(&Resources{SampledAt: sampledAt, CPUCores: 40, CPUPercent: &usage, MemoryTotal: 128 << 30, MemoryUsed: 32 << 30})
	require.NoError(t, m.Heartbeat(ctx))
	nodes, err := m.Nodes(ctx)
	require.NoError(t, err)
	var own *Node
	for i := range nodes {
		if nodes[i].ID == "a" {
			own = &nodes[i]
		}
	}
	require.NotNil(t, own)
	require.NotNil(t, own.Stats.Resources)
	require.Equal(t, sampledAt, own.Stats.Resources.SampledAt)
	require.Equal(t, usage, *own.Stats.Resources.CPUPercent)
	require.NotNil(t, own.Stats.DatabaseLatencyMS)
	require.GreaterOrEqual(t, *own.Stats.DatabaseLatencyMS, 0.0)
	require.Nil(t, own.Stats.IngressOK, "ingress sync is not applicable outside HA")
	require.True(t, m.sampleArchive(ctx).Known)
	require.True(t, m.Fresh())
	original := cfg.JWT.Secret
	cfg.JWT.Secret = "mismatched-node"
	require.NoError(t, m.Heartbeat(ctx))
	require.False(t, m.Fresh())
	require.False(t, m.Begin())
	restarted := NewManager(db, rdb, cfg, "test")
	require.NoError(t, restarted.Heartbeat(ctx))
	require.False(t, restarted.Fresh(), "restart cannot overwrite the durable contract")
	cfg.JWT.Secret = original
	require.NoError(t, m.Heartbeat(ctx))
	require.True(t, m.Fresh())
	require.NoError(t, m.recordHistory(ctx, *m.latest.Load()))
	history, err := m.History(ctx, "a", "24h")
	require.NoError(t, err)
	require.NotEmpty(t, history.Points)
	_, err = m.History(ctx, "not-registered", "24h")
	require.ErrorIs(t, err, ErrInvalidHistory)

}
