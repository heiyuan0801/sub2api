import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import MonitorChannelCards from '../MonitorChannelCards.vue'
import type { MonitorCoverage, MonitorHealth, MonitorMatrixRow, MonitorMetric } from '@/api/channelMonitorV2'

vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({
    locale: { value: 'en-US' },
    t: (key: string, params?: Record<string, unknown>) => `${key}${params ? ` ${Object.entries(params).map(([name, value]) => `${name}=${value}`).join(' ')}` : ''}`,
  }),
}))
vi.mock('@/composables/useChannelMonitorFormat', () => ({ useChannelMonitorFormat: () => ({ providerLabel: (platform: string) => platform }) }))

const coverage: MonitorCoverage = {
  requested_start: '2026-10-17T00:00:00Z', requested_end: '2026-10-17T00:05:00Z',
  coverage_start: '2026-10-17T00:02:00Z', data_through: '2026-10-17T00:03:00Z',
  computed_at: '2026-10-17T00:05:00Z', aggregation_lag_seconds: 120, coverage_complete: false, bucket_seconds: 60,
}
const health: MonitorHealth = { overall: 'warning', error_rate: 'healthy', ttft: 'healthy', cache: 'healthy', minimum_sample: 20 }
function metric(patch: Partial<MonitorMetric> = {}): MonitorMetric {
  return {
    has_data: true, cache_rate_available: true, success_rate: 0.8,
    request_count: 0, success_requests: 0, error_requests: 0, token_count: 0,
    rpm: 1, tpm: 120, error_rate: 0.05, cache_rate: 0.6, cache_rate_numerator: 0, cache_rate_denominator: 0,
    ttft: { sample_count: 0, p50_ms: 1200, avg_ms: 1300, p95_ms: 2000 },
    duration: { sample_count: 0, p50_ms: 2100, avg_ms: 2500, p95_ms: 3100 },
    ...patch,
  }
}
function row(patch: Partial<MonitorMatrixRow> = {}): MonitorMatrixRow {
  return { platform: 'openai', model: 'gpt-test', group_id: 1, group_name: 'default', metrics: metric(), health,
    buckets: [{ bucket_start: '2026-10-17T00:02:00Z', metrics: metric(), health }], ...patch }
}
function render(rows = [row()], extra: Record<string, unknown> = {}) {
  return mount(MonitorChannelCards, { props: { rows, coverage, healthMode: 'overall', ...extra }, global: { stubs: { ProviderIcon: true, EmptyState: { props: ['title', 'description'], template: '<div class="empty-state">{{ title }} {{ description }}</div>' } } } })
}

describe('MonitorChannelCards', () => {
  it('groups model rows by platform + model and emits selection for the actual group', async () => {
    const second = row({ group_id: 2, group_name: 'vip' })
    const wrapper = render([row(), second, row({ platform: 'anthropic', group_id: 3 }), row({ model: 'gpt-another', group_id: 4 })])
    expect(wrapper.findAll('.model-card')).toHaveLength(3)
    expect(wrapper.findAll('.model-card')[0].findAll('.group-row')).toHaveLength(2)
    await wrapper.findAll('.group-link')[1].trigger('click')
    expect(wrapper.emitted('inspect')).toEqual([[{ row: second }]])
    await wrapper.findAll('.filter-action')[1].trigger('click')
    expect(wrapper.emitted('select-group')).toEqual([[second]])
  })

  it('uses explicit success rate, preserves real metrics despite redacted counts, and uses backend overall status', () => {
    const wrapper = render()
    expect(wrapper.find('.success-cell .metric-value').text()).toBe('80.0%')
    expect(wrapper.find('.success-cell .health-badge').classes()).toContain('status-warning')
    expect(wrapper.find('.cache-cell .metric-value').text()).toBe('60.0%')
    expect(wrapper.find('.latency-cell .metric-value').text()).toBe('1.2s')
    expect(wrapper.find('.latency-secondary').text()).toContain('2.5s')
    const buckets = wrapper.findAll('.timeline-bucket')
    expect(buckets).toHaveLength(5)
    expect(buckets.map(bucket => bucket.classes().find(c => c.startsWith('health-')))).toEqual(['health-unknown', 'health-unknown', 'health-warning', 'health-unknown', 'health-unknown'])
  })

  it('distinguishes no samples, insufficient samples, cache unavailable, and a real zero cache rate', () => {
    const wrapper = render([
      row({ metrics: metric({ has_data: false }), health: { ...health, overall: 'healthy' } }),
      row({ group_id: 2, metrics: metric({ cache_rate_available: false }), health: { ...health, overall: 'unknown' } }),
      row({ group_id: 3, metrics: metric({ cache_rate: 0 }) }),
    ])
    const rows = wrapper.findAll('.group-row')
    expect(rows[0].find('.success-cell').text()).toContain('availability.noData')
    expect(rows[0].find('.success-cell .metric-value').text()).toBe('—')
    expect(rows[1].find('.success-cell').text()).toContain('availability.insufficient')
    expect(rows[1].find('.cache-cell .metric-value').text()).toBe('—')
    expect(rows[2].find('.cache-cell .metric-value').text()).toBe('0.00%')
  })

  it('supports one keyboard focus stop per row, navigation and privacy-safe metric tooltips', async () => {
    const wrapper = render()
    const track = wrapper.find('.timeline-track')
    expect(wrapper.findAll('.timeline-bucket[tabindex]')).toHaveLength(0)
    expect(track.attributes('tabindex')).toBe('0')
    await track.trigger('focus')
    expect(track.attributes('aria-valuenow')).toBe('5')
    await track.trigger('keydown', { key: 'Home' })
    expect(track.attributes('aria-valuenow')).toBe('1')
    await track.trigger('keydown', { key: 'ArrowRight' })
    await track.trigger('keydown', { key: 'ArrowRight' })
    expect(track.attributes('aria-valuenow')).toBe('3')
    const tooltip = wrapper.find('[role="tooltip"]')
    expect(tooltip.text()).toContain('successRateValue value=80.0%')
    expect(tooltip.text()).toContain('cacheRateValue value=60.0%')
    expect(tooltip.text()).toContain('ttftValue')
    expect(tooltip.text()).toContain('durationValue')
    expect(tooltip.text()).not.toContain('request_count')
    expect(track.attributes('aria-valuetext')).toBe(tooltip.text())
    await track.trigger('keydown', { key: 'End' })
    expect(track.attributes('aria-valuenow')).toBe('5')
    await track.trigger('keydown', { key: 'Escape' })
    expect(wrapper.find('[role="tooltip"]').exists()).toBe(false)
  })

  it('shows hover details and supports legacy payloads without interpreting redacted zero counts as empty', async () => {
    const legacyMetric = metric({ has_data: undefined, success_rate: undefined, cache_rate_available: undefined })
    const wrapper = render([row({ metrics: legacyMetric, buckets: [{ bucket_start: '2026-10-17T00:02:00Z', metrics: legacyMetric, health }] })])
    expect(wrapper.find('.success-cell .metric-value').text()).toBe('95.0%')
    await wrapper.findAll('.timeline-bucket')[2].trigger('mouseenter')
    expect(wrapper.find('[role="tooltip"]').text()).toContain('successRateValue value=95.0%')
    await wrapper.find('.timeline-track').trigger('mouseleave')
    expect(wrapper.find('[role="tooltip"]').exists()).toBe(false)
  })

  it('retains loading/empty behavior and allows a filtered-empty description', () => {
    expect(render([], { loading: true }).findAll('.model-card')).toHaveLength(3)
    expect(render([], { emptyDescription: 'No groups match your search' }).find('.empty-state').text()).toContain('No groups match your search')
  })

  it('shows image generation duration and P50, keeps group rows together and omits token throughput', async () => {
    const imageMetrics = metric({
      cache_rate_available: false, tpm: 0,
      ttft: { sample_count: 0, p50_ms: null, avg_ms: null, p95_ms: null },
      duration: { sample_count: 0, p50_ms: 18400, avg_ms: 23100, p95_ms: 32000 },
    })
    const image = row({ is_image_model: true, model: 'image-generation', metrics: imageMetrics, buckets: [{ bucket_start: '2026-10-17T00:02:00Z', metrics: imageMetrics, health }] })
    const wrapper = render([image, { ...image, group_id: 2, group_name: 'vip' }])
    expect(wrapper.findAll('.model-card')).toHaveLength(1)
    expect(wrapper.findAll('.group-row')).toHaveLength(2)
    expect(wrapper.find('.image-model-badge').text()).toContain('availability.imageModel')
    expect(wrapper.find('.column-headings').text()).toContain('availability.generationTime')
    expect(wrapper.find('.column-headings').text()).not.toContain('availability.ttft')
    expect(wrapper.find('.latency-cell .metric-value').text()).toBe('32.0s')
    expect(wrapper.find('.generation-primary').text()).toContain('P95')
    expect(wrapper.find('.latency-cell').text()).toContain('generationP50 value=18.4s')
    expect(wrapper.find('.cache-cell .metric-value').text()).toBe('—')
    expect(wrapper.find('.latency-cell').text()).not.toContain('metrics.tpsValue')
    expect(wrapper.find('.latency-cell').text()).toContain('metrics.rpmValue')
    await wrapper.findAll('.timeline-bucket')[2].trigger('mouseenter')
    expect(wrapper.find('[role="tooltip"]').text()).toContain('ttftValue value=—')
    expect(wrapper.find('[role="tooltip"]').text()).toContain('durationValue value=AVG 23.1s')
    expect(wrapper.find('[role="tooltip"]').text()).not.toContain('metrics.tpsValue')
  })

  it('shows missing image durations as unavailable and relies only on the explicit model flag', () => {
    const emptyDuration = { sample_count: 0, p50_ms: null, avg_ms: null, p95_ms: null }
    const imageWrapper = render([row({ is_image_model: true, metrics: metric({ duration: emptyDuration }) })])
    expect(imageWrapper.find('.latency-cell .metric-value').text()).toBe('—')
    expect(imageWrapper.find('.latency-cell').text()).toContain('generationP50 value=—')
    // A model name that looks like an image model must not override the backend projection.
    const dialogWrapper = render([row({ model: 'gpt-image-2', is_image_model: false })])
    expect(dialogWrapper.find('.image-model-badge').exists()).toBe(false)
    expect(dialogWrapper.find('.column-headings').text()).toContain('availability.ttft')
    expect(dialogWrapper.find('.latency-cell .metric-value').text()).toBe('1.2s')
    expect(dialogWrapper.find('.latency-secondary').text()).toContain('averageLatency 2.5s')
    expect(dialogWrapper.find('.latency-cell').text()).toContain('metrics.tpsValue')
  })


  it('uses backend true-success health and generation health, and keeps stale data neutral', async () => {
    const record = row({ health: { ...health, overall: 'healthy', success_rate: 'critical' } })
    const wrapper = render([record])
    expect(wrapper.find('.success-cell .metric-value').classes()).toContain('metric-critical')
    await wrapper.setProps({ stale: true })
    expect(wrapper.find('.health-badge').text()).toContain('availability.staleStatus')
    expect(wrapper.find('.health-badge').classes()).toContain('status-stale')
    expect(wrapper.find('.health-badge').text()).not.toContain('availability.healthy')
    expect(wrapper.find('.success-cell .metric-value').classes()).toContain('metric-stale')
    expect(wrapper.find('.success-cell .metric-value').text()).toBe('80.0%')
    expect(wrapper.findAll('.timeline-bucket')[2].classes()).toContain('health-warning')
    expect(wrapper.find('.stale-note').text()).toContain('availability.historical')
  })

  it('emits the exact clicked interval and supports keyboard inspection without adding bucket focus stops', async () => {
    const record = row()
    const wrapper = render([record])
    await wrapper.findAll('.timeline-bucket')[2].trigger('click')
    expect(wrapper.emitted('inspect')?.[0]).toEqual([{ row: record, bucketStart: '2026-10-17T00:02:00.000Z', bucketEnd: '2026-10-17T00:03:00.000Z' }])
    const track = wrapper.find('.timeline-track')
    await track.trigger('keydown', { key: 'Home' })
    await track.trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('inspect')?.[1]).toEqual([{ row: record, bucketStart: '2026-10-17T00:00:00.000Z', bucketEnd: '2026-10-17T00:01:00.000Z' }])
    expect(wrapper.findAll('.timeline-bucket[tabindex]')).toHaveLength(0)
  })

  it('uses the agreed favorite key and emits favorite changes separately from inspection', async () => {
    const record = row()
    const wrapper = render([record], { favoriteKeys: [JSON.stringify(['openai', 'gpt-test', 1])] })
    expect(wrapper.find('.favorite-action').attributes('aria-pressed')).toBe('true')
    expect(wrapper.find('.favorite-action').text()).toBe('★')
    await wrapper.find('.favorite-action').trigger('click')
    expect(wrapper.emitted('toggle-favorite')).toEqual([[record]])
    expect(wrapper.emitted('inspect')).toBeUndefined()
  })


  it('preserves sorted model appearance and each model group order without re-sorting', () => {
    const wrapper = render([row({ model: 'z-model', group_id: 2, group_name: 'vip' }), row({ model: 'a-model' }), row({ model: 'z-model', group_id: 1, group_name: 'default' })])
    expect(wrapper.findAll('.model-name').map(name => name.text())).toEqual(['z-model', 'a-model'])
    expect(wrapper.findAll('.model-card')[0].findAll('.group-link').map(group => group.text())).toEqual(['vip', 'default'])
  })

  it('prefers the backend image percentile and generation status over a different local latency', () => {
    const wrapper = render([row({ is_image_model: true, health: { ...health, generation: 'critical', generation_latency_ms: 12000, generation_percentile: 'p90' } })])
    expect(wrapper.find('.latency-cell .metric-value').text()).toBe('12.0s')
    expect(wrapper.find('.latency-cell .metric-value').classes()).toContain('metric-critical')
    expect(wrapper.find('.generation-primary').text()).toContain('P90')
  })

})
