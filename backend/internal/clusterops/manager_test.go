package clusterops

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAdmissionConcurrentLimitAndDrain(t *testing.T) {
	cfg := &config.Config{}
	cfg.Cluster.Enabled = true
	cfg.Cluster.MaxConcurrency = 7
	m := NewManager(nil, nil, cfg, "test")
	m.lastOK.Store(time.Now().Unix())
	var accepted atomic.Int64
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if m.Begin() {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, int64(7), accepted.Load())
	for range 7 {
		m.End(200)
	}
	done := m.TrackTask()
	require.Equal(t, int64(1), m.BackgroundActive())
	done()
	done()
	require.Zero(t, m.BackgroundActive())
	c := m.Control()
	c.Maintenance = true
	m.control.Store(&c)
	require.False(t, m.Begin())
	c.Maintenance = false
	m.control.Store(&c)
	m.lastOK.Store(time.Now().Add(-time.Minute).Unix())
	require.False(t, m.Begin())
}
func TestBillingEvidenceClassification(t *testing.T) {
	for _, v := range []struct {
		name string
		r    Request
		want string
	}{
		{"orphan", Request{State: "needs_review"}, "unknown"},
		{"unbilled", Request{State: "finished"}, "no_evidence"},
		{"pending", Request{EvidenceCount: 1}, "settlement_pending"},
		{"missing", Request{EvidenceCount: 1, SettledCount: 1}, "usage_missing"},
		{"equal", Request{EvidenceCount: 1, SettledCount: 1, UsageCount: 1, ExpectedCost: "1.2000000000", RecordedCost: "1.2"}, "settled"},
		{"mismatch", Request{EvidenceCount: 1, SettledCount: 1, UsageCount: 1, ExpectedCost: "1.20", RecordedCost: "0"}, "cost_mismatch"},
		{"large_exact", Request{EvidenceCount: 1, SettledCount: 1, UsageCount: 1, ExpectedCost: "9999999999.0000000200", RecordedCost: "9999999999"}, "cost_mismatch"},
	} {
		t.Run(v.name, func(t *testing.T) { require.Equal(t, v.want, ClassifyBilling(v.r)) })
	}
}
func TestRequestIdentityAndAttemptIsolation(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxkey.RequestID, "local-id")
	require.Equal(t, "local:local-id", BillingRequestID(ctx))
	ctx = context.WithValue(ctx, ctxkey.ClientRequestID, " retry-id ")
	require.Equal(t, "client:retry-id", BillingRequestID(ctx))
	require.Empty(t, AttemptID(ctx))
	require.Equal(t, "attempt", AttemptID(WithAttempt(ctx, "attempt")))
}
func TestClusterAlertsDetectRedundancyAndMismatch(t *testing.T) {
	good := Node{ID: "a", Online: true, Role: "primary", Version: "v1", Control: Control{MaxConcurrency: 100, Revision: 2}, Stats: Stats{DatabaseOK: true, RedisOK: true, RedisLinkOK: true, LeaseOK: true, ReplicationKnown: true, SyncReplicas: 1, IngressRevision: 2, SettingsVersion: "one"}}
	require.Empty(t, EvaluateAlerts([]Node{good}, true))
	other := good
	other.ID = "b"
	other.Role = "worker"
	other.Version = "v2"
	other.Stats.SettingsVersion = "two"
	other.Stats.SyncReplicas = 0
	other.Stats.IngressRevision = 1
	keys := map[string]bool{}
	for _, a := range EvaluateAlerts([]Node{good, other}, true) {
		keys[a.Key] = true
	}
	require.True(t, keys["cluster:versions"])
	require.True(t, keys["cluster:settings"])
	require.True(t, keys["b:single_database_copy"])
	require.True(t, keys["b:ingress_pending"])
}
