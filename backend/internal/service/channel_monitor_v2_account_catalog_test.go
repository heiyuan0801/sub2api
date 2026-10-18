//go:build unit

package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

type monitorAccountCatalogGroups struct {
	GroupRepository
	groups []Group
	err    error
	calls  atomic.Int64
}

func (s *monitorAccountCatalogGroups) ListActive(context.Context) ([]Group, error) {
	s.calls.Add(1)
	return s.groups, s.err
}

type monitorAccountCatalogAccounts struct {
	AccountRepository
	accounts       []Account
	err            error
	calls          atomic.Int64
	wait           <-chan struct{}
	started        chan<- struct{}
	queriedGroup   *int64
	platforms      []string
	includeGrouped bool
}

func (s *monitorAccountCatalogAccounts) ListModelAvailabilityCandidates(_ context.Context, group *int64, platforms []string, includeGrouped bool) ([]Account, error) {
	s.calls.Add(1)
	s.queriedGroup, s.platforms, s.includeGrouped = group, platforms, includeGrouped
	if s.started != nil {
		select {
		case s.started <- struct{}{}:
		default:
		}
	}
	if s.wait != nil {
		<-s.wait
	}
	return append([]Account{}, s.accounts...), s.err
}

func newMonitorAccountCatalog(groups []Group, accounts ...Account) (*ChannelMonitorV2AccountCatalog, *monitorAccountCatalogGroups, *monitorAccountCatalogAccounts) {
	groupRepo := &monitorAccountCatalogGroups{groups: groups}
	accountRepo := &monitorAccountCatalogAccounts{accounts: accounts}
	return NewChannelMonitorV2AccountCatalog(groupRepo, accountRepo), groupRepo, accountRepo
}

func monitorMappedAccount(id int64, platform string, groupIDs []int64, names ...string) Account {
	mapping := make(map[string]any, len(names))
	for _, name := range names {
		mapping[name] = name
	}
	return Account{ID: id, Platform: platform, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true,
		GroupIDs: groupIDs, Credentials: map[string]any{"model_mapping": mapping}}
}

func monitorAccountCatalogModels(groups []PlazaGroup, groupID int64) []string {
	models := []string{}
	for _, group := range groups {
		if group.ID != groupID {
			continue
		}
		for _, model := range group.Models {
			models = append(models, model.Platform+"/"+model.Name)
		}
	}
	return models
}

func TestChannelMonitorV2AccountCatalogReadsGroupAccountsWithoutChannels(t *testing.T) {
	limitedUntil := time.Now().Add(time.Hour)
	account := monitorMappedAccount(1, PlatformOpenAI, []int64{10, 20, 99}, "gpt-6.1-sol", "gpt-image-2.5-flare")
	account.RateLimitResetAt, account.OverloadUntil, account.TempUnschedulableUntil = &limitedUntil, &limitedUntil, &limitedUntil
	account.Credentials["model_mapping"].(map[string]any)["public-alias"] = "private-upstream-model"
	svc, groupRepo, accountRepo := newMonitorAccountCatalog([]Group{
		{ID: 10, Name: "first", Platform: PlatformOpenAI},
		{ID: 20, Name: "second", Platform: PlatformOpenAI},
		{ID: 30, Name: "empty", Platform: PlatformOpenAI},
	}, account, monitorMappedAccount(2, PlatformOpenAI, nil, "unbound"))

	got, err := svc.ListModelCatalog(context.Background())

	require.NoError(t, err)
	require.Equal(t, int64(1), groupRepo.calls.Load())
	require.Equal(t, int64(1), accountRepo.calls.Load(), "all group bindings are read in a single batch")
	require.Nil(t, accountRepo.queriedGroup)
	require.Equal(t, domain.ConcretePlatformIDs(), accountRepo.platforms)
	require.True(t, accountRepo.includeGrouped)
	require.Len(t, got, 3)
	want := []string{"openai/gpt-6.1-sol", "openai/gpt-image-2.5-flare", "openai/public-alias"}
	require.Equal(t, want, monitorAccountCatalogModels(got, 10))
	require.Equal(t, want, monitorAccountCatalogModels(got, 20), "temporary account failures do not remove catalog identities")
	require.Empty(t, monitorAccountCatalogModels(got, 30), "an empty account pool cannot manufacture platform defaults")
	for _, group := range got {
		for _, model := range group.Models {
			require.NotContains(t, model.Name, "upstream-model")
			require.Nil(t, model.Pricing)
			require.Nil(t, model.OfficialPricing)
		}
	}
}

func TestChannelMonitorV2AccountCatalogKeepsPlatformsAndMixedGeminiSeparate(t *testing.T) {
	mixed := monitorMappedAccount(1, PlatformAntigravity, []int64{10, 20, 30, 40}, "gemini-custom", "claude-custom")
	mixed.Extra = map[string]any{"mixed_scheduling": true}
	nonMixed := monitorMappedAccount(2, PlatformAntigravity, []int64{10, 40}, "gemini-hidden", "claude-hidden")
	openAI := monitorMappedAccount(3, PlatformOpenAI, []int64{10, 20, 30, 40}, "openai-only")
	svc, _, _ := newMonitorAccountCatalog([]Group{
		{ID: 10, Name: "gemini", Platform: PlatformGemini},
		{ID: 20, Name: "openai", Platform: PlatformOpenAI},
		{ID: 30, Name: "composite", Platform: PlatformComposite},
		{ID: 40, Name: "claude", Platform: PlatformAnthropic},
	}, mixed, nonMixed, openAI)

	got, err := svc.ListModelCatalog(context.Background())

	require.NoError(t, err)
	geminiModels := monitorAccountCatalogModels(got, 10)
	require.Contains(t, geminiModels, "gemini/gemini-custom")
	require.NotContains(t, geminiModels, "gemini/gemini-hidden")
	for _, model := range geminiModels {
		require.True(t, strings.HasPrefix(model, "gemini/gemini-"), model)
	}
	require.Equal(t, []string{"openai/openai-only"}, monitorAccountCatalogModels(got, 20))
	compositeModels := monitorAccountCatalogModels(got, 30)
	require.Contains(t, compositeModels, "antigravity/gemini-custom")
	require.Contains(t, compositeModels, "antigravity/claude-custom")
	require.Contains(t, compositeModels, "openai/openai-only")
	claudeModels := monitorAccountCatalogModels(got, 40)
	require.Contains(t, claudeModels, "anthropic/claude-custom")
	require.NotContains(t, claudeModels, "anthropic/claude-hidden")
	for _, model := range claudeModels {
		require.True(t, strings.HasPrefix(model, "anthropic/claude-"), model)
	}
}

func TestChannelMonitorV2AccountCatalogUsesProviderMappingsAndSyncedIDs(t *testing.T) {
	kimi := monitorMappedAccount(1, PlatformKimi, []int64{10}, "kimi-k3", "k3-256k")
	zhipu := monitorMappedAccount(2, PlatformZhipu, []int64{20})
	zhipu.SetUpstreamModelMetadataSnapshot(UpstreamModelMetadataSnapshot{Models: map[string]UpstreamModelMetadata{"glm-5.3": {ID: "glm-5.3"}}})
	gemini := monitorMappedAccount(3, PlatformGemini, []int64{30}, "gemini-3.1-pro")
	deepseek := monitorMappedAccount(4, PlatformDeepseek, []int64{40})
	unconfiguredKimi := monitorMappedAccount(5, PlatformKimi, []int64{50})
	unconfiguredZhipu := monitorMappedAccount(6, PlatformZhipu, []int64{60})
	svc, _, _ := newMonitorAccountCatalog([]Group{
		{ID: 10, Name: "kimi", Platform: PlatformKimi}, {ID: 20, Name: "zhipu", Platform: PlatformZhipu},
		{ID: 30, Name: "gemini", Platform: PlatformGemini}, {ID: 40, Name: "deepseek", Platform: PlatformDeepseek},
		{ID: 50, Name: "unconfigured-kimi", Platform: PlatformKimi}, {ID: 60, Name: "unconfigured-zhipu", Platform: PlatformZhipu},
	}, kimi, zhipu, gemini, deepseek, unconfiguredKimi, unconfiguredZhipu)

	got, err := svc.ListModelCatalog(context.Background())

	require.NoError(t, err)
	require.Equal(t, []string{"kimi/k3-256k", "kimi/kimi-k3"}, monitorAccountCatalogModels(got, 10))
	require.Equal(t, []string{"zhipu/glm-5.3"}, monitorAccountCatalogModels(got, 20))
	require.Equal(t, []string{"gemini/gemini-3.1-pro"}, monitorAccountCatalogModels(got, 30))
	require.Contains(t, monitorAccountCatalogModels(got, 40), "deepseek/deepseek-v4-pro")
	require.Empty(t, monitorAccountCatalogModels(got, 50))
	require.Empty(t, monitorAccountCatalogModels(got, 60))
	for _, group := range got {
		for _, model := range group.Models {
			require.NotContains(t, model.Name, "claude", "CN providers must never inherit the helper's Claude default branch")
		}
	}
}

func TestChannelMonitorV2AccountCatalogExpandsWildcardWithoutEmittingPatterns(t *testing.T) {
	account := monitorMappedAccount(1, PlatformOpenAI, []int64{10}, "gpt-6*", "public-alias")
	account.SetUpstreamModelMetadataSnapshot(UpstreamModelMetadataSnapshot{Models: map[string]UpstreamModelMetadata{
		"gpt-6-synced-only": {ID: "gpt-6-synced-only"}, "not-supported": {ID: "not-supported"},
	}})
	svc, _, _ := newMonitorAccountCatalog([]Group{{ID: 10, Platform: PlatformOpenAI}}, account)
	got, err := svc.ListModelCatalog(context.Background())
	require.NoError(t, err)
	models := monitorAccountCatalogModels(got, 10)
	require.Contains(t, models, "openai/gpt-6.1-sol")
	require.Contains(t, models, "openai/gpt-6-synced-only")
	require.Contains(t, models, "openai/public-alias")
	require.NotContains(t, models, "openai/not-supported")
	require.NotContains(t, models, "openai/gpt-5.6-sol")
	for _, model := range models {
		require.NotContains(t, model, "*")
	}
}

func TestChannelMonitorV2AccountCatalogPartialMetadataSupplementsNativeDefaults(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		t.Run(map[bool]string{false: "unmapped", true: "passthrough"}[passthrough], func(t *testing.T) {
			account := monitorMappedAccount(1, PlatformOpenAI, []int64{10})
			if passthrough {
				account.Extra = map[string]any{"openai_passthrough": true}
				account.Credentials["model_mapping"].(map[string]any)["stale-only"] = "stale-only"
			}
			account.SetUpstreamModelMetadataSnapshot(UpstreamModelMetadataSnapshot{Models: map[string]UpstreamModelMetadata{
				"gpt-6-synced-only": {ID: "gpt-6-synced-only"}, "gpt-image-custom": {ID: "gpt-image-custom"},
			}})
			svc, _, _ := newMonitorAccountCatalog([]Group{{ID: 10, Platform: PlatformOpenAI}}, account)
			got, err := svc.ListModelCatalog(context.Background())
			require.NoError(t, err)
			models := monitorAccountCatalogModels(got, 10)
			require.Contains(t, models, "openai/gpt-6-synced-only")
			require.Contains(t, models, "openai/gpt-image-custom")
			require.Contains(t, models, "openai/gpt-6.1-sol")
			require.Contains(t, models, "openai/gpt-image-2.5-flare", "partial capability metadata cannot hide dedicated image models")
			require.NotContains(t, models, "openai/stale-only", "passthrough routing ignores a stale account mapping")
		})
	}
}

func TestChannelMonitorV2AccountCatalogEmptySnapshotFallsBackOnlyForExistingAccount(t *testing.T) {
	account := monitorMappedAccount(1, PlatformOpenAI, []int64{10})
	account.SetUpstreamModelMetadataSnapshot(UpstreamModelMetadataSnapshot{Models: map[string]UpstreamModelMetadata{}})
	svc, _, _ := newMonitorAccountCatalog([]Group{{ID: 10, Platform: PlatformOpenAI}, {ID: 20, Platform: PlatformOpenAI}}, account)
	got, err := svc.ListModelCatalog(context.Background())
	require.NoError(t, err)
	require.Contains(t, monitorAccountCatalogModels(got, 10), "openai/gpt-6.1-sol")
	require.Contains(t, monitorAccountCatalogModels(got, 10), "openai/gpt-image-2.5-flare")
	require.Empty(t, monitorAccountCatalogModels(got, 20))
}

func TestChannelMonitorV2AccountCatalogAppliesGroupAllowlistAndAccountRetirement(t *testing.T) {
	account := monitorMappedAccount(1, PlatformOpenAI, []int64{10, 20}, "gpt-6.1-sol", "gpt-image-2", "gpt-5.5", "public-retired")
	account.Type = AccountTypeOAuth
	account.Credentials["model_mapping"].(map[string]any)["public-retired"] = "gpt-5.5"
	svc, _, _ := newMonitorAccountCatalog([]Group{
		{ID: 10, Platform: PlatformOpenAI, ModelAllowlist: GroupModelAllowlist{Enabled: true, Models: []string{"gpt-6*"}}},
		{ID: 20, Platform: PlatformOpenAI},
	}, account)
	got, err := svc.ListModelCatalog(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"openai/gpt-6.1-sol"}, monitorAccountCatalogModels(got, 10))
	require.Equal(t, []string{"openai/gpt-6.1-sol", "openai/gpt-image-2"}, monitorAccountCatalogModels(got, 20))
}

func TestChannelMonitorV2AccountCatalogSingleflightCacheClonesAndRefreshes(t *testing.T) {
	release := make(chan struct{})
	svc, groupRepo, accountRepo := newMonitorAccountCatalog([]Group{{ID: 10, Name: "group", Platform: PlatformOpenAI,
		I18n: GroupI18n{"en": {Name: "English"}}}}, monitorMappedAccount(1, PlatformOpenAI, []int64{10}, "first"))
	accountRepo.wait = release
	now := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	var workers sync.WaitGroup
	results := make(chan []PlazaGroup, 16)
	errorsFound := make(chan error, 16)
	for range 16 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			groups, err := svc.ListModelCatalog(context.Background())
			results <- groups
			errorsFound <- err
		}()
	}
	close(release)
	workers.Wait()
	close(results)
	close(errorsFound)
	for err := range errorsFound {
		require.NoError(t, err)
	}
	require.Equal(t, int64(1), accountRepo.calls.Load())
	require.Equal(t, int64(1), groupRepo.calls.Load())
	for groups := range results {
		require.Equal(t, "first", groups[0].Models[0].Name)
		groups[0].Name = "mutated"
		groups[0].Models[0].Name = "mutated"
		groups[0].I18n["en"] = GroupLocaleText{Name: "mutated"}
	}
	cached, err := svc.ListModelCatalog(context.Background())
	require.NoError(t, err)
	require.Equal(t, "group", cached[0].Name)
	require.Equal(t, "first", cached[0].Models[0].Name)
	require.Equal(t, "English", cached[0].I18n["en"].Name)
	accountRepo.accounts = []Account{monitorMappedAccount(1, PlatformOpenAI, []int64{10}, "updated")}
	now = now.Add(16 * time.Second)
	refreshed, err := svc.ListModelCatalog(context.Background())
	require.NoError(t, err)
	require.Equal(t, "updated", refreshed[0].Models[0].Name)
	require.Equal(t, int64(2), accountRepo.calls.Load())
}

func TestChannelMonitorV2AccountCatalogErrorsAreNotCachedAsEmptySuccess(t *testing.T) {
	sentinel := errors.New("database unavailable")
	svc, groupRepo, accountRepo := newMonitorAccountCatalog([]Group{{ID: 10, Platform: PlatformOpenAI}}, monitorMappedAccount(1, PlatformOpenAI, []int64{10}, "healthy"))
	groupRepo.err = sentinel
	got, err := svc.ListModelCatalog(context.Background())
	require.ErrorIs(t, err, sentinel)
	require.Nil(t, got)
	require.Zero(t, accountRepo.calls.Load())
	groupRepo.err = nil
	accountRepo.err = sentinel
	got, err = svc.ListModelCatalog(context.Background())
	require.ErrorIs(t, err, sentinel)
	require.Nil(t, got)
	accountRepo.err = nil
	got, err = svc.ListModelCatalog(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"openai/healthy"}, monitorAccountCatalogModels(got, 10))
	require.Equal(t, int64(3), groupRepo.calls.Load())
	require.Equal(t, int64(2), accountRepo.calls.Load())
}

func TestChannelMonitorV2AccountCatalogCancelledCallerDoesNotCancelSharedLoad(t *testing.T) {
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	started := make(chan struct{}, 1)
	svc, _, accounts := newMonitorAccountCatalog([]Group{{ID: 10, Platform: PlatformOpenAI}}, monitorMappedAccount(1, PlatformOpenAI, []int64{10}, "available"))
	accounts.wait, accounts.started = release, started
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	firstDone := make(chan error, 1)
	go func() {
		_, err := svc.ListModelCatalog(ctx)
		firstDone <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("shared account catalog load did not start")
	}
	cancel()
	select {
	case err := <-firstDone:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("cancelled HTTP caller waited for the shared account load")
	}
	type result struct {
		groups []PlazaGroup
		err    error
	}
	secondDone := make(chan result, 1)
	go func() {
		groups, err := svc.ListModelCatalog(context.Background())
		secondDone <- result{groups, err}
	}()
	close(release)
	select {
	case got := <-secondDone:
		require.NoError(t, got.err)
		require.Equal(t, []string{"openai/available"}, monitorAccountCatalogModels(got.groups, 10))
	case <-time.After(5 * time.Second):
		t.Fatal("remaining caller did not receive the shared load result")
	}
	cached, err := svc.ListModelCatalog(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"openai/available"}, monitorAccountCatalogModels(cached, 10))
	require.Equal(t, int64(1), accounts.calls.Load())
}

func TestChannelMonitorV2AccountCatalogDoesNotReadAccountsWithoutActiveGroups(t *testing.T) {
	svc, _, accountRepo := newMonitorAccountCatalog(nil)
	got, err := svc.ListModelCatalog(context.Background())
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got)
	require.Zero(t, accountRepo.calls.Load())
}
