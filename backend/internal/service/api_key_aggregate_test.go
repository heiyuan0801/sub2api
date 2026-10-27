package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/aggregateretry"
	"github.com/stretchr/testify/require"
)

type aggregateRepoStub struct {
	APIKeyRepository
	key *APIKey
	err error
}

func (r *aggregateRepoStub) GetByID(context.Context, int64) (*APIKey, error) { return r.key, r.err }

type aggregateUserStub struct {
	UserRepository
	user *User
}

func (r *aggregateUserStub) GetByID(context.Context, int64) (*User, error) { return r.user, nil }

type aggregateGroupStub struct {
	GroupRepository
	groups map[int64]*Group
	err    error
}

func (r *aggregateGroupStub) GetByID(_ context.Context, id int64) (*Group, error) {
	if r.err != nil {
		return nil, r.err
	}
	if g := r.groups[id]; g != nil {
		return g, nil
	}
	return nil, ErrGroupNotFound
}

type aggregateAccountsStub struct {
	AccountRepository
	accounts map[int64][]Account
	err      error
}

func (r *aggregateAccountsStub) ListSchedulableByGroupID(_ context.Context, id int64) ([]Account, error) {
	return r.accounts[id], r.err
}

type aggregateRPMStub struct{ UserGroupRateRepository }

func (*aggregateRPMStub) GetRPMOverrideByUserAndGroup(_ context.Context, _, id int64) (*int, error) {
	rpm := int(id) * 10
	return &rpm, nil
}

type aggregateSubscriptionStub struct {
	UserSubscriptionRepository
	active bool
}

func (r *aggregateSubscriptionStub) GetActiveByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	if !r.active {
		return nil, errors.New("no subscription")
	}
	return &UserSubscription{ID: 55}, nil
}
func aggregateFixture() (*APIKeyService, *aggregateRepoStub, *APIKey) {
	user := &User{ID: 2, Status: StatusActive, Balance: 100}
	parent := &APIKey{ID: 10, UserID: 2, Status: StatusActive, IsAggregate: true, AggregateGroupIDs: []int64{1, 2, 3}, User: user, Quota: 50, RateLimit5h: 10}
	groups := map[int64]*Group{}
	accounts := map[int64][]Account{}
	for id := int64(1); id <= 3; id++ {
		platform, model := PlatformOpenAI, "gpt-5"
		if id == 3 {
			platform, model = PlatformAnthropic, "claude-sonnet-4-5"
		}
		groups[id] = &Group{ID: id, Platform: platform, Status: StatusActive, Hydrated: true, RateMultiplier: float64(id), SubscriptionType: "standard"}
		accounts[id] = []Account{{ID: id, Platform: platform, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Credentials: map[string]any{"model_mapping": map[string]any{model: model}}}}
	}
	repo := &aggregateRepoStub{key: parent}
	s := &APIKeyService{apiKeyRepo: repo, userRepo: &aggregateUserStub{user: user}, groupRepo: &aggregateGroupStub{groups: groups}, userGroupRateRepo: &aggregateRPMStub{}, userSubRepo: &aggregateSubscriptionStub{}}
	s.SetAggregateGateway(&GatewayService{accountRepo: &aggregateAccountsStub{accounts: accounts}})
	return s, repo, parent
}
func TestAggregateGroupsValidatePermissionAndOrder(t *testing.T) {
	for _, tc := range []string{"valid", "single", "empty", "duplicate", "negative", "missing", "too many", "ordinary", "exclusive", "restricted", "disabled", "subscription"} {
		t.Run(tc, func(t *testing.T) {
			s, _, p := aggregateFixture()
			ids := []int64{3, 1}
			aggregate := true
			switch tc {
			case "single":
				ids = []int64{1}
			case "empty":
				ids = nil
			case "duplicate":
				ids = []int64{1, 1}
			case "negative":
				ids = []int64{-1}
			case "missing":
				ids = []int64{99}
			case "too many":
				ids = make([]int64, 21)
			case "ordinary":
				aggregate = false
			case "exclusive":
				aggregateTestGroups(t, s).groups[3].IsExclusive = true
			case "restricted":
				p.User.RestrictPublicGroups = true
			case "disabled":
				aggregateTestGroups(t, s).groups[3].Status = "disabled"
			case "subscription":
				aggregateTestGroups(t, s).groups[3].SubscriptionType = "subscription"
			}
			err := s.validateAggregateGroups(context.Background(), p.UserID, aggregate, ids)
			if tc == "valid" || tc == "single" {
				require.NoError(t, err)
				require.NotEmpty(t, ids)
			} else {
				require.Error(t, err)
			}
		})
	}
}
func TestAggregateGroupRoutingPriorityAndFreshRevocation(t *testing.T) {
	s, r, p := aggregateFixture()
	ctx := context.Background()
	selected, err := s.ResolveAggregateKey(ctx, p, "gpt-5", CompositeRouteEndpointResponses, "")
	require.NoError(t, err)
	require.Equal(t, int64(1), *selected.GroupID)
	require.Equal(t, p.ID, selected.ID)
	require.Equal(t, 10, *selected.User.UserGroupRPMOverride)
	require.Nil(t, p.GroupID)
	require.Nil(t, p.User.UserGroupRPMOverride)
	r.key.AggregateGroupIDs = []int64{2, 1, 3}
	selected, err = s.ResolveAggregateKey(ctx, p, "gpt-5", CompositeRouteEndpointResponses, "")
	require.NoError(t, err)
	require.Equal(t, int64(2), *selected.GroupID)
	aggregateTestGroups(t, s).groups[2].IsExclusive = true
	selected, err = s.ResolveAggregateKey(ctx, p, "gpt-5", CompositeRouteEndpointResponses, "")
	require.NoError(t, err)
	require.Equal(t, int64(1), *selected.GroupID)
	selected, err = s.ResolveAggregateKey(ctx, p, "claude-sonnet-4-5", CompositeRouteEndpointMessages, "")
	require.NoError(t, err)
	require.Equal(t, int64(3), *selected.GroupID)
	delete(aggregateTestGroups(t, s).groups, 3)
	_, err = s.ResolveAggregateKey(ctx, p, "claude-sonnet-4-5", CompositeRouteEndpointMessages, "")
	require.ErrorIs(t, err, ErrAggregateKeyUnavailable)
}
func TestAggregateRoutingFailsClosedAndDiscoveryHasNoBillingGroup(t *testing.T) {
	for _, tc := range []string{"unknown", "allowlist", "accounts unavailable", "wrong provider", "forced platform", "gemini endpoint", "expired", "disabled user", "db failure"} {
		t.Run(tc, func(t *testing.T) {
			s, r, p := aggregateFixture()
			model, endpoint, forced := "gpt-5", CompositeRouteEndpointResponses, ""
			r.key.AggregateGroupIDs = []int64{1}
			switch tc {
			case "unknown":
				model = "not-configured"
			case "allowlist":
				aggregateTestGroups(t, s).groups[1].ModelAllowlist = GroupModelAllowlist{Enabled: true, Models: []string{"other"}}
			case "accounts unavailable":
				aggregateTestAccounts(t, s).accounts[1][0].Schedulable = false
			case "wrong provider":
				model = "claude-sonnet-4-5"
				aggregateTestAccounts(t, s).accounts[1][0].Credentials = nil
			case "forced platform":
				forced = PlatformAntigravity
			case "gemini endpoint":
				endpoint = CompositeRouteEndpointGemini
			case "expired":
				past := time.Now().Add(-time.Hour)
				p.ExpiresAt = &past
			case "disabled user":
				p.User.Status = "disabled"
			case "db failure":
				aggregateTestAccounts(t, s).err = errors.New("db unavailable")
			}
			_, err := s.ResolveAggregateKey(context.Background(), p, model, endpoint, forced)
			require.Error(t, err)
		})
	}
	s, _, p := aggregateFixture()
	selected, err := s.ResolveAggregateKey(context.Background(), p, "", "", "")
	require.NoError(t, err)
	require.Nil(t, selected.GroupID)
	models, err := s.AggregateModels(context.Background(), p, "", "")
	require.NoError(t, err)
	require.Equal(t, []string{"claude-sonnet-4-5", "gpt-5"}, models)
	aggregateTestGroups(t, s).groups[3].IsExclusive = true
	models, err = s.AggregateModels(context.Background(), p, "", "")
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-5"}, models)
}
func TestAggregateSelectedGroupBillingAndConcurrentIsolation(t *testing.T) {
	s, _, p := aggregateFixture()
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			model, group := "gpt-5", int64(1)
			if i%2 == 1 {
				model, group = "claude-sonnet-4-5", 3
			}
			key, err := s.ResolveAggregateKey(ctx, p, model, CompositeRouteEndpointMessages, "")
			require.NoError(t, err)
			multiplier, image, at := s.aggregateGateway.recordUsageMultipliers(ctx, key, key.User, time.Now())
			require.Equal(t, float64(group), multiplier)
			cost := &CostBreakdown{TotalCost: 0.1, ActualCost: 0.1 * multiplier}
			account := &Account{ID: group, Platform: key.Group.Platform, Type: AccountTypeAPIKey}
			usage := s.aggregateGateway.buildRecordUsageLog(ctx, &recordUsageCoreInput{PricingAt: at}, &ForwardResult{Model: model}, key, key.User, account, nil, model, multiplier, image, 1, 0, false, cost)
			require.Equal(t, group, *usage.GroupID)
			require.Equal(t, p.ID, usage.APIKeyID)
			cmd := buildUsageBillingCommand("request", usage, &postUsageBillingParams{Cost: cost, APIKey: key, User: key.User, Account: account, APIKeyService: apiKeyQuotaUpdaterStub{}})
			require.Equal(t, p.ID, cmd.APIKeyID)
			require.InDelta(t, cost.ActualCost, cmd.BalanceCost, 1e-10)
			require.InDelta(t, cost.ActualCost, cmd.APIKeyQuotaCost, 1e-10)
		}(i)
	}
	wg.Wait()
	require.Nil(t, p.GroupID)
	require.Nil(t, p.Group)
}
func TestAggregateGroupsAuthSnapshotRoundTrip(t *testing.T) {
	s, _, p := aggregateFixture()
	snapshot := s.snapshotFromAPIKey(context.Background(), p)
	out := s.snapshotToAPIKey("test", snapshot)
	require.True(t, out.IsAggregate)
	require.Equal(t, p.AggregateGroupIDs, out.AggregateGroupIDs)
	require.Nil(t, out.GroupID)
	out.AggregateGroupIDs[0] = 99
	require.Equal(t, int64(1), p.AggregateGroupIDs[0])
}

func aggregateTestGroups(t *testing.T, s *APIKeyService) *aggregateGroupStub {
	t.Helper()
	r, ok := s.groupRepo.(*aggregateGroupStub)
	require.True(t, ok)
	return r
}
func aggregateTestAccounts(t *testing.T, s *APIKeyService) *aggregateAccountsStub {
	t.Helper()
	r, ok := s.aggregateGateway.accountRepo.(*aggregateAccountsStub)
	require.True(t, ok)
	return r
}

func TestAggregateRetrySelectsNextGroupAndRechecksPermissions(t *testing.T) {
	for _, mode := range []string{"next", "reordered", "revoked", "allowlist", "key-disabled"} {
		t.Run(mode, func(t *testing.T) {
			svc, repo, key := aggregateFixture()
			state := &aggregateretry.State{}
			ctx := aggregateretry.WithState(context.Background(), state)
			first, err := svc.ResolveAggregateKey(ctx, key, "gpt-5", CompositeRouteEndpointResponses, "")
			require.NoError(t, err)
			require.Equal(t, int64(1), first.Group.ID)
			groups := aggregateTestGroups(t, svc).groups
			switch mode {
			case "reordered":
				repo.key.AggregateGroupIDs = []int64{3, 2, 1}
			case "revoked":
				groups[2].Status = "disabled"
			case "allowlist":
				groups[2].ModelAllowlist = GroupModelAllowlist{Enabled: true, Models: []string{"gpt-other"}}
			case "key-disabled":
				repo.key.Status = "disabled"
			}
			next, err := svc.ResolveAggregateKey(ctx, key, "gpt-5", CompositeRouteEndpointResponses, "")
			if mode == "next" || mode == "reordered" {
				require.NoError(t, err)
				require.Equal(t, int64(2), next.Group.ID)
				require.Equal(t, 2.0, next.Group.RateMultiplier)
				require.Equal(t, 20, *next.User.UserGroupRPMOverride)
				require.Equal(t, first.ID, next.ID)
				require.Equal(t, int64(1), first.Group.ID)
				_, err = svc.ResolveAggregateKey(ctx, key, "gpt-5", CompositeRouteEndpointResponses, "")
				require.ErrorIs(t, err, ErrAggregateKeyUnavailable)
				require.True(t, state.Exhausted)
			} else {
				require.ErrorIs(t, err, ErrAggregateKeyUnavailable)
				require.Nil(t, next)
				require.Equal(t, mode != "key-disabled", state.Exhausted)
			}
		})
	}
}
