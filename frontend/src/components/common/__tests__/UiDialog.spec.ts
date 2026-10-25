import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import UiDialog from '../UiDialog.vue'
import Toggle from '../Toggle.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const wrappers: VueWrapper[] = []
afterEach(() => { wrappers.forEach(w => w.unmount()); wrappers.length = 0; document.body.innerHTML = '' })

describe('Shared shadcn interactions', () => {
  it('forwards form attributes and values through Input', async () => {
    const wrapper = mount(Input, { props: { modelValue: '' }, attrs: { type: 'email', required: true, autocomplete: 'email' } })
    wrappers.push(wrapper)
    await wrapper.get('input').setValue('user@example.test')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['user@example.test'])
    expect(wrapper.get('input').attributes('autocomplete')).toBe('email')
    expect(wrapper.get('input').attributes('required')).toBeDefined()
  })

  it('keeps links semantic when Button uses as-child', () => {
    const wrapper = mount(Button, { props: { asChild: true }, slots: { default: '<a href="/home">Home</a>' } })
    wrappers.push(wrapper)
    expect(wrapper.element.tagName).toBe('A')
    expect(wrapper.attributes('href')).toBe('/home')
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('forwards the disabled state and updates accessible switch state', async () => {
    const wrapper = mount(Toggle, { props: { modelValue: false }, attrs: { disabled: true, 'aria-label': 'Enable feature' } })
    wrappers.push(wrapper)
    await wrapper.get('[role="switch"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
    await wrapper.setProps({ disabled: false })
    await wrapper.get('[role="switch"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([true])
  })

  it('blocks Escape while saving, then dismisses and returns focus', async () => {
    const trigger = document.createElement('button')
    document.body.appendChild(trigger)
    trigger.focus()
    const wrapper = mount(UiDialog, { attachTo: document.body, props: { show: true, title: 'Edit node', closeOnEscape: false }, slots: { default: '<input aria-label="Reason" />' } })
    wrappers.push(wrapper)
    await flushPromises()
    const dialog = document.querySelector('[role="dialog"]')!
    expect(document.getElementById(dialog.getAttribute('aria-labelledby')!)?.textContent).toBe('Edit node')
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }))
    expect(wrapper.emitted('close')).toBeUndefined()
    await wrapper.setProps({ closeOnEscape: true })
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }))
    expect(wrapper.emitted('close')).toHaveLength(1)
    await wrapper.setProps({ show: false })
    await vi.waitFor(() => expect(document.activeElement).toBe(trigger))
  })
})
