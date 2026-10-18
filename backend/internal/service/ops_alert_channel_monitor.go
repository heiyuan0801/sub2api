package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

const opsChannelMonitorPending = "pending"

// ErrOpsAlertCheckpointConflict means another evaluator already advanced this rule.
var ErrOpsAlertCheckpointConflict = errors.New("ops alert checkpoint changed")

// Optional extension of the existing Ops repository; rules and events remain
// owned by Ops. A rule lock and revision comparison serialize checkpoint writes.
type opsChannelMonitorCheckpointRepository interface {
	SaveChannelMonitorAlertEvent(context.Context, *OpsAlertEvent, int64, int64) (*OpsAlertEvent, error)
}

type opsChannelMonitorSelector struct {
	Platform       string
	Model          string
	GroupID        int64
	MinimumSample  int
	Consecutive    int
	Recovery       int
	NotifyRecovery bool
}

// Counts refer to distinct successful aggregation watermarks, not independent
// disjoint windows. SustainedMinutes additionally enforces real elapsed time.
type opsChannelMonitorState struct {
	Revision               int64     `json:"revision"`
	Fingerprint            string    `json:"fingerprint"`
	LastTick               time.Time `json:"last_tick"`
	BreachStartedAt        time.Time `json:"breach_started_at"`
	Breaches               int       `json:"breaches"`
	Recoveries             int       `json:"recoveries"`
	Fired                  bool      `json:"fired"`
	AlertNotified          bool      `json:"alert_notified"`
	RecoveryNotified       bool      `json:"recovery_notified"`
	CooldownUntil          time.Time `json:"cooldown_until"`
	NotificationClaimID    string    `json:"notification_claim_id,omitempty"`
	NotificationClaimUntil time.Time `json:"notification_claim_until,omitempty"`
}

func IsOpsChannelMonitorMetric(metric string) bool {
	switch metric {
	case "channel_monitor_success_rate", "channel_monitor_cache_rate", "channel_monitor_ttft_ms", "channel_monitor_duration_ms":
		return true
	}
	return false
}

func opsChannelMonitorInteger(filters map[string]any, key string, fallback, maximum int) (int, error) {
	value, exists := filters[key]
	if !exists {
		return fallback, nil
	}
	var number float64
	switch v := value.(type) {
	case int:
		number = float64(v)
	case int64:
		number = float64(v)
	case float64:
		number = v
	default:
		return 0, fmt.Errorf("filters.%s must be an integer", key)
	}
	if math.IsNaN(number) || math.IsInf(number, 0) || number != math.Trunc(number) || number < 1 || number > float64(maximum) {
		return 0, fmt.Errorf("filters.%s must be between 1 and %d", key, maximum)
	}
	return int(number), nil
}

func parseOpsChannelMonitorSelector(rule *OpsAlertRule) (opsChannelMonitorSelector, error) {
	selector := opsChannelMonitorSelector{}
	if rule == nil {
		return selector, errors.New("missing rule")
	}
	selector.Platform, _ = rule.Filters["platform"].(string)
	selector.Model, _ = rule.Filters["model"].(string)
	selector.Platform = strings.ToLower(strings.TrimSpace(selector.Platform))
	selector.Model = strings.TrimSpace(selector.Model)
	if strings.ContainsFunc(selector.Platform, unicode.IsControl) || strings.ContainsFunc(selector.Model, unicode.IsControl) {
		return selector, errors.New("platform and model cannot contain control characters")
	}
	if selector.Platform == "" || len(selector.Platform) > 64 || selector.Model == "" || len(selector.Model) > 256 {
		return selector, errors.New("channel monitor rules require exact filters.platform (1..64 characters) and filters.model (1..256 characters)")
	}
	group, err := opsChannelMonitorInteger(rule.Filters, "group_id", 0, int(^uint(0)>>1))
	if err != nil || group <= 0 {
		return selector, errors.New("filters.group_id must be a positive integer")
	}
	selector.GroupID = int64(group)
	if selector.MinimumSample, err = opsChannelMonitorInteger(rule.Filters, "minimum_sample", 20, 1000000); err != nil {
		return selector, err
	}
	if selector.Consecutive, err = opsChannelMonitorInteger(rule.Filters, "consecutive_windows", 3, 60); err != nil {
		return selector, err
	}
	if selector.Recovery, err = opsChannelMonitorInteger(rule.Filters, "recovery_windows", 2, 60); err != nil {
		return selector, err
	}
	if value, exists := rule.Filters["notify_recovery"]; exists {
		var ok bool
		if selector.NotifyRecovery, ok = value.(bool); !ok {
			return selector, errors.New("filters.notify_recovery must be a boolean")
		}
	}
	return selector, nil
}

// ValidateOpsChannelMonitorRule also protects non-HTTP rule callers.
func ValidateOpsChannelMonitorRule(rule *OpsAlertRule) error {
	if rule == nil || !IsOpsChannelMonitorMetric(rule.MetricType) {
		return nil
	}
	if _, err := parseOpsChannelMonitorSelector(rule); err != nil {
		return err
	}
	if math.IsNaN(rule.Threshold) || math.IsInf(rule.Threshold, 0) || rule.Threshold < 0 {
		return errors.New("threshold must be finite and nonnegative")
	}
	if (rule.MetricType == "channel_monitor_success_rate" || rule.MetricType == "channel_monitor_cache_rate") && rule.Threshold > 100 {
		return errors.New("rate threshold must be between 0 and 100")
	}
	switch rule.Operator {
	case ">", "<", ">=", "<=", "==", "!=":
	default:
		return errors.New("invalid alert operator")
	}
	if rule.WindowMinutes != 1 && rule.WindowMinutes != 5 && rule.WindowMinutes != 60 {
		return errors.New("window_minutes must be one of 1, 5, 60")
	}
	if rule.SustainedMinutes < 1 || rule.SustainedMinutes > 1440 || rule.CooldownMinutes < 0 || rule.CooldownMinutes > 1440 {
		return errors.New("invalid sustained_minutes or cooldown_minutes")
	}
	return nil
}

func readOpsChannelMonitorState(event *OpsAlertEvent) opsChannelMonitorState {
	state := opsChannelMonitorState{}
	if event != nil {
		encoded, _ := json.Marshal(event.Dimensions["channel_monitor_state"])
		_ = json.Unmarshal(encoded, &state)
	}
	return state
}

func opsChannelMonitorFingerprint(rule *OpsAlertRule, selector opsChannelMonitorSelector) string {
	encoded, _ := json.Marshal([]any{selector, rule.MetricType, rule.Operator, rule.Threshold, rule.WindowMinutes, rule.SustainedMinutes, rule.CooldownMinutes})
	return fmt.Sprintf("%x", sha256.Sum256(encoded))
}

func opsChannelMonitorMetricValue(metric ChannelMonitorV2Metric, metricType string, minimum int) (float64, bool) {
	if !metric.HasData || metric.RequestCount < int64(minimum) {
		return 0, false
	}
	switch metricType {
	case "channel_monitor_success_rate":
		return metric.SuccessRate * 100, true
	case "channel_monitor_cache_rate":
		return metric.CacheRate * 100, metric.CacheRateAvailable && metric.CacheRateDenominator > 0
	case "channel_monitor_ttft_ms":
		if metric.TTFT.SampleCount >= int64(minimum) && metric.TTFT.P50Ms != nil {
			return float64(*metric.TTFT.P50Ms), true
		}
	case "channel_monitor_duration_ms":
		if duration, _ := channelMonitorV2GenerationLatency(metric.Duration); metric.Duration.SampleCount >= int64(minimum) && duration != nil {
			return float64(*duration), true
		}
	}
	return 0, false
}

func (s *OpsAlertEvaluatorService) checkpointChannelMonitor(ctx context.Context, event *OpsAlertEvent, state opsChannelMonitorState, expectedID, expectedRevision int64) (*OpsAlertEvent, error) {
	repo, ok := s.opsRepo.(opsChannelMonitorCheckpointRepository)
	if !ok {
		return nil, errors.New("ops repository does not support durable monitor checkpoints")
	}
	state.Revision = expectedRevision + 1
	if event.Dimensions == nil {
		event.Dimensions = map[string]any{}
	}
	event.Dimensions["channel_monitor_state"] = state
	return repo.SaveChannelMonitorAlertEvent(ctx, event, expectedID, expectedRevision)
}

func (s *OpsAlertEvaluatorService) deliverChannelMonitorEvent(ctx context.Context, runtime *OpsAlertRuntimeSettings, rule *OpsAlertRule, event *OpsAlertEvent, state opsChannelMonitorState, selector opsChannelMonitorSelector) error {
	if event == nil {
		return nil
	}
	// maybeSendAlertEmail marks delivery on the existing event before this
	// evaluator checkpoints its state. Repair that crash gap without resending.
	if event.EmailSent && state.Fired && ((event.Status == OpsAlertStatusFiring && !state.AlertNotified) || (event.Status == OpsAlertStatusResolved && state.AlertNotified && selector.NotifyRecovery && !state.RecoveryNotified)) {
		if event.Status == OpsAlertStatusFiring {
			state.AlertNotified = true
		} else {
			state.RecoveryNotified = true
		}
		state.NotificationClaimID, state.NotificationClaimUntil = "", time.Time{}
		saved, err := s.checkpointChannelMonitor(ctx, event, state, event.ID, state.Revision)
		if err == nil {
			*event = *saved
		}
		return err
	}
	if !rule.NotifyEmail {
		return nil
	}
	isRecovery := event.Status == OpsAlertStatusResolved && state.Fired && state.AlertNotified && selector.NotifyRecovery && !state.RecoveryNotified
	isAlert := event.Status == OpsAlertStatusFiring && !state.AlertNotified
	if !isAlert && !isRecovery {
		return nil
	}
	if time.Now().UTC().Before(state.NotificationClaimUntil) {
		return nil
	}
	if s.opsService != nil {
		if silenced, err := s.opsService.IsAlertSilenced(ctx, rule.ID, selector.Platform, &selector.GroupID, nil, time.Now().UTC()); err != nil || silenced {
			return err
		}
	}
	copyEvent := *event
	copyEvent.EmailSent = false
	state.NotificationClaimID = uuid.NewString()
	state.NotificationClaimUntil = time.Now().UTC().Add(2 * time.Minute)
	claimed, err := s.checkpointChannelMonitor(ctx, event, state, event.ID, state.Revision)
	if err != nil {
		return err
	}
	claimID := state.NotificationClaimID
	send := s.sendChannelMonitorEmail
	if send == nil {
		send = s.maybeSendAlertEmail
	}
	sent := send(ctx, runtime, rule, &copyEvent)
	// A distinct aggregation tick may have checkpointed while SMTP was running.
	// Merge only this delivery result into the latest durable state.
	current, err := s.opsRepo.GetLatestAlertEvent(ctx, rule.ID)
	if err != nil {
		return err
	}
	if current == nil || current.ID != claimed.ID {
		return ErrOpsAlertCheckpointConflict
	}
	state = readOpsChannelMonitorState(current)
	if state.NotificationClaimID != claimID {
		return ErrOpsAlertCheckpointConflict
	}
	state.NotificationClaimID, state.NotificationClaimUntil = "", time.Time{}
	if sent {
		if isRecovery {
			state.RecoveryNotified = true
		} else {
			state.AlertNotified = true
			current.EmailSent = true
		}
	}
	saved, err := s.checkpointChannelMonitor(ctx, current, state, current.ID, state.Revision)
	if err == nil {
		*event = *saved
	}
	return err
}

func (s *OpsAlertEvaluatorService) evaluateChannelMonitorRule(ctx context.Context, runtime *OpsAlertRuntimeSettings, rule *OpsAlertRule, now time.Time) error {
	if !rule.Enabled || s.channelMonitorRepo == nil || s.channelMonitorSettings == nil || !s.channelMonitorSettings.GetChannelMonitorRuntime(ctx).PassiveAggregationAllowed() {
		return nil
	}
	if err := ValidateOpsChannelMonitorRule(rule); err != nil {
		return err
	}
	selector, _ := parseOpsChannelMonitorSelector(rule)
	config, err := s.channelMonitorRepo.GetConfig(ctx)
	if err != nil || config == nil || !config.Enabled {
		return err
	}
	latest, err := s.opsRepo.GetLatestAlertEvent(ctx, rule.ID)
	if err != nil {
		return err
	}
	state := readOpsChannelMonitorState(latest)
	fingerprint := opsChannelMonitorFingerprint(rule, selector)
	if latest != nil && state.Fingerprint != fingerprint {
		// Editing a selector/threshold closes the old incident without falsely
		// announcing recovery for another model or a changed policy.
		if latest.Status == opsChannelMonitorPending {
			state = opsChannelMonitorState{Revision: state.Revision, Fingerprint: fingerprint}
			latest.Title = rule.Name
			latest.Dimensions = map[string]any{"platform": selector.Platform, "group_id": selector.GroupID, "model": selector.Model, "channel_monitor_state": state}
		} else if latest.Status == OpsAlertStatusFiring {
			latest.Status, latest.ResolvedAt = OpsAlertStatusManualResolved, &now
			state.RecoveryNotified = true
			_, err = s.checkpointChannelMonitor(ctx, latest, state, latest.ID, state.Revision)
			return err
		}
	}
	// Retry failed delivery even if aggregation has not advanced. The existing
	// notification service deduplicates firing and recovery independently.
	if err := s.deliverChannelMonitorEvent(ctx, runtime, rule, latest, state, selector); err != nil {
		return err
	}
	if latest != nil {
		state = readOpsChannelMonitorState(latest)
	}
	watermark, err := s.channelMonitorRepo.GetAggregationWatermark(ctx)
	if err != nil || watermark == nil || !watermark.HasData {
		return err
	}
	tick := watermark.DataThrough.UTC().Truncate(time.Minute)
	if tick.After(now) || tick.IsZero() || !tick.After(state.LastTick) {
		return nil
	}
	refresh := time.Duration(config.RefreshIntervalSeconds) * time.Second
	if refresh < time.Minute {
		refresh = time.Minute
	}
	if now.Sub(tick) > refresh*2 {
		return nil
	}
	start := tick.Add(-time.Duration(rule.WindowMinutes) * time.Minute)
	if watermark.UsageCoverageStart.IsZero() || watermark.ErrorCoverageStart.IsZero() || watermark.UsageCoverageStart.After(start) || watermark.ErrorCoverageStart.After(start) {
		return nil
	}
	filter := ChannelMonitorV2Filter{Platforms: []string{selector.Platform}, Models: []string{selector.Model}, GroupIDs: []int64{selector.GroupID}, Start: start, End: tick, Bucket: time.Minute}
	matrix, err := s.channelMonitorRepo.GetMatrix(ctx, filter, *config, ChannelMonitorV2GroupByPlatformGroupModel, true)
	if err != nil || matrix == nil || matrix.Coverage.DataThrough.Before(tick) || !matrix.Coverage.CoverageComplete {
		return err
	}
	value, valid := 0.0, false
	for _, row := range matrix.Items {
		if row.Platform == selector.Platform && row.Model == selector.Model && row.GroupID != nil && *row.GroupID == selector.GroupID {
			value, valid = opsChannelMonitorMetricValue(row.Metrics, rule.MetricType, selector.MinimumSample)
			break
		}
	}
	breached := valid && compareMetric(value, rule.Operator, rule.Threshold)
	expectedID, expectedRevision := int64(0), state.Revision
	if latest != nil {
		expectedID = latest.ID
	}
	closed := latest == nil || (latest.Status != opsChannelMonitorPending && latest.Status != OpsAlertStatusFiring)
	if closed {
		if !breached {
			return nil
		}
		cooldown := time.Time{}
		if state.Fired && latest != nil {
			cooldown = latest.FiredAt.Add(time.Duration(rule.CooldownMinutes) * time.Minute)
		}
		state = opsChannelMonitorState{Fingerprint: fingerprint, CooldownUntil: cooldown}
		latest = &OpsAlertEvent{RuleID: rule.ID, Severity: rule.Severity, Status: opsChannelMonitorPending,
			Title: rule.Name, FiredAt: now, CreatedAt: now,
			Dimensions: map[string]any{"platform": selector.Platform, "group_id": selector.GroupID, "model": selector.Model}}
	}
	if !state.LastTick.IsZero() && tick.Sub(state.LastTick) > refresh*2 {
		state.Breaches, state.Recoveries, state.BreachStartedAt = 0, 0, time.Time{}
	}
	state.LastTick = tick
	if !valid {
		state.Breaches, state.Recoveries, state.BreachStartedAt = 0, 0, time.Time{}
	} else if breached {
		if state.Breaches == 0 {
			state.BreachStartedAt = tick
		}
		state.Breaches++
		state.Recoveries = 0
	} else {
		state.Recoveries++
		state.Breaches, state.BreachStartedAt = 0, time.Time{}
	}
	latest.MetricValue, latest.ThresholdValue = float64Ptr(value), float64Ptr(rule.Threshold)
	if !valid {
		latest.MetricValue = nil
	}
	latest.Description = buildOpsAlertDescription(rule, value, rule.WindowMinutes, selector.Platform, &selector.GroupID) + " model=" + selector.Model
	if latest.Status == opsChannelMonitorPending && breached && state.Breaches >= selector.Consecutive && tick.Sub(state.BreachStartedAt) >= time.Duration(rule.SustainedMinutes)*time.Minute && !now.Before(state.CooldownUntil) {
		latest.Status, latest.FiredAt, state.Fired = OpsAlertStatusFiring, now, true
	}
	if latest.Status == OpsAlertStatusFiring && valid && !breached && state.Recoveries >= selector.Recovery {
		latest.Status, latest.ResolvedAt = OpsAlertStatusResolved, &now
		if selector.NotifyRecovery && state.AlertNotified {
			latest.EmailSent = false
		}
	}
	saved, err := s.checkpointChannelMonitor(ctx, latest, state, expectedID, expectedRevision)
	if err != nil {
		return err
	}
	return s.deliverChannelMonitorEvent(ctx, runtime, rule, saved, readOpsChannelMonitorState(saved), selector)
}
