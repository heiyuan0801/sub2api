//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGPTImageChatAccountCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name, platform, accountType, baseURL string
		extra                                map[string]any
		want                                 bool
	}{
		{"auto_chat", PlatformOpenAI, AccountTypeAPIKey, "https://chat2api.example/v1", map[string]any{"openai_responses_supported": false}, true},
		{"force_chat_overrides_probe", PlatformOpenAI, AccountTypeAPIKey, "https://chat2api.example", map[string]any{"openai_responses_mode": "force_chat_completions", "openai_responses_supported": true}, true},
		{"responses", PlatformOpenAI, AccountTypeAPIKey, "https://chat2api.example", map[string]any{"openai_responses_supported": true}, false},
		{"force_responses", PlatformOpenAI, AccountTypeAPIKey, "https://chat2api.example", map[string]any{"openai_responses_mode": "force_responses", "openai_responses_supported": false}, false},
		{"unknown_probe", PlatformOpenAI, AccountTypeAPIKey, "https://chat2api.example", nil, false},
		{"oauth", PlatformOpenAI, AccountTypeOAuth, "https://chat2api.example", map[string]any{"openai_responses_supported": false}, false},
		{"official_api", PlatformOpenAI, AccountTypeAPIKey, "https://API.OPENAI.COM./v1", map[string]any{"openai_responses_mode": "force_chat_completions"}, false},
		{"default_official_api", PlatformOpenAI, AccountTypeAPIKey, "", map[string]any{"openai_responses_mode": "force_chat_completions"}, false},
		{"invalid_base_url", PlatformOpenAI, AccountTypeAPIKey, ":bad", map[string]any{"openai_responses_mode": "force_chat_completions"}, false},
		{"other_platform", PlatformGrok, AccountTypeAPIKey, "https://chat2api.example", map[string]any{"openai_responses_supported": false}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			account := &Account{Platform: tc.platform, Type: tc.accountType, Extra: tc.extra, Credentials: map[string]any{"base_url": tc.baseURL}}
			require.Equal(t, tc.want, account.SupportsOpenAIChatCompletionsModel("gpt-image-2"))
			require.True(t, account.SupportsOpenAIChatCompletionsModel("gpt-5.6"), "text routing must remain unchanged")
		})
	}
}

func TestGPTImageChatSchedulersSelectCompatibleAccount(t *testing.T) {
	for _, advanced := range []bool{false, true} {
		for _, model := range []string{"gpt-image-2", "draw-alias"} {
			t.Run(fmt.Sprintf("advanced=%v/model=%s", advanced, model), func(t *testing.T) {
				resetOpenAIAdvancedSchedulerSettingCacheForTest()
				defer resetOpenAIAdvancedSchedulerSettingCacheForTest()
				accounts := []Account{
					{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Priority: 0},
					{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Priority: 1, Extra: map[string]any{"openai_responses_supported": true}},
					{ID: 3, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Priority: 2, Extra: map[string]any{"openai_responses_supported": false}},
				}
				for i := range accounts {
					accounts[i].Status, accounts[i].Schedulable, accounts[i].Concurrency = StatusActive, true, 1
					accounts[i].Credentials = map[string]any{"api_key": "sk-test", "base_url": "https://chat2api.example", "model_mapping": map[string]any{model: "gpt-image-2"}}
				}
				svc := newOpenAICompactionSchedulerTestService(accounts, advanced)
				selection, _, err := svc.SelectAccountWithSchedulerForCapability(context.Background(), nil, "", "", model, nil,
					OpenAIUpstreamTransportAny, OpenAIEndpointCapabilityChatCompletions, false, false, false)
				require.NoError(t, err)
				require.Equal(t, int64(3), selection.Account.ID, "incompatible higher-priority accounts must be filtered")
				releaseLegacySchedulerDecisionSelection(selection)
			})
		}
	}
}

func TestGPTImageChatForwardingPreservesProtocolAndUpscale(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, mode := range []string{"auto", "force_chat_completions"} {
		for _, model := range []string{"gpt-image-2", "gpt-image-2.5", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst"} {
			for _, stream := range []bool{false, true} {
				for _, scale := range []string{"default", "2k", "original"} {
					t.Run(fmt.Sprintf("%s/%s/stream=%v/%s", mode, model, stream, scale), func(t *testing.T) {
						body := map[string]any{"model": model, "messages": []map[string]any{{"role": "user", "content": "draw"}}, "stream": stream}
						if scale == "2k" {
							body["upscale_target"] = "2k"
						}
						if scale == "original" {
							body["upscale"] = false
						}
						encoded, err := json.Marshal(body)
						require.NoError(t, err)
						recorder := httptest.NewRecorder()
						c, _ := gin.CreateTestContext(recorder)
						c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(encoded))
						content := "![image](https://chat2api.example/image.png)"
						upstreamBody := fmt.Sprintf(`{"id":"image-chat","object":"chat.completion","model":%q,"choices":[{"index":0,"message":{"role":"assistant","content":%q},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":10,"total_tokens":15}}`, model, content)
						contentType := "application/json"
						if stream {
							contentType = "text/event-stream"
							upstreamBody = fmt.Sprintf("data: {\"id\":\"image-chat\",\"object\":\"chat.completion.chunk\",\"model\":%q,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":%q}}]}\n\ndata: [DONE]\n\n", model, content)
						}
						upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(upstreamBody))}}
						svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig(), httpUpstream: upstream}
						account := rawChatCompletionsTestAccount()
						account.Extra = map[string]any{openai_compat.ExtraKeyResponsesMode: mode, openai_compat.ExtraKeyResponsesSupported: false}
						result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, encoded, "", "")
						require.NoError(t, err)
						require.NotNil(t, result)
						require.Equal(t, "/v1/chat/completions", upstream.lastReq.URL.Path)
						require.Equal(t, model, gjson.GetBytes(upstream.lastBody, "model").String())
						require.Equal(t, "draw", gjson.GetBytes(upstream.lastBody, "messages.0.content").String())
						require.Equal(t, gjson.GetBytes(encoded, "upscale").Raw, gjson.GetBytes(upstream.lastBody, "upscale").Raw)
						require.Equal(t, gjson.GetBytes(encoded, "upscale_target").Raw, gjson.GetBytes(upstream.lastBody, "upscale_target").Raw)
						require.Contains(t, recorder.Body.String(), content)
						if stream {
							require.Contains(t, recorder.Body.String(), "data: [DONE]")
						}
					})
				}
			}
		}
	}
}

func TestGPTImageChatForwardingRejectsIncompatibleAccounts(t *testing.T) {
	for _, tc := range []string{"responses", "oauth", "official", "mapped_alias"} {
		t.Run(tc, func(t *testing.T) {
			account := rawChatCompletionsTestAccount()
			account.Extra = map[string]any{"openai_responses_supported": true}
			model := "gpt-image-2"
			switch tc {
			case "oauth":
				account.Type = AccountTypeOAuth
			case "official":
				account.Credentials["base_url"] = "https://api.openai.com"
				account.Extra["openai_responses_mode"] = "force_chat_completions"
			case "mapped_alias":
				model = "draw-alias"
				account.Credentials["model_mapping"] = map[string]any{model: "gpt-image-2"}
			}
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			body := []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"draw"}]}`, model))
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
			svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig()}
			result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body, "", "")
			require.Error(t, err)
			require.Nil(t, result)
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Equal(t, "invalid_request_error", gjson.Get(recorder.Body.String(), "error.type").String())
		})
	}
}

func TestGPTImageChatForwardingEnforcesGroupPermissionForMappedAlias(t *testing.T) {
	account := rawChatCompletionsTestAccount()
	account.Extra = map[string]any{"openai_responses_supported": false}
	account.Credentials["model_mapping"] = map[string]any{"draw-alias": "gpt-image-2"}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("api_key", &APIKey{Group: &Group{AllowImageGeneration: false}})
	body := []byte(`{"model":"draw-alias","messages":[{"role":"user","content":"draw"}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig()}
	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body, "", "")
	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), ImageGenerationPermissionMessage())
}
