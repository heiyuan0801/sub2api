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
vi.mock('@/composables/useChannelMonitorFormat', () => ({ useChannelMonitorFormat: () => ({ providerLabel: (platform: string) => platform, providerBadgeClass: () => '' }) }))

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
  return { platform: 'openai', group_id: 1, group_name: 'default', metrics: metric(), health,
    buckets: [{ bucket_start: '2026-10-17T00:02:00Z', metrics: metric(), health }], ...patch }
}
function render(rows = [row()], extra: Record<string, unknown> = {}) {
  return mount(MonitorChannelCards, { props: { rows, coverage, healthMode: 'overall', ...extra }, global: { stubs: { ProviderIcon: true, EmptyState: { props: ['title', 'description'], template: '<div class="empty-state">{{ title }} {{ description }}</div>' } } } })
}

describe('MonitorChannelCards', () => {
  it('renders one card per group under its provider and uses effective rates, including zero', async () => {
    const second = row({ group_id: 2, group_name: 'vip' })
    const wrapper = render([row(), second, row({ platform: 'anthropic', group_id: 3 })], { groupRates: { 1: 0, 2: 0.125 } })
    expect(wrapper.findAll('.provider-section')).toHaveLength(2)
    expect(wrapper.findAll('.channel-card h3').map(node => node.text())).toEqual(['default', 'vip', 'default'])
    expect(wrapper.findAll('.rate-tag').map(node => node.text())).toEqual([
      'channelMonitorV2.channels.userRate value=0.00', 'channelMonitorV2.channels.userRate value=0.125',
    ])
    await wrapper.findAll('.channel-name button')[1].trigger('click')
    expect(wrapper.emitted('inspect')).toEqual([[{ row: second }]])
  })

  it('uses real availability and distinguishes unavailable cache and first token from a real zero cache rate', () => {
    const wrapper = render([
      row({ health: { ...health, success_rate: 'critical' } }),
      row({ group_id: 2, metrics: metric({ cache_rate_available: false, ttft: { sample_count: 0, avg_ms: null, p50_ms: null, p95_ms: null } }) }),
      row({ group_id: 3, metrics: metric({ cache_rate: 0 }) }),
      row({ group_id: 4, metrics: metric({ has_data: false }) }),
    ])
    expect(wrapper.findAll('.channel-card').map(card => card.findAll('dd').map(node => node.text()))).toEqual([
      ['60.0%', '80.0%', '1.2s'], ['—', '80.0%', '—'], ['0.00%', '80.0%', '1.2s'], ['—', '—', '—'],
    ])
    expect(wrapper.findAll('dd')[1].classes()).toContain('tone-critical')
  })

  it('preserves timeline gaps and exposes real historical intervals with exact drill-down', async () => {
    const record = row()
    const wrapper = render([record], { refreshSeconds: 29 })
    expect(wrapper.find('.timeline-heading').text()).toContain('refreshIn seconds=29')
    expect(wrapper.findAll('.timeline-bucket').map(bucket => bucket.classes().find(c => c.startsWith('health-')))).toEqual(['health-unknown', 'health-unknown', 'health-warning', 'health-unknown', 'health-unknown'])
    expect(wrapper.findAll('.history-list button')).toHaveLength(1)
    await wrapper.find('.history-list button').trigger('click')
    const interval = { row: record, bucketStart: '2026-10-17T00:02:00.000Z', bucketEnd: '2026-10-17T00:03:00.000Z' }
    expect(wrapper.emitted('inspect')).toEqual([[interval]])
    const track = wrapper.get('.timeline-track')
    await track.trigger('focus')
    expect(track.attributes('aria-valuenow')).toBe('5')
    await track.trigger('keydown', { key: 'Home' })
    await track.trigger('keydown', { key: 'ArrowRight' })
    await track.trigger('keydown', { key: 'ArrowRight' })
    expect(track.attributes('aria-valuenow')).toBe('3')
    const tooltip = wrapper.get('[role="tooltip"]')
    expect(tooltip.text()).toContain('availabilityValue value=80.0%')
    expect(tooltip.text()).toContain('cacheRateValue value=60.0%')
    expect(tooltip.text()).toContain('ttftValue')
    expect(tooltip.text()).not.toContain('request_count')
    expect(track.attributes('aria-valuetext')).toBe(tooltip.text())
    await track.trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('inspect')).toEqual([[interval], [interval]])
    await track.trigger('keydown', { key: 'Escape' })
    expect(wrapper.find('[role="tooltip"]').exists()).toBe(false)
  })

  it('keeps historical metrics visible and labels stale data', () => {
    const wrapper = render([row()], { stale: true })
    expect(wrapper.find('.timeline-heading').text()).toContain('availability.staleStatus')
    expect(wrapper.findAll('dd')[1].classes()).toContain('tone-unknown')
    expect(wrapper.findAll('dd')[1].text()).toBe('80.0%')
  })

  it('emits group favorites separately from inspection', async () => {
    const record = row()
    const wrapper = render([record], { favoriteKeys: [JSON.stringify(['openai', '', 1])] })
    const button = wrapper.get('.channel-actions button[aria-pressed]')
    expect(button.attributes('aria-pressed')).toBe('true')
    await button.trigger('click')
    expect(wrapper.emitted('toggle-favorite')).toEqual([[record]])
    expect(wrapper.emitted('inspect')).toBeUndefined()
  })
})
