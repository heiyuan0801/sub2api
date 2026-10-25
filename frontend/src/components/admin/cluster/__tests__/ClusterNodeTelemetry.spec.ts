import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import type { ClusterNode } from '@/api/admin/cluster'
import ClusterNodeTelemetry from '../ClusterNodeTelemetry.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string, params?: Record<string, unknown>) => `${key}${params ? ` ${Object.values(params).join(' ')}` : ''}` }) }))
enableAutoUnmount(afterEach)
const now = Date.parse('2026-10-25T08:00:00Z')
function node(): ClusterNode {
  return {
    id: 'hk', instance_id: 'hk-1', role: 'primary', version: 'test', online: true,
    heartbeat_at: new Date(now - 2000).toISOString(), started_at: new Date(now - 3600000).toISOString(),
    control: { maintenance: false, max_concurrency: 100, weight: 100, revision: 1, reason: '' },
    stats: {
      active_requests: 3, requests: 100, errors: 2, rejected: 1, heap_bytes: 1024 ** 2, goroutines: 40,
      disk_total_bytes: 100 * 1024 ** 3, disk_free_bytes: 60 * 1024 ** 3,
      database_ok: true, redis_ok: true, lease_ok: true, control_revision: 1, ingress_revision: 1, ingress_ok: true,
      replication_peers: 1, replication_known: true, sync_replicas: 1, replication_lag_bytes: 0,
      redis_role: 'master', redis_replicas: 1, redis_link_ok: true, settings_version: 'test',
      database_latency_ms: 2.4, redis_latency_ms: 0.8,
      resources: { sampled_at: new Date(now - 4000).toISOString(), hostname: 'host-hk', os: 'linux', arch: 'amd64',
        cpu_cores: 40, cpu_percent: 22.5, cpu_window_seconds: 5, load_1: 2.2, load_5: 1.2, load_15: 0.8,
        memory_total_bytes: 128 * 1024 ** 3, memory_used_bytes: 32 * 1024 ** 3, memory_available_bytes: 96 * 1024 ** 3,
        memory_percent: 25, process_rss_bytes: 128 * 1024 ** 2, uptime_seconds: 86400 }
    }
  }
}
const render = (data = node()) => mount(ClusterNodeTelemetry, { props: { node: data, now, haEnabled: true } })

describe('Cluster node telemetry', () => {
  it('renders measured CPU, system memory, disk and connection latency', () => {
    const wrapper = render()
    expect(wrapper.findAll('[role="progressbar"]').map(bar => bar.attributes('aria-valuenow'))).toEqual(['22.5', '25', '40'])
    expect(wrapper.text()).toContain('32.0 GiB / 128.0 GiB')
    expect(wrapper.text()).toContain('128.0 MiB')
    expect(wrapper.get('[data-connection="database"]').text()).toContain('2.4 ms')
    expect(wrapper.get('[data-connection="redis"]').text()).toContain('0.8 ms')
    expect(wrapper.get('[data-connection="ingress"]').text()).toContain('cluster.connected')
  })
  it('does not show stale values or healthy connections after heartbeat expiry', async () => {
    const wrapper = render()
    await wrapper.setProps({ now: now + 30000 })
    expect(wrapper.text()).toContain('cluster.telemetryStale')
    expect(wrapper.findAll('[role="progressbar"]')).toHaveLength(0)
    expect(wrapper.get('[data-connection="heartbeat"]').text()).toContain('cluster.disconnected')
    expect(wrapper.get('[data-connection="database"]').text()).toContain('cluster.unconfirmed')
    expect(wrapper.text()).not.toContain('22.5%')
    expect(wrapper.text()).not.toContain('2.4 ms')
  })
  it('tolerates old nodes and incomplete CPU sampling without inventing zeros', () => {
    const data = node()
    delete data.stats.resources
    delete data.stats.ingress_ok
    const wrapper = render(data)
    expect(wrapper.text()).toContain('cluster.noData')
    expect(wrapper.findAll('[role="progressbar"]')).toHaveLength(1)
    expect(wrapper.get('[data-connection="ingress"]').text()).toContain('cluster.unconfirmed')
  })
  it('expires only resource samples while fresh heartbeats keep connections current', () => {
    const data = node()
    data.stats.resources!.sampled_at = new Date(now - 60000).toISOString()
    const wrapper = render(data)
    expect(wrapper.text()).toContain('cluster.resourceStale')
    expect(wrapper.findAll('[role="progressbar"]')).toHaveLength(1)
    expect(wrapper.get('[data-connection="database"]').text()).toContain('cluster.connected')
  })
  it('does not mistake a failed sample or a future timestamp for zero usage', async () => {
    const data = node()
    data.stats.resources!.cpu_percent = null
    const wrapper = render(data)
    expect(wrapper.findAll('[role="progressbar"]')).toHaveLength(2)
    await wrapper.setProps({ node: { ...data, stats: { ...data.stats, resources: { ...data.stats.resources!, sampled_at: new Date(now + 60000).toISOString() } } } })
    expect(wrapper.findAll('[role="progressbar"]')).toHaveLength(1)
    expect(wrapper.text()).toContain('cluster.resourceStale')
  })
  it('shows a missing Redis replica and distinguishes disabled HA', async () => {
    const data = node()
    data.stats.redis_replicas = 0
    const wrapper = render(data)
    expect(wrapper.get('[data-connection="redisReplication"]').text()).toContain('cluster.disconnected')
    await wrapper.setProps({ haEnabled: false })
    expect(wrapper.get('[data-connection="lease"]').text()).toContain('cluster.notApplicable')
    expect(wrapper.get('[data-connection="ingress"]').text()).toContain('cluster.notApplicable')
  })
})
