package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestAccountWindowMetricsBatchProjection(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	repo := &usageLogRepository{sql: db}
	mock.ExpectQuery(`SELECT account_id,.*AVG\(first_token_ms\).*GROUP BY account_id`).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"account_id", "requests", "tokens", "cost", "standard_cost", "user_cost", "cache_read", "cache_input", "first_token"}).AddRow(7, 2, 1500, 1, 1, 1, 924, 1000, 7400).AddRow(8, 1, 0, 0, 0, 0, 0, 0, nil))
	stats, err := repo.GetAccountWindowStatsBatch(context.Background(), []int64{7, 8, 9}, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.EqualValues(t, 924, stats[7].CacheReadTokens)
	require.EqualValues(t, 1000, stats[7].CacheInputTokens)
	require.Equal(t, 7400.0, *stats[7].AverageFirstTokenMs)
	require.Nil(t, stats[8].AverageFirstTokenMs)
	require.Zero(t, stats[9].Requests)
	require.NoError(t, mock.ExpectationsWereMet())
}
