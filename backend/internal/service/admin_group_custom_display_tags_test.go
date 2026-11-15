//go:build unit

package service

import (
	"context"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestAdminGroupCustomDisplayTags(t *testing.T) {
	ctx := context.Background()
	repo := &groupRepoStubForAdmin{}
	svc := &adminServiceImpl{groupRepo: repo, cfg: &config.Config{RunMode: config.RunModeSimple}}
	initial := []GroupDisplayTag{{Label: "  高速 ⚡  ", Color: "blue", Icon: "bolt"}}
	created, err := svc.CreateGroup(ctx, &CreateGroupInput{
		Name: "custom tags", Platform: PlatformAnthropic,
		DisplayTags: []string{"recommended"}, CustomDisplayTags: initial,
	})
	require.NoError(t, err)
	expected := []GroupDisplayTag{{Label: "高速 ⚡", Color: "blue", Icon: "bolt"}}
	require.Equal(t, expected, created.CustomDisplayTags)
	require.Equal(t, []string{"recommended"}, created.DisplayTags)
	require.Equal(t, "  高速 ⚡  ", initial[0].Label)

	repo.getByID = created
	wrong := []GroupDisplayTag{{Label: "bad", Color: "blue", Icon: "unknown"}}
	_, err = svc.UpdateGroup(ctx, created.ID, &UpdateGroupInput{CustomDisplayTags: &wrong})
	require.Equal(t, "INVALID_GROUP_CUSTOM_DISPLAY_TAGS", infraerrors.FromError(err).Reason)
	require.Nil(t, repo.updated)

	unchanged, err := svc.UpdateGroup(ctx, created.ID, &UpdateGroupInput{Name: "renamed"})
	require.NoError(t, err)
	require.Equal(t, expected, unchanged.CustomDisplayTags)
	copied := cloneGroupForDuplicate(unchanged, "")
	require.Equal(t, expected, copied.CustomDisplayTags)
	copied.CustomDisplayTags[0].Label = "copy only"
	require.Equal(t, expected, unchanged.CustomDisplayTags)

	empty := []GroupDisplayTag{}
	cleared, err := svc.UpdateGroup(ctx, created.ID, &UpdateGroupInput{CustomDisplayTags: &empty})
	require.NoError(t, err)
	require.Equal(t, empty, cleared.CustomDisplayTags)
	require.Equal(t, []string{"recommended"}, cleared.DisplayTags)
}

func TestNormalizeGroupCustomDisplayTags(t *testing.T) {
	valid := GroupDisplayTag{Label: strings.Repeat("标", 20), Color: "violet", Icon: "gem"}
	got, err := normalizeGroupCustomDisplayTags([]GroupDisplayTag{valid})
	require.NoError(t, err)
	require.Equal(t, []GroupDisplayTag{valid}, got)

	tag := func(label string) GroupDisplayTag {
		return GroupDisplayTag{Label: label, Color: "rose", Icon: "star"}
	}
	for _, tc := range []struct {
		name string
		tags []GroupDisplayTag
	}{
		{"too many", make([]GroupDisplayTag, 6)},
		{"blank", []GroupDisplayTag{tag("　 ")}},
		{"too long", []GroupDisplayTag{tag(strings.Repeat("标", 21))}},
		{"case-folded duplicate", []GroupDisplayTag{tag("Key"), tag(" Key ")}},
		{"control character", []GroupDisplayTag{tag("label\n")}},
		{"bidirectional control", []GroupDisplayTag{tag("name\u202E")}},
		{"invalid UTF-8", []GroupDisplayTag{tag("\xff")}},
		{"invalid enum", []GroupDisplayTag{{Label: "badge", Color: "unknown", Icon: "tag"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := normalizeGroupCustomDisplayTags(tc.tags)
			require.Error(t, err)
			require.Equal(t, "INVALID_GROUP_CUSTOM_DISPLAY_TAGS", infraerrors.FromError(err).Reason)
		})
	}
}
