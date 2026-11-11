package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type RewardActivityHandler struct {
	service *service.RewardActivityService
}

func NewRewardActivityHandler(service *service.RewardActivityService) *RewardActivityHandler {
	return &RewardActivityHandler{service: service}
}

func (h *RewardActivityHandler) Status(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	status, err := h.service.Status(c.Request.Context(), subject.UserID)
	if !response.ErrorFrom(c, err) {
		response.Success(c, status)
	}
}

func (h *RewardActivityHandler) CheckIn(c *gin.Context) {
	h.claim(c, service.RewardCheckIn)
}

func (h *RewardActivityHandler) Draw(c *gin.Context) {
	h.claim(c, service.RewardWheel)
}

func (h *RewardActivityHandler) claim(c *gin.Context, kind string) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var input struct {
		RequestID string `json:"request_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "A request ID is required")
		return
	}
	claim, err := h.service.Claim(c.Request.Context(), subject.UserID, kind, input.RequestID)
	if !response.ErrorFrom(c, err) {
		response.Success(c, claim)
	}
}

func (h *RewardActivityHandler) GetConfig(c *gin.Context) {
	config, err := h.service.GetConfig(c.Request.Context())
	if !response.ErrorFrom(c, err) {
		response.Success(c, config)
	}
}

func (h *RewardActivityHandler) UpdateConfig(c *gin.Context) {
	var config service.RewardActivityConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.BadRequest(c, "Invalid reward activity configuration")
		return
	}
	if !response.ErrorFrom(c, h.service.SaveConfig(c.Request.Context(), config)) {
		response.Success(c, config)
	}
}

func (h *RewardActivityHandler) RecentClaims(c *gin.Context) {
	claims, err := h.service.RecentClaims(c.Request.Context())
	if !response.ErrorFrom(c, err) {
		response.Success(c, claims)
	}
}
