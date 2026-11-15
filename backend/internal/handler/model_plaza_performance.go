package handler

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// Performance returns V2 aggregates for a model visible in the current catalog.
func (h *ModelPlazaHandler) Performance(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	_, subject, authed, ok := h.requireAccess(c)
	if !ok {
		return
	}
	if h.monitorService == nil {
		response.NotFound(c, "Model performance is unavailable")
		return
	}
	model := strings.TrimSpace(c.Query("model"))
	platform := strings.ToLower(strings.TrimSpace(c.Query("platform")))
	if model == "" || len(model) > 256 || model == service.ChannelMonitorV2OtherModel || platform == "" || len(platform) > 64 {
		response.BadRequest(c, "valid model and platform are required")
		return
	}
	window := c.DefaultQuery("range", "24h")
	switch window {
	case "24h", "7d", "30d":
	default:
		response.BadRequest(c, "range must be 24h, 7d, or 30d")
		return
	}
	var groupID int64
	if raw, exists := c.GetQuery("group_id"); exists {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "invalid group_id")
			return
		}
		groupID = id
	}
	groups, ok := h.visibleGroups(c, subject, authed)
	if !ok {
		return
	}
	matches := matchingPlazaPerformanceGroups(groups, model, platform, c.GetHeader("Accept-Language"))
	if len(matches) == 0 {
		response.NotFound(c, "Model is not listed")
		return
	}
	if groupID != 0 {
		found := false
		for _, group := range matches {
			found = found || group.ID == groupID
		}
		if !found {
			response.NotFound(c, "Model is not listed in this group")
			return
		}
	}

	// Commercial visibility and the right to see monitor data are distinct.
	// Signed-out users see public standard groups only; signed-in users follow
	// the same available-group permissions as the channel monitor.
	admin := authed && channelMonitorV2IsAdmin(c)
	allowed := make(map[int64]struct{}, len(matches))
	if authed && !admin {
		available, err := h.apiKeyService.GetAvailableGroups(c.Request.Context(), subject.UserID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		for _, group := range available {
			allowed[group.ID] = struct{}{}
		}
	} else {
		for _, group := range matches {
			if admin || group.SubscriptionType == service.SubscriptionTypeStandard {
				allowed[group.ID] = struct{}{}
			}
		}
	}
	ids := make([]int64, 0, len(matches))
	for _, group := range matches {
		if _, permitted := allowed[group.ID]; permitted {
			ids = append(ids, group.ID)
		}
	}
	filter, err := h.monitorService.ParseFilter(window, []string{platform}, []string{model}, nil)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	filter.RestrictGroups, filter.AllowedGroupIDs = true, ids
	result, err := h.monitorService.ModelPerformance(c.Request.Context(), filter, matches, groupID, admin)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func matchingPlazaPerformanceGroups(groups []service.PlazaGroup, model, platform, language string) []service.PlazaGroup {
	out := make([]service.PlazaGroup, 0, len(groups))
	for _, group := range groups {
		if group.ID <= 0 {
			continue
		}
		for _, entry := range group.Models {
			if strings.EqualFold(strings.TrimSpace(entry.Name), model) && strings.EqualFold(entry.Platform, platform) {
				group.Name, _ = group.I18n.Localize(language, group.Name, group.Description)
				out = append(out, group)
				break
			}
		}
	}
	return out
}
