import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import PlazaCatalog from '../PlazaCatalog.vue'
import type { ModelPlazaGroup } from '@/api/modelPlaza'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('vue-router', () => ({ useRoute: () => ({ query: {} }) }))
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
    global: { stubs: { PlazaModelPricingTable: true, PlazaGroupSection: true, PlatformIcon: true, Icon: true } }
  })
}

describe('PlazaCatalog provider navigation', () => {
  it('filters real group models through provider chips and search, and clears empty results', async () => {
    const wrapper = catalog()
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
    expect(wrapper.findAll('.catalog-card h3').map(node => node.text())).toEqual(['gemini-test'])
    expect(wrapper.findAll('nav button')[0].attributes('aria-pressed')).toBe('true')
  })

  it('shows load failures without displaying stale provider controls', async () => {
    const wrapper = catalog()
    await wrapper.setProps({ error: true })
    expect(wrapper.get('[role="alert"]').text()).toContain('modelPlaza.loadFailed')
    expect(wrapper.find('nav').exists()).toBe(false)
    expect(wrapper.find('.catalog-card').exists()).toBe(false)
    expect(wrapper.get('.filter-button').attributes('disabled')).toBeDefined()
  })
})
