//go:build unit

package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/stretchr/testify/require"
)

type clusterSessionFake struct {
	GatewayCache
	mu     sync.Mutex
	values map[string]string
	fail   bool
}

func (s *clusterSessionFake) GetSessionValues(_ context.Context, keys []string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail {
		return nil, errors.New("offline")
	}
	values := make([]string, len(keys))
	for i, k := range keys {
		values[i] = s.values[k]
	}
	return values, nil
}
func (s *clusterSessionFake) SetSessionValue(_ context.Context, key, value string, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return nil
}
func (s *clusterSessionFake) DeleteSessionValue(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, key)
	return nil
}
func (s *clusterSessionFake) SetSessionAccountID(context.Context, int64, string, int64, time.Duration) error {
	return nil
}

func TestClusterContinuationDoesNotDropForeignAnchor(t *testing.T) {
	cache := &clusterSessionFake{values: map[string]string{}}
	a, b := NewClusterOpenAIWSStateStore(cache, "a"), NewClusterOpenAIWSStateStore(cache, "b")
	ctx := context.Background()
	require.NoError(t, a.BindResponseAccount(ctx, 7, "resp-1", 8, time.Minute))
	a.BindResponseConn("resp-1", "socket-local", time.Minute)
	require.NoError(t, checkClusterContinuation(ctx, a, 7, "resp-1", true))
	require.ErrorIs(t, checkClusterContinuation(ctx, b, 7, "resp-1", true), errClusterContinuation)
	require.ErrorIs(t, checkClusterContinuation(ctx, b, 7, "unknown", true), errClusterContinuation)
	require.NoError(t, checkClusterContinuation(ctx, b, 7, "resp-1", false), "stored responses can recover")
	require.NoError(t, checkClusterContinuation(ctx, b, 7, "", true), "full context without anchor is accepted")
	restarted := NewClusterOpenAIWSStateStore(cache, "a")
	require.ErrorIs(t, checkClusterContinuation(ctx, restarted, 7, "resp-1", true), errClusterContinuation)
	cache.fail = true
	require.ErrorIs(t, checkClusterContinuation(ctx, b, 7, "resp-1", true), errClusterContinuation)
}

func TestClusterCompatDigestSharesStableCacheKey(t *testing.T) {
	cache := &clusterSessionFake{values: map[string]string{}}
	cfg := &config.Config{Cluster: config.ClusterConfig{Enabled: true}}
	a, b := &OpenAIGatewayService{cfg: cfg, cache: cache}, &OpenAIGatewayService{cfg: cfg, cache: cache}
	account := &Account{ID: 10}
	a.bindOpenAICompatAnthropicDigestPromptCacheKey(account, 20, "first-second", "stable-key", "")
	key, chain := b.findOpenAICompatAnthropicDigestPromptCacheKey(account, 20, "first-second-third")
	require.Equal(t, "stable-key", key)
	require.Equal(t, "first-second", chain)
	key, _ = b.findOpenAICompatAnthropicDigestPromptCacheKey(account, 21, "first-second-third")
	require.Empty(t, key, "different API key must not inherit the binding")
	a.bindOpenAICompatSessionTurnState(context.Background(), nil, account, "stable-key", "turn-a")
	b.bindOpenAICompatSessionResponseID(context.Background(), nil, account, "stable-key", "response-b")
	require.Equal(t, "turn-a", b.getOpenAICompatSessionTurnState(context.Background(), nil, account, "stable-key"))
	require.Equal(t, "response-b", a.getOpenAICompatSessionResponseID(context.Background(), nil, account, "stable-key"))
	a.disableOpenAICompatSessionContinuation(context.Background(), nil, account, "stable-key")
	b.bindOpenAICompatSessionResponseID(context.Background(), nil, account, "stable-key", "response-c")
	require.True(t, b.isOpenAICompatSessionContinuationDisabled(context.Background(), nil, account, "stable-key"))
	require.Empty(t, b.getOpenAICompatSessionResponseID(context.Background(), nil, account, "stable-key"))
}

type clusterPartialReservation struct {
	clusterBillingFailure
	released int
}

func (*clusterPartialReservation) ReserveInflightBalance(context.Context, int64, string, float64, float64, time.Duration) (bool, float64, error) {
	return true, 0, nil
}
func (s *clusterPartialReservation) ReleaseInflightBalance(context.Context, int64, string) error {
	s.released++
	return nil
}
func (*clusterPartialReservation) ReserveInflightScope(context.Context, string, string, float64, time.Duration) (float64, error) {
	return 0, errors.New("offline")
}
func (*clusterPartialReservation) ReleaseInflightScope(context.Context, string, string) error {
	return nil
}
func TestClusterFixedChargeReleasesEarlierReservation(t *testing.T) {
	cache := &clusterPartialReservation{}
	cfg := &config.Config{Cluster: config.ClusterConfig{Enabled: true}}
	s := NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, nil)
	defer s.Stop()
	_, err := s.ReserveFixedCharge(context.Background(), &User{ID: 1}, &APIKey{ID: 2, Quota: 5}, nil, nil, 1)
	require.ErrorIs(t, err, ErrBillingServiceUnavailable)
	require.Equal(t, 1, cache.released)
}

type clusterConcurrentSubRepo struct{ UserSubscriptionRepository }

func (*clusterConcurrentSubRepo) GetActiveByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	return &UserSubscription{ID: 1}, nil
}
func TestClusterSubscriptionClearWhileReading(t *testing.T) {
	cfg := &config.Config{}
	cfg.SubscriptionCache.L1Size = 128
	cfg.SubscriptionCache.L1TTLSeconds = 30
	s := NewSubscriptionService(nil, &clusterConcurrentSubRepo{}, nil, nil, cfg)
	defer s.Stop()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = s.GetActiveSubscription(context.Background(), 1, 1)
			}
		}()
	}
	for i := 0; i < 20; i++ {
		s.clearSubscriptionL1()
	}
	wg.Wait()
}

type clusterBillingFailure struct{ BillingCache }

func (*clusterBillingFailure) GetUserBalance(context.Context, int64) (float64, error) {
	return 100, nil
}
func (*clusterBillingFailure) ReserveInflightBalance(context.Context, int64, string, float64, float64, time.Duration) (bool, float64, error) {
	return false, 0, errors.New("offline")
}
func (*clusterBillingFailure) ReleaseInflightBalance(context.Context, int64, string) error {
	return nil
}

func TestClusterBillingFailsClosedWithExplicitOptOut(t *testing.T) {
	for _, mode := range []string{"", "closed", "open"} {
		cfg := &config.Config{Cluster: config.ClusterConfig{Enabled: true, BillingFailureMode: mode}}
		cfg.Billing.InflightReservation.Enabled = true
		svc := NewBillingCacheService(&clusterBillingFailure{}, nil, nil, nil, nil, nil, cfg, nil)
		_, err := svc.ReserveInflight(context.Background(), &User{ID: 1}, nil, nil, 1)
		if mode == "open" {
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, err, ErrBillingServiceUnavailable)
		}
		svc.Stop()
	}
}

type eventSettingsRepo struct {
	SettingRepository
	mu     sync.Mutex
	value  string
	events chan struct{}
}

func (r *eventSettingsRepo) GetAll(context.Context) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return map[string]string{SettingKeyBackendModeEnabled: r.value}, nil
}
func (r *eventSettingsRepo) SubscribeSettingChanges(context.Context) (<-chan struct{}, func(), error) {
	return r.events, func() {}, nil
}
func TestClusterSettingsNotificationAndFreshness(t *testing.T) {
	original := xai.RuntimeModelMappingOptions()
	defer xai.SetRuntimeModelMappingOptions(original)
	r := &eventSettingsRepo{value: "false", events: make(chan struct{}, 1)}
	s := NewSettingService(r, &config.Config{})
	require.Error(t, s.CheckClusterSettingsFresh(context.Background()))
	require.NoError(t, s.InitializeClusterSettings(context.Background()))
	before := s.ClusterSettingsStatus()["version"]
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); s.RunClusterSettingsRefresh(ctx) }()
	defer func() { cancel(); <-done }()
	r.mu.Lock()
	r.value = "true"
	r.mu.Unlock()
	r.events <- struct{}{}
	require.Eventually(t, func() bool { return before != s.ClusterSettingsStatus()["version"] }, 2*time.Second, 10*time.Millisecond)
	require.NoError(t, s.CheckClusterSettingsFresh(ctx))
	s.clusterSettingsLastSuccess.Store(time.Now().Add(-time.Minute).Unix())
	require.Error(t, s.CheckClusterSettingsFresh(ctx))
}
