package middleware

import (
	"net/http"
	"strings"

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
		c.Request = c.Request.WithContext(ctx)
		defer func() { m.FinishRequest(id, c.Writer.Status()) }()
		c.Next()
	}
}
