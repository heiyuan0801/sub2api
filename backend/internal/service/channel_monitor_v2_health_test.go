package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelMonitorV2HealthSeparatesTrueAvailabilityFromIgnoredErrors(t *testing.T) {
	metric := ChannelMonitorV2Metric{RequestCount: 100, ErrorRequests: 100, SuccessRate: 0, ErrorRate: 0}
	health := ChannelMonitorV2HealthFor(metric)
	require.Equal(t, "critical", health.SuccessRate)
	require.Equal(t, "healthy", health.ErrorRate)
	require.Equal(t, "healthy", health.Overall)
	require.Contains(t, health.Reasons, "success_rate_critical")
	require.Contains(t, health.Reasons, "ignored_errors_excluded")
	require.NotContains(t, health.Reasons, "error_rate_critical")
	require.NotContains(t, health.Reasons, "critical_signal_floor")
}

func TestChannelMonitorV2AvailabilityThresholdBoundariesUseTrueCounters(t *testing.T) {
	for _, tc := range []struct {
		errors int64
		want   string
	}{{4, "healthy"}, {5, "warning"}, {20, "critical"}} {
		metric := ChannelMonitorV2Metric{RequestCount: 100, SuccessRequests: 100 - tc.errors, ErrorRequests: tc.errors, SuccessRate: float64(100-tc.errors) / 100}
		require.Equal(t, tc.want, ChannelMonitorV2HealthFor(metric).SuccessRate)
	}
}

func TestChannelMonitorV2HealthCriticalSignalCannotBeAveragedToHealthy(t *testing.T) {
	p50 := int64(10000)
	metric := ChannelMonitorV2Metric{
		RequestCount: 100, SuccessRequests: 100, SuccessRate: 1,
		TTFT:                 ChannelMonitorV2Latency{SampleCount: 100, P50Ms: &p50},
		CacheRateDenominator: 100,
	}
	health := ChannelMonitorV2HealthFor(metric)
	require.Equal(t, "critical", health.TTFT)
	require.Equal(t, 80.0, *health.Score)
	require.Equal(t, "warning", health.Overall)
	require.Contains(t, health.Reasons, "ttft_critical")
	require.Contains(t, health.Reasons, "critical_signal_floor")

	thresholds := DefaultChannelMonitorV2HealthThresholds()
	thresholds.TTFTWeight = 0
	health = ChannelMonitorV2HealthForWithThresholds(metric, thresholds)
	require.Equal(t, "healthy", health.Overall)
	require.NotContains(t, health.Reasons, "critical_signal_floor")
}

func TestChannelMonitorV2GenerationUsesTailLatencyAndIndependentBudgets(t *testing.T) {
	for _, test := range []struct {
		name       string
		p95        *int64
		p90        *int64
		p50        *int64
		samples    int64
		wantMetric string
		wantState  string
	}{
		{name: "p95 preferred", p95: monitorV2Int64(200000), p90: monitorV2Int64(50000), p50: monitorV2Int64(1000), samples: 100, wantMetric: "p95", wantState: "critical"},
		{name: "fallback p90", p90: monitorV2Int64(120000), p50: monitorV2Int64(1000), samples: 100, wantMetric: "p90", wantState: "warning"},
		{name: "fallback p50", p50: monitorV2Int64(50000), samples: 100, wantMetric: "p50", wantState: "healthy"},
		{name: "negative latency skipped", p95: monitorV2Int64(-1), p90: monitorV2Int64(50000), samples: 100, wantMetric: "p90", wantState: "healthy"},
		{name: "insufficient generation samples", p95: monitorV2Int64(200000), samples: 49, wantState: "unknown"},
		{name: "no generation percentile", samples: 100, wantState: "unknown"},
	} {
		t.Run(test.name, func(t *testing.T) {
			metric := ChannelMonitorV2Metric{
				RequestCount: 100, SuccessRequests: 100, SuccessRate: 1,
				Duration: ChannelMonitorV2Latency{SampleCount: test.samples, P95Ms: test.p95, P90Ms: test.p90, P50Ms: test.p50},
				// A fast first byte is not the completed image generation time.
				TTFT: ChannelMonitorV2Latency{SampleCount: 100, P50Ms: monitorV2Int64(100)},
			}
			health := ChannelMonitorV2HealthForContext(metric, DefaultChannelMonitorV2HealthThresholds(), ChannelMonitorV2HealthContext{Model: "gpt-image-2.5"})
			require.Equal(t, test.wantState, health.Generation)
			require.Equal(t, test.wantMetric, health.GenerationPercentile)
			require.Equal(t, "unknown", health.TTFT)
			require.Equal(t, "unknown", health.Cache)
			if test.wantState == "critical" {
				require.NotEqual(t, "healthy", health.Overall)
				require.Contains(t, health.Reasons, "generation_critical")
			}
			if test.wantState == "unknown" {
				require.Nil(t, health.GenerationLatencyMs)
				require.Nil(t, health.GenerationScore)
			}
			mixed := ChannelMonitorV2HealthFor(metric)
			require.Equal(t, "unknown", mixed.Generation)
			require.Equal(t, "healthy", mixed.TTFT)
		})
	}
}

func TestChannelMonitorV2GenerationOverridesUseScopedSpecificity(t *testing.T) {
	thresholds := DefaultChannelMonitorV2HealthThresholds()
	thresholds.GenerationOverrides = []ChannelMonitorV2GenerationOverride{
		{Platform: "openai", WarningGenerationMs: 10000, CriticalGenerationMs: 20000},
		{Model: "gpt-image-2.5", WarningGenerationMs: 30000, CriticalGenerationMs: 40000},
		{GroupID: 7, WarningGenerationMs: 50000, CriticalGenerationMs: 60000},
		{Platform: "openai", Model: "gpt-image-2.5", WarningGenerationMs: 70000, CriticalGenerationMs: 80000},
		{GroupID: 7, Model: "gpt-image-2.5", WarningGenerationMs: 90000, CriticalGenerationMs: 100000},
	}
	for _, test := range []struct {
		name    string
		row     ChannelMonitorV2HealthContext
		warning int64
	}{
		{name: "default", row: ChannelMonitorV2HealthContext{Model: "gpt-image-2"}, warning: 60000},
		{name: "platform", row: ChannelMonitorV2HealthContext{Platform: "openai", Model: "gpt-image-2"}, warning: 10000},
		{name: "model", row: ChannelMonitorV2HealthContext{Model: "gpt-image-2.5"}, warning: 30000},
		{name: "group", row: ChannelMonitorV2HealthContext{GroupID: 7, Model: "gpt-image-2"}, warning: 50000},
		{name: "two conditions beat one", row: ChannelMonitorV2HealthContext{Platform: "openai", GroupID: 8, Model: "gpt-image-2.5"}, warning: 70000},
		{name: "group breaks equal specificity tie", row: ChannelMonitorV2HealthContext{Platform: "openai", GroupID: 7, Model: "gpt-image-2.5"}, warning: 90000},
		{name: "mixed context does not apply generation", row: ChannelMonitorV2HealthContext{GroupID: 7, Platform: "openai"}, warning: 60000},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := channelMonitorV2GenerationThresholdsForContext(thresholds, test.row)
			require.Equal(t, test.warning, got.WarningGenerationMs)
			require.Nil(t, got.GenerationOverrides, "health must not expose other groups' policy selectors")
		})
	}
	require.Len(t, thresholds.GenerationOverrides, 5, "row selection must not mutate saved config")
	cfg := ChannelMonitorV2Config{HealthThresholds: thresholds}
	redactChannelMonitorV2PublicConfig(&cfg)
	require.Nil(t, cfg.HealthThresholds.GenerationOverrides)
	require.Len(t, thresholds.GenerationOverrides, 5)
}

func TestChannelMonitorV2GenerationOverrideValidation(t *testing.T) {
	valid := ChannelMonitorV2GenerationOverride{Platform: "openai", WarningGenerationMs: 60000, CriticalGenerationMs: 180000}
	require.NoError(t, validateChannelMonitorV2GenerationOverrides([]ChannelMonitorV2GenerationOverride{valid}))
	for _, rule := range []ChannelMonitorV2GenerationOverride{
		{WarningGenerationMs: 1, CriticalGenerationMs: 2},
		{GroupID: -1, WarningGenerationMs: 1, CriticalGenerationMs: 2},
		{Model: "gpt-image-2", WarningGenerationMs: 2, CriticalGenerationMs: 1},
		{Model: "gpt-image-2", WarningGenerationMs: 0, CriticalGenerationMs: 1},
	} {
		require.ErrorIs(t, validateChannelMonitorV2GenerationOverrides([]ChannelMonitorV2GenerationOverride{rule}), ErrChannelMonitorV2InvalidConfig)
	}
	duplicate := valid
	duplicate.Platform = " OpenAI "
	require.ErrorIs(t, validateChannelMonitorV2GenerationOverrides([]ChannelMonitorV2GenerationOverride{valid, duplicate}), ErrChannelMonitorV2InvalidConfig)
}

func monitorV2Int64(value int64) *int64 { return &value }
