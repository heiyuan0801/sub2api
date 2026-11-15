package service

import (
	"context"
	"errors"
	"strings"
	"time"
)

type PlazaPerformanceSample struct {
	GroupID      int64
	BucketStart  time.Time
	Successes    int64
	Failures     int64
	TTFTSumMs    int64
	TTFTCount    int64
	LatencySumMs int64
	LatencyCount int64
}

type PlazaPerformanceGeneration struct {
	OutputTokens int64
	DurationMs   int64
}

type PlazaPerformanceFacts struct {
	Buckets    []PlazaPerformanceSample
	Generation map[int64]PlazaPerformanceGeneration
}

type PlazaPerformanceMetrics struct {
	TPS           *float64 `json:"tps"`
	AvgTTFTMs     *float64 `json:"avg_ttft_ms"`
	AvgLatencyMs  *float64 `json:"avg_latency_ms"`
	SuccessRate   *float64 `json:"success_rate"`
	Failures      *int64   `json:"failures"`
	IncidentCount int      `json:"incident_count"`
}

type PlazaPerformanceBucket struct {
	BucketStart time.Time `json:"bucket_start"`
	SuccessRate *float64  `json:"success_rate"`
	Failures    *int64    `json:"failures"`
	Abnormal    bool      `json:"abnormal"`
}

type PlazaPerformanceTrendPoint struct {
	PlazaPerformanceBucket
	AvgTTFTMs  *float64 `json:"avg_ttft_ms"`
	TTFTStatus string   `json:"ttft_status"`
}

type PlazaPerformanceGroup struct {
	ID      int64                    `json:"id"`
	Name    string                   `json:"name"`
	Metrics PlazaPerformanceMetrics  `json:"metrics"`
	Buckets []PlazaPerformanceBucket `json:"buckets"`
}

type PlazaPerformance struct {
	Range         string                       `json:"range"`
	BucketSeconds int                          `json:"bucket_seconds"`
	Summary       PlazaPerformanceMetrics      `json:"summary"`
	Groups        []PlazaPerformanceGroup      `json:"groups"`
	Trend         []PlazaPerformanceTrendPoint `json:"trend"`
}

// ModelPerformance reads the passive monitor's measured values for a single
// catalog model. The caller must provide the visible catalog groups and a
// separate allowlist in filter for viewing their monitoring data.
func (s *ChannelMonitorV2Service) ModelPerformance(ctx context.Context, filter ChannelMonitorV2Filter, groups []PlazaGroup, groupID int64, admin bool) (*PlazaPerformance, error) {
	if len(filter.Models) != 1 || filter.Bucket <= 0 || !filter.Start.Before(filter.End) {
		return nil, ErrChannelMonitorV2InvalidRange
	}
	var facts PlazaPerformanceFacts
	thresholds := DefaultChannelMonitorV2HealthThresholds()
	model := strings.ToLower(filter.Models[0])
	// Every query must remain bound to the catalog allowlist. Monitoring may
	// be off even while the pricing part of the model plaza remains available.
	if filter.RestrictGroups && len(filter.AllowedGroupIDs) > 0 && s.settings != nil &&
		s.settings.GetChannelMonitorRuntime(ctx).PassiveAggregationAllowed() {
		filter.ExactModel = model
		scoped, cfg, err := s.prepareQuery(ctx, filter)
		if err != nil && !errors.Is(err, ErrChannelMonitorDisabled) {
			return nil, err
		}
		if cfg != nil {
			thresholds = NormalizeChannelMonitorV2HealthThresholds(cfg.HealthThresholds)
			// Generation TPS measures decoding speed. Only the V2 RPM/TPM
			// traffic-volume counters honor the HideThroughput setting.
			facts, err = s.repo.GetModelPerformance(ctx, scoped, *cfg)
			if err != nil {
				return nil, err
			}
		}
	}
	return buildPlazaPerformance(filter, groups, groupID, admin, thresholds, facts), nil
}

func buildPlazaPerformance(filter ChannelMonitorV2Filter, groups []PlazaGroup, groupID int64, admin bool, thresholds ChannelMonitorV2HealthThresholds, facts PlazaPerformanceFacts) *PlazaPerformance {
	buckets := int(filter.End.Sub(filter.Start) / filter.Bucket)
	result := &PlazaPerformance{
		Range: filter.Range, BucketSeconds: int(filter.Bucket.Seconds()),
		Groups: make([]PlazaPerformanceGroup, 0, len(groups)),
		Trend:  make([]PlazaPerformanceTrendPoint, 0, buckets),
	}
	series := make(map[int64][]PlazaPerformanceSample, len(groups))
	for _, group := range groups {
		series[group.ID] = make([]PlazaPerformanceSample, buckets)
	}
	for _, fact := range facts.Buckets {
		if fact.BucketStart.Before(filter.Start) || !fact.BucketStart.Before(filter.End) {
			continue
		}
		if samples := series[fact.GroupID]; samples != nil {
			index := int(fact.BucketStart.Sub(filter.Start) / filter.Bucket)
			samples[index].add(fact)
		}
	}
	combined := make([]PlazaPerformanceSample, buckets)
	var combinedGeneration PlazaPerformanceGeneration
	for _, group := range groups {
		samples, generation := series[group.ID], facts.Generation[group.ID]
		include := groupID == 0 || groupID == group.ID
		row := PlazaPerformanceGroup{ID: group.ID, Name: group.Name, Buckets: make([]PlazaPerformanceBucket, 0, buckets)}
		var total PlazaPerformanceSample
		for index, sample := range samples {
			total.add(sample)
			if include {
				combined[index].add(sample)
			}
			bucket := sample.bucket(filter.Start.Add(time.Duration(index)*filter.Bucket), admin)
			row.Buckets = append(row.Buckets, bucket)
			if bucket.Abnormal {
				row.Metrics.IncidentCount++
			}
		}
		row.Metrics = total.metrics(admin, generation, row.Metrics.IncidentCount)
		result.Groups = append(result.Groups, row)
		if include {
			combinedGeneration.OutputTokens += generation.OutputTokens
			combinedGeneration.DurationMs += generation.DurationMs
		}
	}
	var total PlazaPerformanceSample
	incidents := 0
	for index, sample := range combined {
		total.add(sample)
		bucket := sample.bucket(filter.Start.Add(time.Duration(index)*filter.Bucket), admin)
		if bucket.Abnormal {
			incidents++
		}
		point := PlazaPerformanceTrendPoint{PlazaPerformanceBucket: bucket, TTFTStatus: "unknown"}
		point.AvgTTFTMs = performanceRatio(sample.TTFTSumMs, sample.TTFTCount)
		if point.AvgTTFTMs != nil {
			point.TTFTStatus = healthBand(*point.AvgTTFTMs, float64(thresholds.WarningTTFTMs), float64(thresholds.CriticalTTFTMs))
		}
		result.Trend = append(result.Trend, point)
	}
	result.Summary = total.metrics(admin, combinedGeneration, incidents)
	return result
}

func (s *PlazaPerformanceSample) add(other PlazaPerformanceSample) {
	s.Successes += other.Successes
	s.Failures += other.Failures
	s.TTFTSumMs += other.TTFTSumMs
	s.TTFTCount += other.TTFTCount
	s.LatencySumMs += other.LatencySumMs
	s.LatencyCount += other.LatencyCount
}

func (s PlazaPerformanceSample) bucket(start time.Time, admin bool) PlazaPerformanceBucket {
	bucket := PlazaPerformanceBucket{BucketStart: start, SuccessRate: performanceRatio(s.Successes, s.Successes+s.Failures), Abnormal: s.Failures > 0}
	if admin {
		bucket.Failures = &s.Failures
	}
	return bucket
}

func (s PlazaPerformanceSample) metrics(admin bool, generation PlazaPerformanceGeneration, incidents int) PlazaPerformanceMetrics {
	metrics := PlazaPerformanceMetrics{
		AvgTTFTMs: performanceRatio(s.TTFTSumMs, s.TTFTCount), AvgLatencyMs: performanceRatio(s.LatencySumMs, s.LatencyCount),
		SuccessRate: performanceRatio(s.Successes, s.Successes+s.Failures), IncidentCount: incidents,
	}
	if generation.DurationMs > 0 {
		tps := 1000 * float64(generation.OutputTokens) / float64(generation.DurationMs)
		metrics.TPS = &tps
	}
	if admin {
		metrics.Failures = &s.Failures
	}
	return metrics
}

func performanceRatio(numerator, denominator int64) *float64 {
	if denominator <= 0 {
		return nil
	}
	value := float64(numerator) / float64(denominator)
	return &value
}
