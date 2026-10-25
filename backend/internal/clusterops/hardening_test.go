package clusterops

import (
	"context"
	"database/sql"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestContractFingerprintScope(t *testing.T) {
	cfg := &config.Config{}
	cfg.JWT.Secret = "shared"
	cfg.Totp.EncryptionKey = "encryption"
	cfg.Cluster.Enabled = true
	a := NewManager(nil, nil, cfg, "v1")
	one, err := a.contractFingerprint()
	require.NoError(t, err)
	copy := *cfg
	copy.Cluster.NodeID = "different"
	copy.Cluster.Role = "worker"
	copy.Cluster.MaxConcurrency = 500
	b := NewManager(nil, nil, &copy, "v2")
	two, err := b.contractFingerprint()
	require.NoError(t, err)
	require.Equal(t, one, two)
	copy.JWT.Secret = "other"
	two, err = b.contractFingerprint()
	require.NoError(t, err)
	require.NotEqual(t, one, two)
	copy.JWT.Secret = ""
	_, err = b.contractFingerprint()
	require.Error(t, err)
}
func TestOperationalAlertsDebounceRecoveryAndArchive(t *testing.T) {
	now := time.Now()
	old := now.Add(-time.Hour)
	ok := true
	pending := int64(0)
	n := Node{ID: "a", StartedAt: old, Stats: Stats{RuntimeConfigOK: &ok, Archive: &ArchiveHealth{Known: true, Enabled: true, SampledAt: now, LastArchived: &old, Pending: &pending}, Verification: &Verification{CheckedAt: now, RestoreCheckedAt: &now, Storage: &StorageVerification{CheckedAt: now, Status: "pass"}, Endpoints: []IngressProbe{{Status: "fail", ConsecutiveFailures: 2}}}}}
	require.Empty(t, operationalAlerts(n, true, now), "two failed probes are not yet an outage; idle archiving is healthy")
	n.Stats.Verification.Endpoints[0].ConsecutiveFailures = 3
	a := operationalAlerts(n, true, now)
	require.Len(t, a, 1)
	require.Equal(t, "a:ingress_probe_failed", a[0].Key)
	n.Stats.Verification.Endpoints[0].Status = "pass"
	require.Empty(t, operationalAlerts(n, true, now))
	n.Stats.Archive.OldestPending = &old
	n.Stats.Archive.Pending = nil
	status, code := archiveStatus(n.Stats.Archive, now)
	require.Equal(t, "unknown", status)
	require.Equal(t, "archive_queue_unknown", code)
	pending = 1
	n.Stats.Archive.Pending = &pending
	status, code = archiveStatus(n.Stats.Archive, now)
	require.Equal(t, "fail", status)
	require.Equal(t, "archive_backlog", code)
	n.Stats.Verification.CheckedAt = old
	a = operationalAlerts(n, true, now)
	require.Contains(t, a, Alert{Key: "a:verification_stale", NodeID: "a", Severity: "warning", Message: "verification_stale"})
	ok = false
	a = operationalAlerts(n, false, now)
	require.Len(t, a, 1)
	require.Equal(t, "a:runtime_contract", a[0].Key)
}
func TestHistoryRetentionAndUnavailable(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer func() { _ = rdb.Close() }()
	cfg := &config.Config{}
	cfg.Cluster.Enabled = true
	cfg.Cluster.NodeID = "a"
	m := NewManager(nil, rdb, cfg, "test")
	ctx := context.Background()
	require.NoError(t, rdb.ZAdd(ctx, historyKey("a"), redis.Z{Score: float64(time.Now().Add(-8 * 24 * time.Hour).Unix()), Member: "old"}).Err())
	s := Stats{Performance: &Performance{SampledAt: time.Now()}, Active: 3}
	require.NoError(t, m.recordHistory(ctx, s))
	s.Active = 4
	require.NoError(t, m.recordHistory(ctx, s))
	require.EqualValues(t, 1, rdb.ZCard(ctx, historyKey("a")).Val())
	h, err := m.readHistory(ctx, History{Node: "a", Points: []HistoryPoint{}}, 24*time.Hour)
	require.NoError(t, err)
	require.Len(t, h.Points, 1)
	require.EqualValues(t, 4, h.Points[0].Active)
	require.Nil(t, h.Points[0].CPU)
	require.LessOrEqual(t, rdb.TTL(ctx, historyKey("a")).Val(), 8*24*time.Hour)
	_, err = m.History(ctx, "a", "100d")
	require.ErrorIs(t, err, ErrInvalidHistory)
	server.Close()
	_, err = m.readHistory(ctx, History{Node: "a"}, time.Hour)
	require.ErrorIs(t, err, ErrUnavailable)
}
func TestCompletionPhasesHaveSeparateDenominators(t *testing.T) {
	cfg := &config.Config{}
	cfg.Cluster.Enabled = true
	m := NewManager(nil, nil, cfg, "test")
	phase := 20.0
	m.ObserveCompletion(100*time.Millisecond, [3]*float64{&phase, nil, nil})
	m.ObserveCompletion(300*time.Millisecond, [3]*float64{})
	p := m.performance.sample(time.Now(), 2, sql.DBStats{})
	require.EqualValues(t, 2, p.CompletedSamples)
	require.Equal(t, 200.0, *p.DurationMeanMS)
	require.Equal(t, 20.0, *p.AuthMeanMS)
	require.Nil(t, p.UpstreamMeanMS)
	require.EqualValues(t, 1, p.PhaseSamples[0])
}
