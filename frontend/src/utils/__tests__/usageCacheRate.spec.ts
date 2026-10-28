import { describe, expect, it } from 'vitest'
import { formatUsageCacheRatePercent, usageCacheBreakdown } from '../usageCacheRate'

describe('usage cache rate', () => {
  it('counts disjoint inputs without output or write subtotals', () => {
    const row = { input_tokens: 70, cache_read_tokens: 20, cache_creation_tokens: 10, cache_creation_5m_tokens: 10, output_tokens: 900 }
    expect(usageCacheBreakdown(row)).toEqual({ input: 70, read: 20, write: 10, total: 100, percent: 20 })
    expect(formatUsageCacheRatePercent(row)).toBe('20.0%')
  })
  it('supports synchronous and streaming requests', () => {
    for (const stream of [false, true]) {
      const row = { stream, input_tokens: 1, cache_read_tokens: 2 }
      expect(formatUsageCacheRatePercent(row)).toBe('66.7%')
    }
  })
  it('distinguishes no data from zero hits and full hits', () => {
    expect(usageCacheBreakdown({})).toBeNull()
    expect(formatUsageCacheRatePercent({ input_tokens: 100 })).toBe('0.0%')
    expect(formatUsageCacheRatePercent({ cache_creation_tokens: 100 })).toBe('0.0%')
    expect(formatUsageCacheRatePercent({ cache_read_tokens: 100 })).toBe('100.0%')
  })
  it.each(['image', 'video', 'per_request'])('excludes %s billing', billing_mode => {
    expect(usageCacheBreakdown({ billing_mode, input_tokens: 100, cache_read_tokens: 50 })).toBeNull()
  })
  it('distinguishes legacy image billing from explicit token billing', () => {
    expect(usageCacheBreakdown({ image_count: 1, input_tokens: 100 })).toBeNull()
    expect(formatUsageCacheRatePercent({ image_count: 1, billing_mode: 'token', input_tokens: 100, cache_read_tokens: 100 })).toBe('50.0%')
  })
  it.each([-1, NaN, Infinity])('rejects invalid tokens: %s', value => {
    expect(usageCacheBreakdown({ input_tokens: value, cache_read_tokens: 100 })).toBeNull()
    expect(usageCacheBreakdown({ input_tokens: 100, cache_read_tokens: value })).toBeNull()
  })
})
