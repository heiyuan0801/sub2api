import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, shallowMount } from '@vue/test-utils'
import AnnouncementsView from '../AnnouncementsView.vue'

const { route, replace, list, create, publish, getAllIncludingInactive, getPlans, showError } = vi.hoisted(() => ({
  route: { query: {} as Record<string, string> },
  replace: vi.fn(),
  list: vi.fn(),
  create: vi.fn(),
  publish: vi.fn(),
  getAllIncludingInactive: vi.fn(),
  getPlans: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('vue-router', async importOriginal => ({
  ...await importOriginal<typeof import('vue-router')>(),
  useRoute: () => route,
  useRouter: () => ({ replace }),
}))
vi.mock('vue-i18n', async importOriginal => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key }),
}))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError, showSuccess: vi.fn() }) }))
vi.mock('@/api/admin', () => ({ adminAPI: { announcements: { list, create, publish }, groups: { getAllIncludingInactive } } }))
vi.mock('@/api/admin/payment', () => ({ adminPaymentAPI: { getPlans } }))

enableAutoUnmount(afterEach)

beforeEach(() => {
  vi.resetAllMocks()
  list.mockResolvedValue({ items: [], total: 0, pages: 1, page: 1, page_size: 20 })
  getPlans.mockResolvedValue({ data: [] })
  getAllIncludingInactive.mockResolvedValue([
    { id: 7, name: 'Balance group', description: 'Balance access', subscription_type: 'standard', rate_multiplier: 0.15, is_exclusive: false },
  ])
})

function mountAnnouncements() {
  return shallowMount(AnnouncementsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' },
      },
    },
  })
}

describe('standard balance group announcements', () => {
  it('saves a linked rate change as a draft without publishing', async () => {
    route.query = { compose: '1', kind: 'price_increase', group_id: '7', old_rate: '0.1', new_rate: '0.15' }
    create.mockResolvedValue({ id: 12, title: 'Rate change', status: 'draft', kind: 'price_increase', group_id: 7 })
    const wrapper = mountAnnouncements()
    await flushPromises()

    expect((wrapper.get('#notice-target').element as HTMLSelectElement).value).toBe('group:7')
    expect((wrapper.get('#notice-audience').element as HTMLSelectElement).value).toBe('groups')
    expect((wrapper.get('#announcement-form textarea').element as HTMLTextAreaElement).value).toContain('0.1× → 0.15×')

    await wrapper.get('#announcement-form').trigger('submit')
    await flushPromises()
    expect(create).toHaveBeenCalledWith(expect.objectContaining({
      status: 'draft', kind: 'price_increase', group_id: 7, audience: { mode: 'groups', group_ids: [7] },
    }))
    expect(publish).not.toHaveBeenCalled()
    expect(showError).not.toHaveBeenCalled()
  })

  it('defaults new public groups to all eligible users', async () => {
    route.query = { compose: '1', kind: 'new_group', group_id: '7' }
    const wrapper = mountAnnouncements()
    await flushPromises()

    expect((wrapper.get('#notice-kind').element as HTMLSelectElement).value).toBe('new_group')
    expect((wrapper.get('#notice-target').element as HTMLSelectElement).value).toBe('group:7')
    expect((wrapper.get('#notice-audience').element as HTMLSelectElement).value).toBe('all')
  })
})
