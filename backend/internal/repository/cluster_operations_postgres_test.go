package repository

import (
	"context"
	"database/sql"
	"github.com/Wei-Shaw/sub2api/internal/clusterops"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClusterDurableStoresPostgres(t *testing.T) {
	dsn := os.Getenv("SUB2API_CLUSTER_TEST_DSN")
	if dsn == "" {
		t.Skip("set SUB2API_CLUSTER_TEST_DSN for local PostgreSQL integration")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	schema := "cluster_repo_" + uuid.NewString()[:8]
	_, err = db.Exec(`CREATE SCHEMA ` + schema)
	require.NoError(t, err)
	defer func() { _, _ = db.Exec(`DROP SCHEMA ` + schema + ` CASCADE`) }()
	_, err = db.Exec(`SET search_path TO ` + schema)
	require.NoError(t, err)
	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "247_cluster_operations.sql"))
	require.NoError(t, err)
	_, err = db.Exec(string(migration))
	require.NoError(t, err)
	cfg := &config.Config{}
	cfg.Cluster.Enabled = true
	cfg.Cluster.HAEnabled = true
	cfg.Cluster.NodeID = "test"
	m := clusterops.NewManager(db, nil, cfg, "test")
	store := ProvideImageTaskStore(nil, db, cfg, m)
	attempt := uuid.NewString()
	ctx := clusterops.WithAttempt(context.Background(), attempt)
	task := &service.ImageTaskRecord{ID: "imgtask_durable", UserID: 7, APIKeyID: 9, Status: service.ImageTaskStatusProcessing, ExpiresAt: time.Now().Add(time.Hour).Unix()}
	require.NoError(t, store.Save(ctx, task, time.Hour))
	var recorded string
	require.NoError(t, db.QueryRow(`SELECT payload->>'_cluster_attempt_id' FROM cluster_image_tasks WHERE task_id=$1`, task.ID).Scan(&recorded))
	require.Equal(t, attempt, recorded)
	other := ProvideImageTaskStore(nil, db, cfg, clusterops.NewManager(db, nil, cfg, "test"))
	task.Status = service.ImageTaskStatusCompleted
	require.ErrorIs(t, other.Save(ctx, task, time.Hour), service.ErrImageTaskUnavailable)
	require.NoError(t, store.Save(context.Background(), task, time.Hour))
	require.NoError(t, db.QueryRow(`SELECT payload->>'_cluster_attempt_id' FROM cluster_image_tasks WHERE task_id=$1`, task.ID).Scan(&recorded))
	require.Equal(t, attempt, recorded)
	got, err := other.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, service.ImageTaskStatusCompleted, got.Status)
	require.ErrorIs(t, store.Save(ctx, task, time.Hour), service.ErrImageTaskUnavailable)
	repo := &usageBillingRepository{db: db}
	cmd := &service.UsageBillingCommand{RequestID: "bill", APIKeyID: 9, UserID: 7, AccountID: 1, BalanceCost: 1.25}
	cmd.Normalize()
	require.NoError(t, repo.StageClusterBillingEvidence(ctx, cmd, &service.UsageLog{ActualCost: 1.25}, attempt))
	require.NoError(t, repo.StageClusterBillingEvidence(ctx, cmd, &service.UsageLog{ActualCost: 1.25}, attempt))
	cmd.RequestFingerprint = "different"
	require.ErrorIs(t, repo.StageClusterBillingEvidence(ctx, cmd, nil, attempt), service.ErrUsageBillingRequestConflict)
	require.NoError(t, db.QueryRow(`SELECT expected_cost::text FROM cluster_billing_evidence WHERE request_id='bill'`).Scan(&recorded))
	require.Equal(t, "1.2500000000", recorded)
}
