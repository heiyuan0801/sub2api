package service

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	maxGroupCustomDisplayTags = 5
	maxGroupDisplayTagLabel   = 20
)

func normalizeGroupCustomDisplayTags(input []GroupDisplayTag) ([]GroupDisplayTag, error) {
	badRequest := func(message string) error {
		return infraerrors.BadRequest("INVALID_GROUP_CUSTOM_DISPLAY_TAGS", message)
	}
	if len(input) > maxGroupCustomDisplayTags {
		return nil, badRequest("a group can have at most 5 custom display tags")
	}

	tags := make([]GroupDisplayTag, 0, len(input))
	for _, tag := range input {
		if !utf8.ValidString(tag.Label) || strings.ContainsFunc(tag.Label, func(r rune) bool {
			return unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r)
		}) {
			return nil, badRequest("custom display tag labels contain invalid characters")
		}
		tag.Label = strings.TrimSpace(tag.Label)
		if tag.Label == "" || utf8.RuneCountInString(tag.Label) > maxGroupDisplayTagLabel {
			return nil, badRequest("custom display tag labels must contain 1 to 20 characters")
		}
		for _, existing := range tags {
			if strings.EqualFold(existing.Label, tag.Label) {
				return nil, badRequest("custom display tag labels must be unique")
			}
		}
		switch tag.Color {
		case "blue", "violet", "emerald", "orange", "rose", "slate":
		default:
			return nil, badRequest("invalid custom display tag color")
		}
		switch tag.Icon {
		case "tag", "star", "flame", "sparkles", "bolt", "shield", "gem":
		default:
			return nil, badRequest("invalid custom display tag icon")
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

// An update that omits custom tags should not rewrite tags read before a
// concurrent update. The repository supplies the current tags on success.
func updateGroupRecord(ctx context.Context, repo GroupRepository, group *Group, writeCustomTags bool) error {
	if writeCustomTags {
		return repo.Update(ctx, group)
	}
	tags := group.CustomDisplayTags
	group.CustomDisplayTags = nil
	err := repo.Update(ctx, group)
	if group.CustomDisplayTags == nil {
		group.CustomDisplayTags = tags
	}
	return err
}
