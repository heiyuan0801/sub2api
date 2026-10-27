package handler

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/aggregateretry"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAggregateRetryProtocolSignals(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, protocol := range []string{"openai", "messages", "claude", "chat-adapter", "responses-adapter", "gemini"} {
		for _, status := range []int{400, 401, 403, 429, 500, 502, 503, 504, 529} {
			state := &aggregateretry.State{Enabled: true}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(aggregateretry.WithState(context.Background(), state))
			failure := &service.UpstreamFailoverError{StatusCode: status}
			switch protocol {
			case "openai":
				(&OpenAIGatewayHandler{}).handleFailoverExhausted(c, failure, false)
			case "messages":
				(&OpenAIGatewayHandler{}).handleAnthropicFailoverExhausted(c, failure, false)
			case "claude":
				(&GatewayHandler{}).handleFailoverExhausted(c, failure, service.PlatformAnthropic, false)
			case "chat-adapter":
				(&GatewayHandler{}).handleCCFailoverExhausted(c, failure, false)
			case "responses-adapter":
				(&GatewayHandler{}).handleResponsesFailoverExhausted(c, failure, false)
			case "gemini":
				(&GatewayHandler{}).handleGeminiFailoverExhausted(c, failure)
			}
			require.Equal(t, status == 429 || status >= 500, state.Retryable, "protocol=%s status=%d", protocol, status)
		}
	}
}

func TestAggregateRetryNeverReplaysBillableOrBoundFailures(t *testing.T) {
	for _, mode := range []string{"usage", "stream", "written", "stop", "bound", "credentials", "body-size", "image-balance"} {
		t.Run(mode, func(t *testing.T) {
			state := &aggregateretry.State{Enabled: true}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(aggregateretry.WithState(context.Background(), state))
			failure := &service.UpstreamFailoverError{StatusCode: 503}
			switch mode {
			case "usage":
				_, abandon := wrapUsageRecordTaskContext(c.Request.Context(), func(context.Context) {})
				abandon()
			case "written":
				c.Writer.WriteHeaderNow()
			case "stop":
				failure.NextAccountAction = service.NextAccountStop
			case "bound":
				failure.BoundUpstreamPassthrough = true
			case "credentials":
				failure.Stage = service.GatewayFailureStageAccountAuth
			case "body-size":
				failure.StatusCode = 413
			case "image-balance":
				failure.Reason = service.OpenAIImagesInsufficientBalanceReason
			}
			markAggregateUpstreamFailure(c, failure, mode == "stream")
			require.False(t, state.Retryable)
		})
	}
}

func TestAggregateRetrySelectionFailure(t *testing.T) {
	for _, tc := range []struct {
		err   error
		retry bool
	}{{service.ErrNoAvailableAccounts, true}, {nil, true}, {errors.New("database unavailable"), false}, {context.Canceled, false}} {
		state := &aggregateretry.State{Enabled: true}
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(aggregateretry.WithState(context.Background(), state))
		markAggregateSelectionFailure(c, tc.err)
		require.Equal(t, tc.retry, state.Retryable)
	}
}
