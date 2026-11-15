//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestAdminGroupDisplayTags(t *testing.T) {
	ctx := context.Background()
	repo := &groupRepoStubForAdmin{}
	svc := &adminServiceImpl{groupRepo: repo}
	created, err := svc.CreateGroup(ctx, &CreateGroupInput{
		Name: "standard", Platform: PlatformAnthropic, RateMultiplier: 1,
		DisplayTags: []string{"new", "recommended", "new"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"new", "recommended"}, created.DisplayTags)

	repo.getByID = created
	repo.updated = nil
	invalid := []string{"unknown"}
	_, err = svc.UpdateGroup(ctx, created.ID, &UpdateGroupInput{DisplayTags: &invalid})
	require.Equal(t, "INVALID_GROUP_DISPLAY_TAGS", infraerrors.FromError(err).Reason)
	require.Nil(t, repo.updated)

	svc.cfg = &config.Config{RunMode: config.RunModeSimple}
	unchanged, err := svc.UpdateGroup(ctx, created.ID, &UpdateGroupInput{Name: "standard v2"})
	require.NoError(t, err)
	require.Equal(t, []string{"new", "recommended"}, unchanged.DisplayTags)

	copy := cloneGroupForDuplicate(unchanged, "")
	require.Equal(t, unchanged.DisplayTags, copy.DisplayTags)
	copy.DisplayTags[0] = "popular"
	require.Equal(t, "new", unchanged.DisplayTags[0])

	empty := []string{}
	cleared, err := svc.UpdateGroup(ctx, created.ID, &UpdateGroupInput{DisplayTags: &empty})
	require.NoError(t, err)
	require.Equal(t, []string{}, cleared.DisplayTags)
	require.Equal(t, 1.0, cleared.RateMultiplier)
}
