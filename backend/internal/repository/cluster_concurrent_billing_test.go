package repository

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/clusterops"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Two independent connection pools apply the real billing transaction. No
// model/provider requests or production schema are used.
func TestClusterConcurrentBillingPostgres(t *testing.T) {
	dsn := os.Getenv("SUB2API_CLUSTER_TEST_DSN")
	if dsn == "" {
		t.Skip("set SUB2API_CLUSTER_TEST_DSN for isolated PostgreSQL tests")
	}
	ctx := context.Background()
	admin, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = admin.Close() }()
	schema := "cluster_concurrent_" + uuid.NewString()[:8]
	_, err = admin.Exec(`CREATE SCHEMA ` + schema)
	require.NoError(t, err)
	defer func() { _, _ = admin.Exec(`DROP SCHEMA ` + schema + ` CASCADE`) }()
	pools := make([]*sql.DB, 2)
	for i := range pools {
		pools[i], err = sql.Open("postgres", dsn)
		require.NoError(t, err)
		pools[i].SetMaxOpenConns(1)
		pools[i].SetMaxIdleConns(1)
		defer func() { _ = pools[i].Close() }()
		_, err = pools[i].Exec(`SET search_path TO ` + schema)
		require.NoError(t, err)
	}
	db := pools[0]
	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "247_cluster_operations.sql"))
	require.NoError(t, err)
	_, err = db.Exec(string(migration))
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE users(id bigint primary key,balance numeric(20,10),deleted_at timestamptz,updated_at timestamptz);INSERT INTO users VALUES(1,100,NULL,NOW());CREATE TABLE usage_billing_dedup(id bigserial,request_id text,api_key_id bigint,request_fingerprint text,UNIQUE(request_id,api_key_id));CREATE TABLE usage_billing_dedup_archive(request_id text,api_key_id bigint,request_fingerprint text);CREATE TABLE usage_logs(id bigserial,request_id text,api_key_id bigint,actual_cost numeric(20,10));`)
	require.NoError(t, err)
	repos := []*usageBillingRepository{{db: pools[0]}, {db: pools[1]}}
	attempt := uuid.NewString()
	cmd := service.UsageBillingCommand{RequestID: "concurrent", APIKeyID: 2, UserID: 1, AccountID: 3, BalanceCost: 1.25}
	cmd.Normalize()
	_, err = db.Exec(`INSERT INTO cluster_request_journal(attempt_id,request_id,node_id,instance_id,user_id,api_key_id,route) VALUES($1,'concurrent','a','a',1,2,'/v1/responses')`, attempt)
	require.NoError(t, err)
	require.NoError(t, repos[0].StageClusterBillingEvidence(ctx, &cmd, &service.UsageLog{ActualCost: 1.25}, attempt))
	cfg := &config.Config{}
	cfg.Cluster.Enabled = true
	manager := clusterops.NewManager(db, nil, cfg, "test")
	// Provider usage saved, process lost before debit.
	manager.FinishRequest(attempt, 200)
	records, err := manager.Requests(ctx, true, time.Time{})
	require.NoError(t, err)
	require.Equal(t, "settlement_pending", records[0].BillingState)
	var wg sync.WaitGroup
	results := make(chan bool, 32)
	errs := make(chan error, 32)
	for i := range 32 {
		wg.Go(func() {
			copy := cmd
			e := repos[i%2].StageClusterBillingEvidence(ctx, &copy, &service.UsageLog{ActualCost: 1.25}, attempt)
			if e != nil {
				errs <- e
				return
			}
			r, e := repos[i%2].Apply(ctx, &copy)
			errs <- e
			if r != nil {
				results <- r.Applied
			}
		})
	}
	wg.Wait()
	close(results)
	close(errs)
	for e := range errs {
		require.NoError(t, e)
	}
	applied := 0
	for v := range results {
		if v {
			applied++
		}
	}
	require.Equal(t, 1, applied)
	// Debit committed, process lost before usage-log insertion.
	records, err = manager.Requests(ctx, true, time.Time{})
	require.NoError(t, err)
	require.Equal(t, "usage_missing", records[0].BillingState)
	_, err = db.Exec(`INSERT INTO usage_logs(request_id,api_key_id,actual_cost) VALUES('concurrent',2,1.25)`)
	require.NoError(t, err)
	records, err = manager.Requests(ctx, true, time.Time{})
	require.NoError(t, err)
	require.Equal(t, "settled", records[0].BillingState)
	var balance string
	require.NoError(t, db.QueryRow(`SELECT balance::text FROM users WHERE id=1`).Scan(&balance))
	require.Equal(t, "98.7500000000", balance)
	cmd.BalanceCost = 2
	cmd.RequestFingerprint = "different"
	_, err = repos[1].Apply(ctx, &cmd)
	require.ErrorIs(t, err, service.ErrUsageBillingRequestConflict)
}
