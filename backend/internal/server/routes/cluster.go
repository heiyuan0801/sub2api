package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/gin-gonic/gin"
)

func registerClusterRoutes(group *gin.RouterGroup, h *handler.Handlers) {
	c := admin.NewClusterHandler(h.ClusterOps)
	r := group.Group("/cluster")
	r.GET("", c.Snapshot)
	r.GET("/preflight", c.Preflight)
	r.GET("/history", c.History)
	r.PUT("/nodes/:node/control", c.Update)
	r.GET("/requests", c.Requests)
	r.POST("/requests/:id/review", c.Review)
	r.GET("/users/:id/balance", c.Balance)
	r.POST("/users/:id/invalidate-balance", c.InvalidateBalance)
}
