import { mount, flushPromises } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ClusterView from '../ClusterView.vue'
const mocks = vi.hoisted(() => ({ snapshot: vi.fn(), preflight: vi.fn(), requests: vi.fn(), update: vi.fn(), showSuccess: vi.fn(), showError: vi.fn() }))
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
 it('explains historical events and keeps raw text in collapsed details without rendering HTML', async () => {
  mocks.snapshot.mockResolvedValue({ ...snapshot, events: [{ id: 1, node_id: 'hk', kind: 'control_changed', message: 'maintenance=false capacity=100 weight=50: <script>alert(1)</script>', created_at: snapshot.now }] })
  const wrapper = render(); await flushPromises()
  await wrapper.findAll('button').find(b => b.text() === 'cluster.events')!.trigger('mousedown', { button: 0, ctrlKey: false })
  await flushPromises()
  expect(wrapper.text()).toContain('cluster.log.kinds.control_changed')
  expect(wrapper.text()).toContain('cluster.log.messages.controlChanged')
  expect(wrapper.get('details').attributes('open')).toBeUndefined()
  expect(wrapper.get('details pre').text()).toContain('<script>alert(1)</script>')
  expect(wrapper.find('script').exists()).toBe(false)
  wrapper.unmount()
 })
 it('renders deployment check labels and manual status without claiming success', async () => {
  mocks.preflight.mockResolvedValue({ checks: [{ code: 'independent_witness', status: 'manual', detail: 'verify the third quorum member is in an independent fault domain' }] })
  const wrapper = render(); await flushPromises()
  await wrapper.findAll('button').find(b => b.text() === 'cluster.preflight')!.trigger('mousedown', { button: 0, ctrlKey: false })
  await flushPromises()
  expect(wrapper.text()).toContain('cluster.log.checks.independent_witness')
  expect(wrapper.text()).toContain('cluster.log.statuses.manual')
  expect(wrapper.text()).toContain('cluster.log.messages.witness')
  wrapper.unmount()
 })

})
