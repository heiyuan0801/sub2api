import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import type { UserSupportedModelPricing } from '@/api/channels'
import PlazaCatalog from '../PlazaCatalog.vue'
import PlazaModelOverview from '../PlazaModelOverview.vue'
import type { ModelPlazaGroup } from '@/api/modelPlaza'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key, n: String, locale: 'en' }) }))
vi.mock('vue-router', () => ({ useRoute: () => ({ query: {} }), RouterLink: { template: '<a><slot /></a>' } }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ isAuthenticated: false }) }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ cachedPublicSettings: null }) }))
vi.mock('@/composables/useClipboard', () => ({ useClipboard: () => ({ copyToClipboard: vi.fn() }) }))
enableAutoUnmount(afterEach)

function group(id: number, platform: string, name: string): ModelPlazaGroup {
  return {
    id, name: `Group ${id}`, description: '', platform,
    subscription_type: 'standard', rate_multiplier: 1,
    peak_rate_enabled: false, peak_start: '', peak_end: '', peak_rate_multiplier: 1,
    is_exclusive: false, image_rate_independent: false, image_rate_multiplier: 1,
    video_rate_independent: false, video_rate_multiplier: 1,
    long_context_pricing_enabled: true,
    models: [{ name, platform, pricing: null, official_pricing: null }]
  }
}

function catalog() {
  return mount(PlazaCatalog, {
    props: { loading: false, response: { description: '', groups: [group(1, 'openai', 'gpt-test'), group(2, 'gemini', 'gemini-test')] } },
    global: { stubs: { RouterLink: true, PlazaModelPricingTable: true, PlazaGroupSection: true, PlatformIcon: true, Icon: true } }
  })
}

describe('PlazaCatalog provider navigation', () => {
  it('opens model details with the three tabs and all matching groups despite a catalog search', async () => {
    const wrapper = catalog()
    await wrapper.setProps({
      response: { description: '', groups: [group(1, 'openai', 'gpt-test'), group(2, 'openai', 'gpt-test'), group(3, 'gemini', 'gpt-test')] }
    })
    const showModal = Object.getOwnPropertyDescriptor(HTMLDialogElement.prototype, 'showModal')
    Object.defineProperty(HTMLDialogElement.prototype, 'showModal', { configurable: true, value: vi.fn() })
    try {
      await wrapper.get('input[type="search"]').setValue('Group 2')
      await wrapper.get('.offer-detail').trigger('click')
      const dialog = wrapper.get('.model-detail')
      expect(dialog.findAll('[role="tab"]').map(tab => tab.text())).toEqual([
        'publicSite.modelDetail.overview',
        'publicSite.modelDetail.performance',
        'publicSite.modelDetail.api',
      ])
      const overview = wrapper.getComponent(PlazaModelOverview)
      expect(overview.props('groups').map(item => item.id)).toEqual([1, 2])
      expect(overview.get('[aria-pressed="true"]').text()).toBe('Group 2')
      await dialog.get('.model-close').trigger('click')
      expect(wrapper.find('.model-detail').exists()).toBe(false)
    } finally {
      if (showModal) Object.defineProperty(HTMLDialogElement.prototype, 'showModal', showModal)
      else Reflect.deleteProperty(HTMLDialogElement.prototype, 'showModal')
    }
  })

  it('filters real group models through provider chips and search, and clears empty results', async () => {
    const wrapper = catalog()
    await wrapper.get('[aria-label="publicSite.models.cardView"]').trigger('click')
    expect(wrapper.findAll('.catalog-card h3').map(node => node.text())).toEqual(['gpt-test', 'gemini-test'])
    const openAI = wrapper.findAll('nav button').find(button => button.text() === 'OpenAI')!
    await openAI.trigger('click')
    expect(openAI.attributes('aria-pressed')).toBe('true')
    expect(wrapper.findAll('.catalog-card h3').map(node => node.text())).toEqual(['gpt-test'])
    await wrapper.get('input[type="search"]').setValue('gemini')
    expect(wrapper.findAll('.catalog-card')).toHaveLength(0)
    await wrapper.get('.catalog-empty button').trigger('click')
    expect(wrapper.findAll('.catalog-card')).toHaveLength(2)
    expect((wrapper.get('input').element as HTMLInputElement).value).toBe('')
  })

  it('resets a removed provider when refreshed group data changes', async () => {
    const wrapper = catalog()
    await wrapper.findAll('nav button').find(button => button.text() === 'OpenAI')!.trigger('click')
    await wrapper.setProps({ response: { description: '', groups: [group(2, 'gemini', 'gemini-test')] } })
    expect(wrapper.findAll('.offer-detail').map(node => node.text())).toEqual(['gemini-test'])
    expect(wrapper.findAll('nav button')[0].attributes('aria-pressed')).toBe('true')
  })

  it('filters by the vendor of each model in a composite group', async () => {
    const wrapper = catalog()
    const composite = group(1, 'composite', 'gpt-test')
    composite.models = [
      { ...composite.models[0], platform: 'openai' },
      ...group(2, 'gemini', 'gemini-test').models,
    ]
    await wrapper.setProps({ response: { description: '', groups: [composite] } })

    const vendorButtons = wrapper.findAll('nav button')
    expect(vendorButtons.map(button => button.text())).toEqual(['publicSite.models.allProviders', 'Gemini', 'OpenAI'])
    await vendorButtons[2].trigger('click')
    expect(wrapper.findAll('.offer-detail').map(node => node.text())).toEqual(['gpt-test'])
    expect(wrapper.get('tbody').text()).toContain('Group 1')
    expect(wrapper.get('.catalog-totals').findAll('dd').map(node => node.text())).toEqual(['1', '1', '1'])
  })

  it('shows load failures without displaying stale provider controls', async () => {
    const wrapper = catalog()
    await wrapper.setProps({ error: true })
    expect(wrapper.get('[role="alert"]').text()).toContain('modelPlaza.loadFailed')
    expect(wrapper.find('nav').exists()).toBe(false)
    expect(wrapper.find('.catalog-card').exists()).toBe(false)
    expect(wrapper.get('.filter-button').attributes('disabled')).toBeDefined()
  })

  it('shows one effective price per group offer and labels non-token units', async () => {
    const token: UserSupportedModelPricing = {
      billing_mode: 'token',
      input_price: 1e-6, output_price: 2e-6, cache_read_price: 0, cache_write_price: null,
      image_input_price: null, image_output_price: null, per_request_price: null,
      intervals: [],
    }
    const openAI = { ...group(1, 'openai', 'gpt-test'), rate_multiplier: 1, user_rate_multiplier: 0.5 }
    openAI.models[0].pricing = token
    const otherOpenAI = { ...group(2, 'openai', 'gpt-test'), rate_multiplier: 2 }
    otherOpenAI.models[0].pricing = token
    const media = { ...group(3, 'openai', 'gpt-image'), image_rate_independent: true, image_rate_multiplier: 0.25 }
    media.models[0].pricing = { ...token, billing_mode: 'image', per_request_price: 0.04 }
    const wrapper = catalog()
    await wrapper.setProps({ response: { description: '', groups: [openAI, otherOpenAI, media] } })

    const rows = wrapper.findAll('tbody tr')
    expect(rows).toHaveLength(3)
    expect(rows[0].text()).toContain('Group 1')
    expect(rows[0].findAll('.offer-price').map(node => node.text())).toEqual(['$0.50', '$1.00', '$0.00'])
    expect(rows[1].get('.offer-unit').text()).toContain('$0.01')
    expect(rows[1].get('.offer-unit small').text()).toBe('modelPlaza.table.perUnitImage')
    expect(rows[2].findAll('.offer-price').map(node => node.text())).toEqual(['$2.00', '$4.00', '$0.00'])
  })

  it('paginates group offers and resets to the first page after refining search', async () => {
    const wrapper = catalog()
    const largeGroup = group(1, 'openai', 'gpt-test')
    const model = largeGroup.models[0]
    largeGroup.models = Array.from({ length: 26 }, (_, i) => ({ ...model, name: `gpt-test-${i + 1}` }))
    await wrapper.setProps({ response: { description: '', groups: [largeGroup] } })
    expect(wrapper.findAll('.offer-detail')).toHaveLength(24)
    await wrapper.get('[aria-label="pagination.next"]').trigger('click')
    expect(wrapper.findAll('.offer-detail').map(node => node.text())).toEqual(['gpt-test-25', 'gpt-test-26'])
    await wrapper.get('input[type="search"]').setValue('test-1')
    expect(wrapper.findAll('.offer-detail')[0].text()).toBe('gpt-test-1')
    expect(wrapper.find('.catalog-pagination').exists()).toBe(false)
  })
})
