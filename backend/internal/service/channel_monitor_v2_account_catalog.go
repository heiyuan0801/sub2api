package service

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"golang.org/x/sync/singleflight"
)

// ChannelMonitorV2AccountCatalog enumerates the configured accounts bound to
// each active group. It performs no upstream requests or channel/price lookups.
// Monitor configuration and viewer authorization are applied by the caller.
type ChannelMonitorV2AccountCatalog struct {
	groupRepo   GroupRepository
	accountRepo AccountRepository
	now         func() time.Time
	ttl         time.Duration
	mu          sync.RWMutex
	groups      []PlazaGroup
	expiresAt   time.Time
	loads       singleflight.Group
}

func NewChannelMonitorV2AccountCatalog(groupRepo GroupRepository, accountRepo AccountRepository) *ChannelMonitorV2AccountCatalog {
	return &ChannelMonitorV2AccountCatalog{groupRepo: groupRepo, accountRepo: accountRepo, now: time.Now, ttl: 15 * time.Second}
}

func (s *ChannelMonitorV2AccountCatalog) ListModelCatalog(ctx context.Context) ([]PlazaGroup, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if groups, ok := s.cached(); ok {
		return groups, nil
	}
	result := s.loads.DoChan("account-model-catalog", func() (any, error) {
		if groups, ok := s.cached(); ok {
			return groups, nil
		}
		// UI requests can be cancelled when switching platforms. Keep the shared
		// bounded directory load usable for other callers in the same flight.
		loadCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		groups, err := s.load(loadCtx)
		if err != nil {
			return nil, err
		}
		s.mu.Lock()
		s.groups = cloneChannelMonitorV2AccountCatalog(groups)
		s.expiresAt = s.now().Add(s.ttl)
		s.mu.Unlock()
		return groups, nil
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case loaded := <-result:
		if loaded.Err != nil {
			return nil, loaded.Err
		}
		groups, ok := loaded.Val.([]PlazaGroup)
		if !ok {
			return nil, fmt.Errorf("unexpected monitor account catalog result type %T", loaded.Val)
		}
		return cloneChannelMonitorV2AccountCatalog(groups), nil
	}
}

func (s *ChannelMonitorV2AccountCatalog) cached() ([]PlazaGroup, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.groups == nil || !s.now().Before(s.expiresAt) {
		return nil, false
	}
	return cloneChannelMonitorV2AccountCatalog(s.groups), true
}

func cloneChannelMonitorV2AccountCatalog(groups []PlazaGroup) []PlazaGroup {
	clone := make([]PlazaGroup, len(groups))
	for i, group := range groups {
		clone[i] = group
		clone[i].I18n = maps.Clone(group.I18n)
		clone[i].Models = append([]PlazaModel{}, group.Models...)
	}
	return clone
}

func (s *ChannelMonitorV2AccountCatalog) load(ctx context.Context) ([]PlazaGroup, error) {
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list monitor account catalog groups: %w", err)
	}
	out := make([]PlazaGroup, len(groups))
	byID := make(map[int64]int, len(groups))
	for i, group := range groups {
		byID[group.ID] = i
		out[i] = PlazaGroup{
			ID: group.ID, Name: group.Name, Description: group.Description,
			I18n: maps.Clone(group.I18n), Platform: group.Platform,
			RateMultiplier: group.RateMultiplier, IsExclusive: group.IsExclusive,
			Models: []PlazaModel{},
		}
	}
	if len(groups) == 0 {
		return out, nil
	}
	// This repository path ignores rate limits, overload and other temporary
	// scheduling state. One batch includes group bindings and avoids N+1 reads.
	accounts, err := s.accountRepo.ListModelAvailabilityCandidates(ctx, nil, domain.ConcretePlatformIDs(), true)
	if err != nil {
		return nil, fmt.Errorf("list monitor account catalog accounts: %w", err)
	}
	for i := range accounts {
		account := &accounts[i]
		models := channelMonitorV2AccountModelIDs(account)
		for _, groupID := range account.GroupIDs {
			index, exists := byID[groupID]
			if !exists {
				continue
			}
			group := &out[index]
			platform := account.Platform
			if group.Platform != PlatformComposite && group.Platform != account.Platform {
				if !channelMonitorV2MixedAccountAllowed(group.Platform, account) {
					continue
				}
				platform = group.Platform
			}
			for _, model := range models {
				if platform != account.Platform && !channelMonitorV2MixedModelAllowed(platform, model) {
					continue
				}
				group.Models = append(group.Models, PlazaModel{Name: model, Platform: platform})
			}
		}
	}
	for i := range out {
		group := &out[i]
		byPlatform := make(map[string][]string)
		for _, model := range group.Models {
			byPlatform[model.Platform] = append(byPlatform[model.Platform], model.Name)
		}
		group.Models = []PlazaModel{}
		for platform, models := range byPlatform {
			models = dedupeAndSortModelIDs(models)
			models = groups[i].ModelAllowlist.FilterForListing(models)
			for _, model := range models {
				if model != "" && !strings.Contains(model, "*") {
					group.Models = append(group.Models, PlazaModel{Name: model, Platform: platform})
				}
			}
		}
		sort.Slice(group.Models, func(i, j int) bool {
			if group.Models[i].Name != group.Models[j].Name {
				return group.Models[i].Name < group.Models[j].Name
			}
			return group.Models[i].Platform < group.Models[j].Platform
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func channelMonitorV2MixedAccountAllowed(platform string, account *Account) bool {
	return (platform == PlatformAnthropic || platform == PlatformGemini) && account.IsMixedSchedulingEnabled()
}

func channelMonitorV2MixedModelAllowed(platform, model string) bool {
	if platform == PlatformAnthropic {
		return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "claude-")
	}
	return mixedListingModelAllowed(platform, model)
}

func channelMonitorV2AccountModelIDs(account *Account) []string {
	if !domain.IsConcretePlatform(account.Platform) {
		return nil
	}
	mapping := account.GetModelMapping()
	candidates := []string{}
	if snapshot := account.GetUpstreamModelMetadataSnapshot(); snapshot != nil {
		for id := range snapshot.Models {
			id = strings.TrimSpace(id)
			if id != "" && !strings.Contains(id, "*") {
				candidates = append(candidates, id)
			}
		}
	}
	// Synced metadata is a capability subset, not a complete model-ID inventory:
	// partial syncs and dedicated image/video models can be absent from it.
	// Supplement the known native catalog instead of treating metadata as an
	// authoritative whitelist; actual account routing still filters candidates.
	candidates = append(candidates, channelMonitorV2AccountDefaultModels(account)...)
	models := []string{}
	if len(mapping) == 0 || account.IsOpenAIPassthroughEnabled() {
		models = candidates
	} else {
		hasWildcard := false
		for name := range mapping {
			if strings.Contains(name, "*") {
				hasWildcard = true
			} else {
				models = append(models, name)
			}
		}
		if hasWildcard {
			models = append(models, candidates...)
		}
	}
	models = dedupeAndSortModelIDs(models)
	return slices.DeleteFunc(models, func(model string) bool {
		return strings.Contains(model, "*") || !account.IsModelSupported(model)
	})
}

func channelMonitorV2AccountDefaultModels(account *Account) []string {
	switch account.Platform {
	case PlatformAnthropic, PlatformOpenAI, PlatformGemini, PlatformAntigravity, PlatformGrok, PlatformOpenCodeGo, PlatformTypeSafe:
		return defaultModelsListCandidateIDs(account.Platform)
	case PlatformDeepseek:
		return slices.Clone(deepseekServableModels)
	default:
		// Providers without a shared native default catalog rely on their
		// configured mappings/synced IDs. Never borrow another vendor's defaults.
		return nil
	}
}
