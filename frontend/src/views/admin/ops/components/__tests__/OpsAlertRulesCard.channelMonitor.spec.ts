import { enableAutoUnmount, flushPromises, shallowMount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import Select from '@/components/common/Select.vue'
import OpsAlertRulesCard from '../OpsAlertRulesCard.vue'

const mocks = vi.hoisted(() => ({ create: vi.fn(), list: vi.fn(), groups: vi.fn(), showError: vi.fn(), showSuccess: vi.fn() }))
vi.mock('vue-i18n', async importOriginal => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key }),
}))
vi.mock('@vueuse/core', () => ({ useMediaQuery: () => ref(true) }))
vi.mock('@/api', () => ({ adminAPI: { groups: { getAll: mocks.groups } } }))
vi.mock('@/api/admin/ops', () => ({ opsAPI: { listAlertRules: mocks.list, createAlertRule: mocks.create } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: mocks.showError, showSuccess: mocks.showSuccess }) }))
enableAutoUnmount(afterEach)

const metrics = [
  ['channel_monitor_success_rate', '<', 99],
  ['channel_monitor_cache_rate', '<', 50],
  ['channel_monitor_ttft_ms', '>', 10000],
  ['channel_monitor_duration_ms', '>', 180000],
] as const

async function editor() {
  const wrapper = shallowMount(OpsAlertRulesCard, {
    global: { stubs: { BaseDialog: { props: ['show'], template: '<section v-if="show"><slot /><slot name="footer" /></section>' } } },
  })
  await flushPromises()
  await wrapper.findAll('button').find(item => item.text() === 'admin.ops.alertRules.create')!.trigger('click')
  await wrapper.get('input[type="text"]').setValue('Model monitor')
  return wrapper
}
type Editor = Awaited<ReturnType<typeof editor>>
function metricSelect(wrapper: Editor) {
  return wrapper.findAllComponents(Select).find(item => (item.props('options') as Array<{ value: string }>).some(option => option.value === 'channel_monitor_success_rate'))!
}
function groupSelect(wrapper: Editor) { return wrapper.findAllComponents(Select).find(item => item.props('placeholder') === 'admin.ops.alertRules.form.groupPlaceholder')! }
function platformSelect(wrapper: Editor) { return wrapper.findAllComponents(Select).find(item => item.props('placeholder') === 'channelMonitorV2.alerts.selectPlatform')! }
function save(wrapper: Editor) { return wrapper.findAll('button').find(item => item.text() === 'common.save')! }
async function chooseMetric(wrapper: Editor, metric = 'channel_monitor_success_rate') {
  metricSelect(wrapper).vm.$emit('update:modelValue', metric)
  await flushPromises()
}
async function fillScope(wrapper: Editor) {
  platformSelect(wrapper).vm.$emit('update:modelValue', 'openai')
  groupSelect(wrapper).vm.$emit('update:modelValue', 11)
  await wrapper.get('input[maxlength="256"]').setValue('gpt-image-2.5')
  await flushPromises()
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.list.mockResolvedValue([])
  mocks.groups.mockResolvedValue([{ id: 11, name: 'Image group' }])
  mocks.create.mockResolvedValue({ id: 1 })
})

describe('OpsAlertRulesCard passive monitor rules', () => {
  it('offers the four monitor metrics in a distinct option group', async () => {
    const wrapper = await editor()
    const options = metricSelect(wrapper).props('options') as Array<{ value: string; disabled?: boolean }>
    expect(options).toContainEqual(expect.objectContaining({ value: '__group__monitor', disabled: true }))
    expect(options.filter(item => item.value.startsWith('channel_monitor_')).map(item => item.value)).toEqual(metrics.map(([metric]) => metric))
  })

  it.each(metrics)('saves %s with exact scope, explicit sample policy, and notification disabled by default', async (metric, operator, threshold) => {
    const wrapper = await editor()
    await chooseMetric(wrapper, metric)
    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect((checkboxes.at(-2)!.element as HTMLInputElement).checked).toBe(false)
    expect((checkboxes.at(-1)!.element as HTMLInputElement).checked).toBe(false)
    await fillScope(wrapper)
    await wrapper.get('input[max="1000000"]').setValue('50')
    await wrapper.findAll('input[max="60"]')[0].setValue('4')
    await wrapper.findAll('input[max="60"]')[1].setValue('3')
    await wrapper.findAll('label').find(item => item.text() === 'channelMonitorV2.alerts.notifyRecovery')!.get('input').setValue(true)
    await save(wrapper).trigger('click')
    await flushPromises()
    expect(mocks.create).toHaveBeenCalledWith(expect.objectContaining({
      name: 'Model monitor', metric_type: metric, operator, threshold, enabled: false, notify_email: false, window_minutes: 5,
      filters: { platform: 'openai', model: 'gpt-image-2.5', group_id: 11, minimum_sample: 50, consecutive_windows: 4, recovery_windows: 3, notify_recovery: true },
    }))
    expect(mocks.showError).not.toHaveBeenCalled()
  })

  it.each(['platform', 'model', 'group'])('prevents saving when the exact %s scope is missing', async missing => {
    const wrapper = await editor()
    await chooseMetric(wrapper)
    await fillScope(wrapper)
    if (missing === 'platform') platformSelect(wrapper).vm.$emit('update:modelValue', '')
    if (missing === 'model') await wrapper.get('input[maxlength="256"]').setValue('')
    if (missing === 'group') groupSelect(wrapper).vm.$emit('update:modelValue', null)
    await flushPromises()
    await save(wrapper).trigger('click')
    expect(mocks.create).not.toHaveBeenCalled()
    expect(mocks.showError).toHaveBeenCalledOnce()
    expect(wrapper.text()).toContain(missing === 'group' ? 'validation.groupIdRequired' : 'alerts.scopeRequired')
  })

  it.each(['0', '1.5', '1000001'])('rejects invalid sample requirements (%s)', async minimum => {
    const wrapper = await editor()
    await chooseMetric(wrapper)
    await fillScope(wrapper)
    await wrapper.get('input[max="1000000"]').setValue(minimum)
    await save(wrapper).trigger('click')
    expect(mocks.create).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('alerts.invalidCounts')
  })

  it('rejects a percentage threshold over 100 before sending the rule', async () => {
    const wrapper = await editor()
    await chooseMetric(wrapper, 'channel_monitor_cache_rate')
    await fillScope(wrapper)
    await wrapper.get('input[type="number"]:not([min])').setValue('101')
    await save(wrapper).trigger('click')
    expect(mocks.create).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('alerts.invalidThreshold')
  })
})
