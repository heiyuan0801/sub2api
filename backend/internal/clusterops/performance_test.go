package clusterops

import (
	"context"
	"database/sql"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestPerformanceRollingWindowAndPoolDeltas(t *testing.T) {
	var meter performanceMeter
	now := time.Unix(10000, 0)
	first := meter.sample(now, 5, sql.DBStats{MaxOpenConnections: 80, WaitCount: 10, WaitDuration: time.Second})
	require.Nil(t, first.RequestsPerSecond)
	require.Nil(t, first.FirstTokenP95MS)
	require.Nil(t, first.CacheReadPercent)
	ttft := 125
	for range 20 {
		meter.observe(now, &ttft, 10, 10, 80)
	}
	meter.observe(now, nil, 0, 0, 0)
	result := meter.sample(now.Add(5*time.Second), 55, sql.DBStats{MaxOpenConnections: 80, WaitCount: 12, WaitDuration: 1200 * time.Millisecond})
	require.Equal(t, 10.0, *result.RequestsPerSecond)
	require.Equal(t, 250.0, *result.FirstTokenP95MS)
	require.EqualValues(t, 20, result.FirstTokenSamples)
	require.Equal(t, 80.0, *result.CacheReadPercent)
	require.Equal(t, 100.0, *result.DBWaitMeanMS)
	expired := meter.sample(now.Add(61*time.Second), 55, sql.DBStats{})
	require.Nil(t, expired.FirstTokenP95MS)
	require.Nil(t, expired.CacheReadPercent)
}
func TestPerformanceOverflowAndConcurrentObservation(t *testing.T) {
	cfg := &config.Config{}
	cfg.Cluster.Enabled = true
	m := NewManager(nil, nil, cfg, "test")
	ctx := m.WithMetrics(context.Background())
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for range 100 {
				ttft := 700000
				ObserveUsage(ctx, &ttft, 1, 0, 1)
			}
		})
	}
	wg.Wait()
	p := m.performance.sample(time.Now(), 0, sql.DBStats{})
	require.EqualValues(t, 1600, p.UsageSamples)
	require.True(t, p.FirstTokenOverflow)
	require.Nil(t, p.FirstTokenP95MS)
	ObserveUsage(context.Background(), nil, 1, 0, 0)
}
func TestVerificationRejectsFutureAndMalformedReports(t *testing.T) {
	cfg := &config.Config{}
	cfg.Cluster.Enabled = true
	cfg.Cluster.HAEnabled = true
	cfg.Cluster.LeaseFile = t.TempDir() + "/proof"
	m := NewManager(nil, nil, cfg, "test")
	require.Nil(t, m.readVerification())
	v := Verification{CheckedAt: time.Now(), Scope: "local_lab", Endpoints: []IngressProbe{{ID: "a", Status: "pass"}, {ID: "b", Status: "fail"}}}
	require.NoError(t, writePrivateJSON(cfg.Cluster.LeaseFile+".verification", v))
	require.NotNil(t, m.readVerification())
	v.CheckedAt = time.Now().Add(time.Hour)
	require.NoError(t, writePrivateJSON(cfg.Cluster.LeaseFile+".verification", v))
	require.Nil(t, m.readVerification())
}

func TestUsageContextKeepsMetricsAndAttemptWithoutParentCancellation(t *testing.T) {
	cfg := &config.Config{}
	cfg.Cluster.Enabled = true
	m := NewManager(nil, nil, cfg, "test")
	parent, cancel := context.WithCancel(m.WithMetrics(WithAttempt(context.Background(), "attempt")))
	cancel()
	ctx := CopyUsageContext(parent, context.Background())
	require.NoError(t, ctx.Err())
	require.Equal(t, "attempt", AttemptID(ctx))
	token := 100
	ObserveUsage(ctx, &token, 80, 0, 20)
	sample := m.performance.sample(time.Now(), 0, sql.DBStats{})
	require.EqualValues(t, 1, sample.UsageSamples)
	require.Equal(t, 20.0, *sample.CacheReadPercent)
}
