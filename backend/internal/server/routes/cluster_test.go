package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func TestClusterOperationsInheritAdminGate(t *testing.T) {
	r := gin.New()
	group := r.Group("/api/v1/admin")
	group.Use(func(c *gin.Context) { c.AbortWithStatus(401) })
	registerClusterRoutes(group, &handler.Handlers{})
	for _, v := range []struct{ method, path string }{{"GET", ""}, {"GET", "/requests"}, {"GET", "/preflight"}, {"PUT", "/nodes/us/control"}, {"POST", "/requests/request/review"}, {"GET", "/users/1/balance"}, {"POST", "/users/1/invalidate-balance"}} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(v.method, "/api/v1/admin/cluster"+v.path, nil))
		require.Equal(t, 401, w.Code, v.path)
	}
}
