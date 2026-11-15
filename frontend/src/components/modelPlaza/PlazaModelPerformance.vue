<template>
  <div class="performance" :aria-busy="loading">
    <header class="performance-page-heading">
      <h2>{{ t('publicSite.modelDetail.performance') }}</h2>
      <span v-if="lastUpdated" class="performance-updated">{{ t('publicSite.performance.updatedAt', { time: lastUpdated }) }}</span>
    </header>
    <div class="performance-controls">
      <div
        class="performance-ranges"
        role="group"
        :aria-label="t('publicSite.performance.range')"
        :style="{ '--range-index': ranges.indexOf(range) }"
      >
        <span class="performance-range-highlight" aria-hidden="true" />
        <button
          v-for="option in ranges"
          :key="option"
          type="button"
          :aria-pressed="range === option"
          @click="range = option"
        >
          {{ t(`publicSite.performance.ranges.${option}`) }}
        </button>
      </div>
      <div class="performance-group-filter">
        <Layers3 aria-hidden="true" :size="15" />
        <select v-model="groupId" class="performance-select" :aria-label="t('publicSite.performance.group')">
          <option :value="null">{{ t('publicSite.performance.allGroups') }}</option>
          <option v-for="group in groupOptions" :key="group.id" :value="group.id">{{ group.name }}</option>
        </select>
        <ChevronDown aria-hidden="true" :size="14" />
      </div>
      <div class="performance-refresh">
        <label class="performance-auto">
          <input v-model="autoRefresh" type="checkbox" class="sr-only" />
          <span class="performance-switch" aria-hidden="true"></span>
          {{ t('publicSite.performance.autoRefresh') }}
        </label>
        <button
          type="button"
          class="performance-button"
          :disabled="loading"
          :aria-label="t('publicSite.performance.refresh')"
          :title="t('publicSite.performance.refresh')"
          @click="refresh()"
        >
          <RefreshCw aria-hidden="true" :size="15" :class="{ 'animate-spin': loading }" />
        </button>
      </div>
    </div>

    <div v-if="error" class="performance-error" role="status">
      <TriangleAlert aria-hidden="true" :size="16" />
      <span>{{ t('publicSite.performance.loadError') }}</span>
      <button type="button" class="underline underline-offset-4" @click="refresh()">
        {{ t('publicSite.performance.retry') }}
      </button>
    </div>

    <dl class="performance-kpis">
      <div class="performance-kpi">
        <dt>{{ t('publicSite.performance.tps') }}</dt>
        <dd class="performance-value" :class="{ 'is-empty': data?.summary.tps == null }">
          <Transition name="performance-number" mode="out-in">
            <span :key="data?.summary.tps ?? 'empty'">{{ formatTps(data?.summary.tps) }}</span>
          </Transition>
        </dd>
        <dd class="performance-note">{{ t('publicSite.performance.tpsHint') }}</dd>
      </div>
      <div class="performance-kpi">
        <dt>{{ t('publicSite.performance.averageLatency') }}</dt>
        <dd class="performance-value" :class="{ 'is-empty': data?.summary.avg_latency_ms == null }">
          <Transition name="performance-number" mode="out-in">
            <span :key="data?.summary.avg_latency_ms ?? 'empty'">{{ formatLatency(data?.summary.avg_latency_ms) }}</span>
          </Transition>
        </dd>
        <dd class="performance-note">{{ t('publicSite.performance.latencyHint') }}</dd>
      </div>
      <div class="performance-kpi">
        <dt>{{ t('publicSite.performance.successRate') }}</dt>
        <dd class="performance-value" :class="[rateTone(data?.summary.success_rate), { 'is-empty': data?.summary.success_rate == null }]">
          <Transition name="performance-number" mode="out-in">
            <span :key="data?.summary.success_rate ?? 'empty'">{{ formatPercent(data?.summary.success_rate) }}</span>
          </Transition>
        </dd>
        <dd class="performance-note">{{ t('publicSite.performance.successRateHint') }}</dd>
      </div>
    </dl>

    <div class="performance-plots">
      <section class="performance-section performance-trend">
        <header class="performance-heading">
          <div class="performance-title">
            <div>
              <h3>{{ t('publicSite.performance.latencyTitle', { range: rangeLabel }) }}</h3>
              <p>{{ t('publicSite.performance.latencyDescription') }}</p>
            </div>
          </div>
        </header>
        <PlazaPerformanceChart :trend="trend" metric="ttft" :range="range" :loading="loading && !data" />
      </section>

      <section class="performance-section performance-trend">
        <header class="performance-heading">
          <div class="performance-title">
            <div>
              <h3>{{ t('publicSite.performance.availabilityTitle', { range: rangeLabel }) }}</h3>
              <p>{{ t('publicSite.performance.availabilityDescription') }}</p>
            </div>
          </div>
          <span v-if="hasAvailability && incidents > 0" class="performance-incidents">
            <TriangleAlert aria-hidden="true" :size="16" />
            {{ t('publicSite.performance.incidents', { count: incidents }) }}
          </span>
        </header>
        <PlazaPerformanceChart :trend="trend" metric="availability" :range="range" :loading="loading && !data" />
      </section>
    </div>

    <section class="performance-section performance-groups">
      <header class="performance-heading">
        <div class="performance-title">
          <div>
            <h3>{{ t('publicSite.performance.groupTitle') }}</h3>
            <p>{{ t('publicSite.performance.groupDescription') }}</p>
          </div>
        </div>
      </header>
      <div class="performance-table">
        <table role="table" :aria-label="t('publicSite.performance.groupTitle')">
          <thead>
            <tr>
              <th scope="col">{{ t('publicSite.performance.group') }}</th>
              <th scope="col">{{ t('publicSite.performance.tps') }}</th>
              <th scope="col">{{ t('publicSite.performance.averageTtft') }}</th>
              <th scope="col">{{ t('publicSite.performance.averageLatency') }}</th>
              <th scope="col">{{ t('publicSite.performance.successRate') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="group in groups" :key="group.id" :class="{ 'is-selected': groupId === group.id }">
              <th scope="row">
                <button
                  type="button"
                  :aria-pressed="groupId === group.id"
                  @click="groupId = groupId === group.id ? null : group.id"
                >
                  <span>{{ group.name }}</span>
                  <ChevronRight aria-hidden="true" :size="14" />
                </button>
              </th>
              <td>
                <span class="performance-mobile-label" aria-hidden="true">{{ t('publicSite.performance.tps') }}</span>
                {{ formatTps(group.metrics.tps) }}
              </td>
              <td>
                <span class="performance-mobile-label" aria-hidden="true">{{ t('publicSite.performance.averageTtft') }}</span>
                {{ formatLatency(group.metrics.avg_ttft_ms) }}
              </td>
              <td>
                <span class="performance-mobile-label" aria-hidden="true">{{ t('publicSite.performance.averageLatency') }}</span>
                {{ formatLatency(group.metrics.avg_latency_ms) }}
              </td>
              <td>
                <span class="performance-mobile-label" aria-hidden="true">{{ t('publicSite.performance.successRate') }}</span>
                <div class="performance-success">
                  <span v-if="group.buckets.some(bucket => bucket.success_rate != null)" class="performance-spark" aria-hidden="true">
                    <i
                      v-for="bucket in group.buckets"
                      :key="bucket.bucket_start"
                      :class="sparkTone(bucket)"
                    />
                  </span>
                  <span :class="rateTone(group.metrics.success_rate)">{{ formatPercent(group.metrics.success_rate) }}</span>
                </div>
              </td>
            </tr>
            <tr v-if="groups.length === 0">
              <td colspan="5" class="performance-empty" role="status">
                {{ t(loading ? 'publicSite.performance.loading' : 'publicSite.performance.noData') }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronDown, ChevronRight, Layers3, RefreshCw, TriangleAlert } from '@lucide/vue'
import { getModelPlazaPerformance, type PlazaPerformanceBucket, type PlazaPerformanceRange, type PlazaPerformanceResponse } from '@/api/modelPlaza'
import { formatMonitorRate } from '@/features/channel-monitor-v2/monitorFormat'
import PlazaPerformanceChart from './PlazaPerformanceChart.vue'

const props = defineProps<{
  model: string
  platform: string
  initialGroupId?: number | null
}>()

const ranges = ['24h', '7d', '30d'] as const
const { t, locale } = useI18n()
const range = ref<PlazaPerformanceRange>('24h')
const groupId = ref<number | null>(props.initialGroupId ?? null)
const groupOptions = ref<{ id: number; name: string }[]>([])
const autoRefresh = ref(true)
const result = shallowRef<{
  model: string
  platform: string
  range: PlazaPerformanceRange
  groupId: number | null
  receivedAt: Date
  value: PlazaPerformanceResponse
}>()
const loading = ref(false)
const error = ref(false)
let request: AbortController | undefined
let timer: number | undefined

const rangeResult = computed(() => {
  const current = result.value
  return current?.model === props.model && current.platform === props.platform && current.range === range.value ? current : undefined
})
const selectionResult = computed(() => rangeResult.value?.groupId === groupId.value ? rangeResult.value : undefined)
const data = computed(() => selectionResult.value?.value)
const groups = computed(() => rangeResult.value?.value.groups ?? [])
const trend = computed(() => data.value?.trend ?? [])
const rangeLabel = computed(() => t(`publicSite.performance.ranges.${range.value}`))
const hasAvailability = computed(() => trend.value.some(bucket => bucket.success_rate != null))
const incidents = computed(() => data.value?.summary.incident_count ?? 0)
const lastUpdated = computed(() => {
  const receivedAt = selectionResult.value?.receivedAt
  if (!receivedAt) return ''
  return new Intl.DateTimeFormat(locale.value, {
    hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23',
  }).format(receivedAt)
})
const percentFormat = computed(() => new Intl.NumberFormat(locale.value, {
  style: 'percent',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
}))
const secondsFormat = computed(() => new Intl.NumberFormat(locale.value, {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
}))

function formatTps(value?: number | null) {
  return value == null ? t('publicSite.performance.noData') : `${formatMonitorRate(value, locale.value)} t/s`
}

function formatLatency(value?: number | null) {
  if (value == null) return t('publicSite.performance.noData')
  return value < 1000
    ? `${formatMonitorRate(value, locale.value)} ms`
    : `${secondsFormat.value.format(value / 1000)} s`
}

function formatPercent(value?: number | null) {
  if (value == null) return t('publicSite.performance.noData')
  return percentFormat.value.format(value)
}

function rateTone(value?: number | null) {
  if (value == null) return 'muted'
  if (value >= 0.95) return 'healthy'
  return value >= 0.9 ? 'warning' : 'critical'
}

function sparkTone(bucket: PlazaPerformanceBucket) {
  if (bucket.abnormal ?? (bucket.failures ?? 0) > 0) return 'critical'
  return rateTone(bucket.success_rate)
}

async function refresh(reset = false) {
  if (reset) {
    request?.abort()
  } else if (request) return

  const controller = new AbortController()
  request = controller
  loading.value = true
  error.value = false
  const query = { model: props.model, platform: props.platform, range: range.value, groupId: groupId.value }
  try {
    const value = await getModelPlazaPerformance(
      { ...query, groupId: query.groupId ?? undefined },
      { signal: controller.signal },
    )
    if (controller.signal.aborted) return
    if (!value?.summary || !Array.isArray(value.groups) || !Array.isArray(value.trend)) {
      throw new Error('Invalid model performance response')
    }
    result.value = { ...query, receivedAt: new Date(), value }
    groupOptions.value = value.groups.map(({ id, name }) => ({ id, name }))
  } catch {
    if (!controller.signal.aborted) error.value = true
  } finally {
    if (request === controller) {
      request = undefined
      loading.value = false
    }
  }
}

watch(() => [props.model, props.platform], () => {
  groupOptions.value = []
  groupId.value = props.initialGroupId ?? null
})
watch([() => props.model, () => props.platform, range, groupId], () => refresh(true), { immediate: true })

onMounted(() => {
  timer = window.setInterval(() => {
    if (autoRefresh.value && !document.hidden) void refresh()
  }, 30_000)
})

onBeforeUnmount(() => {
  request?.abort()
  window.clearInterval(timer)
})
</script>

<style scoped>
/* Visual reference: DesignBookmark Stat Card and Segmented Control.
   https://designbookmark.com/motion#db-stat-card
   Implemented for Vue with live performance data; no demo animation engine. */
.performance {
  --performance-line: var(--detail-border, #ececec);
  --performance-muted: var(--detail-muted, #747474);
  --performance-accent: var(--detail-accent, #171717);
  --performance-green: var(--detail-success, #168252);
  --performance-amber: var(--detail-warning, #a06013);
  --performance-red: var(--detail-danger, #bf4052);
  display: grid;
  gap: 24px;
  min-width: 0;
  color: var(--detail-ink, #171717);
}
.performance-page-heading { display: flex; flex-wrap: wrap; align-items: baseline; justify-content: space-between; gap: 8px; }
.performance-page-heading h2 { font-size: 1.5rem; line-height: 1.25; letter-spacing: -.03em; font-weight: 600; }
.performance-updated { color: var(--performance-muted); font-size: .75rem; font-variant-numeric: tabular-nums; }

.performance-controls { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; color: var(--performance-muted); font-size: .8rem; }
.performance-ranges { position: relative; display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); padding: 4px; border-radius: 16px; background: var(--detail-subtle, #f8f8f7); }
.performance-range-highlight { position: absolute; inset: 4px auto 4px 4px; width: calc((100% - 8px) / 3); border-radius: 12px; background: var(--detail-ink); transform: translateX(calc(var(--range-index, 0) * 100%)); transition: transform 480ms cubic-bezier(.22, 1.22, .36, 1); }
.performance-ranges button { position: relative; z-index: 1; min-height: 38px; padding: 5px 15px; border-radius: 12px; white-space: nowrap; transition: color 200ms; }
.performance-ranges button:hover { color: var(--detail-ink); }
.performance-ranges button[aria-pressed='true'] { color: var(--detail-bg); }
.performance-group-filter { position: relative; min-width: 0; max-width: min(220px, 100%); }
.performance-group-filter > svg { position: absolute; top: 50%; z-index: 1; transform: translateY(-50%); pointer-events: none; }
.performance-group-filter > svg:first-child { inset-inline-start: 13px; }
.performance-group-filter > svg:last-child { inset-inline-end: 12px; }
.performance-select { width: 100%; min-width: 9rem; min-height: 46px; padding: 5px 32px 5px 36px; border: 0; border-radius: 16px; background: var(--detail-subtle, #f8f8f7); color: var(--detail-ink); appearance: none; text-overflow: ellipsis; }
.dark .performance-select { color-scheme: dark; }
.performance-refresh { display: flex; align-items: center; gap: 12px; margin-inline-start: auto; }
.performance-auto { display: flex; align-items: center; gap: 7px; min-height: 44px; cursor: pointer; white-space: nowrap; }
.performance-switch { position: relative; width: 30px; height: 18px; flex: none; border-radius: 999px; background: var(--performance-line); transition: background-color 200ms; }
.performance-switch::after { position: absolute; inset: 3px auto 3px 3px; width: 12px; border-radius: 50%; background: var(--detail-bg); content: ''; transition: transform 240ms cubic-bezier(.22, 1.22, .36, 1); }
.performance-auto input:checked + .performance-switch { background: var(--detail-ink); }
.performance-auto input:checked + .performance-switch::after { transform: translateX(12px); }
.performance-auto input:focus-visible + .performance-switch { outline: 2px solid var(--performance-accent); outline-offset: 3px; }
.performance-button { display: inline-grid; place-items: center; height: 44px; width: 44px; border-radius: 50%; background: var(--detail-subtle, #f8f8f7); color: var(--detail-ink); transition: transform 180ms, background-color 180ms; }
.performance-button:hover:not(:disabled) { background: var(--detail-accent-soft); transform: rotate(12deg); }
.performance-button:active:not(:disabled) { transform: scale(.94); }
.performance-button:disabled { opacity: .6; }

.performance-kpis { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 14px; }
.performance-kpi { min-width: 0; min-height: 155px; padding: 22px; border-radius: 24px; background: var(--detail-subtle, #f8f8f7); }
.performance-kpi dt { color: var(--performance-muted); font-size: .8125rem; font-weight: 500; }
.performance-value { min-height: 40px; margin: 12px 0 8px; font-size: 34px; line-height: 40px; font-weight: 600; letter-spacing: -.035em; font-variant-numeric: tabular-nums; white-space: nowrap; }
.performance-value > span { display: inline-block; }
.performance-value.is-empty { font-size: 1.1rem; letter-spacing: 0; color: var(--performance-muted); }
.performance-note { font-size: .72rem; line-height: 1.5; color: var(--performance-muted); }
.performance-number-enter-active, .performance-number-leave-active { transition: opacity 140ms, transform 140ms; }
.performance-number-enter-from { opacity: 0; transform: translateY(6px); }
.performance-number-leave-to { opacity: 0; transform: translateY(-4px); }

.performance-plots { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
.performance-section { min-width: 0; }
.performance-heading { display: flex; flex-wrap: wrap; justify-content: space-between; align-items: baseline; gap: 6px; margin-bottom: 16px; }
.performance-title { min-width: 0; }
.performance-title h3 { font-size: .95rem; line-height: 1.45; font-weight: 600; letter-spacing: -.015em; }
.performance-title p { margin-top: 5px; color: var(--performance-muted); font-size: .75rem; line-height: 1.5; }
.performance-trend { display: flex; flex-direction: column; padding: 22px; border-radius: 24px; background: var(--detail-subtle, #f8f8f7); }
.performance-trend .performance-heading { min-height: 72px; margin-bottom: 12px; }
.performance-trend :deep(.performance-plot) { margin-top: auto; }
.performance-incidents { display: inline-flex; align-items: center; gap: 4px; padding: 3px 7px; border-radius: 99px; background: var(--detail-warning-soft); color: var(--performance-amber); font-size: .71rem; }

.performance-groups { padding-top: 4px; }
.performance-table { overflow-x: auto; border: 1px solid var(--performance-line); border-radius: 20px; }
.performance-table table { width: 100%; min-width: 36rem; border-collapse: collapse; text-align: left; font-size: .8125rem; }
.performance-table thead { background: var(--detail-subtle, #f8f8f7); color: var(--performance-muted); font-size: .75rem; }
.performance-table th, .performance-table td { padding: 15px 13px; }
.performance-table th:first-child { padding-inline-start: 20px; }
.performance-table th:last-child, .performance-table td:last-child { padding-inline-end: 20px; }
.performance-table thead th { font-weight: 500; border-bottom: 1px solid var(--performance-line); }
.performance-table tbody tr { border-bottom: 1px solid var(--performance-line); transition: background-color 160ms; }
.performance-table tbody tr:last-child { border-bottom: 0; }
.performance-table tbody tr:hover { background: var(--detail-subtle); }
.performance-table tbody tr.is-selected { background: var(--detail-accent-soft); }
.performance-table tbody th { min-width: 7rem; overflow-wrap: anywhere; }
.performance-table tbody th button { display: inline-flex; align-items: center; gap: 6px; min-height: 30px; color: var(--detail-ink); font-weight: 600; text-align: left; border-radius: 5px; }
.performance-table tbody th button svg { flex: none; color: var(--performance-muted); transition: transform 180ms; }
.performance-table tbody th button:hover svg { transform: translateX(2px); }
.performance-table tbody th button[aria-pressed='true'] svg { color: var(--performance-accent); transform: rotate(90deg); }
.performance-table th:nth-child(n+2):nth-child(-n+4), .performance-table td:nth-child(n+2):nth-child(-n+4) { text-align: right; }
.performance-table td:not(.performance-empty) { white-space: nowrap; font-variant-numeric: tabular-nums; }
.performance-success { display: inline-flex; align-items: center; gap: 8px; }
.performance-spark { display: inline-flex; align-items: flex-end; gap: 2px; min-height: 14px; }
.performance-spark i { width: 3px; height: 12px; border-radius: 2px; background: currentColor; }
.performance-spark .warning { height: 8px; }
.performance-spark .critical { height: 5px; }
.performance-spark .muted { height: 2px; color: var(--performance-line); }
.performance-mobile-label { display: none; }
.healthy { color: var(--performance-green); }
.warning { color: var(--performance-amber); }
.critical { color: var(--performance-red); }
.muted, .performance-empty { color: var(--performance-muted); }
.performance-table .performance-empty { padding: 30px 20px; text-align: center; }
.performance-error { display: flex; align-items: center; gap: 8px; padding: 10px 14px; border-radius: 14px; background: var(--detail-danger-soft); color: var(--performance-red); font-size: .8rem; }
.performance-error button { margin-inline-start: auto; min-height: 30px; }
.performance :is(button, select):focus-visible { outline: 2px solid var(--performance-accent); outline-offset: 3px; }

@media (max-width: 1050px) {
  .performance-spark { display: none; }
  .performance-kpi, .performance-trend { padding: 20px; }
  .performance-value { font-size: 30px; }
}
@media (max-width: 800px) {
  .performance-plots { grid-template-columns: 1fr; }
  .performance-trend .performance-heading { min-height: auto; }
  .performance-table table, .performance-table tbody { display: block; min-width: 0; }
  .performance-table thead { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
  .performance-table tbody tr { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px 16px; padding: 15px 18px 20px; }
  .performance-table tbody th, .performance-table .performance-empty { grid-column: 1 / -1; }
  .performance-table tbody th button { min-height: 36px; }
  .performance-table :is(tbody th, tbody td) { padding: 0; }
  .performance-table tbody th:first-child { padding-inline-start: 0; }
  .performance-table tbody td:last-child { padding-inline-end: 0; }
  .performance-table td:nth-child(n+2):nth-child(-n+4) { text-align: left; }
  .performance-table td:not(.performance-empty) { white-space: normal; }
  .performance-mobile-label { display: block; margin-bottom: 3px; color: var(--performance-muted); font-size: .7rem; line-height: 1.4; }
}
@media (max-width: 600px) {
  .performance { gap: 20px; }
  .performance-page-heading h2 { font-size: 1.35rem; }
  .performance-updated { font-size: .7rem; }
  .performance-controls { gap: 8px; }
  .performance-ranges { width: 100%; }
  .performance-ranges button { min-height: 44px; padding-inline: 4px; white-space: normal; }
  .performance-group-filter { flex: 1 1 7rem; }
  .performance-select { min-width: 0; font-size: 1rem; }
  .performance-kpis { grid-template-columns: 1fr; gap: 10px; }
  .performance-kpi { position: relative; min-height: 114px; padding: 18px 20px; }
  .performance-value { margin: 8px 0 0; font-size: 32px; }
  .performance-note { margin-top: 5px; }
  .performance-trend { padding: 20px 16px; }
  .performance-table tbody th button { min-height: 44px; }
}
@media (prefers-reduced-motion: reduce) {
  .performance *, .performance *::after { transition: none !important; animation: none !important; }
  .performance-number-enter-from, .performance-number-leave-to { transform: none; }
}
</style>
