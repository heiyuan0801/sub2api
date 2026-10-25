package repository

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestClusterSharedSessionAcrossInstances(t *testing.T) {
	r := miniredis.RunT(t)
	aClient, bClient := redis.NewClient(&redis.Options{Addr: r.Addr()}), redis.NewClient(&redis.Options{Addr: r.Addr()})
	t.Cleanup(func() { require.NoError(t, aClient.Close()) })
	t.Cleanup(func() { require.NoError(t, bClient.Close()) })
	cfg := &config.Config{Cluster: config.ClusterConfig{Enabled: true}}
	aCache, bCache := NewGatewayCache(aClient), NewGatewayCache(bClient)
	a, b := service.ProvideDigestSessionStore(aCache, cfg), service.ProvideDigestSessionStore(bCache, cfg)
	a.Save(1, "user-scope", "a-b", "session-1", 99, "")
	id, account, chain, ok := b.Find(1, "user-scope", "a-b-c")
	require.True(t, ok)
	require.Equal(t, "session-1", id)
	require.EqualValues(t, 99, account)
	require.Equal(t, "a-b", chain)
	_, _, _, ok = b.Find(2, "user-scope", "a-b-c")
	require.False(t, ok)
	wa, wb := service.NewClusterOpenAIWSStateStore(aCache, "a"), service.NewClusterOpenAIWSStateStore(bCache, "b")
	wa.BindSessionTurnState(1, "key-scoped-session", "opaque-turn", time.Minute)
	value, ok := wb.GetSessionTurnState(1, "key-scoped-session")
	require.True(t, ok)
	require.Equal(t, "opaque-turn", value)
	wb.DeleteSessionTurnState(1, "key-scoped-session")
	_, ok = wa.GetSessionTurnState(1, "key-scoped-session")
	require.False(t, ok, "must not resurrect stale local turn state")
	wa.BindResponseConn("r1", "local-socket", time.Minute)
	require.NoError(t, wa.BindResponseAccount(context.Background(), 1, "r1", 42, time.Minute))
	bound, err := wb.GetResponseAccount(context.Background(), 1, "r1")
	require.NoError(t, err)
	require.EqualValues(t, 42, bound)
	_, ok = wb.GetResponseConn("r1")
	require.False(t, ok, "socket handles are never shared")
	r.FastForward(6 * time.Minute)
	_, _, _, ok = b.Find(1, "user-scope", "a-b-c")
	require.False(t, ok)
}

type clusterSubscriptionRepo struct {
	service.UserSubscriptionRepository
	expiry atomic.Int64
}

func (r *clusterSubscriptionRepo) GetActiveByUserIDAndGroupID(context.Context, int64, int64) (*service.UserSubscription, error) {
	return &service.UserSubscription{ID: 1, UserID: 1, GroupID: 2, ExpiresAt: time.Unix(r.expiry.Load(), 0)}, nil
}

func TestClusterSubscriptionInvalidationAndMissedEventRecovery(t *testing.T) {
	r := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: r.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	cache := NewBillingCache(client)
	cfg := &config.Config{}
	cfg.SubscriptionCache.L1Size = 128
	cfg.SubscriptionCache.L1TTLSeconds = 300
	billing := service.NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, nil)
	defer billing.Stop()
	repo := &clusterSubscriptionRepo{}
	repo.expiry.Store(100)
	a, b := service.NewSubscriptionService(nil, repo, billing, nil, cfg), service.NewSubscriptionService(nil, repo, billing, nil, cfg)
	defer a.Stop()
	defer b.Stop()
	ctx := context.Background()
	require.Eventually(t, func() bool {
		n, _ := client.PubSubNumSub(ctx, subCacheInvalidateChannel).Result()
		return n[subCacheInvalidateChannel] == 2
	}, time.Second, 10*time.Millisecond)
	first, err := b.GetActiveSubscription(ctx, 1, 2)
	require.NoError(t, err)
	require.EqualValues(t, 100, first.ExpiresAt.Unix())
	repo.expiry.Store(200)
	require.NoError(t, billing.InvalidateSubscription(ctx, 1, 2))
	require.Eventually(t, func() bool { s, e := b.GetActiveSubscription(ctx, 1, 2); return e == nil && s.ExpiresAt.Unix() == 200 }, 2*time.Second, 10*time.Millisecond)
	// Simulate an update whose Pub/Sub delivery was lost while B was disconnected.
	repo.expiry.Store(300)
	require.NoError(t, client.Incr(ctx, "cluster:subscription:revision").Err())
	require.Eventually(t, func() bool { s, e := b.GetActiveSubscription(ctx, 1, 2); return e == nil && s.ExpiresAt.Unix() == 300 }, 7*time.Second, 10*time.Millisecond)
}
