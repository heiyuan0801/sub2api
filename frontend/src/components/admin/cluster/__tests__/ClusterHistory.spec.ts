import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ClusterHistory from '../ClusterHistory.vue'
import type { ClusterNode } from '@/api/admin/cluster'
const { history } = vi.hoisted(() => ({ history: vi.fn() }))
vi.mock('@/api/admin/cluster', () => ({ clusterApi: { history } }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (s: string) => s }) }))
const nodes = [{ id: 'a' }, { id: 'b' }] as ClusterNode[]
describe('Cluster history', () => {
  it('loads selected node and range without inventing missing measurements', async () => {
    history.mockResolvedValue({ points: [{ at: new Date().toISOString(), active: 2 }] })
    const wrapper = mount(ClusterHistory, { props: { nodes } }); await flushPromises()
    expect(history).toHaveBeenLastCalledWith('a', '24h', expect.any(AbortSignal))
    expect(wrapper.findAll('circle')).toHaveLength(0)
    await wrapper.findAll('select')[1]!.setValue('7d'); await flushPromises()
    expect(history).toHaveBeenLastCalledWith('a', '7d', expect.any(AbortSignal))
    await wrapper.findAll('select')[0]!.setValue('b'); await flushPromises()
    expect(history).toHaveBeenLastCalledWith('b', '7d', expect.any(AbortSignal))
    wrapper.unmount()
  })
  it('does not bridge a missing five-minute sample', async () => {
    const now = Date.now()
    history.mockResolvedValue({ points: [
      { at: new Date(now - 540000).toISOString(), performance: { requests_per_second: 2 } },
      { at: new Date(now).toISOString(), performance: { requests_per_second: 3 } }
    ] })
    const wrapper = mount(ClusterHistory, { props: { nodes } }); await flushPromises()
    expect(wrapper.findAll('path')).toHaveLength(3)
    wrapper.unmount()
  })
  it('shows unavailable history as an error and cancels on unmount', async () => {
    history.mockRejectedValue(new Error('offline'))
    const wrapper = mount(ClusterHistory, { props: { nodes } }); await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('historyView.failed')
    const signal = history.mock.calls.at(-1)![2] as AbortSignal
    wrapper.unmount(); expect(signal.aborted).toBe(true)
  })
})
