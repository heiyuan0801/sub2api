import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, shallowMount } from '@vue/test-utils'
import { ref } from 'vue'
import NotificationPreferencesDialog from '../NotificationPreferencesDialog.vue'

const { preferences, setGroupSubscription, getAvailable, getPlans, getActive, showError } = vi.hoisted(() => ({
  preferences: vi.fn(),
  setGroupSubscription: vi.fn(),
  getAvailable: vi.fn(),
  getPlans: vi.fn(),
  getActive: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/announcements', () => ({ default: { preferences, setGroupSubscription } }))
vi.mock('@/api/groups', () => ({ default: { getAvailable } }))
vi.mock('@/api/payment', () => ({ paymentAPI: { getPlans } }))
vi.mock('@/stores/subscriptions', () => ({ useSubscriptionStore: () => ({ fetchActiveSubscriptions: getActive }) }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key, locale: ref('en') }) }))

enableAutoUnmount(afterEach)

beforeEach(() => {
  vi.resetAllMocks()
  preferences.mockResolvedValue({ muted_group_ids: [3, 5] })
  getAvailable.mockResolvedValue([
    { id: 3, name: 'Standard access', subscription_type: 'standard' },
    { id: 4, name: 'Active group', subscription_type: 'subscription' },
  ])
  getPlans.mockResolvedValue({ data: [{ group_id: 6, group_name: 'Published plan' }] })
  getActive.mockResolvedValue([{ group_id: 4, group: { name: 'Active group' } }])
})

function mountPreferences() {
  return shallowMount(NotificationPreferencesDialog, {
    props: { show: true },
    global: { stubs: { BaseDialog: { template: '<div><slot /></div>' } } },
  })
}

describe('Group notification preferences', () => {
  it('merges standard and subscription groups and preserves a hidden group preference', async () => {
    const wrapper = mountPreferences()
    await flushPromises()

    const standard = wrapper.get('[role="switch"][aria-label="Standard access"]')
    expect(standard.attributes('aria-checked')).toBe('false')
    expect(wrapper.get('[role="switch"][aria-label="Active group"]').attributes('aria-checked')).toBe('true')
    expect(wrapper.get('[role="switch"][aria-label="Published plan"]').attributes('aria-checked')).toBe('true')
    expect(wrapper.get('[role="switch"][aria-label="operations.notificationGroup #5"]').attributes('aria-checked')).toBe('false')

    setGroupSubscription.mockResolvedValueOnce({ subscribed: true }).mockResolvedValueOnce({ subscribed: false })
    await standard.trigger('click')
    await flushPromises()
    expect(setGroupSubscription).toHaveBeenNthCalledWith(1, 3, true)
    expect(standard.attributes('aria-checked')).toBe('true')

    await standard.trigger('click')
    await flushPromises()
    expect(setGroupSubscription).toHaveBeenNthCalledWith(2, 3, false)
    expect(standard.attributes('aria-checked')).toBe('false')
  })

  it('keeps subscription controls available when the standard group list fails', async () => {
    getAvailable.mockRejectedValueOnce(new Error('offline'))
    const wrapper = mountPreferences()
    await flushPromises()

    expect(wrapper.get('[role="status"]').text()).toContain('operations.groupPreferencesPartial')
    expect(wrapper.get('[role="switch"][aria-label="Active group"]').attributes('aria-checked')).toBe('true')
    expect(wrapper.get('[role="switch"][aria-label="Published plan"]').attributes('aria-checked')).toBe('true')
  })
})
