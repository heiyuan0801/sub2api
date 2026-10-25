package repository

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestBuildRedisOptions(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:                "localhost",
			Port:                6379,
			Username:            "app-user",
			Password:            "secret",
			DB:                  2,
			DialTimeoutSeconds:  5,
			ReadTimeoutSeconds:  3,
			WriteTimeoutSeconds: 4,
			PoolSize:            100,
			MinIdleConns:        10,
		},
	}

	opts := buildRedisOptions(cfg)
	require.Equal(t, "localhost:6379", opts.Addr)
	require.Equal(t, "app-user", opts.Username)
	require.Equal(t, "secret", opts.Password)
	require.Equal(t, 2, opts.DB)
	require.Equal(t, 5*time.Second, opts.DialTimeout)
	require.Equal(t, 3*time.Second, opts.ReadTimeout)
	require.Equal(t, 4*time.Second, opts.WriteTimeout)
	require.Equal(t, 100, opts.PoolSize)
	require.Equal(t, 10, opts.MinIdleConns)
	require.Nil(t, opts.TLSConfig)

	// Test case with TLS enabled
	cfgTLS := &config.Config{
		Redis: config.RedisConfig{
			Host:      "localhost",
			EnableTLS: true,
		},
	}
	optsTLS := buildRedisOptions(cfgTLS)
	require.NotNil(t, optsTLS.TLSConfig)
	require.Equal(t, "localhost", optsTLS.TLSConfig.ServerName)
}

func TestBuildRedisSentinelOptions(t *testing.T) {
	c := &config.Config{Redis: config.RedisConfig{SentinelMasterName: "cache", SentinelAddrs: []string{"a:26379", "b:26379", "c:26379"}, SentinelUsername: "monitor", SentinelPassword: "monitor-password", Username: "app", Password: "app-password", DB: 4, EnableTLS: true, PoolSize: 32, ReadTimeoutSeconds: 2}}
	o := buildRedisFailoverOptions(c)
	require.Equal(t, "cache", o.MasterName)
	require.Len(t, o.SentinelAddrs, 3)
	require.Equal(t, "monitor-password", o.SentinelPassword)
	require.Equal(t, "app-password", o.Password)
	require.Equal(t, 4, o.DB)
	require.Equal(t, 32, o.PoolSize)
	require.Equal(t, -1, o.MaxRetries)
	require.Equal(t, 2*time.Second, o.ReadTimeout)
	require.NotNil(t, o.TLSConfig)
	require.Empty(t, o.TLSConfig.ServerName, "TLS must verify each discovered host, not the obsolete standalone host")
}
