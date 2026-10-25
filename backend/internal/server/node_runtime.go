package server

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/clusterha"
	"github.com/Wei-Shaw/sub2api/internal/clusterlb"
	"github.com/Wei-Shaw/sub2api/internal/clusterops"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// NodeRuntime owns readiness and the lifetime of in-flight requests, including
// hijacked WebSockets which net/http.Server.Shutdown does not wait for.
type NodeRuntime struct {
	operations     *clusterops.Manager
	cfg            config.ClusterConfig
	checks         []func(context.Context) error
	draining       atomic.Bool
	active         atomic.Int64
	mu             sync.Mutex
	hijacked       map[*trackedNodeConn]struct{}
	closing        bool
	settingsStatus func() map[string]any
}

func ProvideNodeRuntime(cfg *config.Config, db *sql.DB, rdb *redis.Client) *NodeRuntime {
	n := newNodeRuntime(cfg.Cluster, func(ctx context.Context) error {
		if !cfg.Cluster.HAEnabled {
			return db.PingContext(ctx)
		}
		var readonly bool
		if err := db.QueryRowContext(ctx, "SELECT pg_is_in_recovery() OR current_setting('transaction_read_only')::boolean").Scan(&readonly); err != nil {
			return err
		}
		if readonly {
			return fmt.Errorf("database writer unavailable")
		}
		return nil
	}, func(ctx context.Context) error { return rdb.Ping(ctx).Err() })
	if cfg.Cluster.HAEnabled {
		n.checks = append(n.checks, func(context.Context) error { return n.checkProof() })
	}
	return n
}

func (n *NodeRuntime) checkProof() error {
	return clusterha.CheckProof(n.cfg.LeaseFile, n.cfg.LeaseToken, n.cfg.Role, time.Now())
}

func newNodeRuntime(cfg config.ClusterConfig, checks ...func(context.Context) error) *NodeRuntime {
	return &NodeRuntime{cfg: cfg, checks: checks, hijacked: make(map[*trackedNodeConn]struct{})}
}

func (n *NodeRuntime) BeginDrain() { n.draining.Store(true) }

func (n *NodeRuntime) Ready(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if n.operations.Enabled() && (!n.operations.Fresh() || n.operations.Control().Maintenance) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
		return
	}
	if n.draining.Load() || (c.Query("node") != "" && (!n.cfg.Enabled || c.Query("node") != n.cfg.NodeID)) ||
		(c.Query("role") != "" && (!n.cfg.Enabled || c.Query("role") != n.cfg.Role)) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	results := make(chan error, len(n.checks))
	for _, check := range n.checks {
		go func(check func(context.Context) error) { results <- check(ctx) }(check)
	}
	for range n.checks {
		select {
		case err := <-results:
			if err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
				return
			}
		case <-ctx.Done():
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
	}
	if n.draining.Load() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
		return
	}
	result := gin.H{"status": "ready"}
	if c.Query("details") == "1" && n.cfg.Enabled {
		result["node"] = n.cfg.NodeID
		result["role"] = n.cfg.Role
		result["active_requests"] = n.active.Load()
		if n.settingsStatus != nil {
			result["settings"] = n.settingsStatus()
		}
	}
	c.JSON(http.StatusOK, result)
}

func (n *NodeRuntime) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if n.cfg.HAEnabled && n.checkProof() != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"type": "ha_lease_unavailable", "message": "Node is waiting for cluster ownership"}})
			return
		}
		if !n.cfg.Enabled || c.Request.URL.Path == "/health" || c.Request.URL.Path == "/readyz" {
			c.Next()
			return
		}
		if n.cfg.Role == "worker" && !clusterlb.WorkerPath(c.Request.URL.Path) {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"type": "primary_required", "message": "This endpoint must be routed to the primary node"}})
			return
		}
		n.active.Add(1)
		defer n.active.Add(-1)
		c.Writer = &nodeResponseWriter{ResponseWriter: c.Writer, node: n}
		c.Next()
	}
}

// Drain lets load balancer health checks remove the node before closing the
// listener. Requests arriving during that propagation interval still complete.
func (n *NodeRuntime) Drain(ctx context.Context, srv *http.Server, propagation time.Duration) error {
	n.BeginDrain()
	timer := time.NewTimer(propagation)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
	}
	err := srv.Shutdown(ctx)
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for n.hasActiveRequests() {
		select {
		case <-ctx.Done():
			n.closeHijacked()
			_ = srv.Close()
			return ctx.Err()
		case <-ticker.C:
		}
	}
	if err != nil {
		n.closeHijacked()
		_ = srv.Close()
	}
	return err
}

func (n *NodeRuntime) hasActiveRequests() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.active.Load() > 0 || len(n.hijacked) > 0 || n.operations.BackgroundActive() > 0
}

func (n *NodeRuntime) closeHijacked() {
	n.mu.Lock()
	n.closing = true
	connections := make([]*trackedNodeConn, 0, len(n.hijacked))
	for conn := range n.hijacked {
		connections = append(connections, conn)
	}
	n.mu.Unlock()
	for _, conn := range connections {
		_ = conn.Close()
	}
}

type nodeResponseWriter struct {
	gin.ResponseWriter
	node *NodeRuntime
}

func (w *nodeResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, rw, err := w.ResponseWriter.Hijack()
	if err != nil {
		return nil, nil, err
	}
	tracked := &trackedNodeConn{Conn: conn, node: w.node}
	w.node.mu.Lock()
	if w.node.closing {
		w.node.mu.Unlock()
		_ = conn.Close()
		return nil, nil, net.ErrClosed
	}
	w.node.hijacked[tracked] = struct{}{}
	w.node.mu.Unlock()
	return tracked, rw, nil
}

type trackedNodeConn struct {
	net.Conn
	node *NodeRuntime
	once sync.Once
}

func (c *trackedNodeConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() {
		c.node.mu.Lock()
		delete(c.node.hijacked, c)
		c.node.mu.Unlock()
	})
	return err
}
