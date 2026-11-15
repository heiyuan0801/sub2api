package service

import (
	"context"
	"slices"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// PlanPresentation governs sale visibility. It does not grant API access or
// change the quotas of the underlying subscription group.
type PlanPresentation struct {
	DisplayTags        []string `json:"display_tags"`
	VisibilityScope    string   `json:"visibility_scope"`
	VisibilityUserIDs  []int64  `json:"visibility_user_ids"`
	VisibilityGroupIDs []int64  `json:"visibility_group_ids"`
}

func PlanPresentationFromPlan(plan *dbent.SubscriptionPlan) PlanPresentation {
	return PlanPresentation{
		DisplayTags: plan.DisplayTags, VisibilityScope: plan.VisibilityScope,
		VisibilityUserIDs: plan.VisibilityUserIds, VisibilityGroupIDs: plan.VisibilityGroupIds,
	}
}

func normalizePlanPresentation(ctx context.Context, client *dbent.Client, presentation PlanPresentation) (PlanPresentation, error) {
	bad := infraerrors.BadRequest("INVALID_PLAN_PRESENTATION", "invalid plan tags or visibility audience")
	presentation.VisibilityScope = strings.TrimSpace(presentation.VisibilityScope)
	if presentation.VisibilityScope == "" {
		presentation.VisibilityScope = "all"
	}
	tags := make([]string, 0, len(presentation.DisplayTags))
	for _, tag := range presentation.DisplayTags {
		if tag != "recommended" && tag != "popular" && tag != "new" {
			return presentation, bad
		}
		if !slices.Contains(tags, tag) {
			tags = append(tags, tag)
		}
	}
	presentation.DisplayTags = tags
	switch presentation.VisibilityScope {
	case "all":
		presentation.VisibilityUserIDs = []int64{}
		presentation.VisibilityGroupIDs = []int64{}
	case "users":
		ids := normalizedAudienceIDs(presentation.VisibilityUserIDs)
		if len(ids) == 0 {
			return presentation, bad
		}
		count, err := client.User.Query().Where(user.IDIn(ids...)).Count(ctx)
		if err != nil {
			return presentation, err
		}
		if count != len(ids) {
			return presentation, bad
		}
		presentation.VisibilityUserIDs, presentation.VisibilityGroupIDs = ids, []int64{}
	case "groups":
		ids := normalizedAudienceIDs(presentation.VisibilityGroupIDs)
		if len(ids) == 0 {
			return presentation, bad
		}
		count, err := client.Group.Query().Where(group.IDIn(ids...)).Count(ctx)
		if err != nil {
			return presentation, err
		}
		if count != len(ids) {
			return presentation, bad
		}
		presentation.VisibilityGroupIDs, presentation.VisibilityUserIDs = ids, []int64{}
	default:
		return presentation, bad
	}
	return presentation, nil
}

func normalizedAudienceIDs(ids []int64) []int64 {
	if len(ids) > 1000 {
		return nil
	}
	result := slices.Clone(ids)
	slices.Sort(result)
	result = slices.Compact(result)
	if len(result) > 0 && result[0] <= 0 {
		return nil
	}
	return result
}

func planViewerGroups(ctx context.Context, client *dbent.Client, userID int64) (map[int64]bool, error) {
	allowed, err := client.User.Query().Where(user.IDEQ(userID)).QueryAllowedGroups().
		Where(group.StatusEQ(StatusActive)).IDs(ctx)
	if err != nil {
		return nil, err
	}
	groups := make(map[int64]bool, len(allowed))
	for _, id := range allowed {
		groups[id] = true
	}
	now := time.Now()
	var subs []struct {
		GroupID int64 `json:"group_id"`
	}
	err = client.UserSubscription.Query().Where(
		usersubscription.UserIDEQ(userID), usersubscription.StatusEQ(SubscriptionStatusActive),
		usersubscription.StartsAtLTE(now), usersubscription.ExpiresAtGT(now),
	).Select(usersubscription.FieldGroupID).Scan(ctx, &subs)
	for _, sub := range subs {
		groups[sub.GroupID] = true
	}
	return groups, err
}

func canSeePaymentPlan(plan *dbent.SubscriptionPlan, userID int64, groups map[int64]bool) bool {
	switch plan.VisibilityScope {
	case "", "all":
		return true
	case "users":
		return userID > 0 && slices.Contains(plan.VisibilityUserIds, userID)
	case "groups":
		return slices.ContainsFunc(plan.VisibilityGroupIds, func(id int64) bool { return groups[id] })
	default:
		return false
	}
}

func checkPaymentPlanAccess(ctx context.Context, client *dbent.Client, plan *dbent.SubscriptionPlan, userID int64) error {
	var groups map[int64]bool
	if plan.VisibilityScope == "groups" && userID > 0 {
		var err error
		groups, err = planViewerGroups(ctx, client, userID)
		if err != nil {
			return err
		}
	}
	if !plan.ForSale || !canSeePaymentPlan(plan, userID, groups) {
		return infraerrors.NotFound("PLAN_NOT_AVAILABLE", "plan not found or not for sale")
	}
	return nil
}
