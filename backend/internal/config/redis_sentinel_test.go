package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedisSentinelConfiguration(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("REDIS_SENTINEL_MASTER_NAME", "sub2api")
	t.Setenv("REDIS_SENTINEL_ADDRS", "127.0.0.1:26379,127.0.0.1:26380,127.0.0.1:26381")
	c, err := LoadForBootstrap()
	require.NoError(t, err)
	require.Len(t, c.Redis.SentinelAddrs, 3)
	c.Redis.SentinelAddrs[2] = c.Redis.SentinelAddrs[0]
	require.Error(t, c.Redis.ValidateSentinel())
	c.Redis.SentinelAddrs = []string{"127.0.0.1:26379"}
	require.Error(t, c.Redis.ValidateSentinel())
}
