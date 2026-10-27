import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AggregateKeyGroups from '../AggregateKeyGroups.vue'
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ cachedPublicSettings: null }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const groups = [
  { id: 1, name: 'OpenAI Basic', platform: 'openai' as const, rate_multiplier: 1 },
  { id: 2, name: 'OpenAI Pro', platform: 'openai' as const, rate_multiplier: 2 },
  { id: 3, name: 'Claude', platform: 'anthropic' as const, rate_multiplier: 3 },
]
const stubs = {
  Select: { name: 'Select', props: ['options', 'disabled'], emits: ['update:modelValue'], template: '<button :disabled="disabled">Add</button>' },
  GroupBadge: { name: 'GroupBadge', props: ['name', 'rateMultiplier', 'userRateMultiplier'], template: '<span>{{ name }} {{ userRateMultiplier ?? rateMultiplier }}x</span>' },
  Icon: true,
}
describe('aggregate group routes', () => {
  it('adds from the dropdown without duplicates and preserves route order', async () => {
    const w = mount(AggregateKeyGroups, { props: { groups, modelValue: [1, 3] }, global: { stubs } })
    const select = w.findComponent({ name: 'Select' })
    expect(select.props('options')).toEqual([{ value: 2, label: 'OpenAI Pro' }])
    await w.get('[aria-label="keys.aggregate.up Claude"]').trigger('click')
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual([[3, 1]])
    await w.setProps({ modelValue: [3, 1] })
    select.vm.$emit('update:modelValue', 2)
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual([[3, 1, 2]])
    const count = w.emitted('update:modelValue')!.length
    select.vm.$emit('update:modelValue', 1)
    expect(w.emitted('update:modelValue')).toHaveLength(count)
    await w.get('[aria-label="keys.aggregate.remove Claude"]').trigger('click')
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual([[1]])
  })
  it('filters additions by provider while preserving routes from other providers', async () => {
    const w = mount(AggregateKeyGroups, { props: { groups, modelValue: [1, 99], provider: 'anthropic', userRates: { 1: 0.5 } }, global: { stubs } })
    const select = w.findComponent({ name: 'Select' })
    expect(select.props('options')).toEqual([{ value: 3, label: 'Claude' }])
    expect(w.text()).toContain('OpenAI Basic 0.5x')
    expect(w.text()).toContain('keys.aggregate.unavailable #99')
    await w.setProps({ provider: 'openai' })
    expect(select.props('options')).toEqual([{ value: 2, label: 'OpenAI Pro' }])
    expect(w.emitted('update:modelValue')).toBeUndefined()
    await w.get('[aria-label="keys.aggregate.remove 99"]').trigger('click')
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual([[1]])
  })
  it('caps additions at twenty routes without preventing removal', async () => {
    const ids = Array.from({ length: 20 }, (_, i) => i + 3)
    const w = mount(AggregateKeyGroups, { props: { groups, modelValue: ids }, global: { stubs } })
    const select = w.findComponent({ name: 'Select' })
    expect(select.props('disabled')).toBe(true)
    select.vm.$emit('update:modelValue', 1)
    expect(w.emitted('update:modelValue')).toBeUndefined()
    await w.get('[aria-label="keys.aggregate.remove Claude"]').trigger('click')
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual([ids.slice(1)])
  })
})
