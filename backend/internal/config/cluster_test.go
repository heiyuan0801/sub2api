package config

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClusterConfigLoadAndSecrets(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("CLUSTER_ENABLED", "true")
	t.Setenv("CLUSTER_NODE_ID", "hk-worker")
	t.Setenv("CLUSTER_ROLE", "worker")
	t.Setenv("TOTP_ENCRYPTION_KEY", strings.Repeat("a", 64))
	cfg, err := LoadForBootstrap()
	require.NoError(t, err)
	require.True(t, cfg.IsClusterWorker())
	require.False(t, cfg.RunsBackgroundJobs())
	require.Equal(t, 10, cfg.Cluster.DrainSeconds)
	require.Equal(t, 120*time.Second, cfg.ShutdownTimeout())
	require.True(t, cfg.Cluster.StrictBilling())
	t.Setenv("CLUSTER_BILLING_FAILURE_MODE", "open")
	cfg, err = LoadForBootstrap()
	require.NoError(t, err)
	require.False(t, cfg.Cluster.StrictBilling())
	t.Setenv("JWT_SECRET", "")
	_, err = LoadForBootstrap()
	require.ErrorContains(t, err, "explicit shared")
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	t.Setenv("TOTP_ENCRYPTION_KEY", "")
	_, err = LoadForBootstrap()
	require.ErrorContains(t, err, "explicit shared")
}

func TestClusterValidation(t *testing.T) {
	valid := ClusterConfig{Enabled: true, NodeID: "us-primary", Role: "primary", DrainSeconds: 10, ShutdownTimeoutSeconds: 120}
	require.NoError(t, valid.Validate())
	for _, mutate := range []func(*ClusterConfig){
		func(c *ClusterConfig) { c.NodeID = "bad\nnode" },
		func(c *ClusterConfig) { c.NodeID = "" },
		func(c *ClusterConfig) { c.Role = "standby" },
		func(c *ClusterConfig) { c.DrainSeconds = -1 },
		func(c *ClusterConfig) { c.ShutdownTimeoutSeconds = 0 },
		func(c *ClusterConfig) { c.BillingFailureMode = "typo" },
	} {
		cfg := valid
		mutate(&cfg)
		require.Error(t, cfg.Validate())
	}
	cfg := &Config{}
	require.True(t, cfg.RunsBackgroundJobs())
	require.Equal(t, 5*time.Second, cfg.ShutdownTimeout())
	require.True(t, (*Config)(nil).RunsBackgroundJobs())
}
