//go:build unit

package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/stretchr/testify/require"
)

type clusterStartupCache struct {
	ConcurrencyCache
	cleanups int
}

func (c *clusterStartupCache) CleanupStaleProcessSlots(context.Context, string) error {
	c.cleanups++
	return nil
}

func TestClusterStartupPreservesPeerSlots(t *testing.T) {
	for _, role := range []string{"primary", "worker", "single"} {
		cache := &clusterStartupCache{}
		cfg := &config.Config{Cluster: config.ClusterConfig{Enabled: role != "single", Role: role}}
		ProvideConcurrencyService(cache, nil, cfg)
		want := 0
		if role == "single" {
			want = 1
		}
		require.Equal(t, want, cache.cleanups, role)
	}
}

type clusterSettingsRepo struct {
	SettingRepository
	values map[string]string
	err    error
}

func (r *clusterSettingsRepo) GetAll(context.Context) (map[string]string, error) {
	return r.values, r.err
}

func TestClusterSettingsRefreshRecoversChanges(t *testing.T) {
	originalMapping := xai.RuntimeModelMappingOptions()
	t.Cleanup(func() { xai.SetRuntimeModelMappingOptions(originalMapping) })
	repo := &clusterSettingsRepo{values: map[string]string{SettingKeyBackendModeEnabled: "true"}}
	svc := NewSettingService(repo, &config.Config{})
	updates := 0
	svc.SetOnUpdateCallback(func() { updates++ })
	digest, err := svc.refreshClusterSettings(context.Background(), [sha256.Size]byte{})
	require.NoError(t, err)
	require.Equal(t, 1, updates)
	_, err = svc.refreshClusterSettings(context.Background(), digest)
	require.NoError(t, err)
	require.Equal(t, 1, updates)
	repo.err = errors.New("offline")
	require.Error(t, svc.InitializeClusterSettings(context.Background()))
	_, err = svc.refreshClusterSettings(context.Background(), digest)
	require.Error(t, err)
	require.Equal(t, 1, updates)
	repo.err = nil
	repo.values[SettingKeyBackendModeEnabled] = "false"
	next, err := svc.refreshClusterSettings(context.Background(), digest)
	require.NoError(t, err)
	require.NotEqual(t, digest, next)
	require.Equal(t, 2, updates)
	cached, ok := backendModeCache.Load().(*cachedBackendMode)
	require.True(t, ok)
	require.False(t, cached.value)
}

func TestClusterWorkerSkipsScheduledProviders(t *testing.T) {
	cfg := &config.Config{Cluster: config.ClusterConfig{Enabled: true, Role: "worker"}}
	// Nil repositories would panic if these providers started their first scan.
	a := ProvideAccountExpiryService(nil, cfg)
	a.Stop()
	p := ProvideProxyExpiryService(nil, cfg)
	p.Stop()
	c := ProvideCNProviderBalanceCheckService(nil, nil, nil, cfg)
	c.Stop()
	v := ProvideChannelMonitorV2Aggregator(nil, nil, nil, cfg)
	v.Stop()
}
