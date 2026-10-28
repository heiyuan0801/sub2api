import { beforeEach, describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import UpstreamBalanceCell from '../UpstreamBalanceCell.vue'
import type { Account } from '@/types'
const mocks = vi.hoisted(() => ({ probe: vi.fn(), error: vi.fn() }))
vi.mock('@/api/admin/accounts', () => ({ probeUpstreamBalance: mocks.probe }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: mocks.error }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const now = Date.parse('2026-10-27T12:00:00Z')
function account(amount?: number, status = 'ok', queried_at = '2026-10-27T12:00:00Z') {
 return { id: 1, type: 'apikey', extra: { upstream_billing_probe: { status: 'unqueried', balance: { amount, status, unit: 'USD', kind: 'wallet', queried_at } } } } as Account
}
beforeEach(() => vi.clearAllMocks())
describe('upstream balance', () => {
 it('distinguishes zero funds, missing data, failed queries and stale snapshots', async () => {
  const wrapper = mount(UpstreamBalanceCell, { props: { account: account(0), now } })
  expect(wrapper.get('[data-testid="upstream-balance-value"]').text()).toBe('$0.00')
  await wrapper.setProps({ account: account(undefined, 'failed') })
  expect(wrapper.get('[data-testid="upstream-balance-value"]').text()).toBe('—')
  expect(wrapper.text()).toContain('upstreamBalance.failed')
  await wrapper.setProps({ account: account(123.95, 'ok', '2026-10-27T10:00:00Z') })
  expect(wrapper.text()).toContain('$123.95')
  expect(wrapper.text()).toContain('upstreamBalance.stale')
 })
 it('queries only on demand, prevents duplicate clicks, and emits the returned snapshot', async () => {
  let resolve!: (value: unknown) => void
  mocks.probe.mockImplementation(() => new Promise(r => { resolve = r }))
  const wrapper = mount(UpstreamBalanceCell, { props: { account: account(), now } })
  expect(mocks.probe).not.toHaveBeenCalled()
  await wrapper.get('button').trigger('click')
  await wrapper.get('button').trigger('click')
  expect(mocks.probe).toHaveBeenCalledTimes(1)
  const snapshot = account(9.95).extra!.upstream_billing_probe
  resolve({ snapshot }); await flushPromises()
  expect(wrapper.emitted('updated')?.[0]).toEqual([snapshot])
 })
 it('does not expose queries for OAuth accounts', () => {
  const wrapper = mount(UpstreamBalanceCell, { props: { account: { ...account(), type: 'oauth' }, now } })
  expect(wrapper.find('button').exists()).toBe(false)
 })
})
