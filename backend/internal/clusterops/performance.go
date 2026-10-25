package clusterops

import (
	"context"
	"database/sql"
	"math"
	"sync"
	"time"
)

// Fixed-size rolling histogram: no request IDs, prompts or unbounded labels.
// Percentiles are upper bucket bounds, not falsely precise individual samples.
var latencyBounds = [...]float64{25, 50, 100, 250, 500, 1000, 2000, 4000, 8000, 16000, 30000, 60000, 120000, 300000, 600000}

type performanceBucket struct {
	second                  int64
	count, cacheRead, input uint64
	latencies               [16]uint64
	durationCount           uint64
	durationMS              float64
	phaseCount              [3]uint64
	phaseMS                 [3]float64
}
type performanceMeter struct {
	mu         sync.Mutex
	buckets    [60]performanceBucket
	previousAt time.Time
	requests   uint64
	db         sql.DBStats
}
type Performance struct {
	CompletedSamples   uint64    `json:"completed_samples"`
	DurationMeanMS     *float64  `json:"duration_mean_ms"`
	AuthMeanMS         *float64  `json:"auth_mean_ms"`
	RoutingMeanMS      *float64  `json:"routing_mean_ms"`
	UpstreamMeanMS     *float64  `json:"upstream_mean_ms"`
	PhaseSamples       [3]uint64 `json:"phase_samples"`
	SampledAt          time.Time `json:"sampled_at"`
	WindowSeconds      float64   `json:"window_seconds"`
	RequestsPerSecond  *float64  `json:"requests_per_second"`
	FirstTokenP95MS    *float64  `json:"first_token_p95_ms"`
	FirstTokenSamples  uint64    `json:"first_token_samples"`
	FirstTokenOverflow bool      `json:"first_token_overflow"`
	CacheReadPercent   *float64  `json:"cache_read_percent"`
	UsageSamples       uint64    `json:"usage_samples"`
	DBInUse            int       `json:"db_in_use"`
	DBOpen             int       `json:"db_open"`
	DBLimit            int       `json:"db_limit"`
	DBWaits            int64     `json:"db_waits"`
	DBWaitMeanMS       *float64  `json:"db_wait_mean_ms"`
	RedisConnections   uint32    `json:"redis_connections"`
	RedisIdle          uint32    `json:"redis_idle"`
	RedisPoolTimeouts  uint32    `json:"redis_pool_timeouts"`
}
type meterKey struct{}

func (m *Manager) WithMetrics(ctx context.Context) context.Context {
	if !m.Enabled() {
		return ctx
	}
	return context.WithValue(ctx, meterKey{}, &m.performance)
}

// CopyUsageContext preserves only cluster bookkeeping across the bounded usage
// worker context. It does not copy request cancellation or arbitrary values.
func CopyUsageContext(parent, base context.Context) context.Context {
	if parent == nil {
		return base
	}
	if id := AttemptID(parent); id != "" {
		base = WithAttempt(base, id)
	}
	if meter, ok := parent.Value(meterKey{}).(*performanceMeter); ok {
		base = context.WithValue(base, meterKey{}, meter)
	}
	return base
}

// ObserveUsage only receives normalized usage after a successful unique ledger
// application. Duplicated billing does not inflate token or latency samples.
func ObserveUsage(ctx context.Context, firstToken *int, input, creation, read int) {
	m, _ := ctx.Value(meterKey{}).(*performanceMeter)
	if m != nil {
		m.observe(time.Now(), firstToken, input, creation, read)
	}
}
func (m *performanceMeter) observe(now time.Time, firstToken *int, input, creation, read int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sec := now.Unix()
	b := &m.buckets[sec%60]
	if b.second != sec {
		*b = performanceBucket{second: sec}
	}
	b.count++
	if input >= 0 && creation >= 0 && read >= 0 {
		b.input += uint64(input) + uint64(creation) + uint64(read)
		b.cacheRead += uint64(read)
	}
	if firstToken != nil && *firstToken >= 0 {
		i := 0
		for i < len(latencyBounds) && float64(*firstToken) > latencyBounds[i] {
			i++
		}
		b.latencies[i]++
	}
}
func (m *performanceMeter) sample(now time.Time, requests uint64, db sql.DBStats) *Performance {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := &Performance{SampledAt: now, DBInUse: db.InUse, DBOpen: db.OpenConnections, DBLimit: db.MaxOpenConnections}
	if !m.previousAt.IsZero() && now.After(m.previousAt) && requests >= m.requests {
		p.WindowSeconds = now.Sub(m.previousAt).Seconds()
		p.RequestsPerSecond = finiteMetric(float64(requests-m.requests) / p.WindowSeconds)
		p.DBWaits = max(0, db.WaitCount-m.db.WaitCount)
		if p.DBWaits > 0 && db.WaitDuration >= m.db.WaitDuration {
			p.DBWaitMeanMS = finiteMetric(float64(db.WaitDuration-m.db.WaitDuration) / float64(time.Millisecond) / float64(p.DBWaits))
		}
	}
	m.previousAt, m.requests, m.db = now, requests, db
	var histogram [16]uint64
	var total, read uint64
	var duration float64
	var phases [3]float64
	for _, b := range m.buckets {
		if b.second <= now.Unix()-60 || b.second > now.Unix() {
			continue
		}
		p.UsageSamples += b.count
		p.CompletedSamples += b.durationCount
		duration += b.durationMS
		for i := range phases {
			phases[i] += b.phaseMS[i]
			p.PhaseSamples[i] += b.phaseCount[i]
		}
		total += b.input
		read += b.cacheRead
		for i, n := range b.latencies {
			histogram[i] += n
			p.FirstTokenSamples += n
		}
	}
	if p.CompletedSamples > 0 {
		p.DurationMeanMS = finiteMetric(duration / float64(p.CompletedSamples))
	}
	phaseTargets := []**float64{&p.AuthMeanMS, &p.RoutingMeanMS, &p.UpstreamMeanMS}
	for i := range phases {
		if p.PhaseSamples[i] > 0 {
			*phaseTargets[i] = finiteMetric(phases[i] / float64(p.PhaseSamples[i]))
		}
	}
	if total > 0 {
		p.CacheReadPercent = finiteMetric(100 * float64(read) / float64(total))
	}
	if p.FirstTokenSamples > 0 {
		target := uint64(math.Ceil(float64(p.FirstTokenSamples) * 0.95))
		var n uint64
		for i, count := range histogram {
			n += count
			if n >= target {
				if i < len(latencyBounds) {
					value := latencyBounds[i]
					p.FirstTokenP95MS = &value
				} else {
					p.FirstTokenOverflow = true
				}
				break
			}
		}
	}
	return p
}

// Completion is observed once even on an upstream error. Optional handler
// timings have separate denominators and may overlap; they are not a waterfall.
func (m *Manager) ObserveCompletion(elapsed time.Duration, phases [3]*float64) {
	if !m.Enabled() {
		return
	}
	meter := &m.performance
	meter.mu.Lock()
	defer meter.mu.Unlock()
	sec := time.Now().Unix()
	b := &meter.buckets[sec%60]
	if b.second != sec {
		*b = performanceBucket{second: sec}
	}
	b.durationCount++
	b.durationMS += float64(elapsed) / float64(time.Millisecond)
	for i, v := range phases {
		if v != nil && finiteMetric(*v) != nil {
			b.phaseCount[i]++
			b.phaseMS[i] += *v
		}
	}
}
