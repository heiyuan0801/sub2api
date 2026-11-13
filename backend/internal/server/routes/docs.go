package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func docsNoStore(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Writer.Header().Add("Vary", "Authorization")
	c.Header("X-Robots-Tag", "noindex")
	c.Next()
}

func RegisterDocsRoutes(v1 *gin.RouterGroup, h *handler.DocsHandler, optionalJWT middleware.OptionalJWTAuthMiddleware, settings *service.SettingService, limiter *middleware.PanelRateLimiter) {
	if h == nil {
		return
	}
	docs := v1.Group("/docs", docsNoStore, limiter.PublicIP(), gin.HandlerFunc(optionalJWT), middleware.BackendModeUserGuard(settings))
	docs.GET("", h.Navigation)
	docs.GET("/search", h.Search)
	docs.GET("/articles/:slug", h.Publication)
	docs.GET("/images/:image", h.Image)
}

func registerAdminDocsRoutes(admin *gin.RouterGroup, h *handler.DocsHandler) {
	if h == nil {
		return
	}
	docs := admin.Group("/docs", docsNoStore)
	docs.GET("", h.AdminNavigation)
	docs.POST("/categories", h.SaveCategory)
	docs.PUT("/categories/order", h.Sort)
	docs.PUT("/categories/:id", h.SaveCategory)
	docs.DELETE("/categories/:id", h.DeleteCategory)
	docs.PUT("/categories/:id/articles/order", h.Sort)
	docs.POST("/articles", h.SaveArticle)
	docs.GET("/articles/:id", h.Article)
	docs.PUT("/articles/:id", h.SaveArticle)
	docs.DELETE("/articles/:id", h.DeleteArticle)
	docs.PUT("/articles/:id/publication", h.ChangePublication)
	docs.POST("/articles/:id/images", h.UploadImage)
	docs.GET("/articles/:id/images/:image", h.Image)
}
