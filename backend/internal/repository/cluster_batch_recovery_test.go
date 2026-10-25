//go:build unit

package repository

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestClusterBatchQueueRepairDoesNotDuplicateOrStealWork(t *testing.T) {
	ctx := context.Background()
	q, _ := newBatchImageQueueTest(t)
	id := "imgbatch_repair"
	require.NoError(t, q.rdb.Set(ctx, q.inflightPrefix+id, id, time.Hour).Err())
	added, err := q.EnsureQueued(ctx, id)
	require.NoError(t, err)
	require.True(t, added)
	added, err = q.EnsureQueued(ctx, id)
	require.NoError(t, err)
	require.False(t, added)
	require.Equal(t, int64(1), q.rdb.LLen(ctx, q.readyKey).Val())
	for _, kind := range []string{"active", "delayed", "locked"} {
		id := "imgbatch_" + kind
		switch kind {
		case "active":
			require.NoError(t, q.rdb.ZAdd(ctx, q.activeKey, redis.Z{Score: float64(time.Now().UnixMilli()), Member: id}).Err())
		case "delayed":
			require.NoError(t, q.rdb.ZAdd(ctx, q.delayedKey, redis.Z{Score: float64(time.Now().UnixMilli()), Member: id}).Err())
		case "locked":
			require.NoError(t, q.rdb.Set(ctx, q.lockPrefix+id, "owner", time.Minute).Err())
		}
		added, err = q.EnsureQueued(ctx, id)
		require.NoError(t, err)
		require.False(t, added)
	}
	_, err = q.EnsureQueued(ctx, "invalid")
	require.Error(t, err)
}
