package service

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/aggregateretry"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const MaxAggregateGroups = 20

var ErrAggregateKeyUnavailable = infraerrors.BadRequest("AGGREGATE_GROUP_UNAVAILABLE", "No authorized group supports this model")

// SetAggregateGateway runs before serving. Channel configuration already depends
// on key invalidation, so constructor injection would create a dependency cycle.
func (s *APIKeyService) SetAggregateGateway(gateway *GatewayService) { s.aggregateGateway = gateway }

func (s *APIKeyService) validateAggregateGroups(ctx context.Context, userID int64, aggregate bool, ids []int64) error {
	invalid := infraerrors.BadRequest("AGGREGATE_GROUPS_INVALID", "Select 1-20 distinct groups available to you")
	if !aggregate {
		if len(ids) > 0 {
			return invalid
		}
		return nil
	}
	if len(ids) == 0 || len(ids) > MaxAggregateGroups {
		return invalid
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	seen := map[int64]bool{}
	for _, id := range ids {
		if id <= 0 || seen[id] {
			return invalid
		}
		seen[id] = true
		group, err := s.groupRepo.GetByID(ctx, id)
		if errors.Is(err, ErrGroupNotFound) {
			return invalid
		}
		if err != nil {
			return err
		}
		if group == nil || !group.IsActive() || !s.canUserBindGroup(ctx, user, group) {
			return ErrGroupNotAllowed
		}
	}
	return nil
}

// Reload permissions so revoked groups cannot survive in a cached aggregate key.
func (s *APIKeyService) aggregateGroups(ctx context.Context, key *APIKey) (*APIKey, []*Group, error) {
	current, err := s.apiKeyRepo.GetByID(ctx, key.ID)
	if err != nil {
		return nil, nil, err
	}
	if current == nil || !current.IsAggregate || current.UserID != key.UserID ||
		(current.Status != StatusActive && current.Status != StatusAPIKeyExpired && current.Status != StatusAPIKeyQuotaExhausted) ||
		len(current.AggregateGroupIDs) == 0 || len(current.AggregateGroupIDs) > MaxAggregateGroups {
		return nil, nil, ErrAggregateKeyUnavailable
	}
	user, err := s.userRepo.GetByID(ctx, key.UserID)
	if err != nil {
		return nil, nil, err
	}
	if user == nil || !user.IsActive() {
		return nil, nil, ErrAggregateKeyUnavailable
	}
	out := *current
	out.Key = key.Key
	userCopy := *user
	userCopy.UserGroupRPMOverride = nil
	out.User = &userCopy
	out.GroupID, out.Group = nil, nil
	out.AggregateGroupIDs = slices.Clone(current.AggregateGroupIDs)
	s.compileAPIKeyIPRules(&out)
	groups := make([]*Group, 0, len(out.AggregateGroupIDs))
	seen := map[int64]bool{}
	for _, id := range out.AggregateGroupIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		group, err := s.groupRepo.GetByID(ctx, id)
		if errors.Is(err, ErrGroupNotFound) {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		if group != nil && group.IsActive() && s.canUserBindGroup(ctx, out.User, group) {
			groups = append(groups, group)
		}
	}
	return &out, groups, nil
}

// ResolveAggregateKey selects the next authorized group before each attempt. Quotas and idempotency
// stay on this key; group-dependent billing uses the actual selected group.
func (s *APIKeyService) ResolveAggregateKey(ctx context.Context, key *APIKey, model, endpoint, forced string) (*APIKey, error) {
	if key == nil || !key.IsAggregate {
		return key, nil
	}
	out, groups, err := s.aggregateGroups(ctx, key)
	if err != nil {
		return nil, err
	}
	if model == "" {
		return out, nil
	}
	if !out.IsActive() || out.IsExpired() || out.IsQuotaExhausted() {
		return nil, ErrAggregateKeyUnavailable
	}
	if s.aggregateGateway == nil || s.aggregateGateway.accountRepo == nil {
		return nil, errors.New("aggregate routing is not initialized")
	}
	state := aggregateretry.FromContext(ctx)
	if state != nil {
		remaining := state.Plan(out.AggregateGroupIDs)
		byID := make(map[int64]*Group, len(groups))
		for _, group := range groups {
			byID[group.ID] = group
		}
		groups = nil
		for _, id := range remaining {
			if group := byID[id]; group != nil {
				groups = append(groups, group)
			}
		}
	}
	for _, group := range groups {
		accounts, err := s.aggregateGateway.accountRepo.ListSchedulableByGroupID(ctx, group.ID)
		if err != nil {
			return nil, err
		}
		supported, err := s.aggregateGroupSupports(ctx, group, accounts, model, endpoint, forced)
		if err != nil {
			return nil, err
		}
		if !supported {
			continue
		}
		selected := *group
		// Legacy fallbacks can leave the selected set. Only the aggregate retry
		// coordinator may advance to another explicitly selected group.
		selected.FallbackGroupID, selected.FallbackGroupIDOnInvalidRequest = nil, nil
		out.Group, out.GroupID = &selected, &selected.ID
		if s.userGroupRateRepo != nil {
			override, err := s.userGroupRateRepo.GetRPMOverrideByUserAndGroup(ctx, out.UserID, group.ID)
			if err != nil {
				return nil, err
			}
			out.User.UserGroupRPMOverride = override
		}
		if state != nil {
			state.Select(group.ID)
		}
		return out, nil
	}
	if state != nil && state.GroupID != 0 {
		state.Exhausted = true
	}
	return nil, ErrAggregateKeyUnavailable
}

func (s *APIKeyService) aggregateGroupSupports(ctx context.Context, group *Group, accounts []Account, model, endpoint, forced string) (bool, error) {
	if !group.ModelAllowlist.Allows(model) {
		return false, nil
	}
	target, upstream := group.Platform, model
	if target == PlatformComposite {
		if s.aggregateGateway.compositeResolver == nil {
			return false, nil
		}
		decision, err := s.aggregateGateway.compositeResolver.Resolve(ctx, group.ID, model, endpoint)
		if err != nil {
			return false, err
		}
		if !decision.Matched {
			return false, nil
		}
		target, upstream = decision.TargetPlatform, decision.UpstreamModel
	}
	if forced != "" && target != forced {
		return false, nil
	}
	if endpoint == CompositeRouteEndpointGemini && target != PlatformGemini && target != PlatformAntigravity {
		return false, nil
	}
	channelMapped := false
	if channel := s.aggregateGateway.channelService; channel != nil {
		if channel.IsModelRestricted(ctx, group.ID, upstream) {
			return false, nil
		}
		mapped := channel.ResolveChannelMapping(ctx, group.ID, upstream).MappedModel
		channelMapped = mapped != upstream
		upstream = mapped
	}
	for i := range accounts {
		// Model helpers memoize on Account; keep that memoization request-local.
		accountCopy := accounts[i]
		account := &accountCopy
		if !account.IsSchedulable() {
			continue
		}
		mixed := account.Platform == PlatformAntigravity && mixedListingAccountAllowed(target, account) && mixedListingModelAllowed(target, upstream)
		if account.Platform != target && !mixed {
			continue
		}
		if !account.IsModelSupported(upstream) {
			continue
		}
		// An empty mapping permits native models, not every provider's models.
		_, explicit := account.ResolveMappedModel(upstream)
		platform, known := DetectModelPlatform(upstream)
		catalogued := slices.Contains(channelMonitorV2AccountModelIDs(account), upstream)
		if explicit || catalogued || channelMapped || (known && (platform == target || target == PlatformAntigravity)) {
			return true, nil
		}
	}
	return false, nil
}

// AggregateModels returns a union without making upstream requests.
func (s *APIKeyService) AggregateModels(ctx context.Context, key *APIKey, endpoint, forced string) ([]string, error) {
	_, groups, err := s.aggregateGroups(ctx, key)
	if err != nil {
		return nil, err
	}
	if s.aggregateGateway == nil || s.aggregateGateway.accountRepo == nil {
		return nil, errors.New("aggregate routing is not initialized")
	}
	models := []string{}
	for _, group := range groups {
		accounts, err := s.aggregateGateway.accountRepo.ListSchedulableByGroupID(ctx, group.ID)
		if err != nil {
			return nil, err
		}
		candidates := slices.Clone(group.ModelAllowlist.Models)
		for i := range accounts {
			accountCopy := accounts[i]
			candidates = append(candidates, channelMonitorV2AccountModelIDs(&accountCopy)...)
		}
		if group.Platform == PlatformComposite && s.aggregateGateway.compositeResolver != nil {
			aliases, err := s.aggregateGateway.compositeResolver.ListExactPublicModels(ctx, group.ID, endpoint, true)
			if err != nil {
				return nil, err
			}
			candidates = append(candidates, aliases...)
		}
		for _, model := range dedupeAndSortModelIDs(candidates) {
			if strings.Contains(model, "*") {
				continue
			}
			supported, err := s.aggregateGroupSupports(ctx, group, accounts, model, endpoint, forced)
			if err != nil {
				return nil, err
			}
			if supported {
				models = append(models, model)
			}
		}
	}
	return dedupeAndSortModelIDs(models), nil
}
