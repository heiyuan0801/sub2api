import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { monitorFreshness, monitorRowKey, preferenceKey, readPreferences, savePreferences, sortMonitorRows } from '../monitorPreferences'
import type { MonitorCoverage, MonitorMatrixRow, MonitorMetric } from '@/api/channelMonitorV2'

function metric(patch: Partial<MonitorMetric> = {}): MonitorMetric {
  return { has_data: true, success_rate: 0.8, request_count: 0, success_requests: 0, error_requests: 0, token_count: 0, rpm: 0, tpm: 0, error_rate: 0.2, cache_rate: 0, cache_rate_numerator: 0, cache_rate_denominator: 0,
    ttft: { sample_count: 0, p50_ms: 1000, p95_ms: 1500, avg_ms: 1100 }, duration: { sample_count: 0, p50_ms: 2000, p95_ms: 4000, avg_ms: 3000 }, ...patch }
}
function row(model: string, patch: Partial<MonitorMatrixRow> = {}): MonitorMatrixRow { return { platform: 'openai', model, group_id: 1, group_name: 'default', metrics: metric(), health: { overall: 'healthy', ttft: 'healthy', error_rate: 'healthy', minimum_sample: 20 }, buckets: [], ...patch } }
const now = Date.parse('2026-10-17T15:00:00Z')
function coverage(patch: Partial<MonitorCoverage> = {}): MonitorCoverage { return { requested_start: '2026-10-16T15:00:00Z', requested_end: '2026-10-17T15:00:00Z', coverage_start: '2026-10-16T15:00:00Z', data_through: '2026-10-17T15:00:00Z', computed_at: '2026-10-17T15:00:00Z', aggregation_lag_seconds: 0, coverage_complete: true, bucket_seconds: 60, ...patch } }

beforeEach(() => { localStorage.clear() })
afterEach(() => { vi.restoreAllMocks() })

describe('monitor preference storage', () => {
  it('isolates saved settings by authenticated user and never creates an anonymous key', () => {
    expect(preferenceKey(undefined)).toBeNull()
    for (const invalid of [0, -1, 1.2, NaN, Infinity, Number.MAX_SAFE_INTEGER + 1]) expect(preferenceKey(invalid)).toBeNull()
    const first = preferenceKey(10)
    const second = preferenceKey(11)
    expect(first).not.toBe(second)
    savePreferences(first, { q: 'private model', sort: 'success' })
    expect(readPreferences(first)).toEqual({ q: 'private model', sort: 'success' })
    expect(readPreferences(second)).toEqual({})
    const set = vi.spyOn(Storage.prototype, 'setItem')
    savePreferences(null, { q: 'anonymous' })
    expect(set).not.toHaveBeenCalled()
  })

  it('returns an empty preference set for malformed JSON, arrays, primitives or storage access failures', () => {
    for (const raw of ['{bad', '[]', 'null', '42', '"text"']) { localStorage.setItem('test', raw); expect(readPreferences('test')).toEqual({}) }
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new DOMException('Disabled', 'SecurityError') })
    expect(readPreferences('test')).toEqual({})
    expect(readPreferences(null)).toEqual({})
  })

  it('bounds data and rejects malformed groups, favorites and unsupported sort values', () => {
    const validKey = monitorRowKey(row('safe'))
    localStorage.setItem('test', JSON.stringify({ q: 'x'.repeat(257), range: '24h', platform: ['openai', false, 'x'.repeat(257)], model: ['safe'], group: [1, 0, -1, '2', 2.2, 2], sort: 'invalid', favorites_only: 'true', favorites: [validKey, validKey, 'broken', '{}', '["x","y",-1]', '["x","y","1"]', '["x","y",null]', 12] }))
    expect(readPreferences('test')).toEqual({ range: '24h', platform: ['openai'], model: ['safe'], group: [1, 2], favorites: [validKey, '["x","y",null]'] })
    localStorage.setItem('limits', JSON.stringify({ platform: Array.from({ length: 150 }, (_, i) => `p${i}`), model: Array.from({ length: 150 }, (_, i) => `m${i}`), group: Array.from({ length: 150 }, (_, i) => i + 1), favorites: Array.from({ length: 600 }, (_, i) => monitorRowKey(row(`m${i}`))) }))
    const preferences = readPreferences('limits')
    expect(preferences.platform).toHaveLength(100)
    expect(preferences.model).toHaveLength(100)
    expect(preferences.group).toHaveLength(100)
    expect(preferences.favorites).toHaveLength(500)
  })

  it('does not break the page when saving is blocked or storage is full', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new DOMException('Full', 'QuotaExceededError') })
    expect(() => savePreferences('test', { sort: 'health' })).not.toThrow()
  })

  it('keeps platform, model and group in the row key without delimiter collisions', () => {
    expect(monitorRowKey({ platform: 'openai' })).toBe('["openai","",null]')
    expect(monitorRowKey(row('model', { group_id: 2 }))).not.toBe(monitorRowKey(row('model', { group_id: 3 })))
    expect(monitorRowKey(row('a:b', { platform: 'c' }))).not.toBe(monitorRowKey(row('b', { platform: 'c:a' })))
  })
})

describe('sortMonitorRows', () => {
  it('places favorites first and does not mutate the original rows', () => {
    const healthy = row('healthy')
    const critical = row('critical', { health: { ...healthy.health, overall: 'critical' } })
    const rows = [critical, healthy]
    expect(sortMonitorRows(rows, 'health', [monitorRowKey(healthy)])).toEqual([healthy, critical])
    expect(rows).toEqual([critical, healthy])
  })

  it('sorts critical, warning, unknown and healthy states in severity order', () => {
    const base = row('model')
    const records = (['healthy', 'unknown', 'critical', 'warning'] as const).map((state, i) => row(`m${i}`, { health: { ...base.health, overall: state } }))
    expect(sortMonitorRows(records, 'health', []).map(item => item.health.overall)).toEqual(['critical', 'warning', 'unknown', 'healthy'])
  })

  it('shows lower success rates first, supports legacy rates and places no-data rows last', () => {
    const good = row('good', { metrics: metric({ success_rate: 0.95 }) })
    const bad = row('bad', { metrics: metric({ success_rate: 0.1 }) })
    const legacy = row('legacy', { metrics: metric({ success_rate: undefined, error_rate: 0.3 }) })
    const empty = row('empty', { metrics: metric({ has_data: false, success_rate: 0 }) })
    expect(sortMonitorRows([good, empty, legacy, bad], 'success', []).map(item => item.model)).toEqual(['bad', 'legacy', 'good', 'empty'])
  })

  it('uses backend generation latency for images and keeps missing latencies last', () => {
    const missing = { sample_count: 0, p50_ms: null, p95_ms: null, avg_ms: null }
    const noLatency = row('empty', { metrics: metric({ ttft: missing, duration: missing }) })
    const dialog = row('chat', { metrics: metric({ ttft: { ...missing, p50_ms: 20000 } }) })
    const image = row('image', { is_image_model: true, health: { ...row('x').health, generation_latency_ms: 30000 }, metrics: metric({ duration: { ...missing, avg_ms: 1000, p95_ms: 50000 } }) })
    const legacyImage = row('legacy-image', { is_image_model: true, metrics: metric({ duration: { ...missing, p90_ms: 25000 } }) })
    expect(sortMonitorRows([noLatency, dialog, legacyImage, image], 'latency', []).map(item => item.model)).toEqual(['image', 'legacy-image', 'chat', 'empty'])
  })

  it('sorts names deterministically, then platform and group labels', () => {
    const first = row('a', { platform: 'anthropic', group_name: 'a' })
    const second = row('a', { platform: 'anthropic', group_name: 'b' })
    const third = row('a', { platform: 'openai' })
    const fourth = row('b')
    expect(sortMonitorRows([fourth, third, second, first], 'name', [])).toEqual([first, second, third, fourth])
  })
})

describe('monitorFreshness', () => {
  it('treats a missing snapshot as not yet loaded and uses a refresh-aware grace period', () => {
    expect(monitorFreshness(null, 60, now)).toEqual({ stale: false, ageSeconds: 0, threshold: 180 })
    expect(monitorFreshness(undefined, 300, now).threshold).toBe(900)
    expect(monitorFreshness(coverage(), 60, now).stale).toBe(false)
  })

  it('uses the oldest computation/watermark and aggregation lag rather than a successful response time', () => {
    expect(monitorFreshness(coverage({ data_through: '2026-10-17T14:55:00Z' }), 60, now)).toMatchObject({ stale: true, ageSeconds: 300 })
    expect(monitorFreshness(coverage({ computed_at: '2026-10-17T14:55:00Z' }), 60, now)).toMatchObject({ stale: true, ageSeconds: 300 })
    expect(monitorFreshness(coverage({ aggregation_lag_seconds: 400 }), 60, now)).toMatchObject({ stale: true, ageSeconds: 400 })
    expect(monitorFreshness(coverage({ aggregation_lag_seconds: 400 }), 300, now).stale).toBe(false)
  })

  it('uses a strict age boundary and never returns negative age for allowed clock skew', () => {
    expect(monitorFreshness(coverage({ data_through: '2026-10-17T14:57:00Z' }), 60, now).stale).toBe(false)
    expect(monitorFreshness(coverage({ data_through: '2026-10-17T14:56:59Z' }), 60, now).stale).toBe(true)
    expect(monitorFreshness(coverage({ computed_at: '2026-10-17T15:02:00Z', data_through: '2026-10-17T15:02:00Z' }), 60, now)).toMatchObject({ stale: false, ageSeconds: 0 })
  })

  it('marks invalid timestamps and excessive future computation times stale', () => {
    expect(monitorFreshness(coverage({ computed_at: 'bad' }), 60, now).stale).toBe(true)
    expect(monitorFreshness(coverage({ data_through: '' }), 60, now).stale).toBe(true)
    expect(monitorFreshness(coverage({ computed_at: '2026-10-17T15:06:00Z' }), 60, now).stale).toBe(true)
  })

  it('rejects a watermark implausibly far in the future even when computed_at is current', () => {
    expect(monitorFreshness(coverage({ data_through: '2026-10-17T15:06:00Z' }), 60, now).stale).toBe(true)
  })
})
