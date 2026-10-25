import { defineComponent, h, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { useModalVisibility } from '../useModalVisibility'
describe('lazy modal initialization', () => {
  it('initializes an already-open mount after setup and follows later visibility', async () => {
    const shown = ref(true), callback = vi.fn()
    const fixture = defineComponent({ setup() {
      useModalVisibility(() => shown.value, () => callback(initialized))
      const initialized = true
      return () => h('div')
    } })
    const wrapper = mount(fixture)
    expect(callback).toHaveBeenCalledTimes(1); expect(callback).toHaveBeenCalledWith(true)
    shown.value = false; await wrapper.vm.$nextTick(); expect(callback).toHaveBeenCalledTimes(2)
    wrapper.unmount(); shown.value = true; await Promise.resolve(); expect(callback).toHaveBeenCalledTimes(2)
  })
  it('does not initialize a hidden modal', () => {
    const callback = vi.fn()
    const wrapper = mount(defineComponent({ setup() { useModalVisibility(() => false, callback); return () => h('div') } }))
    expect(callback).not.toHaveBeenCalled(); wrapper.unmount()
  })
})
