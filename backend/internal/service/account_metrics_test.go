package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

type accountMetricsRepo struct {
	UsageLogRepository
	usageErr     error
	telemetryErr error
}

func (r *accountMetricsRepo) GetAccountWindowStatsBatch(context.Context, []int64, time.Time) (map[int64]*usagestats.AccountStats, error) {
	return map[int64]*usagestats.AccountStats{1: {Requests: 3}}, r.usageErr
}
func (r *accountMetricsRepo) GetAccountWindowStats(context.Context, int64, time.Time) (*usagestats.AccountStats, error) {
	return &usagestats.AccountStats{Requests: 3}, r.usageErr
}
func (r *accountMetricsRepo) GetAccountErrorStatsBatch(context.Context, []int64, time.Time, time.Time) (map[int64]*AccountErrorStats, error) {
	return map[int64]*AccountErrorStats{1: {AbnormalRequests: 2, RecoveredRequests: 1, FailedRequests: 1}}, r.telemetryErr
}
func TestAccountMetricsTelemetry(t *testing.T) {
	repo := &accountMetricsRepo{}
	svc := &AccountUsageService{usageLogRepo: repo}
	stats, err := svc.GetTodayStatsBatch(context.Background(), []int64{1, 1, 0})
	require.NoError(t, err)
	require.Len(t, stats, 1)
	require.EqualValues(t, 2, stats[1].ErrorStats.AbnormalRequests)
	repo.telemetryErr = errors.New("telemetry unavailable")
	stats, err = svc.GetTodayStatsBatch(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Nil(t, stats[1].ErrorStats)
	require.EqualValues(t, 3, stats[1].Requests)
	repo.usageErr = errors.New("database unavailable")
	stats, err = svc.GetTodayStatsBatch(context.Background(), []int64{1})
	require.Error(t, err)
	require.Nil(t, stats, "a failed query must not turn into zero usage")
}
