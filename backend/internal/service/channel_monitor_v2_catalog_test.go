package service

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
)

type monitorCatalogStub struct {
	groups []PlazaGroup
	err    error
	calls  int
}

func (s *monitorCatalogStub) ListModelCatalog(context.Context) ([]PlazaGroup, error) {
	s.calls++
	return s.groups, s.err
}

type catalogMonitorRepo struct {
	channelMonitorV2RepoStub
	received       ChannelMonitorV2Filter
	receivedConfig ChannelMonitorV2Config
}

func (s *catalogMonitorRepo) GetMatrix(_ context.Context, f ChannelMonitorV2Filter, cfg ChannelMonitorV2Config, g ChannelMonitorV2GroupBy, _ bool) (*ChannelMonitorV2Matrix, error) {
	s.received = f
	s.receivedConfig = cfg
	return &ChannelMonitorV2Matrix{GroupBy: g, Items: []ChannelMonitorV2MatrixRow{}}, nil
}
func (s *catalogMonitorRepo) GetDimensions(_ context.Context, f ChannelMonitorV2Filter, cfg ChannelMonitorV2Config) (*ChannelMonitorV2Dimensions, error) {
	s.received = f
	s.receivedConfig = cfg
	return &ChannelMonitorV2Dimensions{}, nil
}
func (s *catalogMonitorRepo) GetModels(_ context.Context, f ChannelMonitorV2Filter, cfg ChannelMonitorV2Config, _ bool) (*ChannelMonitorV2List[ChannelMonitorV2ModelRow], error) {
	s.received = f
	s.receivedConfig = cfg
	return &ChannelMonitorV2List[ChannelMonitorV2ModelRow]{}, nil
}

func TestChannelMonitorV2CatalogIsScopedBeforeRepository(t *testing.T) {
	for _, endpoint := range []string{"matrix", "models", "dimensions"} {
		t.Run(endpoint, func(t *testing.T) {
			repo := &catalogMonitorRepo{channelMonitorV2RepoStub: channelMonitorV2RepoStub{config: ChannelMonitorV2Config{Enabled: true, GroupIDs: []int64{1, 2, 3}}}}
			catalog := &monitorCatalogStub{groups: []PlazaGroup{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 99}}}
			svc := NewChannelMonitorV2Service(repo)
			svc.SetModelCatalogReader(catalog)
			filter := ChannelMonitorV2Filter{RestrictGroups: true, AllowedGroupIDs: []int64{2, 3, 99}, GroupIDs: []int64{3}, Platforms: []string{"openai"}, Models: []string{"chosen"}}
			var err error
			switch endpoint {
			case "matrix":
				_, err = svc.Matrix(context.Background(), filter, ChannelMonitorV2GroupByPlatformGroupModel, false)
			case "models":
				_, err = svc.Models(context.Background(), filter, false)
			case "dimensions":
				_, err = svc.Dimensions(context.Background(), filter)
			}
			require.NoError(t, err)
			require.Equal(t, 1, catalog.calls)
			require.Equal(t, []PlazaGroup{{ID: 2}, {ID: 3}}, repo.received.Catalog, "catalog keeps authorized picker options independent of current selection")
			require.Equal(t, filter.GroupIDs, repo.received.GroupIDs)
			require.Equal(t, filter.Models, repo.received.Models)
			require.True(t, repo.received.RestrictGroups)
		})
	}
}
func TestChannelMonitorV2CatalogFailureDoesNotFallbackToGlobal(t *testing.T) {
	repo := &catalogMonitorRepo{channelMonitorV2RepoStub: channelMonitorV2RepoStub{config: ChannelMonitorV2Config{Enabled: true}}}
	svc := NewChannelMonitorV2Service(repo)
	catalog := &monitorCatalogStub{err: errors.New("catalog unavailable")}
	svc.SetModelCatalogReader(catalog)
	_, err := svc.Matrix(context.Background(), ChannelMonitorV2Filter{}, ChannelMonitorV2GroupByPlatformGroupModel, false)
	require.ErrorContains(t, err, "catalog unavailable")
	require.Nil(t, repo.received.Catalog)
}
func TestChannelMonitorV2EmptyViewerScopeSkipsCatalog(t *testing.T) {
	repo := &catalogMonitorRepo{channelMonitorV2RepoStub: channelMonitorV2RepoStub{config: ChannelMonitorV2Config{Enabled: true, GroupIDs: []int64{1}}}}
	svc := NewChannelMonitorV2Service(repo)
	catalog := &monitorCatalogStub{groups: []PlazaGroup{{ID: 1}}}
	svc.SetModelCatalogReader(catalog)
	_, err := svc.Matrix(context.Background(), ChannelMonitorV2Filter{RestrictGroups: true, AllowedGroupIDs: []int64{2}}, ChannelMonitorV2GroupByPlatformGroupModel, false)
	require.NoError(t, err)
	require.Zero(t, catalog.calls)
	require.NotNil(t, repo.received.Catalog)
	require.Empty(t, repo.received.Catalog)
}

func (s *catalogMonitorRepo) GetSnapshot(_ context.Context, f ChannelMonitorV2Filter, cfg ChannelMonitorV2Config, _ bool) (*ChannelMonitorV2Snapshot, error) {
	s.received = f
	s.receivedConfig = cfg
	s.receivedConfig.Platforms = append([]ChannelMonitorV2PlatformConfig(nil), cfg.Platforms...)
	return &ChannelMonitorV2Snapshot{Config: cfg}, nil
}
func (s *catalogMonitorRepo) GetErrors(_ context.Context, f ChannelMonitorV2Filter, cfg ChannelMonitorV2Config, _ bool) (*ChannelMonitorV2List[ChannelMonitorV2ErrorRow], error) {
	s.received = f
	s.receivedConfig = cfg
	return &ChannelMonitorV2List[ChannelMonitorV2ErrorRow]{}, nil
}
func (s *catalogMonitorRepo) GetUsers(_ context.Context, f ChannelMonitorV2Filter, cfg ChannelMonitorV2Config, _ bool) (*ChannelMonitorV2List[ChannelMonitorV2UserRow], error) {
	s.received = f
	s.receivedConfig = cfg
	return &ChannelMonitorV2List[ChannelMonitorV2UserRow]{}, nil
}
func TestChannelMonitorV2DiscoversPlatformsOnlyInsideSelectedAuthorizedMonitors(t *testing.T) {
	for _, endpoint := range []string{"dimensions", "models", "matrix", "snapshot", "errors", "users"} {
		t.Run(endpoint, func(t *testing.T) {
			saved := ChannelMonitorV2Config{Enabled: true, GroupIDs: []int64{1, 2, 3, 4, 7}, Platforms: []ChannelMonitorV2PlatformConfig{
				{Platform: PlatformOpenAI, Enabled: true, Models: []string{"explicit-whitelist"}},
				{Platform: PlatformKimi, Enabled: false},
			}}
			repo := &catalogMonitorRepo{channelMonitorV2RepoStub: channelMonitorV2RepoStub{config: saved}}
			catalog := &monitorCatalogStub{groups: []PlazaGroup{
				{ID: 1, Platform: PlatformOpenAI}, {ID: 2, Platform: PlatformKimi}, {ID: 3, Platform: PlatformZhipu},
				{ID: 4, Platform: PlatformGemini}, {ID: 5, Platform: PlatformGrok}, {ID: 6, Platform: PlatformMiniMax},
				{ID: 7, Platform: PlatformComposite, Models: []PlazaModel{{Platform: PlatformDeepseek, Name: "deepseek-test"}}},
			}}
			svc := NewChannelMonitorV2Service(repo)
			svc.SetModelCatalogReader(catalog)
			filter := ChannelMonitorV2Filter{RestrictGroups: true, AllowedGroupIDs: []int64{2, 4, 5, 6, 7}, Platforms: []string{PlatformGemini}}
			var err error
			switch endpoint {
			case "dimensions":
				_, err = svc.Dimensions(context.Background(), filter)
			case "models":
				_, err = svc.Models(context.Background(), filter, false)
			case "matrix":
				_, err = svc.Matrix(context.Background(), filter, ChannelMonitorV2GroupByPlatformGroupModel, false)
			case "snapshot":
				_, err = svc.Snapshot(context.Background(), filter, false)
			case "errors":
				_, err = svc.ErrorsForViewer(context.Background(), filter, false)
			case "users":
				_, err = svc.Users(context.Background(), filter, 42, false)
			}
			require.NoError(t, err)
			require.Equal(t, 1, catalog.calls)
			ids := []int64{}
			for _, g := range repo.received.Catalog {
				ids = append(ids, g.ID)
			}
			require.Equal(t, []int64{2, 4, 7}, ids, "monitor selection and viewer permissions must both apply")
			policies := map[string]ChannelMonitorV2PlatformConfig{}
			for _, p := range repo.receivedConfig.Platforms {
				policies[p.Platform] = p
			}
			require.Len(t, policies, 4)
			require.False(t, policies[PlatformKimi].Enabled, "explicit platform disable cannot be auto-enabled")
			require.Equal(t, []string{"explicit-whitelist"}, policies[PlatformOpenAI].Models)
			require.True(t, policies[PlatformGemini].Enabled)
			require.Empty(t, policies[PlatformGemini].Models)
			require.True(t, policies[PlatformDeepseek].Enabled)
			require.NotContains(t, policies, PlatformZhipu, "unauthorized group must not create platform")
			require.NotContains(t, policies, PlatformGrok, "unselected monitor group must not create platform")
			require.NotContains(t, policies, PlatformMiniMax)
			require.NotContains(t, policies, PlatformComposite)
			require.Equal(t, saved, repo.config, "runtime discovery must not change saved configuration")
		})
	}
}
