package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"time"
)

// ClusterConfig is opt-in. One primary owns management and scheduled work;
// workers serve gateway traffic against the same PostgreSQL and Redis databases.
type ClusterConfig struct {
	Enabled                bool   `mapstructure:"enabled"`
	NodeID                 string `mapstructure:"node_id"`
	Role                   string `mapstructure:"role"`
	DrainSeconds           int    `mapstructure:"drain_seconds"`
	ShutdownTimeoutSeconds int    `mapstructure:"shutdown_timeout_seconds"`
	// Empty is strict when clustered; "open" is an explicit availability tradeoff.
	BillingFailureMode string `mapstructure:"billing_failure_mode"`
	HAEnabled          bool   `mapstructure:"ha_enabled"`
	LeaseFile          string `mapstructure:"lease_file"`
	LeaseToken         string `mapstructure:"lease_token"`
	MaxConcurrency     int64  `mapstructure:"max_concurrency"`
}

var clusterNodeIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

func (c *Config) IsClusterWorker() bool {
	return c != nil && c.Cluster.Enabled && c.Cluster.Role == "worker"
}

func (c *Config) RunsBackgroundJobs() bool { return !c.IsClusterWorker() }

func (c *Config) ShutdownTimeout() time.Duration {
	if c != nil && c.Cluster.Enabled {
		return time.Duration(c.Cluster.ShutdownTimeoutSeconds) * time.Second
	}
	return 5 * time.Second
}

func (c ClusterConfig) Validate() error {
	if c.HAEnabled && (!c.Enabled || !filepath.IsAbs(c.LeaseFile) || c.LeaseToken == "" || c.BillingFailureMode == "open") {
		return fmt.Errorf("HA requires cluster mode, strict billing and a private supervisor lease proof")
	}
	if !c.Enabled {
		return nil
	}
	if !clusterNodeIDPattern.MatchString(c.NodeID) {
		return fmt.Errorf("cluster.node_id must be 1-64 letters, digits, underscores or hyphens, starting with a letter or digit")
	}
	if c.Role != "primary" && c.Role != "worker" {
		return fmt.Errorf("cluster.role must be primary or worker")
	}
	if c.DrainSeconds < 0 || c.DrainSeconds > 300 {
		return fmt.Errorf("cluster.drain_seconds must be between 0 and 300")
	}
	if c.ShutdownTimeoutSeconds < 5 || c.ShutdownTimeoutSeconds > 3600 {
		return fmt.Errorf("cluster.shutdown_timeout_seconds must be between 5 and 3600")
	}
	if c.BillingFailureMode != "" && c.BillingFailureMode != "closed" && c.BillingFailureMode != "open" {
		return fmt.Errorf("cluster.billing_failure_mode must be closed or open")
	}
	if c.MaxConcurrency < 0 || c.MaxConcurrency > 20000 {
		return fmt.Errorf("cluster.max_concurrency must be between 0 and 20000")
	}
	return nil
}

func (c ClusterConfig) StrictBilling() bool { return c.Enabled && c.BillingFailureMode != "open" }
