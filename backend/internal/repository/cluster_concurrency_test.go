package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestClusterNodeStartupDoesNotDeleteLivePeerReservations(t *testing.T) {
	r := miniredis.RunT(t)
	clientA := redis.NewClient(&redis.Options{Addr: r.Addr()})
	clientB := redis.NewClient(&redis.Options{Addr: r.Addr()})
	t.Cleanup(func() { _ = clientA.Close(); _ = clientB.Close() })
	a, b := NewConcurrencyCache(clientA, 5, 30), NewConcurrencyCache(clientB, 5, 30)
	ctx := context.Background()
	ok, err := a.AcquireAccountSlot(ctx, 1, 1, "live-peer-request")
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = a.AcquireUserSlot(ctx, 2, 1, "live-peer-request")
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = a.IncrementWaitCount(ctx, 2, 5)
	require.NoError(t, err)
	require.True(t, ok)
	for _, role := range []string{"primary", "worker"} {
		service.ProvideConcurrencyService(b, nil, &config.Config{Cluster: config.ClusterConfig{Enabled: true, Role: role}})
		count, err := b.GetAccountConcurrency(ctx, 1)
		require.NoError(t, err)
		require.Equal(t, 1, count)
		wait, err := clientB.Get(ctx, waitQueueKey(2)).Int()
		require.NoError(t, err)
		require.Equal(t, 1, wait)
		ok, err := b.AcquireAccountSlot(ctx, 1, 1, "new-node-request")
		require.NoError(t, err)
		require.False(t, ok, "must not exceed shared concurrency")
	}
	require.NoError(t, a.ReleaseAccountSlot(ctx, 1, "live-peer-request"))
	ok, err = b.AcquireAccountSlot(ctx, 1, 1, "new-node-request")
	require.NoError(t, err)
	require.True(t, ok)
}
