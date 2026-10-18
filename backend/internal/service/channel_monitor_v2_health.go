package service

import (
	"fmt"
	"strings"
)

func validateChannelMonitorV2GenerationOverrides(rules []ChannelMonitorV2GenerationOverride) error {
	if len(rules) > 200 {
		return fmt.Errorf("%w: generation_overrides must contain at most 200 rules", ErrChannelMonitorV2InvalidConfig)
	}
	seen := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		platform := strings.ToLower(strings.TrimSpace(rule.Platform))
		model := strings.TrimSpace(rule.Model)
		if rule.GroupID < 0 || (platform == "" && model == "" && rule.GroupID == 0) {
			return fmt.Errorf("%w: generation override requires a platform, model, or positive group_id", ErrChannelMonitorV2InvalidConfig)
		}
		if rule.WarningGenerationMs <= 0 || rule.CriticalGenerationMs <= rule.WarningGenerationMs {
			return fmt.Errorf("%w: generation override critical_generation_ms must exceed positive warning_generation_ms", ErrChannelMonitorV2InvalidConfig)
		}
		key := fmt.Sprintf("%s\x00%s\x00%d", platform, model, rule.GroupID)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("%w: duplicate generation override selectors", ErrChannelMonitorV2InvalidConfig)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func channelMonitorV2GenerationThresholdsForContext(thresholds ChannelMonitorV2HealthThresholds, row ChannelMonitorV2HealthContext) ChannelMonitorV2HealthThresholds {
	thresholds = NormalizeChannelMonitorV2HealthThresholds(thresholds)
	bestPriority := 0
	var selected *ChannelMonitorV2GenerationOverride
	for i := range thresholds.GenerationOverrides {
		rule := &thresholds.GenerationOverrides[i]
		selectors, priority := 0, 0
		if rule.Platform != "" {
			if rule.Platform != strings.ToLower(strings.TrimSpace(row.Platform)) {
				continue
			}
			selectors++
			priority += 1
		}
		if rule.Model != "" {
			if rule.Model != strings.TrimSpace(row.Model) {
				continue
			}
			selectors++
			priority += 2
		}
		if rule.GroupID > 0 {
			if rule.GroupID != row.GroupID {
				continue
			}
			selectors++
			priority += 4
		}
		priority += selectors * 8
		if selectors > 0 && priority > bestPriority {
			selected, bestPriority = rule, priority
		}
	}
	if selected != nil && ChannelMonitorV2IsImageModel(row.Model) {
		thresholds.WarningGenerationMs = selected.WarningGenerationMs
		thresholds.CriticalGenerationMs = selected.CriticalGenerationMs
	}
	// Full operator override rules remain on admin config only. Health exposes
	// the effective row budget without leaking other groups or model policies.
	thresholds.GenerationOverrides = nil
	return thresholds
}

func channelMonitorV2GenerationLatency(duration ChannelMonitorV2Latency) (*int64, string) {
	for _, sample := range []struct {
		value *int64
		name  string
	}{{duration.P95Ms, "p95"}, {duration.P90Ms, "p90"}, {duration.P50Ms, "p50"}} {
		if sample.value != nil && *sample.value >= 0 {
			return sample.value, sample.name
		}
	}
	return nil, ""
}

func channelMonitorV2AppendHealthReason(reasons []string, metric, state string) []string {
	if state == "warning" || state == "critical" {
		return append(reasons, metric+"_"+state)
	}
	return reasons
}
