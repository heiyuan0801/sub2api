package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestModelPlazaPerformanceProjectionKeepsMissingTTFTAndPrivateCounts(t *testing.T) {
	start := time.Date(2026, 11, 15, 1, 0, 0, 0, time.UTC)
	filter := ChannelMonitorV2Filter{Range: "24h", Start: start, End: start.Add(24 * time.Hour), Bucket: time.Hour}
	facts := PlazaPerformanceFacts{
		Buckets: []PlazaPerformanceSample{
			{GroupID: 1, BucketStart: start, Successes: 3, Failures: 1, LatencySumMs: 9000, LatencyCount: 3},
			{GroupID: 2, BucketStart: start, Successes: 6, TTFTSumMs: 10000, TTFTCount: 2, LatencySumMs: 30000, LatencyCount: 6},
			{GroupID: 2, BucketStart: start.Add(2 * time.Hour), Successes: 3, TTFTSumMs: 2000, TTFTCount: 1},
		},
		Generation: map[int64]PlazaPerformanceGeneration{
			2: {OutputTokens: 840, DurationMs: 4200},
		},
	}
	groups := []PlazaGroup{{ID: 1, Name: "First"}, {ID: 2, Name: "Second"}}
	public := buildPlazaPerformance(filter, groups, 0, false, DefaultChannelMonitorV2HealthThresholds(), facts)
	require.Len(t, public.Trend, 24)
	require.InDelta(t, 12.0/13.0, *public.Summary.SuccessRate, .0001)
	require.Equal(t, 4000.0, *public.Summary.AvgTTFTMs)
	require.Equal(t, 200.0, *public.Summary.TPS)
	require.Equal(t, 1, public.Summary.IncidentCount)
	require.Nil(t, public.Summary.Failures)
	require.True(t, public.Trend[0].Abnormal)
	require.Nil(t, public.Trend[0].Failures)
	require.Equal(t, "warning", public.Trend[0].TTFTStatus)
	require.Nil(t, public.Trend[1].AvgTTFTMs)
	require.Nil(t, public.Trend[1].SuccessRate)
	require.Nil(t, public.Groups[0].Metrics.AvgTTFTMs)
	require.Equal(t, 3000.0, *public.Groups[0].Metrics.AvgLatencyMs)

	selected := buildPlazaPerformance(filter, groups, 1, true, DefaultChannelMonitorV2HealthThresholds(), facts)
	require.Len(t, selected.Groups, 2)
	require.Nil(t, selected.Summary.AvgTTFTMs)
	require.Nil(t, selected.Summary.TPS)
	require.Equal(t, 0.75, *selected.Summary.SuccessRate)
	require.Equal(t, int64(1), *selected.Trend[0].Failures)
	require.Nil(t, selected.Trend[2].SuccessRate)
}
