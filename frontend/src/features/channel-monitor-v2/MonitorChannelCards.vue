<template>
  <section class="channel-cards" :aria-label="t('channelMonitorV2.cards.aria')" :aria-busy="loading">
    <div v-if="loading && rows.length === 0" class="model-list" aria-hidden="true">
      <div v-for="index in 3" :key="index" class="model-card animate-pulse">
        <div class="model-heading">
          <div class="h-10 w-10 rounded-xl bg-gray-100 dark:bg-dark-700"></div>
          <div class="h-4 w-48 max-w-full rounded bg-gray-100 dark:bg-dark-700"></div>
        </div>
        <div class="skeleton-row"><div class="h-6 w-full rounded bg-gray-100 dark:bg-dark-700"></div></div>
      </div>
    </div>

    <EmptyState
      v-else-if="rows.length === 0"
      :title="t('channelMonitorV2.matrix.emptyTitle')"
      :description="emptyDescription || t('channelMonitorV2.empty.description')"
    />

    <div v-else class="model-list">
      <p v-if="stale" class="stale-note" role="status">{{ t('channelMonitorV2.availability.staleStatus') }} · {{ t('channelMonitorV2.availability.historical') }}</p>
      <article v-for="model in modelCards" :key="model.key" class="model-card">
        <header class="model-heading">
          <span class="provider-icon"><ProviderIcon :provider="model.platform" :size="25" /></span>
          <div class="min-w-0">
            <h2 class="model-name">{{ model.name }}<span v-if="model.isImage" class="image-model-badge">{{ t('channelMonitorV2.availability.imageModel') }}</span></h2>
            <p class="provider-name">{{ providerLabel(model.platform) }}</p>
          </div>
          <span class="group-count">{{ model.entries.length }}</span>
        </header>

        <div class="column-headings metric-grid" aria-hidden="true">
          <span>{{ t('channelMonitorV2.availability.group') }}</span>
          <span>{{ t('channelMonitorV2.availability.successRate') }}</span>
          <span>{{ t('channelMonitorV2.availability.trend') }}</span>
          <span>{{ t('channelMonitorV2.availability.cacheRate') }}</span>
          <span>{{ t(model.isImage ? 'channelMonitorV2.availability.generationTime' : 'channelMonitorV2.availability.ttft') }}</span>
        </div>

        <section v-for="entry in model.entries" :key="entry.key" class="group-row metric-grid" :aria-label="groupName(entry.row)">
          <div class="group-cell">
            <span class="mobile-label">{{ t('channelMonitorV2.availability.group') }}</span>
            <button
              type="button"
              class="group-link"
              :aria-label="t('channelMonitorV2.availability.inspect', { group: groupName(entry.row) })"
              @click="emit('inspect', { row: entry.row })"
            >
              <span>{{ groupName(entry.row) }}</span>
              <svg aria-hidden="true" width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path d="m9 5 7 7-7 7" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" /></svg>
            </button>
            <div class="group-actions">
              <button class="row-action favorite-action" type="button" :class="{ 'is-favorite': isFavorite(entry.row) }" :aria-pressed="isFavorite(entry.row)" :aria-label="t(isFavorite(entry.row) ? 'channelMonitorV2.availability.unfavorite' : 'channelMonitorV2.availability.favorite')" @click="emit('toggle-favorite', entry.row)">{{ isFavorite(entry.row) ? '★' : '☆' }}</button>
              <button v-if="entry.row.group_id != null && entry.row.group_id > 0" class="row-action filter-action" type="button" :aria-label="t('channelMonitorV2.availability.filterGroup')" @click="emit('select-group', entry.row)"><Icon name="filter" size="xs" /></button>
            </div>
          </div>

          <div class="success-cell">
            <span class="mobile-label">{{ t('channelMonitorV2.availability.successRate') }}</span>
            <span class="metric-value" :class="healthTextClass(entry.row.health.success_rate ?? entry.row.health.error_rate)">{{ hasMetrics(entry.row) ? successRate(entry.row.metrics) : '—' }}</span>
            <span class="health-badge" :class="stale ? 'status-stale' : `status-${rowState(entry.row)}`"><span class="status-dot"></span>{{ stale ? t('channelMonitorV2.availability.staleStatus') : healthLabel(rowState(entry.row), hasMetrics(entry.row)) }}</span>
          </div>

          <div class="history-cell">
            <span class="mobile-label">{{ t('channelMonitorV2.availability.trend') }}</span>
            <div
              :id="`${componentId}-${entry.key}-track`"
              class="timeline-track"
              :class="{ 'timeline-active': activeTimeline === entry.key }"
              :style="{ gridTemplateColumns: `repeat(${Math.max(1, bucketStarts.length)}, minmax(0, 1fr))`, columnGap: `min(2px, calc(20% / ${Math.max(1, bucketStarts.length)}))` }"
              role="slider"
              :tabindex="entry.slots.length ? 0 : -1"
              :aria-label="t('channelMonitorV2.availability.timelineLabel', { model: model.name, group: groupName(entry.row) })"
              :aria-valuemin="1"
              :aria-valuemax="Math.max(1, entry.slots.length)"
              :aria-valuenow="selectedIndex(entry) + 1"
              :aria-valuetext="entry.slots[selectedIndex(entry)]?.tooltip"
              :aria-describedby="`${componentId}-timeline-help`"
              @focus="activateTimeline(entry)"
              @blur="activeTimeline = null"
              @mouseleave="leaveTimeline($event)"
              @keydown="navigateTimeline($event, entry)"
            >
              <span
                v-for="(slot, index) in entry.slots"
                :key="slot.start"
                class="timeline-bucket"
                :class="[`health-${bucketState(slot.bucket, model.isImage)}`, { 'bucket-selected': activeTimeline === entry.key && selectedIndex(entry) === index }]"
                aria-hidden="true"
                @mouseenter="activateTimeline(entry, index)"
                @click="selectBucket($event, entry, index)"
              ></span>
              <span v-if="activeTimeline === entry.key" class="bucket-tooltip" role="tooltip">{{ entry.slots[selectedIndex(entry)]?.tooltip }}</span>
            </div>
            <div class="timeline-axis"><time>{{ windowStart }}</time><time>{{ windowEnd }}</time></div>
          </div>

          <div class="cache-cell">
            <span class="mobile-label">{{ t('channelMonitorV2.availability.cacheRate') }}</span>
            <span class="metric-value">{{ cacheValue(entry.row.metrics, hasMetrics(entry.row)) }}</span>
          </div>

          <div class="latency-cell" :title="latencyDetail(model.isImage ? entry.row.metrics.duration : entry.row.metrics.ttft)">
            <span class="mobile-label">{{ t(model.isImage ? 'channelMonitorV2.availability.generationTime' : 'channelMonitorV2.availability.ttft') }}</span>
            <template v-if="model.isImage">
              <div class="generation-primary"><span class="metric-value" :class="healthTextClass(entry.row.health.generation)">{{ hasMetrics(entry.row) && generationValue(entry.row).value != null ? formatMonitorMs(generationValue(entry.row).value) : '—' }}</span><span class="latency-secondary">{{ generationValue(entry.row).percentile }}</span></div>
              <span class="latency-secondary">{{ t('channelMonitorV2.availability.generationP50', { value: hasMetrics(entry.row) && entry.row.metrics.duration.p50_ms != null ? formatMonitorMs(entry.row.metrics.duration.p50_ms) : '—' }) }}</span>
              <span class="latency-secondary">{{ t('channelMonitorV2.availability.averageLatency') }} {{ hasMetrics(entry.row) && entry.row.metrics.duration.avg_ms != null ? formatMonitorMs(entry.row.metrics.duration.avg_ms) : '—' }}</span>
            </template>
            <template v-else>
              <span class="metric-value" :class="healthTextClass(ttftDisplayState(entry.row.health.ttft, entry.row.metrics.ttft))">{{ hasMetrics(entry.row) ? formatMonitorMs(entry.row.metrics.ttft.p50_ms) : '—' }}</span>
              <span class="latency-secondary">{{ t('channelMonitorV2.availability.averageLatency') }} {{ hasMetrics(entry.row) ? formatMonitorMs(entry.row.metrics.duration.avg_ms) : '—' }}</span>
            </template>
            <span v-if="showThroughput && hasMetrics(entry.row)" class="latency-secondary"><template v-if="!model.isImage">{{ t('channelMonitorV2.metrics.tpsValue', { value: formatMonitorTokensPerSecond(entry.row.metrics.tpm) }) }} · </template>{{ t('channelMonitorV2.metrics.rpmValue', { value: formatMonitorThroughput(entry.row.metrics.rpm) }) }}</span>
          </div>
        </section>
      </article>
    </div>
    <p :id="`${componentId}-timeline-help`" class="sr-only">{{ t('channelMonitorV2.availability.timelineHelp') }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import type { HealthState, LatencyMetric, MonitorCoverage, MonitorMatrixBucket, MonitorMatrixRow, MonitorMetric } from '@/api/channelMonitorV2'
import Icon from '@/components/icons/Icon.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ProviderIcon from '@/components/user/monitor/ProviderIcon.vue'
import { useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'
import { formatLatencyPrivacy, formatMonitorMs, formatMonitorPercent, formatMonitorSuccessRateFromError, formatMonitorThroughput, formatMonitorTokensPerSecond, ttftDisplayState } from './monitorFormat'

const props = withDefaults(defineProps<{
  rows: MonitorMatrixRow[]
  coverage: MonitorCoverage
  healthMode: 'overall' | 'success' | 'ttft' | 'cache'
  showThroughput?: boolean
  loading?: boolean
  emptyDescription?: string
  favoriteKeys?: string[]
  stale?: boolean
}>(), { showThroughput: true, loading: false, emptyDescription: '', favoriteKeys: () => [], stale: false })
const emit = defineEmits<{
  'select-group': [row: MonitorMatrixRow]
  'toggle-favorite': [row: MonitorMatrixRow]
  inspect: [selection: { row: MonitorMatrixRow; bucketStart?: string; bucketEnd?: string }]
}>()
const { t, locale } = useI18n()
const { providerLabel } = useChannelMonitorFormat()
const componentId = useId()
const activeTimeline = ref<string | null>(null)
const timelineIndices = reactive<Record<string, number>>({})

interface TimelineSlot { start: number; bucket?: MonitorMatrixBucket; tooltip: string }
interface CardEntry { key: string; row: MonitorMatrixRow; slots: TimelineSlot[] }
interface ModelCard { key: string; platform: string; name: string; isImage: boolean; entries: CardEntry[] }

const bucketStep = computed(() => Math.max(60, Number(props.coverage.bucket_seconds) || 60) * 1000)
const rangeEnd = computed(() => {
  const start = Date.parse(props.coverage.requested_start)
  const requestedEnd = Date.parse(props.coverage.requested_end || '')
  return Number.isFinite(requestedEnd) && requestedEnd > start ? requestedEnd : Date.parse(props.coverage.data_through)
})
const bucketStarts = computed(() => {
  const start = Date.parse(props.coverage.requested_start)
  if (!Number.isFinite(start) || !Number.isFinite(rangeEnd.value) || rangeEnd.value <= start) return []
  const starts: number[] = []
  // Include every interval of the selected window, including gaps and partial coverage.
  for (let cursor = Math.floor(start / bucketStep.value) * bucketStep.value; cursor < rangeEnd.value; cursor += bucketStep.value) starts.push(cursor)
  return starts
})
const windowStart = computed(() => formatTime(Date.parse(props.coverage.requested_start)))
const windowEnd = computed(() => formatTime(rangeEnd.value))
const modelCards = computed(() => {
  const cards = new Map<string, ModelCard>()
  for (const row of props.rows) {
    const key = JSON.stringify([row.platform, row.model || ''])
    let card = cards.get(key)
    if (!card) {
      card = { key, platform: row.platform, isImage: Boolean(row.is_image_model), name: row.model === '__other__' ? t('channelMonitorV2.otherModels') : row.model || t('channelMonitorV2.availability.modelFallback'), entries: [] }
      cards.set(key, card)
    }
    card.isImage ||= Boolean(row.is_image_model)
    const buckets = new Map((row.buckets || []).map(bucket => [Date.parse(bucket.bucket_start), bucket]))
    card.entries.push({
      key: encodeURIComponent(JSON.stringify([row.platform, row.model || '', row.group_id ?? null])),
      row,
      slots: bucketStarts.value.map(start => {
        const bucket = buckets.get(start)
        return { start, bucket, tooltip: bucketTooltip(start, bucket, Boolean(row.is_image_model)) }
      }),
    })
  }
  return [...cards.values()]
})

function hasMetrics(row: MonitorMatrixRow) {
  // User payloads redact counts, so presence is authoritative and bucket existence is the legacy fallback.
  return row.metrics.has_data ?? Boolean(row.buckets?.some(bucket => bucket.metrics.has_data ?? true))
}
function bucketHasMetrics(bucket: MonitorMatrixBucket) {
  return bucket.metrics.has_data ?? true
}
function groupName(row: MonitorMatrixRow) {
  return row.group_name || (row.group_id ? t('channelMonitorV2.availability.groupFallback', { id: row.group_id }) : t('channelMonitorV2.availability.allGroups'))
}
function rowState(row: MonitorMatrixRow): HealthState { return hasMetrics(row) ? row.health.overall : 'unknown' }
function bucketState(bucket?: MonitorMatrixBucket, isImage = false): HealthState {
  if (!bucket || !bucketHasMetrics(bucket)) return 'unknown'
  if (props.healthMode === 'success') return bucket.health.success_rate ?? bucket.health.error_rate
  if (props.healthMode === 'ttft' && isImage) return bucket.health.generation || 'unknown'
  if (props.healthMode === 'ttft') return ttftDisplayState(bucket.health.ttft, bucket.metrics.ttft) || 'unknown'
  if (props.healthMode === 'cache') return bucket.metrics.cache_rate_available === false ? 'unknown' : bucket.health.cache || 'unknown'
  return bucket.health.overall
}
function healthLabel(state: HealthState, hasData: boolean) {
  return t(`channelMonitorV2.availability.${state === 'unknown' ? hasData ? 'insufficient' : 'noData' : state}`)
}
function healthTextClass(state?: HealthState) { return props.stale ? 'metric-stale' : `metric-${state || 'unknown'}` }
function isFavorite(row: MonitorMatrixRow) { return props.favoriteKeys.includes(JSON.stringify([row.platform, row.model || '', row.group_id ?? null])) }
function generationValue(row: MonitorMatrixRow) {
  if (row.health.generation_latency_ms != null) return { value: row.health.generation_latency_ms, percentile: row.health.generation_percentile?.toUpperCase() || '' }
  const duration = row.metrics.duration
  if (duration.p95_ms != null) return { value: duration.p95_ms, percentile: 'P95' }
  if (duration.p90_ms != null) return { value: duration.p90_ms, percentile: 'P90' }
  if (duration.p50_ms != null) return { value: duration.p50_ms, percentile: 'P50' }
  return { value: duration.avg_ms, percentile: duration.avg_ms != null ? 'AVG' : '' }
}
function successRate(metrics: MonitorMetric) {
  return metrics.success_rate == null ? formatMonitorSuccessRateFromError(metrics.error_rate) : formatMonitorPercent(metrics.success_rate)
}
function cacheValue(metrics: MonitorMetric, hasData: boolean) {
  return !hasData || metrics.cache_rate_available === false ? '—' : formatMonitorPercent(metrics.cache_rate)
}
function latencyDetail(metric: LatencyMetric) {
  const detail = formatLatencyPrivacy(metric.p50_ms, metric.p90_ms, metric.avg_ms, metric.p95_ms)
  return detail === '-' ? '—' : detail
}
function formatTime(timestamp: number) {
  if (!Number.isFinite(timestamp)) return '—'
  return new Intl.DateTimeFormat(locale.value || undefined, { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(timestamp))
}
function bucketTooltip(start: number, bucket?: MonitorMatrixBucket, isImage = false) {
  const time = `${formatTime(Math.max(start, Date.parse(props.coverage.requested_start)))} – ${formatTime(Math.min(start + bucketStep.value, rangeEnd.value))}`
  if (!bucket || !bucketHasMetrics(bucket)) return t('channelMonitorV2.cards.noDataAt', { time })
  const lines = [
    time,
    healthLabel(bucketState(bucket, isImage), true),
    t('channelMonitorV2.metrics.successRateValue', { value: successRate(bucket.metrics) }),
    t('channelMonitorV2.metrics.cacheRateValue', { value: cacheValue(bucket.metrics, true) }),
    t('channelMonitorV2.metrics.ttftValue', { value: latencyDetail(bucket.metrics.ttft) }),
    t('channelMonitorV2.metrics.durationValue', { value: latencyDetail(bucket.metrics.duration) }),
  ]
  if (props.showThroughput) {
    if (!isImage) lines.push(t('channelMonitorV2.metrics.tpsValue', { value: formatMonitorTokensPerSecond(bucket.metrics.tpm) }))
    lines.push(t('channelMonitorV2.metrics.rpmValue', { value: formatMonitorThroughput(bucket.metrics.rpm) }))
  }
  if (props.stale) lines.unshift(t('channelMonitorV2.availability.staleStatus'), t('channelMonitorV2.availability.historical'))
  return lines.join('\n')
}
function selectedIndex(entry: CardEntry) { return Math.min(Math.max(0, timelineIndices[entry.key] ?? entry.slots.length - 1), Math.max(0, entry.slots.length - 1)) }
function activateTimeline(entry: CardEntry, index = selectedIndex(entry)) {
  timelineIndices[entry.key] = index
  activeTimeline.value = entry.key
}
function leaveTimeline(event: MouseEvent) { if (document.activeElement !== event.currentTarget) activeTimeline.value = null }
function selectBucket(event: MouseEvent, entry: CardEntry, index: number) {
  activateTimeline(entry, index)
  ;(event.currentTarget as HTMLElement).parentElement?.focus()
  inspectBucket(entry, index)
}
function inspectBucket(entry: CardEntry, index: number) {
  const slot = entry.slots[index]
  if (!slot) return
  emit('inspect', { row: entry.row, bucketStart: new Date(Math.max(slot.start, Date.parse(props.coverage.requested_start))).toISOString(), bucketEnd: new Date(Math.min(slot.start + bucketStep.value, rangeEnd.value)).toISOString() })
}
function navigateTimeline(event: KeyboardEvent, entry: CardEntry) {
  const current = selectedIndex(entry)
  if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); inspectBucket(entry, current); return }
  const next = event.key === 'ArrowRight' ? current + 1 : event.key === 'ArrowLeft' ? current - 1 : event.key === 'Home' ? 0 : event.key === 'End' ? entry.slots.length - 1 : null
  if (event.key === 'Escape') { activeTimeline.value = null; event.preventDefault(); return }
  if (next == null) return
  event.preventDefault()
  activateTimeline(entry, Math.min(Math.max(0, next), Math.max(0, entry.slots.length - 1)))
}
</script>

<style scoped>
.channel-cards { min-width: 0; container-type: inline-size; }
.model-list { display: grid; min-width: 0; gap: 1rem; }
.model-card { min-width: 0; border: 1px solid #e9e9e9; border-radius: 14px; background: #fff; box-shadow: 0 4px 14px rgb(0 0 0 / 3%); }
.model-heading { display: flex; min-width: 0; align-items: center; gap: 0.75rem; padding: 1rem 1.25rem; }
.provider-icon { display: inline-grid; width: 2.5rem; height: 2.5rem; flex: none; place-items: center; border: 1px solid #e5e5e5; border-radius: 12px; color: #171717; }
.model-name { overflow-wrap: anywhere; color: #171717; font-size: 0.875rem; font-weight: 650; line-height: 1.5; }
.image-model-badge { display: inline-block; margin-inline-start: 0.5rem; border: 1px solid #e5e5e5; border-radius: 5px; padding: 0.0625rem 0.375rem; color: #737373; font-size: 0.625rem; font-weight: 500; vertical-align: middle; white-space: nowrap; }
.provider-name { margin-top: 0.125rem; color: #737373; font-size: 0.75rem; }
.group-count { margin-left: auto; flex: none; border: 1px solid #e5e5e5; border-radius: 999px; padding: 0.125rem 0.5rem; color: #737373; font-size: 0.6875rem; }
.metric-grid { display: grid; min-width: 0; grid-template-columns: minmax(90px, 1fr) 92px minmax(180px, 4fr) 86px minmax(120px, 1fr); column-gap: 1.25rem; align-items: center; padding-right: 1.25rem; padding-left: 1.25rem; }
.column-headings { border-top: 1px solid #ececec; padding-top: 0.5rem; padding-bottom: 0.5rem; color: #737373; font-size: 0.75rem; }
.group-row { border-top: 1px solid #ececec; padding-top: 1rem; padding-bottom: 1rem; }
.group-row > div { min-width: 0; }
.group-link { display: inline-flex; max-width: 100%; align-items: center; gap: 0.5rem; border-radius: 4px; color: #0284c7; text-align: left; font-size: 0.8125rem; line-height: 1.5; }
.group-link > span, .group-actions { display: flex; gap: 0.25rem; margin-top: 0.375rem; }
.row-action { display: inline-grid; height: 1.625rem; width: 1.625rem; flex: none; place-items: center; border-radius: 5px; color: #737373; font-size: 1.125rem; }
.row-action:hover { background: #f5f5f5; color: #171717; }
.row-action:focus-visible { outline: 2px solid #737373; outline-offset: 2px; }
.row-action.is-favorite { color: #d97706; }
.stale-note { color: #737373; font-size: 0.75rem; }
.group-name { overflow-wrap: anywhere; }
.group-link svg { flex: none; color: #737373; }
.group-link:hover { color: #0369a1; text-decoration: underline; }
.group-link:focus-visible { outline: 2px solid #737373; outline-offset: 4px; }
.group-name { color: #525252; font-size: 0.8125rem; }
.success-cell, .cache-cell, .latency-cell { display: flex; align-items: flex-start; flex-direction: column; gap: 0.375rem; }
.metric-value { color: #171717; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; font-size: 0.875rem; font-weight: 600; font-variant-numeric: tabular-nums; overflow-wrap: anywhere; }
.success-cell .metric-value { font-size: 1rem; }
.metric-stale { color: #737373; }
.metric-healthy { color: #059669; }
.metric-warning { color: #d97706; }
.metric-critical { color: #e11d48; }
.health-badge { display: inline-flex; max-width: 100%; align-items: center; gap: 0.375rem; border-radius: 999px; padding: 0.1875rem 0.5rem; background: #f5f5f5; color: #737373; font-size: 0.6875rem; line-height: 1.5; }
.status-dot { width: 0.3125rem; height: 0.3125rem; flex: none; border-radius: 50%; background: currentColor; }
.status-healthy { background: #ecfdf5; color: #059669; }
.status-warning { background: #fffbeb; color: #d97706; }
.status-critical { background: #fff1f2; color: #e11d48; }
.generation-primary { display: flex; align-items: baseline; gap: 0.375rem; min-width: 0; flex-wrap: wrap; }
.latency-secondary { color: #737373; font-size: 0.625rem; line-height: 1.6; overflow-wrap: anywhere; }
.timeline-track { position: relative; display: grid; height: 24px; gap: 2px; outline-offset: 4px; }
.timeline-track:focus-visible { border-radius: 2px; outline: 2px solid #737373; }
.timeline-bucket { min-width: 0; border-radius: 1px; }
.bucket-selected { box-shadow: inset 0 0 0 2px rgb(0 0 0 / 35%); }
.health-healthy { background: #00bd83; }
.health-warning { background: #ff9900; }
.health-critical { background: #fa2d44; }
.health-unknown { background: #e5e5e5; }
.timeline-axis { display: flex; justify-content: space-between; gap: 0.5rem; margin-top: 0.5rem; color: #737373; font-size: 0.5625rem; font-variant-numeric: tabular-nums; }
.timeline-axis time:last-child { text-align: right; }
.bucket-tooltip { position: absolute; z-index: 40; right: 0; bottom: calc(100% + 10px); width: min(100%, 26rem); min-width: min(16rem, 100%); border: 1px solid #404040; border-radius: 10px; background: #171717; padding: 0.75rem; color: #fff; font-size: 0.6875rem; line-height: 1.75; overflow-wrap: anywhere; white-space: pre-line; pointer-events: none; box-shadow: 0 6px 20px rgb(0 0 0 / 15%); }
.mobile-label { display: none; }
.skeleton-row { border-top: 1px solid #ececec; padding: 1.5rem 1.25rem; }
.dark .model-card { border-color: #404040; background: #262626; }
.dark .image-model-badge { border-color: #525252; color: #a3a3a3; }
.dark .provider-icon { border-color: #525252; color: #e5e5e5; }
.dark .model-name, .dark .metric-value { color: #e5e5e5; }
.dark .provider-name, .dark .column-headings, .dark .group-name, .dark .latency-secondary, .dark .timeline-axis { color: #a3a3a3; }
.dark .group-count { border-color: #525252; color: #a3a3a3; }
.dark .group-row, .dark .column-headings, .dark .skeleton-row { border-color: #404040; }
.dark .metric-stale { color: #a3a3a3; }
.dark .row-action:hover { background: #404040; color: #f5f5f5; }
.dark .stale-note { color: #a3a3a3; }
.dark .metric-healthy { color: #34d399; }
.dark .metric-warning { color: #fbbf24; }
.dark .metric-critical { color: #fb7185; }
.dark .health-badge { background: #404040; color: #a3a3a3; }
.dark .status-healthy { background: rgb(16 185 129 / 15%); color: #34d399; }
.dark .status-warning { background: rgb(245 158 11 / 15%); color: #fbbf24; }
.dark .status-critical { background: rgb(244 63 94 / 15%); color: #fb7185; }
.dark .health-unknown { background: #525252; }
.dark .bucket-selected { box-shadow: inset 0 0 0 2px rgb(255 255 255 / 80%); }
@container (max-width: 959px) {
  .metric-grid { grid-template-columns: minmax(90px, 1fr) 90px minmax(170px, 3fr) 80px minmax(95px, 1fr); column-gap: 0.75rem; }
}
@container (max-width: 759px) {
  .column-headings { display: none; }
  .group-row { grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) minmax(0, 1fr); gap: 1rem 0.75rem; }
  .group-cell { grid-column: 1 / -1; }
  .history-cell { grid-column: 1 / -1; grid-row: 3; }
  .success-cell { grid-column: 1; grid-row: 2; }
  .cache-cell { grid-column: 2; grid-row: 2; }
  .latency-cell { grid-column: 3; grid-row: 2; }
  .mobile-label { display: block; margin-bottom: 0.125rem; color: #737373; font-size: 0.625rem; line-height: 1.5; }
  .group-cell > .mobile-label { display: none; }
  .model-heading, .metric-grid { padding-right: 1rem; padding-left: 1rem; }
  .dark .mobile-label { color: #a3a3a3; }
}
@container (max-width: 359px) {
  .group-row { gap: 0.875rem 0.5rem; }
  .success-cell .metric-value { font-size: 0.875rem; }
  .metric-value { font-size: 0.75rem; }
}
</style>
