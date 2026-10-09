package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestNonOpenAIChatCompletionsRejectsGPTImageModelsBeforeScheduling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, model := range []string{"gpt-image-1", "gpt-image-1.5", "gpt-image-2"} {
		for _, tc := range []struct {
			name string
			call func(*gin.Context)
		}{
			{
				name: "gateway",
				call: (&GatewayHandler{}).ChatCompletions,
			},
		} {
			t.Run(tc.name+"/"+model, func(t *testing.T) {
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				body := []byte(`{"model":"` + model + `","messages":[{"role":"user","content":"draw"}]}`)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
				setImageChatTestAuth(c)

				tc.call(c)

				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Equal(t, "invalid_request_error", gjson.Get(recorder.Body.String(), "error.type").String())
				require.Contains(t, gjson.Get(recorder.Body.String(), "error.message").String(), "Chat Completions")
				_, selected := c.Get(opsAccountIDKey)
				require.False(t, selected, "rejection must happen before account selection")
			})
		}
	}
}

func TestOpenAIChatCompletionsImageModelReachesAccountRouting(t *testing.T) {
	var acquireCalls atomic.Int64
	cache := &concurrencyCacheMock{
		acquireUserSlotFn: func(context.Context, int64, int, string) (bool, error) {
			acquireCalls.Add(1)
			return false, errors.New("stop before scheduling in handler test")
		},
	}
	h := newOpenAIImageChatRejectionHandlerWithCache(t, cache)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(
		`{"model":"gpt-image-2","messages":[{"role":"user","content":"draw"}]}`,
	))
	setImageChatTestAuth(c)

	h.ChatCompletions(c)

	require.Equal(t, int64(1), acquireCalls.Load(), "image models must reach normal routing")
	require.NotContains(t, recorder.Body.String(), "This model is not supported on the Chat Completions endpoint")
}

func newOpenAIImageChatRejectionHandler(t *testing.T) *OpenAIGatewayHandler {
	t.Helper()
	return newOpenAIImageChatRejectionHandlerWithCache(t, &concurrencyCacheMock{})
}

func newOpenAIImageChatRejectionHandlerWithCache(t *testing.T, cache *concurrencyCacheMock) *OpenAIGatewayHandler {
	t.Helper()
	return &OpenAIGatewayHandler{
		gatewayService:      &service.OpenAIGatewayService{},
		billingCacheService: &service.BillingCacheService{},
		apiKeyService:       &service.APIKeyService{},
		concurrencyHelper:   NewConcurrencyHelper(service.NewConcurrencyService(cache), SSEPingFormatNone, time.Second),
	}
}

func setImageChatTestAuth(c *gin.Context) {
	apiKey := &service.APIKey{ID: 4348, UserID: 4348, User: &service.User{ID: 4348}}
	c.Set(string(middleware.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: 1})
}

func TestOpenAIChatCompletionsImageModelDisabledGroup(t *testing.T) {
	h := newOpenAIImageChatRejectionHandler(t)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"gpt-image-2","messages":[{"role":"user","content":"draw"}]}`))
	setImageChatTestAuth(c)
	value, _ := c.Get(string(middleware.ContextKeyAPIKey))
	apiKey := value.(*service.APIKey)
	apiKey.Group = &service.Group{AllowImageGeneration: false}
	h.ChatCompletions(c)
	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), service.ImageGenerationPermissionMessage())
}

func TestOpenAIChatCompletionsImageModelConcurrencyLimit(t *testing.T) {
	h := newOpenAIImageChatRejectionHandler(t)
	h.cfg = &config.Config{}
	h.cfg.Gateway.ImageConcurrency.Enabled = true
	h.cfg.Gateway.ImageConcurrency.MaxConcurrentRequests = 1
	h.imageLimiter = &imageConcurrencyLimiter{}
	release, acquired := h.imageLimiter.Acquire(context.Background(), true, 1, false, 0, 0)
	require.True(t, acquired)
	defer release()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"gpt-image-2","messages":[{"role":"user","content":"draw"}]}`))
	setImageChatTestAuth(c)
	h.ChatCompletions(c)
	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Contains(t, recorder.Body.String(), "Image generation concurrency limit exceeded")
}

type imageChatHandlerUpstream struct {
	service.HTTPUpstream
	request *http.Request
	body    []byte
}

func (u *imageChatHandlerUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.request = req
	u.body, _ = io.ReadAll(req.Body)
	contentType := "application/json"
	response := `{"id":"image-chat","object":"chat.completion","model":"gpt-image-2","choices":[{"index":0,"message":{"role":"assistant","content":"![image](https://chat2api.example/image.png)"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":10,"total_tokens":15}}`
	if gjson.GetBytes(u.body, "stream").Bool() {
		contentType = "text/event-stream"
		response = "data: {\"id\":\"image-chat\",\"object\":\"chat.completion.chunk\",\"model\":\"gpt-image-2\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"![image](https://chat2api.example/image.png)\"}}]}\n\ndata: [DONE]\n\n"
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(response))}, nil
}

func TestOpenAIChatCompletionsImageModelHandlerForwarding(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprintf("stream=%v", stream), func(t *testing.T) {
			cfg := &config.Config{RunMode: config.RunModeSimple}
			cfg.Security.URLAllowlist.AllowInsecureHTTP = true
			account := service.Account{ID: 1, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
				Status: service.StatusActive, Schedulable: true, Concurrency: 1,
				Credentials: map[string]any{"api_key": "sk-test", "base_url": "http://chat2api.example"},
				Extra:       map[string]any{"openai_responses_supported": false}}
			upstream := &imageChatHandlerUpstream{}
			cache := &concurrencyCacheMock{
				acquireUserSlotFn:    func(context.Context, int64, int, string) (bool, error) { return true, nil },
				acquireAccountSlotFn: func(context.Context, int64, int, string) (bool, error) { return true, nil },
			}
			concurrency := service.NewConcurrencyService(cache)
			billingCache := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
			t.Cleanup(billingCache.Stop)
			gateway := service.NewOpenAIGatewayService(codexModelsFailoverAccountRepo{accounts: []service.Account{account}},
				nil, nil, nil, nil, nil, nil, cfg, nil, concurrency, nil, nil, billingCache, upstream, nil, nil, nil, nil, nil, nil, nil, nil)
			h := NewOpenAIGatewayHandler(gateway, concurrency, billingCache, &service.APIKeyService{}, nil, nil, nil, nil, cfg)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(fmt.Sprintf(`{"model":"gpt-image-2","messages":[{"role":"user","content":"draw"}],"stream":%v,"upscale_target":"2k"}`, stream)))
			setImageChatTestAuth(c)
			h.ChatCompletions(c)
			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			require.NotNil(t, upstream.request, "handler must reach the upstream instead of rejecting GPT image models")
			require.Equal(t, "/v1/chat/completions", upstream.request.URL.Path)
			require.Equal(t, "gpt-image-2", gjson.GetBytes(upstream.body, "model").String())
			require.Equal(t, "2k", gjson.GetBytes(upstream.body, "upscale_target").String())
			require.Contains(t, recorder.Body.String(), "https://chat2api.example/image.png")
		})
	}
}
