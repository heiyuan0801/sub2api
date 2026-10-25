package middleware

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/clusterops"
	"github.com/gin-gonic/gin"
)

// ClusterGatewayOperations runs after authentication on every gateway alias.
func ClusterGatewayOperations(m *clusterops.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.Enabled() || (c.Request.Method == http.MethodGet && !strings.EqualFold(c.GetHeader("Upgrade"), "websocket")) || strings.HasSuffix(c.Request.URL.Path, "/count_tokens") {
			c.Next()
			return
		}
		key, ok := GetAPIKeyFromContext(c)
		if !ok || key == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if !m.Begin() {
			status := http.StatusTooManyRequests
			code := "node_capacity_exceeded"
			if m.Control().Maintenance || !m.Fresh() {
				status = http.StatusServiceUnavailable
				code = "node_draining"
			}
			c.Header("Retry-After", "3")
			c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"type": "overloaded_error", "code": code, "message": "Node temporarily unavailable; retry according to request idempotency semantics"}})
			return
		}
		defer func() { m.End(c.Writer.Status()) }()
		ctx, id, err := m.StartRequest(c.Request.Context(), key.UserID, key.ID, c.Request.URL.Path)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "request_journal_unavailable", "message": "Request could not be durably recorded"}})
			return
		}
		c.Request = c.Request.WithContext(m.WithMetrics(ctx))
		defer func() { m.FinishRequest(id, c.Writer.Status()) }()
		started := time.Now()
		defer func() {
			var phases [3]*float64
			for i, k := range []string{service.OpsAuthLatencyMsKey, service.OpsRoutingLatencyMsKey, service.OpsUpstreamLatencyMsKey} {
				if raw, ok := c.Get(k); ok {
					if v, ok := raw.(int64); ok {
						f := float64(v)
						phases[i] = &f
					}
				}
			}
			m.ObserveCompletion(time.Since(started), phases)
		}()
		c.Next()
	}
}
