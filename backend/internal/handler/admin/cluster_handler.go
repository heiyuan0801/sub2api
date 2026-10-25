package admin

import (
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/clusterops"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ClusterHandler struct{ manager *clusterops.Manager }

func NewClusterHandler(m *clusterops.Manager) *ClusterHandler { return &ClusterHandler{m} }
func (h *ClusterHandler) Snapshot(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if h.manager == nil {
		response.Success(c, clusterops.Snapshot{Nodes: []clusterops.Node{}, Alerts: []clusterops.Alert{}, Events: []clusterops.Event{}})
		return
	}
	v, e := h.manager.Snapshot(c.Request.Context())
	if e != nil {
		clusterError(c, e)
		return
	}
	response.Success(c, v)
}
func (h *ClusterHandler) Preflight(c *gin.Context) {
	if h.manager == nil {
		clusterError(c, clusterops.ErrUnavailable)
		return
	}
	v, e := h.manager.Preflight(c.Request.Context())
	if e != nil {
		clusterError(c, e)
		return
	}
	response.Success(c, v)
}
func (h *ClusterHandler) Update(c *gin.Context) {
	var v clusterops.Control
	if e := c.ShouldBindJSON(&v); e != nil {
		response.BadRequest(c, "Invalid node control")
		return
	}
	if v.MaxConcurrency < 1 || v.MaxConcurrency > 20000 || v.Weight < 1 || v.Weight > 256 || len(v.Reason) < 3 || len(v.Reason) > 500 {
		response.BadRequest(c, "Capacity must be 1-20000, weight 1-256 and reason 3-500 bytes")
		return
	}
	s, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}
	if h.manager == nil {
		clusterError(c, clusterops.ErrUnavailable)
		return
	}
	if e := h.manager.UpdateControl(c.Request.Context(), c.Param("node"), v, s.UserID); e != nil {
		clusterError(c, e)
		return
	}
	response.Success(c, gin.H{"saved": true, "status": "pending_node_acknowledgement"})
}
func (h *ClusterHandler) Requests(c *gin.Context) {
	if h.manager == nil {
		response.Success(c, []clusterops.Request{})
		return
	}
	var before time.Time
	if raw := c.Query("before"); raw != "" {
		var e error
		before, e = time.Parse(time.RFC3339Nano, raw)
		if e != nil {
			response.BadRequest(c, "Invalid request cursor")
			return
		}
	}
	v, e := h.manager.Requests(c.Request.Context(), c.Query("review") == "true", before)
	if e != nil {
		clusterError(c, e)
		return
	}
	response.Success(c, v)
}
func (h *ClusterHandler) Review(c *gin.Context) {
	if _, e := uuid.Parse(c.Param("id")); e != nil {
		response.BadRequest(c, "Invalid request ID")
		return
	}
	var v struct {
		Note string `json:"note"`
	}
	if e := c.ShouldBindJSON(&v); e != nil || len(v.Note) < 3 || len(v.Note) > 1000 {
		response.BadRequest(c, "Review note must contain 3-1000 bytes")
		return
	}
	s, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}
	if h.manager == nil {
		clusterError(c, clusterops.ErrUnavailable)
		return
	}
	if e := h.manager.ReviewRequest(c.Request.Context(), c.Param("id"), v.Note, s.UserID); e != nil {
		clusterError(c, e)
		return
	}
	response.Success(c, gin.H{"reviewed": true, "money_changed": false})
}
func (h *ClusterHandler) Balance(c *gin.Context) {
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id <= 0 {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	if h.manager == nil {
		clusterError(c, clusterops.ErrUnavailable)
		return
	}
	v, e := h.manager.BalanceStatus(c.Request.Context(), id)
	if e != nil {
		clusterError(c, e)
		return
	}
	response.Success(c, v)
}
func (h *ClusterHandler) InvalidateBalance(c *gin.Context) {
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id <= 0 {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	s, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}
	if h.manager == nil {
		clusterError(c, clusterops.ErrUnavailable)
		return
	}
	if e = h.manager.InvalidateBalance(c.Request.Context(), id, s.UserID); e != nil {
		clusterError(c, e)
		return
	}
	response.Success(c, gin.H{"invalidated": true, "money_changed": false})
}
func clusterError(c *gin.Context, e error) {
	switch {
	case errors.Is(e, clusterops.ErrConflict), errors.Is(e, clusterops.ErrLastNode):
		response.ErrorWithDetails(c, 409, e.Error(), "CLUSTER_CONTROL_CONFLICT", nil)
	case errors.Is(e, sql.ErrNoRows):
		response.NotFound(c, "Record not found")
	case errors.Is(e, clusterops.ErrUnavailable):
		response.ErrorWithDetails(c, 503, "Cluster operations are unavailable", "CLUSTER_UNAVAILABLE", nil)
	default:
		response.InternalError(c, "Unable to complete cluster operation")
	}
}
