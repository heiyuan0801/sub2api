import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import MonitorIncidentDialog from '../MonitorIncidentDialog.vue'
import type { MonitorCoverage, MonitorHealth, MonitorMatrixRow, MonitorMetric } from '@/api/channelMonitorV2'

vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ locale: { value: 'en-US' }, te: () => true, t: (key: string) => key }),
}))
const health: MonitorHealth = { overall: 'warning', error_rate: 'healthy', success_rate: 'critical', ttft: 'healthy', minimum_sample: 20 }
function metric(patch: Partial<MonitorMetric> = {}): MonitorMetric {
  return { has_data: true, cache_rate_available: false, success_rate: 0.8, request_count: 98765, success_requests: 98760, error_requests: 5,
    token_count: 0, rpm: 1, tpm: 0, error_rate: 0.01, cache_rate: 0, cache_rate_numerator: 0, cache_rate_denominator: 0,
    ttft: { sample_count: 0, p50_ms: 1200, p90_ms: 1800, avg_ms: 1400, p95_ms: 2400 },
    duration: { sample_count: 0, p50_ms: 18000, p90_ms: 28000, avg_ms: 23000, p95_ms: 32000 }, ...patch }
}
const coverage: MonitorCoverage = { requested_start: '2026-10-17T00:00:00Z', requested_end: '2026-10-17T00:05:00Z', data_through: '2026-10-17T00:05:00Z', coverage_start: '2026-10-17T00:00:00Z', computed_at: '2026-10-17T00:05:00Z', bucket_seconds: 60, aggregation_lag_seconds: 0, coverage_complete: true }
function row(patch: Partial<MonitorMatrixRow> = {}): MonitorMatrixRow { return { platform: 'openai', model: 'gpt-test', group_id: 1, group_name: 'default', metrics: metric(), health, buckets: [{ bucket_start: '2026-10-17T00:02:00Z', metrics: metric({ success_rate: 0.2 }), health }], ...patch } }
const errors = [{ category: 'timeout', rate: 0.5, count: 12345, details: [{ message: 'private upstream error', error_type: 'upstream', status_code: 502, count: 12345 }] }]
function render(extra: Record<string, unknown> = {}) { return mount(MonitorIncidentDialog, { props: { show: true, row: row(), coverage, errors, ...extra }, global: { stubs: { BaseDialog: { props: ['show', 'title'], template: '<section v-if="show"><slot /><slot name="footer" /></section>' }, RouterLink: { name: 'RouterLink', props: ['to'], template: '<a><slot /></a>' } } } }) }

describe('MonitorIncidentDialog', () => {
  it('shows rates and latency while withholding absolute counts and raw errors from ordinary users', () => {
    const wrapper = render()
    expect(wrapper.find('.incident-metrics').text()).toContain('80.0%')
    expect(wrapper.find('.incident-metrics').text()).toContain('1.2s')
    expect(wrapper.find('.incident-metrics').text()).toContain('1.8s')
    expect(wrapper.find('.incident-metrics').text()).toContain('2.4s')
    expect(wrapper.find('.incident-metrics').text()).toContain('—')
    expect(wrapper.find('.failure-list').text()).toContain('50.0%')
    expect(wrapper.text()).not.toContain('private upstream error')
    expect(wrapper.text()).not.toContain('12345')
    expect(wrapper.text()).not.toContain('98765')
    expect(wrapper.find('.incident-admin').exists()).toBe(false)
  })

  it('uses the selected bucket rather than full-window row metrics and supports snapshot overrides', async () => {
    const wrapper = render({ window: { start: '2026-10-17T00:02:00Z', end: '2026-10-17T00:03:00Z' } })
    expect(wrapper.find('.incident-metrics').text()).toContain('20.0%')
    await wrapper.setProps({ metrics: metric({ success_rate: 0.4 }) })
    expect(wrapper.find('.incident-metrics').text()).toContain('40.0%')
    expect(wrapper.find('.incident-metrics').text()).not.toContain('80.0%')
  })

  it('does not show full-window metrics for a selected interval with no data', () => {
    const wrapper = render({ window: { start: '2026-10-17T00:03:00Z', end: '2026-10-17T00:04:00Z' } })
    expect(wrapper.text()).toContain('incident.windowNoData')
    expect(wrapper.find('.incident-metrics').text()).not.toContain('80.0%')
  })

  it('shows image generation percentiles and backend generation status', () => {
    const wrapper = render({ row: row({ is_image_model: true, health: { ...health, generation: 'warning', generation_latency_ms: 17000, generation_percentile: 'p95' } }) })
    expect(wrapper.find('.incident-model-tag').exists()).toBe(true)
    expect(wrapper.find('.incident-metrics').text()).toContain('17.0s')
    expect(wrapper.find('.incident-metrics').text()).toContain('28.0s')
    expect(wrapper.find('.incident-metrics').text()).toContain('32.0s')
    expect(wrapper.find('.incident-metrics .tone-warning').exists()).toBe(true)
  })

  it('reveals admin summaries and links only supported date queries to the real usage route', () => {
    const wrapper = render({ isAdmin: true })
    expect(wrapper.find('.admin-request-count').text()).toContain('98765')
    expect(wrapper.find('.admin-details').text()).toContain('private upstream error')
    const route = wrapper.findComponent({ name: 'RouterLink' }).props('to')
    expect(route.path).toBe('/admin/usage')
    expect(Object.keys(route.query).sort()).toEqual(['end_date', 'start_date'])
    expect(route.query.start_date).toMatch(/^2026-10-17$/)
    expect(wrapper.text()).toContain('incident.logsHint')
  })

  it('renders loading and retry states without exposing an old failure list', async () => {
    const wrapper = render({ loading: true })
    expect(wrapper.find('.failure-list').exists()).toBe(false)
    expect(wrapper.find('[role="status"]').text()).toContain('common.loading')
    await wrapper.setProps({ loading: false, error: 'Request failed' })
    expect(wrapper.find('[role="alert"]').text()).toContain('Request failed')
    await wrapper.find('[role="alert"] button').trigger('click')
    expect(wrapper.emitted('retry')).toEqual([[]])
    expect(wrapper.find('.failure-list').exists()).toBe(false)
  })

  it('keeps stale numbers visible with neutral status colors', () => {
    const wrapper = render({ stale: true })
    expect(wrapper.text()).toContain('freshness.title')
    expect(wrapper.find('.incident-metrics').text()).toContain('80.0%')
    expect(wrapper.find('.tone-critical').exists()).toBe(false)
  })
})
