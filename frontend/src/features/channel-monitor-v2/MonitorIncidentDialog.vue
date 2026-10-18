<template>
  <BaseDialog :show="show" :title="t('channelMonitorV2.incident.title')" width="wide" @close="emit('close')">
    <div v-if="row" class="incident-content">
      <div class="incident-heading">
        <h4>{{ row.model === '__other__' ? t('channelMonitorV2.otherModels') : row.model || t('channelMonitorV2.availability.modelFallback') }}<span v-if="row.is_image_model" class="incident-model-tag">{{ t('channelMonitorV2.availability.imageModel') }}</span></h4>
        <p>{{ row.platform }} · {{ row.group_name || (row.group_id ? t('channelMonitorV2.availability.groupFallback', { id: row.group_id }) : t('channelMonitorV2.availability.allGroups')) }}</p>
      </div>
      <p class="incident-period">{{ t('channelMonitorV2.incident.period') }}: {{ periodLabel }}</p>
      <p v-if="stale" class="incident-warning" role="status">{{ t('channelMonitorV2.freshness.title') }} · {{ t('channelMonitorV2.availability.historical') }}</p>
      <div v-if="loading" class="incident-loading" role="status">{{ t('common.loading') }}</div>
      <div v-else-if="error" class="incident-warning" role="alert"><p>{{ error }}</p><button type="button" class="btn btn-secondary btn-sm mt-3" @click="emit('retry')">{{ t('channelMonitorV2.incident.retry') }}</button></div>
      <template v-else>
        <p v-if="!hasData" class="incident-empty">{{ t('channelMonitorV2.incident.windowNoData') }}</p>
        <dl class="incident-metrics">
          <div><dt>{{ t('channelMonitorV2.availability.successRate') }}</dt><dd :class="metricTone(currentHealth?.success_rate ?? currentHealth?.error_rate)">{{ hasData && currentMetrics ? formatMonitorPercent(currentMetrics.success_rate ?? 1 - currentMetrics.error_rate) : '—' }}</dd></div>
          <div><dt>{{ t('channelMonitorV2.availability.cacheRate') }}</dt><dd>{{ hasData && currentMetrics && currentMetrics.cache_rate_available !== false ? formatMonitorPercent(currentMetrics.cache_rate) : '—' }}</dd></div>
          <div><dt>{{ t(row.is_image_model ? 'channelMonitorV2.availability.generationTime' : 'channelMonitorV2.availability.ttft') }}<template v-if="row.is_image_model && generationMetric.percentile"> · {{ generationMetric.percentile }}</template></dt><dd :class="metricTone(row.is_image_model ? currentHealth?.generation : currentHealth?.ttft)">{{ latencyValue(row.is_image_model ? generationMetric.value : currentMetrics?.ttft.p50_ms) }}</dd></div>
          <div><dt>{{ t('channelMonitorV2.availability.averageLatency') }}</dt><dd>{{ latencyValue(currentMetrics?.duration.avg_ms) }}</dd></div>
          <div><dt>{{ row.is_image_model ? t('channelMonitorV2.availability.generationTime') : t('channelMonitorV2.metrics.ttftP50') }} · {{ t('channelMonitorV2.incident.p90') }}</dt><dd>{{ latencyValue(latencyMetric?.p90_ms) }}</dd></div>
          <div><dt>{{ row.is_image_model ? t('channelMonitorV2.availability.generationTime') : t('channelMonitorV2.metrics.ttftP50') }} · {{ t('channelMonitorV2.incident.p95') }}</dt><dd>{{ latencyValue(latencyMetric?.p95_ms) }}</dd></div>
          <div v-if="isAdmin" class="admin-request-count"><dt>{{ t('channelMonitorV2.incident.requestCount') }}</dt><dd>{{ hasData ? currentMetrics?.request_count ?? 0 : '—' }}</dd></div>
          <div v-if="isAdmin" class="admin-error-count"><dt>{{ t('channelMonitorV2.incident.errorCount') }}</dt><dd>{{ hasData ? currentMetrics?.error_requests ?? 0 : '—' }}</dd></div>
        </dl>
        <section v-if="currentHealth?.reasons?.length" class="incident-failures">
          <h4>{{ t('channelMonitorV2.incident.healthReasons') }}</h4>
          <ul class="mt-2 list-disc space-y-1 pl-4 text-xs leading-6"><template v-for="reason in currentHealth.reasons" :key="reason"><li v-if="te(`channelMonitorV2.healthReasons.${reason}`)">{{ t(`channelMonitorV2.healthReasons.${reason}`) }}</li></template></ul>
        </section>
        <section class="incident-failures" :aria-label="t('channelMonitorV2.incident.failures')">
          <h4>{{ t('channelMonitorV2.incident.failures') }}</h4>
          <p v-if="errors.length === 0" class="incident-empty">{{ t('channelMonitorV2.incident.noFailures') }}</p>
          <ul v-else class="failure-list">
            <li v-for="failure in errors" :key="failure.category">
              <div class="failure-line"><span class="failure-label">{{ categoryLabel(failure.category) }}<small v-if="failure.ignored">{{ t('channelMonitorV2.incident.ignored') }}</small></span><span class="failure-rate">{{ t('channelMonitorV2.incident.share') }} {{ formatMonitorPercent(failure.rate) }}<span v-if="isAdmin"> · {{ failure.count }}</span></span></div>
              <div class="failure-bar" role="img" :aria-label="`${categoryLabel(failure.category)} ${formatMonitorPercent(failure.rate)}`"><span :style="{ width: `${Math.max(0, Math.min(100, failure.rate * 100))}%` }"></span></div>
            </li>
          </ul>
        </section>
        <section v-if="isAdmin" class="incident-admin" :aria-label="t('channelMonitorV2.incident.adminSummary')">
          <h4>{{ t('channelMonitorV2.incident.adminSummary') }}</h4>
          <p v-if="adminDetails.length === 0" class="incident-empty">{{ t('channelMonitorV2.incident.errorSummaryEmpty') }}</p>
          <ul v-else class="admin-details"><li v-for="(detail, index) in adminDetails" :key="index"><p>{{ categoryLabel(detail.category) }} · {{ detail.error_type || '—' }} · HTTP {{ detail.status_code ?? detail.upstream_status_code ?? '—' }} · {{ detail.count }}</p><pre v-if="detail.message">{{ detail.message }}</pre></li></ul>
          <div v-if="logRoute" class="incident-log-link"><router-link :to="logRoute" class="btn btn-secondary btn-sm">{{ t('channelMonitorV2.incident.openLogs') }}</router-link><p>{{ t('channelMonitorV2.incident.logsHint') }}</p></div>
        </section>
      </template>
    </div>
    <template #footer><button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('channelMonitorV2.incident.close') }}</button></template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { HealthState, MonitorCoverage, MonitorErrorRow, MonitorHealth, MonitorMatrixRow, MonitorMetric } from '@/api/channelMonitorV2'
import { formatMonitorMs, formatMonitorPercent } from './monitorFormat'

const props = withDefaults(defineProps<{
  show: boolean
  row: MonitorMatrixRow | null
  coverage?: MonitorCoverage | null
  window?: { start: string; end: string } | null
  metrics?: MonitorMetric | null
  health?: MonitorHealth | null
  loading?: boolean
  error?: string
  errors?: MonitorErrorRow[]
  isAdmin?: boolean
  stale?: boolean
}>(), { coverage: null, window: null, loading: false, error: '', errors: () => [], isAdmin: false, stale: false })
const emit = defineEmits<{ close: []; retry: [] }>()
const { t, te, locale } = useI18n()
const selectedBucket = computed(() => {
  if (!props.window) return undefined
  const start = Date.parse(props.window.start)
  const step = Math.max(60, props.coverage?.bucket_seconds || 60) * 1000
  return props.row?.buckets.find(bucket => Date.parse(bucket.bucket_start) <= start && Date.parse(bucket.bucket_start) + step > start)
})
const currentMetrics = computed(() => props.metrics !== undefined ? props.metrics : props.window ? selectedBucket.value?.metrics : props.row?.metrics)
const currentHealth = computed(() => props.health !== undefined ? props.health : props.window ? selectedBucket.value?.health : props.row?.health)
const hasData = computed(() => {
  const m = currentMetrics.value
  if (!m) return false
  return m.has_data ?? Boolean(selectedBucket.value || props.row?.buckets.length || m.request_count > 0 || m.ttft.p50_ms != null || m.duration.p50_ms != null || m.error_rate > 0 || (m.success_rate ?? 0) > 0)
})
const generationMetric = computed(() => {
  if (currentHealth.value?.generation_latency_ms != null) return { value: currentHealth.value.generation_latency_ms, percentile: currentHealth.value.generation_percentile?.toUpperCase() || '' }
  const duration = currentMetrics.value?.duration
  if (duration?.p95_ms != null) return { value: duration.p95_ms, percentile: 'P95' }
  if (duration?.p90_ms != null) return { value: duration.p90_ms, percentile: 'P90' }
  if (duration?.p50_ms != null) return { value: duration.p50_ms, percentile: 'P50' }
  return { value: duration?.avg_ms, percentile: duration?.avg_ms != null ? 'AVG' : '' }
})
const latencyMetric = computed(() => props.row?.is_image_model ? currentMetrics.value?.duration : currentMetrics.value?.ttft)
const period = computed(() => ({ start: props.window?.start || props.coverage?.requested_start || '', end: props.window?.end || props.coverage?.requested_end || props.coverage?.data_through || '' }))
function formatTime(raw: string) {
  const date = new Date(raw)
  return Number.isFinite(date.getTime()) ? new Intl.DateTimeFormat(locale.value || undefined, { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(date) : '—'
}
const periodLabel = computed(() => `${formatTime(period.value.start)} – ${formatTime(period.value.end)}`)
function localDate(raw: string) {
  const date = new Date(raw)
  if (!Number.isFinite(date.getTime())) return null
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}
// UsageView only supports date/user query hydration. Do not imply unsupported model/group filters were applied.
const logRoute = computed(() => {
  const start = localDate(period.value.start)
  const end = localDate(period.value.end)
  return props.isAdmin && start && end ? { path: '/admin/usage', query: { start_date: start, end_date: end } } : null
})
const adminDetails = computed(() => props.isAdmin ? props.errors.flatMap(row => (row.details || []).map(detail => ({ ...detail, category: row.category }))) : [])
function categoryLabel(category: string) { const key = `channelMonitorV2.errorCategories.${category}`; return te(key) ? t(key) : category }
function latencyValue(value?: number | null) { return hasData.value && value != null ? formatMonitorMs(value) : '—' }
function metricTone(state?: HealthState) { return props.stale ? 'tone-unknown' : `tone-${state || 'unknown'}` }
</script>

<style scoped>
.incident-content { min-width: 0; color: #262626; }
.incident-heading h4 { overflow-wrap: anywhere; font-size: 1rem; font-weight: 650; }
.incident-heading p, .incident-period { margin-top: 0.5rem; color: #737373; font-size: 0.75rem; line-height: 1.75; overflow-wrap: anywhere; }
.incident-model-tag { display: inline-block; margin-inline-start: 0.5rem; border: 1px solid #e5e5e5; border-radius: 5px; padding: 1px 6px; font-size: 0.625rem; font-weight: 500; }
.incident-warning { margin-top: 1rem; border: 1px solid #d4d4d4; border-radius: 10px; padding: 0.875rem; color: #737373; font-size: 0.8125rem; overflow-wrap: anywhere; }
.incident-loading { padding: 2rem 0; color: #737373; font-size: 0.875rem; text-align: center; }
.incident-metrics { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.75rem; margin-top: 1.25rem; }
.incident-metrics > div { min-width: 0; border: 1px solid #e5e5e5; border-radius: 10px; padding: 0.875rem; }
.incident-metrics dt { color: #737373; font-size: 0.6875rem; line-height: 1.75; overflow-wrap: anywhere; }
.incident-metrics dd { margin-top: 0.5rem; font-family: ui-monospace, monospace; font-size: 1rem; font-weight: 600; overflow-wrap: anywhere; font-variant-numeric: tabular-nums; }
.tone-healthy { color: #059669; }.tone-warning { color: #d97706; }.tone-critical { color: #e11d48; }.tone-unknown { color: #737373; }
.incident-failures, .incident-admin { margin-top: 1.5rem; }
.incident-failures h4, .incident-admin h4 { font-size: 0.875rem; font-weight: 600; }
.incident-empty { margin-top: 0.75rem; color: #737373; font-size: 0.75rem; line-height: 1.75; }
.failure-list { display: grid; gap: 1rem; margin-top: 1rem; }
.failure-line { display: flex; align-items: flex-start; justify-content: space-between; gap: 0.75rem; font-size: 0.75rem; }
.failure-label { min-width: 0; overflow-wrap: anywhere; }.failure-label small { display: block; margin-top: 0.25rem; color: #737373; font-size: 0.625rem; }.failure-rate { flex: none; font-variant-numeric: tabular-nums; }
.failure-bar { height: 4px; margin-top: 0.5rem; overflow: hidden; border-radius: 4px; background: #f5f5f5; }.failure-bar > span { display: block; height: 100%; border-radius: inherit; background: #fa2d44; }
.admin-details { display: grid; gap: 0.75rem; margin-top: 1rem; }.admin-details > li { min-width: 0; border: 1px solid #e5e5e5; border-radius: 8px; padding: 0.75rem; }.admin-details p { font-size: 0.6875rem; overflow-wrap: anywhere; }.admin-details pre { margin-top: 0.5rem; white-space: pre-wrap; overflow-wrap: anywhere; font-size: 0.6875rem; line-height: 1.75; }
.incident-log-link { margin-top: 1rem; }.incident-log-link p { margin-top: 0.5rem; color: #737373; font-size: 0.6875rem; line-height: 1.75; }
.dark .incident-content { color: #e5e5e5; }.dark .incident-heading p, .dark .incident-period, .dark .incident-metrics dt, .dark .incident-empty, .dark .failure-label small, .dark .incident-log-link p, .dark .incident-warning, .dark .tone-unknown { color: #a3a3a3; }.dark .incident-metrics > div, .dark .incident-model-tag, .dark .admin-details > li, .dark .incident-warning { border-color: #525252; }.dark .failure-bar { background: #404040; }.dark .tone-healthy { color: #34d399; }.dark .tone-warning { color: #fbbf24; }.dark .tone-critical { color: #fb7185; }
@media (max-width: 600px) { .incident-metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); }.incident-metrics > div { padding: 0.75rem; }.failure-line { flex-direction: column; gap: 0.25rem; } }
</style>
