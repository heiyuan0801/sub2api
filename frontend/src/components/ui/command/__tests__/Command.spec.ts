import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import en from '@/i18n/locales/en/design'
import Command from '../Command.vue'
import type { CommandGroup } from '../types'

const groups: CommandGroup[] = [
  { id: 'pages', label: 'Pages', items: [{ id: 'cafe', label: 'Café', keywords: ['usage'] }] },
  { id: 'actions', label: 'Actions', items: [{ id: 'disabled', label: 'Restart', disabled: true }] }
]

function renderCommand() {
  return mount(Command, {
    attachTo: document.body,
    props: { groups },
    global: { plugins: [createI18n({ legacy: false, locale: 'en', messages: { en } })] }
  })
}

const originalScrollIntoView = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'scrollIntoView')

beforeEach(() => {
  Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', { configurable: true, value: vi.fn() })
})

afterEach(() => {
  if (originalScrollIntoView) {
    Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', originalScrollIntoView)
  } else {
    Reflect.deleteProperty(HTMLElement.prototype, 'scrollIntoView')
  }
})

describe('Command', () => {
  it('matches accents and keywords and renders an accessible empty state', async () => {
    const wrapper = renderCommand()
    try {
      const input = wrapper.get('input[role="combobox"]')
      await input.setValue('cafe')
      expect(wrapper.findAll('[role="option"]').map(item => item.text())).toEqual(['Café'])
      expect(wrapper.text()).not.toContain('Actions')
      await input.setValue('usage')
      expect(wrapper.get('[role="option"]').text()).toBe('Café')
      await input.setValue('unmatched')
      expect(wrapper.findAll('[role="option"]')).toHaveLength(0)
      expect(wrapper.get('[role="status"]').text()).toBe('No matching commands.')
    } finally {
      wrapper.unmount()
    }
  })

  it('activates an enabled command with the keyboard and ignores disabled commands', async () => {
    const wrapper = renderCommand()
    try {
      const input = wrapper.get('input[role="combobox"]')
      await wrapper.get('[role="option"][aria-disabled="true"]').trigger('click')
      expect(wrapper.emitted('select')).toBeUndefined()
      input.element.focus()
      await input.setValue('usage')
      await flushPromises()
      await input.trigger('keydown', { key: 'Enter' })
      expect(wrapper.emitted('select')).toEqual([[groups[0].items[0]]])
    } finally {
      wrapper.unmount()
    }
  })
})
