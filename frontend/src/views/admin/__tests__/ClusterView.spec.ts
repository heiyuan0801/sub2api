import { mount, flushPromises } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ClusterView from '../ClusterView.vue'
const mocks = vi.hoisted(() => ({ snapshot: vi.fn(), update: vi.fn(), showSuccess: vi.fn(), showError: vi.fn() }))
vi.mock('@/api/admin/cluster', () => ({ clusterApi: mocks }))
vi.mock('@/stores/app', () => ({ useAppStore: () => mocks }))
vi.mock('@/components/layout/AppLayout.vue', () => ({ default: { template: '<div><slot /></div>' } }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const node = { id: 'us', instance_id: 'instance', role: 'primary', online: true, version: 'v1', control: { maintenance: false, max_concurrency: 100, weight: 50, revision: 7, reason: '' }, stats: { active_requests: 3, control_revision: 7, ingress_revision: 6, heap_bytes: 1048576, database_ok: true, redis_ok: true, lease_ok: true } }
const snapshot = { enabled: true, ha_enabled: true, nodes: [node], alerts: [], events: [], review_count: 0, now: '2026-10-25T00:00:00Z' }
const render = () => mount(ClusterView, { global: { stubs: { AppLayout: { template: '<div><slot /></div>' }, BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /></div>' } } } })
beforeEach(() => { vi.useFakeTimers(); vi.clearAllMocks(); mocks.snapshot.mockResolvedValue(snapshot); mocks.update.mockResolvedValue({}) })
afterEach(() => vi.useRealTimers())
describe('Cluster controls', () => {
 it('shows pending ingress acknowledgement and sends the original revision', async () => {
  const wrapper = render(); await flushPromises()
  expect(wrapper.text()).toContain('cluster.pending')
  await wrapper.findAll('button').find(b => b.text() === 'cluster.manage')!.trigger('click')
  await wrapper.find('textarea').setValue('planned maintenance')
  await wrapper.find('form').trigger('submit'); await flushPromises()
  expect(mocks.update).toHaveBeenCalledWith('us', expect.objectContaining({ revision: 7, reason: 'planned maintenance', weight: 50, max_concurrency: 100 }))
  wrapper.unmount()
 })
 it('keeps stale data visibly marked after refresh failure and stops polling on unmount', async () => {
  const wrapper = render(); await flushPromises()
  mocks.snapshot.mockRejectedValueOnce(new Error('offline'))
  await wrapper.find('button').trigger('click'); await flushPromises()
  expect(wrapper.find('[role="alert"]').text()).toBe('cluster.refreshError')
  expect(wrapper.text()).toContain('us')
  const calls = mocks.snapshot.mock.calls.length
  wrapper.unmount(); await vi.advanceTimersByTimeAsync(20000)
  expect(mocks.snapshot).toHaveBeenCalledTimes(calls)
 })
})
