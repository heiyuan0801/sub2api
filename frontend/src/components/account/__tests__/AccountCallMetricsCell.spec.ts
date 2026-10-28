import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AccountCallMetricsCell from '../AccountCallMetricsCell.vue'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
describe('account call metrics', () => {
  it('renders a weighted input cache rate and first-token milliseconds as seconds', () => {
    const wrapper = mount(AccountCallMetricsCell, { props: { stats: { requests: 20, tokens: 2000, cost: 1, cache_read_tokens: 924, cache_input_tokens: 1000, average_first_token_ms: 7400 } } })
    expect(wrapper.text()).toContain('92.4%')
    expect(wrapper.text()).toContain('7.4s')
  })
  it('does not invent missing latency or zero-denominator cache metrics', () => {
    const wrapper = mount(AccountCallMetricsCell, { props: { stats: { requests: 2, tokens: 0, cost: 1, cache_read_tokens: 0, cache_input_tokens: 0, average_first_token_ms: null } } })
    expect(wrapper.text()).not.toContain('0.0%')
    expect(wrapper.text()).not.toContain('0.0s')
    expect(wrapper.text()).toContain('—')
  })
})

it('shows failures even when no request produced billable usage', () => {
  const wrapper = mount(AccountCallMetricsCell, { props: { stats: { requests: 0, tokens: 0, cost: 0, error_stats: { abnormal_requests: 2, recovered_requests: 1, failed_requests: 1 } } } })
  expect(wrapper.text()).toContain('callMetrics.errors')
  expect(wrapper.text()).toContain('callMetrics.recovered')
  expect(wrapper.text()).toContain('callMetrics.finalFailed')
  expect(wrapper.text()).not.toContain('callMetrics.unused')
})
it('distinguishes unavailable error telemetry from zero errors', () => {
  const wrapper = mount(AccountCallMetricsCell, { props: { stats: { requests: 0, tokens: 0, cost: 0, error_stats: null } } })
  expect(wrapper.text()).toContain('callMetrics.errorsUnavailable')
  expect(wrapper.text()).not.toContain('callMetrics.finalFailed')
})
