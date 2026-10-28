import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { ref } from 'vue'
import UsageCacheRate from '../UsageCacheRate.vue'
import en from '@/i18n/locales/en/dashboard'
import zh from '@/i18n/locales/zh/dashboard'

const currentLocale = ref('zh')
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: currentLocale,
    t: (key: string, params: Record<string, string> = {}) => {
      if (key === 'common.close') return 'Close'
      const messages = currentLocale.value === 'zh' ? zh : en
      const text = key.split('.').reduce<unknown>((value, part) => (value as Record<string, unknown>)?.[part], messages)
      return String(text ?? key).replace(/\{(\w+)\}/g, (_, name: string) => params[name] ?? '')
    }
  })
}))

const wrappers: ReturnType<typeof mount>[] = []
function render(row = { input_tokens: 70, cache_read_tokens: 20, cache_creation_tokens: 10 }, locale = 'zh') {
  currentLocale.value = locale
  const wrapper = mount(UsageCacheRate, {
    props: { row }, attachTo: document.body
  })
  wrappers.push(wrapper)
  return wrapper
}
afterEach(() => { wrappers.forEach(w => w.unmount()); wrappers.length = 0; document.body.innerHTML = '' })

describe('UsageCacheRate', () => {
  it('opens localized details and closes when the displayed row changes', async () => {
    const wrapper = render()
    expect(wrapper.text()).toContain('20.0%')
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(document.body.textContent).toContain('缓存明细')
    expect(document.body.textContent).toContain('总输入 Token')
    expect(document.body.textContent).toContain('100')
    await wrapper.setProps({ row: { input_tokens: 100, cache_read_tokens: 0, cache_creation_tokens: 0 } })
    await flushPromises()
    expect(wrapper.text()).toContain('0.0%')
    expect(document.querySelector('[role="dialog"]')).toBeNull()
  })
  it('shows an unavailable state without an interactive trigger', () => {
    const wrapper = render({ input_tokens: 0, cache_read_tokens: 0, cache_creation_tokens: 0 })
    expect(wrapper.text()).toBe('—')
    expect(wrapper.find('button').exists()).toBe(false)
  })
  it('provides an English accessible trigger', () => {
    const wrapper = render(undefined, 'en')
    expect(wrapper.get('button').attributes('aria-label')).toBe('View cache details, cache rate 20.0%')
  })
})
