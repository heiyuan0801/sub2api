//go:build unit

package repository

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestSchedulerCacheGPTImageChatAdmissionSurvivesProjection(t *testing.T) {
	for _, tc := range []struct {
		name, base string
		supported  bool
		mode       string
		chat, want bool
	}{
		{"native_chat", "https://chat2api.example/v1", false, "", true, true},
		{"forced_chat", "https://chat2api.example/v1", true, "force_chat_completions", true, true},
		{"responses", "https://chat2api.example/v1", true, "", true, false},
		{"official", "https://api.openai.com/v1", false, "", true, false},
		{"default_url", "", false, "", true, false},
		{"chat_disabled", "https://chat2api.example/v1", false, "", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cache := newSchedulerCacheUnit(t)
			ctx := context.Background()
			account := service.Account{
				ID: 9059, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
				Status: "active", Schedulable: true,
				Credentials: map[string]any{"base_url": tc.base, "openai_capabilities": map[string]any{"chat_completions": tc.chat}, "access_token": "excluded"},
				Extra:       map[string]any{"openai_responses_supported": tc.supported, "openai_responses_mode": tc.mode},
			}
			bucket := service.SchedulerBucket{GroupID: 19, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
			token, err := cache.CaptureBucketWriteToken(ctx, bucket)
			require.NoError(t, err)
			require.NoError(t, cache.SetSnapshot(ctx, bucket, token, []service.Account{account}))
			cached, hit, err := cache.GetSnapshot(ctx, bucket)
			require.NoError(t, err)
			require.True(t, hit)
			require.Len(t, cached, 1)
			eligible := func(a *service.Account) bool {
				return a.SupportsOpenAIEndpointCapability(service.OpenAIEndpointCapabilityChatCompletions) && a.SupportsOpenAIChatCompletionsModel("gpt-image-2")
			}
			require.Equal(t, tc.want, eligible(&account))
			require.Equal(t, tc.want, eligible(cached[0]))
			require.Empty(t, cached[0].GetCredential("access_token"))
		})
	}
}

func TestSchedulerCacheGPTImageChatOldProjectionMissesUntilRebuilt(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	bucket := service.SchedulerBucket{GroupID: 19, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	old := service.Account{ID: 9059, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: "active", Schedulable: true, Extra: map[string]any{"openai_responses_supported": false}}
	payload, err := json.Marshal(old)
	require.NoError(t, err)
	require.NoError(t, cache.rdb.Set(ctx, "sched:meta:9059", payload, 0).Err())
	require.NoError(t, cache.rdb.Set(ctx, schedulerBucketKey(schedulerReadyPrefix, bucket), "1", 0).Err())
	require.NoError(t, cache.rdb.Set(ctx, schedulerBucketKey(schedulerActivePrefix, bucket), "1", 0).Err())
	require.NoError(t, cache.rdb.ZAdd(ctx, schedulerSnapshotKey(bucket, "1"), redis.Z{Score: 50, Member: "9059"}).Err())
	_, hit, err := cache.GetSnapshot(ctx, bucket)
	require.NoError(t, err)
	require.False(t, hit, "legacy metadata must cause DB fallback instead of excluding image candidates")
	old.Credentials = map[string]any{"base_url": "https://chat2api.example/v1", "openai_capabilities": []any{"chat_completions"}}
	require.NoError(t, cache.SetAccount(ctx, &old))
	accounts, hit, err := cache.GetSnapshot(ctx, bucket)
	require.NoError(t, err)
	require.True(t, hit)
	require.Len(t, accounts, 1)
	require.True(t, accounts[0].SupportsOpenAIChatCompletionsModel("gpt-image-2"))
}
