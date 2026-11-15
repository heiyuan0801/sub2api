package repository

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"golang.org/x/sync/singleflight"
)

func (r *channelMonitorV2Repository) GetModelPerformance(ctx context.Context, filter service.ChannelMonitorV2Filter, cfg service.ChannelMonitorV2Config) (service.PlazaPerformanceFacts, error) {
	var result service.PlazaPerformanceFacts
	groupIDs, empty := channelMonitorV2ScopedGroupIDs(filter, cfg)
	if empty || len(groupIDs) == 0 || len(filter.Platforms) != 1 || filter.ExactModel == "" {
		return result, nil
	}
	platform := filter.Platforms[0]
	if !containsString(channelMonitorV2EnabledPlatforms(cfg), platform) || !channelMonitorV2ModelSelected(filter, cfg, platform, filter.ExactModel) {
		return result, nil
	}
	coverage, err := r.loadCoverage(ctx, filter)
	if err != nil {
		return result, err
	}
	effectiveFilter := channelMonitorV2CommonCoverageFilter(filter, *coverage)
	if !effectiveFilter.Start.Before(effectiveFilter.End) {
		return result, nil
	}
	facts, err := r.loadFacts(ctx, effectiveFilter, cfg, true)
	if err != nil || len(facts) == 0 {
		return result, err
	}
	ignoredByDimension, _, err := r.loadIgnoredErrorCountsByMatrixKey(ctx, effectiveFilter, cfg, service.ChannelMonitorV2GroupByPlatformGroupModel)
	if err != nil {
		return result, err
	}

	// Multiple case variants in V2 can collapse to one model identity. Combine
	// by group and bucket before removing ignored counts so each error is only
	// subtracted once.
	type bucketKey struct {
		group int64
		start string
	}
	samples := make(map[bucketKey]service.PlazaPerformanceSample)
	generationModels := make(map[string]struct{})
	for _, fact := range facts {
		if fact.Platform != platform || channelMonitorV2ModelIdentity(fact.Model) != filter.ExactModel || !slices.Contains(groupIDs, fact.GroupID) {
			continue
		}
		key := bucketKey{group: fact.GroupID, start: fact.BucketStart}
		sample, exists := samples[key]
		if !exists {
			start, err := time.Parse(time.RFC3339Nano, fact.BucketStart)
			if err != nil {
				return result, err
			}
			sample = service.PlazaPerformanceSample{GroupID: fact.GroupID, BucketStart: start}
		}
		sample.Successes += fact.Success
		sample.Failures += fact.Errors
		sample.TTFTSumMs += fact.TTFTSum
		sample.TTFTCount += fact.TTFTCount
		sample.LatencySumMs += fact.DurationSum
		sample.LatencyCount += fact.DurationCount
		samples[key] = sample
		if fact.Output > 0 && fact.TTFTCount > 0 && fact.DurationCount > 0 {
			generationModels[fact.Model] = struct{}{}
		}
	}
	result.Buckets = make([]service.PlazaPerformanceSample, 0, len(samples))
	for key, sample := range samples {
		id := channelMonitorV2MatrixDimensionKey(service.ChannelMonitorV2GroupByPlatformGroupModel, cfg, platform, key.group, filter.ExactModel)
		sample.Failures = max(0, sample.Failures-ignoredByDimension[id][key.start])
		result.Buckets = append(result.Buckets, sample)
	}
	if len(generationModels) > 0 {
		models := make([]string, 0, len(generationModels))
		for model := range generationModels {
			models = append(models, model)
		}
		slices.Sort(models)
		// The fixed rollups omit a leading bucket if common coverage starts
		// partway through it. Use that same lower boundary for the raw sums.
		if start := effectiveFilter.Start.Truncate(filter.Bucket); start.Before(effectiveFilter.Start) {
			effectiveFilter.Start = start.Add(filter.Bucket)
		}
		result.Generation, err = r.cachedModelPlazaGeneration(ctx, effectiveFilter, platform, groupIDs, models)
		if err != nil {
			return result, fmt.Errorf("load model output speed: %w", err)
		}
	}
	return result, nil
}

type plazaTPSCacheEntry struct {
	rates   map[int64]service.PlazaPerformanceGeneration
	expires time.Time
}

type plazaTPSCache struct {
	mu      sync.Mutex
	entries map[string]plazaTPSCacheEntry
	load    singleflight.Group
}

func (c *plazaTPSCache) get(key string) (map[int64]service.PlazaPerformanceGeneration, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	return entry.rates, ok && time.Now().Before(entry.expires)
}

func (c *plazaTPSCache) put(key string, rates map[int64]service.PlazaPerformanceGeneration) {
	const limit = 128
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[string]plazaTPSCacheEntry)
	}
	now := time.Now()
	oldestKey := ""
	oldest := now.Add(time.Minute)
	for currentKey, entry := range c.entries {
		if !now.Before(entry.expires) {
			delete(c.entries, currentKey)
		} else if entry.expires.Before(oldest) {
			oldestKey, oldest = currentKey, entry.expires
		}
	}
	if len(c.entries) >= limit {
		delete(c.entries, oldestKey)
	}
	c.entries[key] = plazaTPSCacheEntry{rates: rates, expires: now.Add(45 * time.Second)}
}

func (r *channelMonitorV2Repository) cachedModelPlazaGeneration(ctx context.Context, filter service.ChannelMonitorV2Filter, platform string, groupIDs []int64, models []string) (map[int64]service.PlazaPerformanceGeneration, error) {
	if !filter.Start.Before(filter.End) {
		return nil, nil
	}
	groupIDs = slices.Clone(groupIDs)
	slices.Sort(groupIDs)
	key := fmt.Sprintf("%s/%d/%d/%v/%q", platform, filter.Start.Unix(), filter.End.Unix(), groupIDs, models)
	if rates, ok := r.plazaTPS.get(key); ok {
		return rates, nil
	}
	pending := r.plazaTPS.load.DoChan(key, func() (any, error) {
		if rates, ok := r.plazaTPS.get(key); ok {
			return rates, nil
		}
		queryCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		rates, err := r.loadModelPlazaGeneration(queryCtx, filter, platform, groupIDs, models)
		if errors.Is(queryCtx.Err(), context.DeadlineExceeded) {
			// Continue serving latency and availability if the optional,
			// indexed throughput query exceeds its budget.
			rates, err = nil, nil
		}
		if err == nil {
			r.plazaTPS.put(key, rates)
		}
		return rates, err
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-pending:
		rates, _ := result.Val.(map[int64]service.PlazaPerformanceGeneration)
		return rates, result.Err
	}
}

func (r *channelMonitorV2Repository) loadModelPlazaGeneration(ctx context.Context, filter service.ChannelMonitorV2Filter, platform string, groupIDs []int64, models []string) (map[int64]service.PlazaPerformanceGeneration, error) {
	// Use the exact model spellings present in the scoped rollups to retain
	// the effective-requested-model index (migration 226). Both the output
	// and elapsed generation time must come from the same measured request.
	rows, err := r.db.QueryContext(ctx, `
		SELECT ul.group_id, SUM(ul.output_tokens), SUM(ul.duration_ms::bigint - ul.first_token_ms)
		FROM usage_logs ul
		LEFT JOIN groups g ON g.id = ul.group_id
		LEFT JOIN accounts a ON a.id = ul.account_id
		WHERE COALESCE(NULLIF(BTRIM(ul.requested_model), ''), ul.model) = ANY($1)
		  AND ul.created_at >= $2 AND ul.created_at < $3
		  AND ul.group_id = ANY($4)
		  AND `+channelMonitorV2PlatformSQL+` = $5
		  AND `+usageLogSuccessFilterUL+`
		  AND COALESCE(ul.request_type, 0) NOT IN (4, 6)
		  AND ul.output_tokens > 0 AND ul.first_token_ms >= 0 AND ul.duration_ms > ul.first_token_ms
		GROUP BY ul.group_id`, pq.Array(models), filter.Start, filter.End, pq.Array(groupIDs), platform)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	result := make(map[int64]service.PlazaPerformanceGeneration)
	for rows.Next() {
		var id int64
		var generation service.PlazaPerformanceGeneration
		if err := rows.Scan(&id, &generation.OutputTokens, &generation.DurationMs); err != nil {
			return nil, err
		}
		result[id] = generation
	}
	return result, rows.Err()
}
