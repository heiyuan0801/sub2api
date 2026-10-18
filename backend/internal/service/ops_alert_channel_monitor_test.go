//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type monitorAlertOpsRepo struct {
	OpsRepository
	events []*OpsAlertEvent
}

func cloneMonitorAlert(event *OpsAlertEvent) *OpsAlertEvent {
	if event == nil {
		return nil
	}
	encoded, _ := json.Marshal(event)
	var clone OpsAlertEvent
	_ = json.Unmarshal(encoded, &clone)
	return &clone
}
func (r *monitorAlertOpsRepo) GetLatestAlertEvent(context.Context, int64) (*OpsAlertEvent, error) {
	if len(r.events) == 0 {
		return nil, nil
	}
	return cloneMonitorAlert(r.events[len(r.events)-1]), nil
}
func (r *monitorAlertOpsRepo) SaveChannelMonitorAlertEvent(_ context.Context, event *OpsAlertEvent, expectedID, revision int64) (*OpsAlertEvent, error) {
	var latest *OpsAlertEvent
	if len(r.events) > 0 {
		latest = r.events[len(r.events)-1]
	}
	if (latest == nil && expectedID != 0) || (latest != nil && (latest.ID != expectedID || readOpsChannelMonitorState(latest).Revision != revision)) {
		return nil, ErrOpsAlertCheckpointConflict
	}
	copyEvent := cloneMonitorAlert(event)
	if copyEvent.ID == 0 {
		copyEvent.ID = int64(len(r.events) + 1)
		r.events = append(r.events, copyEvent)
	} else {
		r.events[len(r.events)-1] = copyEvent
	}
	return cloneMonitorAlert(copyEvent), nil
}

type monitorAlertMetricsRepo struct {
	ChannelMonitorV2Repository
	config     ChannelMonitorV2Config
	watermark  ChannelMonitorV2AggregationWatermark
	metric     ChannelMonitorV2Metric
	lastFilter ChannelMonitorV2Filter
	queries    int
	err        error
}

func (r *monitorAlertMetricsRepo) GetConfig(context.Context) (*ChannelMonitorV2Config, error) {
	return &r.config, r.err
}
func (r *monitorAlertMetricsRepo) GetAggregationWatermark(context.Context) (*ChannelMonitorV2AggregationWatermark, error) {
	return &r.watermark, r.err
}
func (r *monitorAlertMetricsRepo) GetMatrix(_ context.Context, filter ChannelMonitorV2Filter, _ ChannelMonitorV2Config, grouping ChannelMonitorV2GroupBy, admin bool) (*ChannelMonitorV2Matrix, error) {
	if !admin || grouping != ChannelMonitorV2GroupByPlatformGroupModel {
		return nil, errors.New("wrong monitoring scope")
	}
	r.lastFilter, r.queries = filter, r.queries+1
	group := int64(11)
	return &ChannelMonitorV2Matrix{Coverage: ChannelMonitorV2Coverage{DataThrough: r.watermark.DataThrough, CoverageComplete: true}, Items: []ChannelMonitorV2MatrixRow{{Platform: "openai", GroupID: &group, Model: "gpt-image-demo", Metrics: r.metric}}}, r.err
}

type monitorAlertRuntime struct{ enabled bool }

func (r *monitorAlertRuntime) GetChannelMonitorRuntime(context.Context) ChannelMonitorRuntime {
	return ChannelMonitorRuntime{Enabled: r.enabled, Mode: ChannelMonitorModeV2}
}

func newMonitorAlertFixture() (*OpsAlertEvaluatorService, *monitorAlertOpsRepo, *monitorAlertMetricsRepo, *OpsAlertRule, *[]string, time.Time) {
	base := time.Date(2026, 10, 17, 14, 0, 0, 0, time.UTC)
	duration := int64(45000)
	metrics := &monitorAlertMetricsRepo{config: ChannelMonitorV2Config{Enabled: true, RefreshIntervalSeconds: 60}, metric: ChannelMonitorV2Metric{HasData: true, RequestCount: 100, SuccessRate: .8, Duration: ChannelMonitorV2Latency{SampleCount: 100, P50Ms: &duration}}, watermark: ChannelMonitorV2AggregationWatermark{HasData: true, DataThrough: base, UsageCoverageStart: base.Add(-time.Hour), ErrorCoverageStart: base.Add(-time.Hour)}}
	repo := &monitorAlertOpsRepo{}
	sent := []string{}
	svc := &OpsAlertEvaluatorService{opsRepo: repo, channelMonitorRepo: metrics, channelMonitorSettings: &monitorAlertRuntime{enabled: true}, sendChannelMonitorEmail: func(_ context.Context, _ *OpsAlertRuntimeSettings, _ *OpsAlertRule, event *OpsAlertEvent) bool {
		sent = append(sent, event.Status)
		return true
	}}
	rule := &OpsAlertRule{ID: 1, Name: "Image payment-independent monitor", Enabled: true, NotifyEmail: true, Severity: "P2", MetricType: "channel_monitor_success_rate", Operator: "<", Threshold: 90, WindowMinutes: 5, SustainedMinutes: 1, CooldownMinutes: 10, Filters: map[string]any{"platform": "openai", "group_id": 11, "model": "gpt-image-demo", "minimum_sample": 20, "consecutive_windows": 3, "recovery_windows": 2, "notify_recovery": true}}
	return svc, repo, metrics, rule, &sent, base
}
func evaluateMonitorTick(t *testing.T, svc *OpsAlertEvaluatorService, metrics *monitorAlertMetricsRepo, rule *OpsAlertRule, tick time.Time) {
	t.Helper()
	metrics.watermark.DataThrough = tick
	require.NoError(t, svc.evaluateChannelMonitorRule(context.Background(), defaultOpsAlertRuntimeSettings(), rule, tick.Add(10*time.Second)))
}

func TestOpsChannelMonitorPersistsConsecutiveAlertAndRecovery(t *testing.T) {
	svc, repo, metrics, rule, sent, base := newMonitorAlertFixture()
	evaluateMonitorTick(t, svc, metrics, rule, base)
	require.Equal(t, opsChannelMonitorPending, repo.events[0].Status)
	for i := 0; i < 5; i++ {
		evaluateMonitorTick(t, svc, metrics, rule, base)
	}
	require.Equal(t, 1, readOpsChannelMonitorState(repo.events[0]).Breaches)
	require.Equal(t, 1, metrics.queries, "the same aggregation watermark must not be evaluated twice")
	require.Empty(t, *sent)
	// Rebuilding the evaluator retains progress solely through existing Ops events.
	restarted := *svc
	evaluateMonitorTick(t, &restarted, metrics, rule, base.Add(time.Minute))
	evaluateMonitorTick(t, &restarted, metrics, rule, base.Add(2*time.Minute))
	require.Equal(t, OpsAlertStatusFiring, repo.events[0].Status)
	require.Equal(t, []string{OpsAlertStatusFiring}, *sent)
	require.Equal(t, []string{"openai"}, metrics.lastFilter.Platforms)
	require.Equal(t, []int64{11}, metrics.lastFilter.GroupIDs)
	require.Equal(t, []string{"gpt-image-demo"}, metrics.lastFilter.Models)
	metrics.metric.SuccessRate = .99
	evaluateMonitorTick(t, &restarted, metrics, rule, base.Add(3*time.Minute))
	require.Equal(t, OpsAlertStatusFiring, repo.events[0].Status)
	evaluateMonitorTick(t, &restarted, metrics, rule, base.Add(4*time.Minute))
	require.Equal(t, OpsAlertStatusResolved, repo.events[0].Status)
	require.Equal(t, []string{OpsAlertStatusFiring, OpsAlertStatusResolved}, *sent)
	evaluateMonitorTick(t, &restarted, metrics, rule, base.Add(4*time.Minute))
	require.Len(t, *sent, 2)
	require.NotEqual(t, opsAlertEmailSourceID(&OpsAlertEvent{ID: 1, Status: OpsAlertStatusFiring}), opsAlertEmailSourceID(&OpsAlertEvent{ID: 1, Status: OpsAlertStatusResolved}))
}

func TestOpsChannelMonitorRequiresElapsedSustainedTime(t *testing.T) {
	svc, repo, metrics, rule, sent, base := newMonitorAlertFixture()
	rule.SustainedMinutes = 3
	rule.Filters["consecutive_windows"] = 1
	for i := 0; i < 3; i++ {
		evaluateMonitorTick(t, svc, metrics, rule, base.Add(time.Duration(i)*time.Minute))
	}
	require.Equal(t, opsChannelMonitorPending, repo.events[0].Status)
	require.Empty(t, *sent)
	evaluateMonitorTick(t, svc, metrics, rule, base.Add(3*time.Minute))
	require.Equal(t, OpsAlertStatusFiring, repo.events[0].Status)
}

func TestOpsChannelMonitorInsufficientSamplesNeverFireOrRecover(t *testing.T) {
	svc, repo, metrics, rule, sent, base := newMonitorAlertFixture()
	metrics.metric.RequestCount = 2
	evaluateMonitorTick(t, svc, metrics, rule, base)
	require.Empty(t, repo.events)
	metrics.metric.RequestCount = 100
	for i := 1; i <= 3; i++ {
		evaluateMonitorTick(t, svc, metrics, rule, base.Add(time.Duration(i)*time.Minute))
	}
	require.Equal(t, OpsAlertStatusFiring, repo.events[0].Status)
	metrics.metric.HasData = false
	metrics.metric.SuccessRate = 1
	for i := 4; i <= 7; i++ {
		evaluateMonitorTick(t, svc, metrics, rule, base.Add(time.Duration(i)*time.Minute))
	}
	require.Equal(t, OpsAlertStatusFiring, repo.events[0].Status)
	require.Equal(t, 0, readOpsChannelMonitorState(repo.events[0]).Recoveries)
	require.Len(t, *sent, 1)
}

func TestOpsChannelMonitorUnavailableImageCacheAndTTFTDoNotAlert(t *testing.T) {
	for _, kind := range []string{"channel_monitor_cache_rate", "channel_monitor_ttft_ms"} {
		t.Run(kind, func(t *testing.T) {
			svc, repo, metrics, rule, sent, base := newMonitorAlertFixture()
			rule.MetricType = kind
			for i := 0; i < 5; i++ {
				evaluateMonitorTick(t, svc, metrics, rule, base.Add(time.Duration(i)*time.Minute))
			}
			require.Empty(t, repo.events)
			require.Empty(t, *sent)
		})
	}
	metric := ChannelMonitorV2Metric{HasData: true, RequestCount: 100, CacheRateAvailable: true, CacheRateDenominator: 500, CacheRate: 0}
	value, ok := opsChannelMonitorMetricValue(metric, "channel_monitor_cache_rate", 20)
	require.True(t, ok, "an actual zero cache hit rate remains a valid sample")
	require.Zero(t, value)
}

func TestOpsChannelMonitorCooldownStillTracksRecoveryAndNewIncident(t *testing.T) {
	svc, repo, metrics, rule, sent, base := newMonitorAlertFixture()
	for i := 0; i < 3; i++ {
		evaluateMonitorTick(t, svc, metrics, rule, base.Add(time.Duration(i)*time.Minute))
	}
	metrics.metric.SuccessRate = .99
	for i := 3; i < 5; i++ {
		evaluateMonitorTick(t, svc, metrics, rule, base.Add(time.Duration(i)*time.Minute))
	}
	metrics.metric.SuccessRate = .8
	for i := 5; i < 8; i++ {
		evaluateMonitorTick(t, svc, metrics, rule, base.Add(time.Duration(i)*time.Minute))
	}
	require.Len(t, repo.events, 2)
	require.Equal(t, opsChannelMonitorPending, repo.events[1].Status)
	require.Equal(t, 3, readOpsChannelMonitorState(repo.events[1]).Breaches)
	require.Len(t, *sent, 2, "cooldown suppresses a new firing notification")
	metrics.metric.SuccessRate = .99
	for i := 8; i < 10; i++ {
		evaluateMonitorTick(t, svc, metrics, rule, base.Add(time.Duration(i)*time.Minute))
	}
	require.Equal(t, 0, readOpsChannelMonitorState(repo.events[1]).Breaches, "recovery is tracked inside the cooldown")
	metrics.metric.SuccessRate = .8
	for i := 10; i <= 13; i++ {
		evaluateMonitorTick(t, svc, metrics, rule, base.Add(time.Duration(i)*time.Minute))
	}
	require.Equal(t, OpsAlertStatusFiring, repo.events[1].Status)
	require.Len(t, *sent, 3)
}

func TestOpsChannelMonitorRetriesNotificationWithoutCountingWindowTwice(t *testing.T) {
	svc, repo, metrics, rule, sent, base := newMonitorAlertFixture()
	attempts := 0
	actualSender := svc.sendChannelMonitorEmail
	svc.sendChannelMonitorEmail = func(ctx context.Context, runtime *OpsAlertRuntimeSettings, r *OpsAlertRule, event *OpsAlertEvent) bool {
		attempts++
		if attempts == 1 {
			return false
		}
		return actualSender(ctx, runtime, r, event)
	}
	for i := 0; i < 3; i++ {
		evaluateMonitorTick(t, svc, metrics, rule, base.Add(time.Duration(i)*time.Minute))
	}
	require.False(t, readOpsChannelMonitorState(repo.events[0]).AlertNotified)
	evaluateMonitorTick(t, svc, metrics, rule, base.Add(2*time.Minute))
	require.Equal(t, 2, attempts)
	require.Len(t, *sent, 1)
	require.Equal(t, 3, readOpsChannelMonitorState(repo.events[0]).Breaches)
	require.True(t, readOpsChannelMonitorState(repo.events[0]).AlertNotified)
}

func TestOpsChannelMonitorDisabledStaleOrMissingCoverageDoesNotNotify(t *testing.T) {
	for _, kind := range []string{"disabled", "stale", "partial", "disabled-rule"} {
		t.Run(kind, func(t *testing.T) {
			svc, repo, metrics, rule, sent, base := newMonitorAlertFixture()
			switch kind {
			case "disabled":
				metrics.config.Enabled = false
			case "stale":
				metrics.watermark.DataThrough = base.Add(-time.Hour)
			case "partial":
				metrics.watermark.UsageCoverageStart = base
			case "disabled-rule":
				rule.Enabled = false
			}
			require.NoError(t, svc.evaluateChannelMonitorRule(context.Background(), defaultOpsAlertRuntimeSettings(), rule, base))
			require.Empty(t, repo.events)
			require.Empty(t, *sent)
		})
	}
}

func TestOpsChannelMonitorRuleValidation(t *testing.T) {
	_, _, _, rule, _, _ := newMonitorAlertFixture()
	require.NoError(t, ValidateOpsChannelMonitorRule(rule))
	for _, patch := range []map[string]any{{"model": ""}, {"group_id": 1.5}, {"minimum_sample": 0}, {"consecutive_windows": 61}, {"recovery_windows": "2"}, {"notify_recovery": "yes"}} {
		copyRule := *rule
		copyRule.Filters = map[string]any{}
		for key, value := range rule.Filters {
			copyRule.Filters[key] = value
		}
		for key, value := range patch {
			copyRule.Filters[key] = value
		}
		require.Error(t, ValidateOpsChannelMonitorRule(&copyRule))
	}
}

func TestOpsChannelMonitorDeliveryClaimPreventsAnotherEvaluatorSending(t *testing.T) {
	svc, _, metrics, rule, sent, base := newMonitorAlertFixture()
	actualSender := svc.sendChannelMonitorEmail
	svc.sendChannelMonitorEmail = func(ctx context.Context, runtime *OpsAlertRuntimeSettings, r *OpsAlertRule, event *OpsAlertEvent) bool {
		otherEvaluator := *svc
		require.NoError(t, otherEvaluator.evaluateChannelMonitorRule(ctx, runtime, r, metrics.watermark.DataThrough.Add(10*time.Second)))
		return actualSender(ctx, runtime, r, event)
	}
	for i := 0; i < 3; i++ {
		evaluateMonitorTick(t, svc, metrics, rule, base.Add(time.Duration(i)*time.Minute))
	}
	require.Equal(t, []string{OpsAlertStatusFiring}, *sent)
}

func TestOpsChannelMonitorGenerationDurationUsesItsOwnSamples(t *testing.T) {
	svc, repo, metrics, rule, sent, base := newMonitorAlertFixture()
	rule.MetricType, rule.Operator, rule.Threshold = "channel_monitor_duration_ms", ">", 30000
	for i := 0; i < 3; i++ {
		evaluateMonitorTick(t, svc, metrics, rule, base.Add(time.Duration(i)*time.Minute))
	}
	require.Equal(t, OpsAlertStatusFiring, repo.events[0].Status)
	require.Equal(t, float64(45000), *repo.events[0].MetricValue)
	require.Equal(t, []string{OpsAlertStatusFiring}, *sent)
}

func TestOpsChannelMonitorGenerationLatencyMatchesHealthTailPercentile(t *testing.T) {
	_, _, metrics, _, _, _ := newMonitorAlertFixture()
	tail := int64(95000)
	metrics.metric.Duration.P95Ms = &tail
	value, ok := opsChannelMonitorMetricValue(metrics.metric, "channel_monitor_duration_ms", 20)
	require.True(t, ok)
	require.Equal(t, float64(95000), value)
}

func TestOpsChannelMonitorEditedPendingRuleRemainsHiddenAndResetsProgress(t *testing.T) {
	svc, repo, metrics, rule, sent, base := newMonitorAlertFixture()
	evaluateMonitorTick(t, svc, metrics, rule, base)
	rule.Threshold = 95
	evaluateMonitorTick(t, svc, metrics, rule, base.Add(time.Minute))
	require.Len(t, repo.events, 1)
	require.Equal(t, opsChannelMonitorPending, repo.events[0].Status)
	require.Equal(t, 1, readOpsChannelMonitorState(repo.events[0]).Breaches)
	require.Empty(t, *sent)
}

func TestOpsChannelMonitorDeliveryMergesManualResolutionInsteadOfReopening(t *testing.T) {
	svc, repo, metrics, rule, sent, base := newMonitorAlertFixture()
	rule.NotifyEmail = false
	for i := 0; i < 3; i++ {
		evaluateMonitorTick(t, svc, metrics, rule, base.Add(time.Duration(i)*time.Minute))
	}
	rule.NotifyEmail = true
	actualSender := svc.sendChannelMonitorEmail
	svc.sendChannelMonitorEmail = func(ctx context.Context, runtime *OpsAlertRuntimeSettings, r *OpsAlertRule, event *OpsAlertEvent) bool {
		// Simulate the existing manual-resolution endpoint during SMTP delivery.
		repo.events[0].Status = OpsAlertStatusManualResolved
		resolvedAt := base.Add(3 * time.Minute)
		repo.events[0].ResolvedAt = &resolvedAt
		return actualSender(ctx, runtime, r, event)
	}
	evaluateMonitorTick(t, svc, metrics, rule, base.Add(3*time.Minute))
	require.Equal(t, OpsAlertStatusManualResolved, repo.events[0].Status)
	require.Len(t, repo.events, 2)
	require.Equal(t, opsChannelMonitorPending, repo.events[1].Status)
	require.Len(t, *sent, 1)
}

func TestOpsChannelMonitorRepairsPersistedAlertDeliveryAfterRestart(t *testing.T) {
	svc, repo, metrics, rule, sent, base := newMonitorAlertFixture()
	rule.NotifyEmail = false
	for i := 0; i < 3; i++ {
		evaluateMonitorTick(t, svc, metrics, rule, base.Add(time.Duration(i)*time.Minute))
	}
	// SMTP succeeded and the existing notifier saved email_sent before a crash.
	repo.events[0].EmailSent = true
	rule.NotifyEmail = true
	evaluateMonitorTick(t, svc, metrics, rule, base.Add(2*time.Minute))
	require.True(t, readOpsChannelMonitorState(repo.events[0]).AlertNotified)
	require.Empty(t, *sent, "a durable delivery flag repairs the state without SMTP")
}

func TestOpsChannelMonitorRepairsPersistedRecoveryDeliveryAfterRestart(t *testing.T) {
	svc, repo, metrics, rule, sent, base := newMonitorAlertFixture()
	for i := 0; i < 3; i++ {
		evaluateMonitorTick(t, svc, metrics, rule, base.Add(time.Duration(i)*time.Minute))
	}
	metrics.metric.SuccessRate = .99
	svc.sendChannelMonitorEmail = func(context.Context, *OpsAlertRuntimeSettings, *OpsAlertRule, *OpsAlertEvent) bool { return false }
	for i := 3; i < 5; i++ {
		evaluateMonitorTick(t, svc, metrics, rule, base.Add(time.Duration(i)*time.Minute))
	}
	require.Equal(t, OpsAlertStatusResolved, repo.events[0].Status)
	require.False(t, repo.events[0].EmailSent, "each notification phase has its own durable flag")
	repo.events[0].EmailSent = true
	evaluateMonitorTick(t, svc, metrics, rule, base.Add(4*time.Minute))
	require.True(t, readOpsChannelMonitorState(repo.events[0]).RecoveryNotified)
	require.Len(t, *sent, 1)
}
