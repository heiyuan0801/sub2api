package handler

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/aggregateretry"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// Local auth/validation/balance errors and partial billable responses never
// signal replay. The outer writer independently enforces the no-output boundary.
func markAggregateUpstreamFailure(c *gin.Context, failure *service.UpstreamFailoverError, streamStarted bool) {
	if c == nil || c.Request == nil || streamStarted || c.Writer.Written() || failure == nil ||
		!failure.ShouldRetryNextAccount() || failure.BoundUpstreamPassthrough || failure.IsCredentialFailure() ||
		failure.IsOpenAIRequestBodyTooLarge() || failure.Reason == service.OpenAIHTTPContinuationUnsupportedReason ||
		failure.Reason == service.OpenAIImagesInsufficientBalanceReason || service.IsOpenAISilentRefusalErrorBody(failure.ResponseBody) {
		return
	}
	state := aggregateretry.FromContext(c.Request.Context())
	if state == nil || !state.Enabled || state.BillingSealed() {
		return
	}
	switch failure.StatusCode {
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout, 529:
		state.Retryable = true
	}
}

// No request has been sent when the scheduler exhausts a temporarily blocked
// pool. This includes the next client request arriving during a fault cooldown.
func markAggregateSelectionFailure(c *gin.Context, err error) {
	if err != nil && !isOpsNoAvailableAccountError(err) {
		return
	}
	markAggregateUpstreamFailure(c, &service.UpstreamFailoverError{StatusCode: http.StatusServiceUnavailable}, false)
}
