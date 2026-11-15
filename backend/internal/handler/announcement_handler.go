package handler

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AnnouncementHandler handles user announcement operations
type AnnouncementHandler struct {
	announcementService *service.AnnouncementService
}

// NewAnnouncementHandler creates a new user announcement handler
func NewAnnouncementHandler(announcementService *service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{
		announcementService: announcementService,
	}
}

// List handles listing announcements visible to current user
// GET /api/v1/announcements
func (h *AnnouncementHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	unreadOnly := parseBoolQuery(c.Query("unread_only"))

	items, err := h.announcementService.ListForUser(c.Request.Context(), subject.UserID, unreadOnly)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.UserAnnouncement, 0, len(items))
	for i := range items {
		out = append(out, *dto.UserAnnouncementFromService(&items[i]))
	}
	response.Success(c, out)
}

// MarkRead marks an announcement as read for current user
// POST /api/v1/announcements/:id/read
func (h *AnnouncementHandler) MarkRead(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	announcementID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || announcementID <= 0 {
		response.BadRequest(c, "Invalid announcement ID")
		return
	}

	if err := h.announcementService.MarkRead(c.Request.Context(), subject.UserID, announcementID); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "ok"})
}

func (h *AnnouncementHandler) History(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	page, size := response.ParsePagination(c)
	items, total, unread, err := h.announcementService.HistoryForUser(
		c.Request.Context(), subject.UserID, parseBoolQuery(c.Query("unread_only")),
		pagination.PaginationParams{Page: page, PageSize: size},
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.UserAnnouncement, 0, len(items))
	for i := range items {
		out = append(out, *dto.UserAnnouncementFromService(&items[i]))
	}
	response.Success(c, gin.H{
		"items": out, "total": total, "page": page, "page_size": size,
		"pages": max(1, (total+int64(size)-1)/int64(size)), "unread_count": unread,
	})
}

func (h *AnnouncementHandler) GetByID(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid announcement ID")
		return
	}
	item, err := h.announcementService.GetForUser(c.Request.Context(), subject.UserID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.UserAnnouncementFromService(item))
}

func (h *AnnouncementHandler) MarkAllRead(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	updated, err := h.announcementService.MarkAllRead(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"updated": updated})
}

func (h *AnnouncementHandler) Click(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid announcement ID")
		return
	}
	url, err := h.announcementService.RecordClick(c.Request.Context(), subject.UserID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"action_url": url})
}

func (h *AnnouncementHandler) Preferences(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	groups, err := h.announcementService.MutedGroups(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"muted_group_ids": groups})
}

func (h *AnnouncementHandler) UpdatePreference(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	id, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid group ID")
		return
	}
	var body struct {
		Subscribed *bool `json:"subscribed" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "Invalid subscription preference")
		return
	}
	if err := h.announcementService.SetGroupSubscription(c.Request.Context(), subject.UserID, id, *body.Subscribed); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"group_id": id, "subscribed": *body.Subscribed})
}

func parseBoolQuery(v string) bool {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
