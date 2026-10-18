import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type {
  HealthState,
  MonitorConfig,
  MonitorCoverage,
  MonitorHealth,
  MonitorMatrixRow,
  MonitorMetric,
  MonitorSnapshot,
} from '@/api/channelMonitorV2'

const mocks = vi.hoisted(() => ({
  getDimensions: vi.fn(),
  getSnapshot: vi.fn(),
  getMatrix: vi.fn(),
  getModels: vi.fn(),
  getErrors: vi.fn(),
  getUsers: vi.fn(),
  replace: vi.fn(),
  showError: vi.fn(),
  auth: { isAdmin: false, user: { id: 42 } },
  route: { query: {} as Record<string, string> },
}))

vi.mock('@/api/channelMonitorV2', () => ({
  getDimensions: mocks.getDimensions,
  getSnapshot: mocks.getSnapshot,
  getMatrix: mocks.getMatrix,
  getModels: mocks.getModels,
  getErrors: mocks.getErrors,
  getUsers: mocks.getUsers,
}))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => mocks.auth }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: mocks.showError }) }))
vi.mock('vue-router', () => ({ useRoute: () => mocks.route, useRouter: () => ({ replace: mocks.replace }) }))
vi.mock('@/utils/featureFlags', () => ({
  isChannelMonitorThroughputHidden: () => false,
  isChannelMonitorUserRankingHidden: () => false,
}))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({
    locale: { value: 'en-US' },
    te: () => false,
    t: (key: string, params?: Record<string, unknown>) => `${key}${params ? ` ${Object.entries(params).map(([name, value]) => `${name}=${value}`).join(' ')}` : ''}`,
  }),
}))

import ChannelStatusV2View from '../ChannelStatusV2View.vue'
import FilterMultiSelect from '@/features/channel-monitor-v2/FilterMultiSelect.vue'

const coverage: MonitorCoverage = {
  requested_start: '2026-10-16T15:00:00Z',
  requested_end: '2026-10-17T15:00:00Z',
  coverage_start: '2026-10-16T15:00:00Z',
  data_through: '2026-10-17T14:30:00Z',
  computed_at: '2026-10-17T14:30:00Z',
  aggregation_lag_seconds: 0,
  coverage_complete: true,
  bucket_seconds: 3600,
}
const config: MonitorConfig = {
  version: 1,
  enabled: true,
  refresh_interval_seconds: 300,
  platforms: [],
  group_ids: [],
  health_thresholds: {
    minimum_sample: 20,
    warning_error_rate: 0.1,
    critical_error_rate: 0.3,
    target_ttft_ms: 1000,
    warning_ttft_ms: 3000,
    critical_ttft_ms: 5000,
    warning_cache_rate: 0.5,
    critical_cache_rate: 0.2,
    error_weight: 0.5,
    ttft_weight: 0.3,
    cache_weight: 0.2,
  },
}
function metric(patch: Partial<MonitorMetric> = {}): MonitorMetric {
  return {
    has_data: true,
    cache_rate_available: true,
    success_rate: 0.8,
    // Public responses may redact counts while keeping these real metrics.
    success_requests: 0,
    error_requests: 0,
    request_count: 0,
    token_count: 0,
    rpm: 1,
    tpm: 120,
    error_rate: 0.05,
    cache_rate: 0.6,
    cache_rate_numerator: 0,
    cache_rate_denominator: 0,
    ttft: { sample_count: 0, p50_ms: 1200, avg_ms: 1300, p95_ms: 2000 },
    duration: { sample_count: 0, p50_ms: 2100, avg_ms: 2500, p95_ms: 3100 },
    ...patch,
  }
}
function health(overall: HealthState): MonitorHealth {
  return { overall, error_rate: overall, ttft: overall, cache: overall, minimum_sample: 20 }
}
function row(model: string, groupId: number, state: HealthState, patch: Partial<MonitorMetric> = {}): MonitorMatrixRow {
  return {
    platform: model.startsWith('claude') ? 'anthropic' : 'openai',
    model,
    group_id: groupId,
    group_name: groupId === 1 ? 'default' : 'vip',
    metrics: metric(patch),
    health: health(state),
    buckets: [],
  }
}
const rows = [
  row('gpt-alpha', 1, 'healthy'),
  row('gpt-warning', 2, 'warning'),
  row('claude-critical', 3, 'critical'),
  row('gpt-empty', 4, 'critical', { has_data: false }),
  row('gpt-sparse', 5, 'unknown'),
]
const severityOrder = [rows[2], rows[1], rows[4], rows[0], rows[3]]
function snapshot(): MonitorSnapshot {
  return { config, coverage, metrics: metric(), health: health('healthy'), trend: [] }
}

const CardsStub = defineComponent({
  name: 'MonitorChannelCards',
  props: ['rows', 'coverage', 'loading', 'healthMode', 'showThroughput', 'emptyDescription', 'favoriteKeys', 'stale'],
  emits: ['toggle-favorite', 'inspect', 'select-group'],
  setup: () => () => h('div', { 'data-testid': 'model-cards' }),
})
const IncidentStub = defineComponent({
  name: 'MonitorIncidentDialog',
  props: ['show', 'row', 'coverage', 'window', 'metrics', 'health', 'errors', 'loading', 'error', 'isAdmin', 'stale'],
  emits: ['close', 'retry'],
  setup: props => () => props.show ? h('div', { 'data-testid': 'incident-dialog' }) : null,
})
const SelectStub = defineComponent({
  name: 'MonitorSelectStub',
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () => h('select', {
      value: props.modelValue,
      onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
    }, (props.options as Array<{ value: string; label: string }>).map(option => h('option', { value: option.value }, option.label)))
  },
})
const MetricStub = defineComponent({
  name: 'MetricCell',
  props: ['label', 'value', 'detail', 'state'],
  setup: props => () => h('div', { 'data-metric': props.label }, String(props.value)),
})
function render() {
  return mount(ChannelStatusV2View, {
    attachTo: document.body,
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        Icon: true,
        LoadingSpinner: true,
        FilterMultiSelect: true,
        Select: SelectStub,
        MetricCell: MetricStub,
        MonitorChannelCards: CardsStub,
        MonitorIncidentDialog: IncidentStub,
        MonitorTrendChart: true,
        MonitorRankBadge: true,
        RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
      },
    },
  })
}
let wrapper: ReturnType<typeof render> | undefined
const displayedRows = () => wrapper!.getComponent(CardsStub).props('rows') as MonitorMatrixRow[]
const dimensionPicker = (dimension: 'group' | 'model' | 'platform') => wrapper!.findAllComponents(FilterMultiSelect)
  .find(item => item.props('label') === `channelMonitorV2.filters.${dimension}`)!

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date', 'setInterval', 'clearInterval'] })
  vi.setSystemTime(new Date('2026-10-17T14:31:00Z'))
  vi.clearAllMocks()
  localStorage.clear()
  mocks.auth.isAdmin = false
  mocks.auth.user.id = 42
  mocks.route.query = {}
  mocks.getDimensions.mockReset().mockResolvedValue({ platforms: [], groups: [], models: [] })
  mocks.getSnapshot.mockReset().mockResolvedValue(snapshot())
  mocks.getMatrix.mockReset().mockResolvedValue({ coverage, group_by: 'platform_group_model', items: rows })
  mocks.getModels.mockReset().mockResolvedValue({ coverage, items: [] })
  mocks.getErrors.mockReset().mockResolvedValue({ coverage, items: [] })
  mocks.getUsers.mockReset().mockResolvedValue({ coverage, items: [] })
  mocks.replace.mockResolvedValue(undefined)
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
  vi.useRealTimers()
})

describe('ChannelStatusV2View model availability', () => {
  function platformCatalog() {
    return {
      platforms: [
        { value: 'gemini', label: 'Gemini', request_count: 0 },
        { value: 'deepseek', label: 'DeepSeek', request_count: 0 },
        { value: 'grok', label: 'Grok', request_count: 0 },
        { value: 'anthropic', label: 'Claude', request_count: 1 },
        { value: 'openai', label: 'OpenAI', request_count: 1 },
      ],
      groups: [
        { id: 1, name: 'default', platform: 'openai', request_count: 1 },
        { id: 2, name: 'vip', platform: 'openai', request_count: 0 },
        { id: 3, name: 'Claude group', platform: 'anthropic', request_count: 1 },
      ],
      models: [
        { value: 'gpt-alpha', label: 'gpt-alpha', platform: 'openai', group_ids: [1], request_count: 1 },
        { value: 'gpt-image-new', label: 'gpt-image-new', platform: 'openai', group_ids: [2], request_count: 0 },
        { value: 'claude-critical', label: 'claude-critical', platform: 'anthropic', group_ids: [3], request_count: 1 },
      ],
    }
  }

  it('waits for the initial catalog before enabling platform switching without aborting its request', async () => {
    let resolveCatalog!: (value: ReturnType<typeof platformCatalog>) => void
    mocks.getDimensions.mockImplementationOnce(() => new Promise(resolve => { resolveCatalog = resolve }))
    wrapper = render()
    await flushPromises()
    const signal = mocks.getDimensions.mock.calls[0][2] as AbortSignal
    expect(wrapper.get('[data-testid="platform-all"]').attributes('disabled')).toBeDefined()
    await wrapper.get('[data-testid="platform-all"]').trigger('click')
    expect(signal.aborted).toBe(false)
    expect(mocks.getSnapshot).toHaveBeenCalledTimes(1)
    resolveCatalog(platformCatalog())
    await flushPromises()
    expect(wrapper.get('[data-testid="platform-all"]').attributes('disabled')).toBeUndefined()
    expect(wrapper.find('[data-testid="platform-openai"]').exists()).toBe(true)
  })

  it('normalizes raw production platform labels while preserving a custom display label', async () => {
    const catalog = platformCatalog()
    mocks.getDimensions.mockResolvedValue({
      ...catalog,
      platforms: [
        ...catalog.platforms.map(platform => ({ ...platform, label: platform.value })),
        { value: 'custom-provider', label: 'Custom Provider', request_count: 0 },
      ],
    })
    wrapper = render()
    await flushPromises()
    expect(wrapper.get('[data-testid="platform-openai"]').text()).toBe('OpenAI')
    expect(wrapper.get('[data-testid="platform-anthropic"]').text()).toBe('Claude')
    expect(wrapper.get('[data-testid="platform-grok"]').text()).toBe('Grok')
    expect(wrapper.get('[data-testid="platform-deepseek"]').text()).toBe('DeepSeek')
    expect(wrapper.get('[data-testid="platform-custom-provider"]').text()).toBe('Custom Provider')
    expect(dimensionPicker('platform').props('options')).toContainEqual({ value: 'anthropic', label: 'Claude' })
  })

  it('dynamically offers no-traffic Gemini, Kimi and Zhipu groups with their account-supported models', async () => {
    const providers = [
      { value: 'kimi', label: 'Kimi', groupId: 80, group: 'Kimi account group', models: ['kimi-preview-chat', 'kimi-preview-reasoner'] },
      { value: 'zhipu', label: 'Zhipu GLM', groupId: 81, group: 'GLM account group', models: ['glm-preview-chat', 'glm-image-preview'] },
      { value: 'gemini', label: 'Gemini', groupId: 82, group: 'Gemini account group', models: ['gemini-preview-pro', 'gemini-image-preview'] },
    ]
    const catalog = platformCatalog()
    mocks.getDimensions.mockResolvedValue({
      platforms: [...catalog.platforms.filter(item => item.value !== 'gemini'), ...providers.map(provider => ({ value: provider.value, label: provider.value, request_count: 0 }))],
      groups: [...catalog.groups, ...providers.map(provider => ({ id: provider.groupId, name: provider.group, platform: provider.value, platforms: [provider.value], request_count: 0 }))],
      models: [...catalog.models, ...providers.flatMap(provider => provider.models.map(model => ({ value: model, label: model, platform: provider.value, group_ids: [provider.groupId], request_count: 0 })))],
    })
    mocks.getMatrix.mockImplementation(async (filter) => ({ coverage, group_by: 'platform_group_model', items: providers
      .filter(provider => !filter.platforms.length || filter.platforms.includes(provider.value))
      .flatMap(provider => provider.models.map(model => ({ ...row(model, provider.groupId, 'unknown', { has_data: false }), platform: provider.value, group_name: provider.group }))) }))
    wrapper = render()
    await flushPromises()

    for (const provider of providers) {
      const shortcut = wrapper.get(`[data-testid="platform-${provider.value}"]`)
      expect(shortcut.text()).toBe(provider.label)
      await shortcut.trigger('click')
      await flushPromises()
      expect(shortcut.attributes('aria-pressed')).toBe('true')
      expect(dimensionPicker('group').props('options')).toEqual([{ value: String(provider.groupId), label: `${provider.value} / ${provider.group}` }])
      expect(dimensionPicker('model').props('options')).toEqual(provider.models.map(model => ({ value: model, label: `${provider.value} / ${model}` })))
      expect(displayedRows()).toHaveLength(2)
      expect(displayedRows().every(item => item.platform === provider.value && item.metrics.has_data === false && item.health.overall === 'unknown')).toBe(true)
      expect(mocks.getSnapshot).toHaveBeenLastCalledWith({ range: '24h', platforms: [provider.value], groupIds: [], models: [] }, false, expect.any(AbortSignal))
    }
    expect(mocks.getDimensions).toHaveBeenCalledTimes(1)
  })

  it('shows only the scoped dimensions and never adds a platform from an unselected group or global config', async () => {
    mocks.getSnapshot.mockResolvedValue({ ...snapshot(), config: { ...config, group_ids: [1], platforms: [
      { platform: 'openai', enabled: true, models: [] },
      { platform: 'kimi', enabled: true, models: [] },
      { platform: 'zhipu', enabled: true, models: [] },
    ] } })
    mocks.getDimensions.mockResolvedValue({
      platforms: [{ value: 'openai', label: 'openai', request_count: 0 }],
      groups: [{ id: 1, name: 'Selected group', platform: 'openai', platforms: ['openai'], request_count: 0 }],
      models: [{ value: 'gpt-selected', label: 'gpt-selected', platform: 'openai', group_ids: [1], request_count: 0 }],
    })
    wrapper = render()
    await flushPromises()
    expect(wrapper.find('[data-testid="platform-openai"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="platform-kimi"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="platform-zhipu"]').exists()).toBe(false)
    expect(dimensionPicker('group').props('options')).toEqual([{ value: '1', label: 'openai / Selected group' }])
    expect(dimensionPicker('model').props('options')).toEqual([{ value: 'gpt-selected', label: 'openai / gpt-selected' }])
  })

  it('shows enabled platform shortcuts from the catalog and switches scope without stale group/model selections', async () => {
    mocks.getDimensions.mockResolvedValue(platformCatalog())
    mocks.route.query = { platform: 'openai', group: '1', model: 'gpt-alpha', q: 'search', status: 'healthy', sort: 'name' }
    wrapper = render()
    await flushPromises()

    expect(wrapper.findAll('.availability-platforms button').map(item => item.text())).toEqual([
      'channelMonitorV2.filters.allPlatforms', 'OpenAI', 'Claude', 'Grok', 'DeepSeek', 'Gemini',
    ])
    expect(wrapper.get('[data-testid="platform-openai"]').attributes('aria-pressed')).toBe('true')
    mocks.getSnapshot.mockClear()
    mocks.getMatrix.mockClear()
    await wrapper.get('[data-testid="platform-anthropic"]').trigger('click')
    await flushPromises()

    const scoped = { range: '24h', platforms: ['anthropic'], groupIds: [], models: [] }
    expect(mocks.getSnapshot).toHaveBeenCalledTimes(1)
    expect(mocks.getSnapshot).toHaveBeenCalledWith(scoped, false, expect.any(AbortSignal))
    expect(mocks.getMatrix).toHaveBeenCalledTimes(1)
    expect(mocks.getMatrix).toHaveBeenCalledWith(scoped, 'platform_group_model', false, expect.any(AbortSignal))
    expect(dimensionPicker('group').props('options')).toEqual([{ value: '3', label: 'anthropic / Claude group' }])
    expect(dimensionPicker('model').props('options')).toEqual([{ value: 'claude-critical', label: 'anthropic / claude-critical' }])
    expect(wrapper.get('[data-testid="platform-anthropic"]').attributes('aria-pressed')).toBe('true')
    expect(mocks.replace).toHaveBeenLastCalledWith(expect.objectContaining({ query: expect.objectContaining({ platform: 'anthropic', group: undefined, model: undefined, q: 'search', status: 'healthy', sort: 'name' }) }))
    expect(JSON.parse(localStorage.getItem('channel-monitor-v2:v1:42')!)).toMatchObject({ platform: ['anthropic'], group: [], model: [] })
    expect(mocks.getDimensions).toHaveBeenCalledTimes(1)

    // Enabled providers without any calls remain selectable, and incompatible
    // old dimensions are cleared even when the new provider has no options.
    await wrapper.get('[data-testid="platform-deepseek"]').trigger('click')
    await flushPromises()
    expect(dimensionPicker('group').props('options')).toEqual([])
    expect(dimensionPicker('model').props('options')).toEqual([])
    expect(mocks.getSnapshot).toHaveBeenLastCalledWith({ ...scoped, platforms: ['deepseek'] }, false, expect.any(AbortSignal))
    await wrapper.get('[data-testid="platform-all"]').trigger('click')
    await flushPromises()
    expect(dimensionPicker('group').props('options')).toHaveLength(3)
    expect(dimensionPicker('model').props('options')).toHaveLength(3)
    expect(wrapper.get('[data-testid="platform-all"]').attributes('aria-pressed')).toBe('true')
  })

  it('cascades model choices by selected groups and clears a model outside the next group', async () => {
    mocks.getDimensions.mockResolvedValue(platformCatalog())
    mocks.route.query = { platform: 'openai', model: 'gpt-alpha' }
    wrapper = render()
    await flushPromises()
    expect(dimensionPicker('model').props('options')).toHaveLength(2)

    dimensionPicker('group').vm.$emit('update:modelValue', ['2'])
    await flushPromises()
    expect(dimensionPicker('model').props('options')).toEqual([{ value: 'gpt-image-new', label: 'openai / gpt-image-new' }])
    expect(mocks.getSnapshot).toHaveBeenLastCalledWith({ range: '24h', platforms: ['openai'], groupIds: [2], models: [] }, false, expect.any(AbortSignal))
    expect(mocks.replace).toHaveBeenLastCalledWith(expect.objectContaining({ query: expect.objectContaining({ group: '2', model: undefined }) }))
  })

  it('keeps compatible dimensions on platform selection and supports older catalogs without model group links', async () => {
    const catalog = platformCatalog()
    mocks.getDimensions.mockResolvedValue({ ...catalog, models: catalog.models.map(model => ({ value: model.value, label: model.label, platform: model.platform, request_count: model.request_count })) })
    mocks.route.query = { group: '1', model: 'gpt-alpha' }
    wrapper = render()
    await flushPromises()

    await wrapper.get('[data-testid="platform-openai"]').trigger('click')
    await flushPromises()
    expect(mocks.getSnapshot).toHaveBeenLastCalledWith({ range: '24h', platforms: ['openai'], groupIds: [1], models: ['gpt-alpha'] }, false, expect.any(AbortSignal))
    expect(dimensionPicker('model').props('options')).toHaveLength(2)
    dimensionPicker('platform').vm.$emit('update:modelValue', ['anthropic', 'grok'])
    await flushPromises()
    expect(mocks.getSnapshot).toHaveBeenLastCalledWith({ range: '24h', platforms: ['anthropic', 'grok'], groupIds: [], models: [] }, false, expect.any(AbortSignal))
    expect(dimensionPicker('model').props('options')).toHaveLength(1)
  })

  it('includes a composite group under every platform advertised by its catalog models', async () => {
    const catalog = platformCatalog()
    mocks.getDimensions.mockResolvedValue({
      ...catalog,
      groups: [...catalog.groups, { id: 9, name: 'Mixed group', platform: 'composite', platforms: ['openai', 'anthropic'], request_count: 0 }],
      models: catalog.models.map(model => ({ ...model, group_ids: [...model.group_ids, 9] })),
    })
    mocks.route.query = { platform: 'openai', group: '9', model: 'gpt-alpha' }
    wrapper = render()
    await flushPromises()
    expect(dimensionPicker('group').props('options')).toContainEqual({ value: '9', label: 'composite / Mixed group' })
    await wrapper.get('[data-testid="platform-anthropic"]').trigger('click')
    await flushPromises()
    expect(dimensionPicker('group').props('options')).toContainEqual({ value: '9', label: 'composite / Mixed group' })
    expect(dimensionPicker('model').props('options')).toEqual([{ value: 'claude-critical', label: 'anthropic / claude-critical' }])
    expect(mocks.getSnapshot).toHaveBeenLastCalledWith({ range: '24h', platforms: ['anthropic'], groupIds: [9], models: [] }, false, expect.any(AbortSignal))
    await wrapper.get('[data-testid="platform-grok"]').trigger('click')
    await flushPromises()
    expect(dimensionPicker('group').props('options')).toEqual([])
    expect(mocks.getSnapshot).toHaveBeenLastCalledWith({ range: '24h', platforms: ['grok'], groupIds: [], models: [] }, false, expect.any(AbortSignal))
  })

  it('loads a 24-hour model/group view using the authenticated user API', async () => {
    wrapper = render()
    await flushPromises()

    const filter = { range: '24h', platforms: [], groupIds: [], models: [] }
    expect(mocks.getDimensions).toHaveBeenCalledWith(filter, false, expect.any(AbortSignal))
    expect(mocks.getSnapshot).toHaveBeenCalledWith(filter, false, expect.any(AbortSignal))
    expect(mocks.getMatrix).toHaveBeenCalledWith(filter, 'platform_group_model', false, expect.any(AbortSignal))
    expect(mocks.getModels).toHaveBeenCalledWith(filter, false, expect.any(AbortSignal))
    expect(wrapper.get('[aria-pressed="true"]').text()).toContain('availability.range24h')
    expect(displayedRows()).toEqual(severityOrder)
    expect(wrapper.find('a[href="/admin/channels/monitor"]').exists()).toBe(false)
  })

  it.each(['form', 'button'] as const)('applies case-insensitive model search through the %s submission', async (submission) => {
    wrapper = render()
    await flushPromises()
    await wrapper.get('input[type="search"]').setValue(' CLAUDE ')
    expect(displayedRows()).toHaveLength(5)

    // Enter submits the search form natively; jsdom needs that submit event explicitly.
    if (submission === 'form') await wrapper.get('form[role="search"]').trigger('submit')
    else await wrapper.get('form[role="search"] button[type="submit"]').trigger('click')
    await flushPromises()

    expect(displayedRows().map(item => item.model)).toEqual(['claude-critical'])
    expect(mocks.replace).toHaveBeenLastCalledWith(expect.objectContaining({ query: expect.objectContaining({ q: 'CLAUDE' }) }))
    expect(mocks.getMatrix).toHaveBeenCalledTimes(1)
    await wrapper.get('input[type="search"]').setValue('')
    expect(displayedRows()).toHaveLength(5)
  })

  it('counts missing and insufficient samples separately and filters by actual status', async () => {
    wrapper = render()
    await flushPromises()
    expect(wrapper.get('[data-testid="summary-total"]').text()).toMatch(/5$/)
    for (const status of ['healthy', 'warning', 'critical']) {
      expect(wrapper.get(`[data-testid="summary-${status}"]`).text()).toMatch(/1$/)
    }
    expect(wrapper.text()).toContain('pendingSummary empty=1 insufficient=1')

    const status = wrapper.get('select[aria-label="channelMonitorV2.availability.filterStatus"]')
    await status.setValue('critical')
    expect(displayedRows().map(item => item.model)).toEqual(['claude-critical'])
    // Filtering must not turn the absent sample into a failed model or collapse the summary.
    expect(wrapper.get('[data-testid="summary-total"]').text()).toMatch(/5$/)
    await status.setValue('noData')
    expect(displayedRows().map(item => item.model)).toEqual(['gpt-empty'])
    await status.setValue('insufficient')
    expect(displayedRows().map(item => item.model)).toEqual(['gpt-sparse'])
  })

  it('includes image models by default and filters by backend model type without name guessing', async () => {
    const image = { ...row('gpt-image-2.5-flare', 20, 'healthy'), is_image_model: true }
    const text = { ...row('image-analysis-chat', 21, 'healthy'), is_image_model: false }
    mocks.getMatrix.mockResolvedValue({ coverage, group_by: 'platform_group_model', items: [image, text] })
    wrapper = render()
    await flushPromises()

    expect(displayedRows()).toEqual([image, text])
    const type = wrapper.get('select[aria-label="channelMonitorV2.availability.modelType"]')
    await type.setValue('image')
    expect(displayedRows()).toEqual([image])
    expect(wrapper.get('[data-testid="summary-total"]').text()).toMatch(/1$/)
    expect(mocks.replace).toHaveBeenLastCalledWith(expect.objectContaining({ query: expect.objectContaining({ model_type: 'image' }) }))
    await type.setValue('text')
    expect(displayedRows()).toEqual([text])
    await type.setValue('all')
    expect(displayedRows()).toEqual([image, text])
  })

  it('retains the last snapshot while a background refresh runs and shows a failure notice', async () => {
    wrapper = render()
    await flushPromises()
    let rejectRefresh!: (error: unknown) => void
    mocks.getSnapshot.mockImplementationOnce(() => new Promise((_resolve, reject) => { rejectRefresh = reject }))

    await vi.advanceTimersByTimeAsync(300000)
    await flushPromises()
    expect(mocks.getSnapshot).toHaveBeenCalledTimes(2)
    expect(wrapper.get('.availability-results').attributes('aria-busy')).toBe('true')
    expect(displayedRows()).toEqual(severityOrder)
    expect(wrapper.get('[data-metric="channelMonitorV2.metrics.successRate"]').text()).toBe('80.0%')

    rejectRefresh({ message: 'Snapshot unavailable' })
    await flushPromises()
    expect(displayedRows()).toEqual(severityOrder)
    expect(wrapper.get('[role="alert"]').text()).toContain('Snapshot unavailable')
    expect(wrapper.get('[role="alert"]').text()).toContain('availability.stale')
    expect(wrapper.get('.availability-results').attributes('aria-busy')).toBe('false')

    await wrapper.get('[role="alert"] button').trigger('click')
    await flushPromises()
    expect(mocks.getSnapshot).toHaveBeenCalledTimes(3)
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('clears stale model data and explains when the backend disables monitoring', async () => {
    wrapper = render()
    await flushPromises()
    mocks.getSnapshot.mockRejectedValueOnce({ reason: 'CHANNEL_MONITOR_DISABLED', message: 'Feature disabled' })
    await wrapper.get('.availability-header .availability-action').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="model-cards"]').exists()).toBe(false)
    expect(wrapper.find('[data-metric="channelMonitorV2.metrics.successRate"]').exists()).toBe(false)
    expect(wrapper.get('[role="alert"]').text()).toContain('availability.disabled')
    expect(wrapper.text()).toContain('availability.disabledHint')
    expect(wrapper.get('[data-testid="summary-total"]').text()).toMatch(/0$/)
    expect(wrapper.get('[role="alert"]').text()).not.toContain('availability.stale')
  })

  it('uses explicit successful-call rate instead of the health error rate complement', async () => {
    mocks.getSnapshot.mockResolvedValue({ ...snapshot(), metrics: metric({ success_rate: 0.8, error_rate: 0.05, cache_rate_available: false }) })
    wrapper = render()
    await flushPromises()

    expect(wrapper.get('[data-metric="channelMonitorV2.metrics.successRate"]').text()).toBe('80.0%')
    expect(wrapper.get('[data-metric="channelMonitorV2.metrics.cacheRate"]').text()).toBe('—')
  })

  it('offers the existing monitor settings entry to admins and requests admin statistics', async () => {
    mocks.auth.isAdmin = true
    wrapper = render()
    await flushPromises()

    expect(wrapper.get('a[href="/admin/channels/monitor"]').text()).toContain('availability.settings')
    expect(mocks.getSnapshot).toHaveBeenCalledWith(expect.any(Object), true, expect.any(AbortSignal))
    expect(mocks.getMatrix).toHaveBeenCalledWith(expect.any(Object), 'platform_group_model', true, expect.any(AbortSignal))
  })

  it('persists favorites and display filters per authenticated user and restores them after remount', async () => {
    wrapper = render()
    await flushPromises()
    wrapper.getComponent(CardsStub).vm.$emit('toggle-favorite', rows[0])
    await wrapper.get('input[type="search"]').setValue('gpt')
    await wrapper.get('form[role="search"]').trigger('submit')
    await wrapper.get('select[aria-label="channelMonitorV2.preferences.sortLabel"]').setValue('name')
    await wrapper.findAll('button').find(item => item.text() === 'channelMonitorV2.preferences.favoritesOnly')!.trigger('click')
    await flushPromises()

    const favorite = JSON.stringify(['openai', 'gpt-alpha', 1])
    expect(JSON.parse(localStorage.getItem('channel-monitor-v2:v1:42')!)).toMatchObject({ q: 'gpt', sort: 'name', favorites_only: true, favorites: [favorite] })
    expect(displayedRows()).toEqual([rows[0]])
    wrapper.unmount()
    wrapper = render()
    await flushPromises()
    expect(wrapper.get('input[type="search"]').element).toHaveProperty('value', 'gpt')
    expect(wrapper.get('select[aria-label="channelMonitorV2.preferences.sortLabel"]').element).toHaveProperty('value', 'name')
    expect(wrapper.getComponent(CardsStub).props('favoriteKeys')).toEqual([favorite])
    expect(displayedRows()).toEqual([rows[0]])

    wrapper.unmount()
    mocks.auth.user.id = 84
    wrapper = render()
    await flushPromises()
    expect(wrapper.get('input[type="search"]').element).toHaveProperty('value', '')
    expect(wrapper.getComponent(CardsStub).props('favoriteKeys')).toEqual([])
    expect(displayedRows()).toEqual(severityOrder)
    expect(JSON.parse(localStorage.getItem('channel-monitor-v2:v1:42')!).favorites).toEqual([favorite])
  })

  it('gives explicit URL values including empty filters precedence over stored preferences', async () => {
    localStorage.setItem('channel-monitor-v2:v1:42', JSON.stringify({
      range: '7d', platform: ['openai'], group: [1], model: ['gpt-alpha'], q: 'gpt', status: 'healthy', model_type: 'image', sort: 'name', favorites_only: true,
    }))
    mocks.route.query = { range: '24h', platform: '', group: '', model: '', q: 'CLAUDE', status: 'all', model_type: 'all', sort: 'success', favorites_only: '0' }
    wrapper = render()
    await flushPromises()
    expect(mocks.getSnapshot).toHaveBeenCalledWith({ range: '24h', platforms: [], groupIds: [], models: [] }, false, expect.any(AbortSignal))
    expect(wrapper.get('input[type="search"]').element).toHaveProperty('value', 'CLAUDE')
    expect(wrapper.get('select[aria-label="channelMonitorV2.preferences.sortLabel"]').element).toHaveProperty('value', 'success')
    expect(displayedRows()).toEqual([rows[2]])
  })

  it('marks stale HTTP-200 data as historical and withholds current health counts', async () => {
    const staleCoverage = { ...coverage, computed_at: '2026-10-17T13:00:00Z', data_through: '2026-10-17T13:00:00Z', aggregation_lag_seconds: 5460 }
    mocks.getSnapshot.mockResolvedValue({ ...snapshot(), coverage: staleCoverage })
    mocks.getMatrix.mockResolvedValue({ coverage: staleCoverage, group_by: 'platform_group_model', items: rows })
    wrapper = render()
    await flushPromises()
    expect(wrapper.get('[data-testid="freshness-warning"]').text()).toContain('freshness.title')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.getComponent(CardsStub).props('stale')).toBe(true)
    expect(wrapper.get('[data-testid="summary-total"]').text()).toMatch(/5$/)
    for (const status of ['healthy', 'warning', 'critical']) expect(wrapper.get(`[data-testid="summary-${status}"]`).text()).toMatch(/—$/)
    expect(wrapper.get('[data-metric="channelMonitorV2.metrics.successRate"]').text()).toBe('80.0%')
  })

  it.each([false, true])('loads exact row scope and clicked interval for incident details (bucket=%s)', async bucket => {
    wrapper = render()
    await flushPromises()
    const selected = rows[2]
    const window = bucket ? { bucketStart: '2026-10-17T14:00:00.000Z', bucketEnd: '2026-10-17T15:00:00.000Z' } : {}
    const interval = bucket ? { start: window.bucketStart, end: window.bucketEnd } : { start: coverage.requested_start, end: coverage.requested_end }
    mocks.getMatrix.mockResolvedValueOnce({ coverage, items: [selected] })
    mocks.getErrors.mockResolvedValueOnce({ coverage, items: [{ category: 'timeout', rate: 1, count: 0 }] })
    wrapper.getComponent(CardsStub).vm.$emit('inspect', { row: selected, ...window })
    await flushPromises()
    const scoped = { range: '24h', platforms: ['anthropic'], groupIds: [3], models: ['claude-critical'], ...interval }
    expect(mocks.getMatrix).toHaveBeenLastCalledWith(scoped, 'platform_group_model', false, expect.any(AbortSignal))
    expect(mocks.getErrors).toHaveBeenLastCalledWith(scoped, false, expect.any(AbortSignal))
    const dialog = wrapper.getComponent(IncidentStub)
    expect(dialog.props('window')).toEqual(interval)
    expect(dialog.props('metrics')).toEqual(selected.metrics)
    expect(dialog.props('errors')).toEqual([{ category: 'timeout', rate: 1, count: 0 }])
    expect(dialog.props('loading')).toBe(false)
  })

  it('aborts old incident requests and prevents late responses replacing a newer row', async () => {
    wrapper = render()
    await flushPromises()
    let resolveFirst!: (value: unknown) => void
    mocks.getMatrix.mockImplementationOnce(() => new Promise(resolve => { resolveFirst = resolve }))
    wrapper.getComponent(CardsStub).vm.$emit('inspect', { row: rows[0] })
    await flushPromises()
    const previousSignal = mocks.getMatrix.mock.calls.at(-1)![3] as AbortSignal
    const current = { ...rows[2], metrics: metric({ success_rate: 0.3 }) }
    mocks.getMatrix.mockResolvedValueOnce({ coverage, items: [current] })
    mocks.getErrors.mockResolvedValueOnce({ coverage, items: [{ category: 'upstream_5xx', rate: 0.7, count: 0 }] })
    wrapper.getComponent(CardsStub).vm.$emit('inspect', { row: rows[2], bucketStart: '2026-10-17T14:00:00.000Z', bucketEnd: '2026-10-17T15:00:00.000Z' })
    await flushPromises()
    expect(previousSignal.aborted).toBe(true)
    resolveFirst({ coverage, items: [rows[0]] })
    await flushPromises()
    const dialog = wrapper.getComponent(IncidentStub)
    expect(dialog.props('row')).toEqual(rows[2])
    expect(dialog.props('metrics')).toEqual(current.metrics)
    expect(dialog.props('errors')).toEqual([{ category: 'upstream_5xx', rate: 0.7, count: 0 }])
  })

  it('closes open details when monitoring is disabled and ignores in-flight incident responses', async () => {
    wrapper = render()
    await flushPromises()
    let resolveIncident!: (value: unknown) => void
    mocks.getMatrix.mockImplementationOnce(() => new Promise(resolve => { resolveIncident = resolve }))
    wrapper.getComponent(CardsStub).vm.$emit('inspect', { row: rows[0] })
    await flushPromises()
    const signal = mocks.getMatrix.mock.calls.at(-1)![3] as AbortSignal
    expect(wrapper.getComponent(IncidentStub).props('show')).toBe(true)
    mocks.getSnapshot.mockRejectedValueOnce({ reason: 'CHANNEL_MONITOR_DISABLED', message: 'Feature disabled' })
    await wrapper.get('.availability-header .availability-action').trigger('click')
    await flushPromises()
    expect(signal.aborted).toBe(true)
    expect(wrapper.getComponent(IncidentStub).props('show')).toBe(false)
    expect(wrapper.find('[data-testid="model-cards"]').exists()).toBe(false)

    resolveIncident({ coverage, items: [rows[0]] })
    await flushPromises()
    expect(wrapper.getComponent(IncidentStub).props('show')).toBe(false)
    expect(wrapper.getComponent(IncidentStub).props('row')).toBeNull()
    expect(wrapper.getComponent(IncidentStub).props('metrics')).toBeNull()
    expect(wrapper.get('[role="alert"]').text()).toContain('availability.disabled')
  })
})
