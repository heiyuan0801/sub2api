package admin

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AnnouncementHandler handles admin announcement management
type AnnouncementHandler struct {
	announcementService *service.AnnouncementService
}

// NewAnnouncementHandler creates a new admin announcement handler
func NewAnnouncementHandler(announcementService *service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{
		announcementService: announcementService,
	}
}

type CreateAnnouncementRequest struct {
	Title      string                        `json:"title" binding:"required"`
	Content    string                        `json:"content" binding:"required"`
	Status     string                        `json:"status" binding:"omitempty,oneof=draft active archived"`
	NotifyMode string                        `json:"notify_mode" binding:"omitempty,oneof=silent popup"`
	Targeting  service.AnnouncementTargeting `json:"targeting"`
	StartsAt   *int64                        `json:"starts_at"` // Unix seconds, 0/empty = immediate
	EndsAt     *int64                        `json:"ends_at"`   // Unix seconds, 0/empty = never
	Kind       string                        `json:"kind"`
	PlanID     *int64                        `json:"plan_id"`
	GroupID    *int64                        `json:"group_id"`
	Audience   *service.AnnouncementAudience `json:"audience"`
}

type UpdateAnnouncementRequest struct {
	Title      *string                        `json:"title"`
	Content    *string                        `json:"content"`
	Status     *string                        `json:"status" binding:"omitempty,oneof=draft active archived"`
	NotifyMode *string                        `json:"notify_mode" binding:"omitempty,oneof=silent popup"`
	Targeting  *service.AnnouncementTargeting `json:"targeting"`
	StartsAt   *int64                         `json:"starts_at"` // Unix seconds, 0 = clear
	EndsAt     *int64                         `json:"ends_at"`   // Unix seconds, 0 = clear
	Kind       *string                        `json:"kind"`
	PlanID     *int64                         `json:"plan_id"`
	GroupID    *int64                         `json:"group_id"`
	Audience   *service.AnnouncementAudience  `json:"audience"`
}

// List handles listing announcements with filters
// GET /api/v1/admin/announcements
func (h *AnnouncementHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	status := strings.TrimSpace(c.Query("status"))
	search := strings.TrimSpace(c.Query("search"))
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")
	if len(search) > 200 {
		search = search[:200]
	}

	params := pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	items, paginationResult, err := h.announcementService.List(
		c.Request.Context(),
		params,
		service.AnnouncementListFilters{Status: status, Search: search},
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.Announcement, 0, len(items))
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	publications, err := h.announcementService.Publications(c.Request.Context(), ids)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	for i := range items {
		item := dto.AnnouncementFromService(&items[i])
		if publication, ok := publications[items[i].ID]; ok {
			item.ApplyPublication(publication)
		}
		out = append(out, *item)
	}
	response.Paginated(c, out, paginationResult.Total, page, pageSize)
}

// GetByID handles getting an announcement by ID
// GET /api/v1/admin/announcements/:id
func (h *AnnouncementHandler) GetByID(c *gin.Context) {
	announcementID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || announcementID <= 0 {
		response.BadRequest(c, "Invalid announcement ID")
		return
	}

	item, err := h.announcementService.GetByID(c.Request.Context(), announcementID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	h.respondAnnouncement(c, item)
}

// Create handles creating a new announcement
// POST /api/v1/admin/announcements
func (h *AnnouncementHandler) Create(c *gin.Context) {
	var req CreateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	input := &service.CreateAnnouncementInput{
		Title:      req.Title,
		Content:    req.Content,
		Status:     req.Status,
		NotifyMode: req.NotifyMode,
		Targeting:  req.Targeting,
		ActorID:    &subject.UserID,
	}
	if req.Kind != "" || req.PlanID != nil || req.GroupID != nil || req.Audience != nil {
		if req.Audience == nil {
			response.ErrorFrom(c, service.ErrAnnouncementInvalidPublication)
			return
		}
		input.Publication = &service.AnnouncementPublication{Kind: req.Kind, PlanID: req.PlanID, GroupID: req.GroupID, Audience: *req.Audience}
	}

	if req.StartsAt != nil && *req.StartsAt > 0 {
		t := time.Unix(*req.StartsAt, 0)
		input.StartsAt = &t
	}
	if req.EndsAt != nil && *req.EndsAt > 0 {
		t := time.Unix(*req.EndsAt, 0)
		input.EndsAt = &t
	}

	created, err := h.announcementService.Create(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	h.respondAnnouncement(c, created)
}

// Update handles updating an announcement
// PUT /api/v1/admin/announcements/:id
func (h *AnnouncementHandler) Update(c *gin.Context) {
	announcementID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || announcementID <= 0 {
		response.BadRequest(c, "Invalid announcement ID")
		return
	}

	var req UpdateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	input := &service.UpdateAnnouncementInput{
		Title:      req.Title,
		Content:    req.Content,
		Status:     req.Status,
		NotifyMode: req.NotifyMode,
		Targeting:  req.Targeting,
		ActorID:    &subject.UserID,
		Kind:       req.Kind,
		Audience:   req.Audience,
	}
	if req.PlanID != nil {
		planID := req.PlanID
		if *req.PlanID == 0 {
			planID = nil
		}
		input.PlanID = &planID
	}
	if req.GroupID != nil {
		groupID := req.GroupID
		if *req.GroupID == 0 {
			groupID = nil
		}
		input.GroupID = &groupID
	}

	if req.StartsAt != nil {
		if *req.StartsAt == 0 {
			var cleared *time.Time = nil
			input.StartsAt = &cleared
		} else {
			t := time.Unix(*req.StartsAt, 0)
			ptr := &t
			input.StartsAt = &ptr
		}
	}

	if req.EndsAt != nil {
		if *req.EndsAt == 0 {
			var cleared *time.Time = nil
			input.EndsAt = &cleared
		} else {
			t := time.Unix(*req.EndsAt, 0)
			ptr := &t
			input.EndsAt = &ptr
		}
	}

	updated, err := h.announcementService.Update(c.Request.Context(), announcementID, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	h.respondAnnouncement(c, updated)
}

func (h *AnnouncementHandler) respondAnnouncement(c *gin.Context, item *service.Announcement) {
	publications, err := h.announcementService.Publications(c.Request.Context(), []int64{item.ID})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := dto.AnnouncementFromService(item)
	if publication, ok := publications[item.ID]; ok {
		out.ApplyPublication(publication)
	}
	response.Success(c, out)
}

func adminAnnouncementID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid announcement ID")
		return 0, false
	}
	return id, true
}

func (h *AnnouncementHandler) Publish(c *gin.Context) {
	var req struct {
		StartsAt int64 `json:"starts_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.StartsAt < 0 {
		response.ErrorFrom(c, service.ErrAnnouncementInvalidSchedule)
		return
	}
	var startsAt *time.Time
	if req.StartsAt > 0 {
		at := time.Unix(req.StartsAt, 0)
		startsAt = &at
	}
	id, ok := adminAnnouncementID(c)
	if !ok {
		return
	}
	actor, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	middleware2.SetAuditAction(c, "admin.announcements.publish")
	if err := h.announcementService.Publish(c.Request.Context(), id, actor.UserID, startsAt); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.GetByID(c)
}

func (h *AnnouncementHandler) Withdraw(c *gin.Context) {
	id, ok := adminAnnouncementID(c)
	if !ok {
		return
	}
	actor, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	middleware2.SetAuditAction(c, "admin.announcements.withdraw")
	if err := h.announcementService.Withdraw(c.Request.Context(), id, actor.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.GetByID(c)
}

func (h *AnnouncementHandler) Retry(c *gin.Context) {
	id, ok := adminAnnouncementID(c)
	if !ok {
		return
	}
	actor, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	middleware2.SetAuditAction(c, "admin.announcements.retry")
	if err := h.announcementService.RetryDelivery(c.Request.Context(), id, actor.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.GetByID(c)
}

func (h *AnnouncementHandler) Stats(c *gin.Context) {
	id, ok := adminAnnouncementID(c)
	if !ok {
		return
	}
	stats, err := h.announcementService.DeliveryStats(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

func (h *AnnouncementHandler) Deliveries(c *gin.Context) {
	id, ok := adminAnnouncementID(c)
	if !ok {
		return
	}
	page, size := response.ParsePagination(c)
	items, total, err := h.announcementService.ListDeliveries(c.Request.Context(), id, pagination.PaginationParams{Page: page, PageSize: size})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, size)
}

// Delete handles deleting an announcement
// DELETE /api/v1/admin/announcements/:id
func (h *AnnouncementHandler) Delete(c *gin.Context) {
	announcementID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || announcementID <= 0 {
		response.BadRequest(c, "Invalid announcement ID")
		return
	}

	if err := h.announcementService.Delete(c.Request.Context(), announcementID); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Announcement deleted successfully"})
}

// ListReadStatus handles listing users read status for an announcement
// GET /api/v1/admin/announcements/:id/read-status
func (h *AnnouncementHandler) ListReadStatus(c *gin.Context) {
	announcementID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || announcementID <= 0 {
		response.BadRequest(c, "Invalid announcement ID")
		return
	}

	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.DefaultQuery("sort_by", "email"),
		SortOrder: c.DefaultQuery("sort_order", "asc"),
	}
	search := strings.TrimSpace(c.Query("search"))
	if len(search) > 200 {
		search = search[:200]
	}

	items, paginationResult, err := h.announcementService.ListUserReadStatus(
		c.Request.Context(),
		announcementID,
		params,
		search,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Paginated(c, adminAnnouncementReadResponse(c, items), paginationResult.Total, page, pageSize)
}
