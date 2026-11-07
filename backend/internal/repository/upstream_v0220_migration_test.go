package repository

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Wei-Shaw/sub2api/migrations"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// This destructive rehearsal requires a dedicated, empty local test database.
func TestUpstreamV0220MigrationPreservesCustomSchema(t *testing.T) {
	dsn := os.Getenv("SUB2API_MERGE_TEST_DSN")
	if dsn == "" {
		t.Skip("SUB2API_MERGE_TEST_DSN is required for an isolated upgrade rehearsal")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var tables int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&tables))
	require.Zero(t, tables, "upgrade rehearsal requires an empty database")
	baseline := fstest.MapFS{}
	files, err := fs.Glob(migrations.FS, "*.sql")
	require.NoError(t, err)
	for _, name := range files {
		if name == "249_mcp_oauth.sql" {
			continue
		}
		data, err := migrations.FS.ReadFile(name)
		require.NoError(t, err)
		baseline[name] = &fstest.MapFile{Data: data}
	}
	require.NoError(t, applyMigrationsFS(ctx, db, baseline))
	_, err = db.ExecContext(ctx, "INSERT INTO users (email, password_hash, balance) VALUES ('merge-fixture@example.invalid', 'fixture', 12.34)")
	require.NoError(t, err)
	var before int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM schema_migrations").Scan(&before))
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- ApplyMigrations(ctx, db)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	var after, custom, mcpTables int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM schema_migrations").Scan(&after))
	require.Equal(t, before+1, after)
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM schema_migrations WHERE filename IN ('247_cluster_operations.sql', '248_aggregate_api_keys.sql', '249_aggregate_group_routing.sql')").Scan(&custom))
	require.Equal(t, 3, custom)
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('mcp_oauth_grants', 'mcp_oauth_codes', 'mcp_oauth_credentials')").Scan(&mcpTables))
	require.Equal(t, 3, mcpTables)
	var balance float64
	require.NoError(t, db.QueryRowContext(ctx, "SELECT balance FROM users WHERE email = 'merge-fixture@example.invalid'").Scan(&balance))
	require.InDelta(t, 12.34, balance, 0.000001)
	require.NoError(t, ApplyMigrations(ctx, db), "a subsequent restart must be idempotent")
}
