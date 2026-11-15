package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *PaymentHandler) ChangePlanPrice(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	actor, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	var req service.PlanPriceChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	middleware.SetAuditAction(c, "admin.payment.plan.change_price")
	result, err := h.configService.ChangePlanPrice(c.Request.Context(), id, actor.UserID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, result)
}

func (h *PaymentHandler) ListPlanPriceChanges(c *gin.Context) {
	planID, errPlan := strconv.ParseInt(c.DefaultQuery("plan_id", "0"), 10, 64)
	beforeID, errBefore := strconv.ParseInt(c.DefaultQuery("before_id", "0"), 10, 64)
	if errPlan != nil || errBefore != nil || planID < 0 || beforeID < 0 {
		response.BadRequest(c, "Invalid price history filter")
		return
	}
	result, err := h.configService.ListPlanPriceChanges(c.Request.Context(), planID, beforeID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *PaymentHandler) CancelPlanPriceChange(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	actor, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	middleware.SetAuditAction(c, "admin.payment.plan.cancel_price_change")
	if err := h.configService.CancelPlanPriceChange(c.Request.Context(), id, actor.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"cancelled": true})
}

func (h *PaymentHandler) ReorderPlans(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	middleware.SetAuditAction(c, "admin.payment.plan.reorder")
	if err := h.configService.ReorderPlans(c.Request.Context(), req.IDs); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"reordered": true})
}
