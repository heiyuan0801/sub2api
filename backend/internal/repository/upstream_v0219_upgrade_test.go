package repository

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

// Run only against a fresh dedicated database. This verifies upgrading the
// customized v0.2.18 migration set without renaming previously applied files.
func TestUpstreamV0219UpgradePostgres(t *testing.T) {
	dsn := os.Getenv("SUB2API_UPSTREAM_MERGE_TEST_DSN")
	if dsn == "" {
		t.Skip("set SUB2API_UPSTREAM_MERGE_TEST_DSN to a fresh sub2api_merge_test database")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var database string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT current_database()").Scan(&database))
	require.Equal(t, "sub2api_merge_test", database, "requires isolated database")
	var migrationTable sql.NullString
	require.NoError(t, db.QueryRowContext(ctx, "SELECT to_regclass('schema_migrations')::text").Scan(&migrationTable))
	require.False(t, migrationTable.Valid, "requires a fresh database")
	baseline := fstest.MapFS{}
	names, err := fs.Glob(migrations.FS, "*.sql")
	require.NoError(t, err)
	for _, name := range names {
		if name == "247_scoped_admin_api_keys.sql" || name == "248_group_image_count_billing.sql" {
			continue
		}
		body, err := fs.ReadFile(migrations.FS, name)
		require.NoError(t, err)
		baseline[name] = &fstest.MapFile{Data: body}
	}
	require.NoError(t, applyMigrationsFS(ctx, db, baseline))
	var groupID int64
	require.NoError(t, db.QueryRowContext(ctx, "INSERT INTO groups(name, platform, rate_multiplier) VALUES ('upstream-merge-fixture', 'openai', 1.25) RETURNING id").Scan(&groupID))
	require.NoError(t, ApplyMigrations(ctx, db))
	require.NoError(t, ApplyMigrations(ctx, db), "repeated startup must remain idempotent")
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE filename IN (
 '247_cluster_operations.sql','247_scoped_admin_api_keys.sql',
 '248_aggregate_api_keys.sql','248_group_image_count_billing.sql','249_aggregate_group_routing.sql')`).Scan(&count))
	require.Equal(t, 5, count, "upstream and custom migrations must coexist")
	var enabled bool
	var rate float64
	require.NoError(t, db.QueryRowContext(ctx, "SELECT image_count_billing, rate_multiplier FROM groups WHERE id=$1", groupID).Scan(&enabled, &rate))
	require.False(t, enabled, "existing groups must not silently switch billing modes")
	require.Equal(t, 1.25, rate)
	var table sql.NullString
	require.NoError(t, db.QueryRowContext(ctx, "SELECT to_regclass('scoped_admin_api_keys')::text").Scan(&table))
	require.True(t, table.Valid)
}
