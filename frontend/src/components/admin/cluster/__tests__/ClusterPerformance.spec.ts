import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import type { ClusterNode } from '@/api/admin/cluster'
import ClusterPerformance from '../ClusterPerformance.vue'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const now = Date.parse('2026-10-25T12:00:00Z')
const sampled = new Date(now - 2000).toISOString()
function fixture() {
  return { online: true, heartbeat_at: sampled, stats: {
    performance: { sampled_at: sampled, requests_per_second: 12, first_token_p95_ms: 500, first_token_samples: 100, cache_read_percent: 80, db_waits: 2, db_wait_mean_ms: 125 },
    ingress_traffic: { sampled_at: sampled, queued: 3, active: 10, bytes_in: 1048576, bytes_out: 2097152, cross_node_bytes: 1048576 },
    verification: { checked_at: sampled, scope: 'local_lab', artifacts_match: true, endpoints: [{ id: 'us', status: 'pass', latency_ms: 20, certificate_expires_at: null }, { id: 'hk', status: 'fail', reason: 'tls_failed', latency_ms: 5, certificate_expires_at: null }] }
  }} as ClusterNode
}
describe('Cluster performance', () => {
  it('renders real measurements, report scope and per-entry failures', () => {
    const wrapper = mount(ClusterPerformance, { props: { node: fixture(), now } })
    expect(wrapper.text()).toContain('12.0 /s')
    expect(wrapper.text()).toContain('80.0%')
    expect(wrapper.text()).toContain('500.0 ms')
    expect(wrapper.text()).toContain('cluster.performance.lab')
    expect(wrapper.text()).toContain('cluster.performance.tls_failed')
    expect(wrapper.findAll('[data-testid="ingress-proof"]')).toHaveLength(2)
  })
  it('expires telemetry and never labels stale reports healthy', () => {
    const wrapper = mount(ClusterPerformance, { props: { node: fixture(), now: now + 310000 } })
    expect(wrapper.text()).not.toContain('12.0 /s')
    expect(wrapper.text()).not.toContain('cluster.log.statuses.pass')
    expect(wrapper.text()).toContain('cluster.performance.staleProof')
  })
  it('supports old nodes without manufacturing zeros or verification', () => {
    const wrapper = mount(ClusterPerformance, { props: { node: { ...fixture(), stats: {} } as ClusterNode, now } })
    expect(wrapper.text()).toContain('cluster.performance.noProof')
    expect(wrapper.text()).not.toContain('0.0%')
    expect(wrapper.text()).not.toContain('cluster.log.statuses.pass')
  })
})
