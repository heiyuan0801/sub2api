<template>
  <section class="channel-cards" :aria-label="t('channelMonitorV2.cards.aria')">
    <div v-if="loading && rows.length === 0" class="channel-grid" aria-hidden="true">
      <div v-for="index in 4" :key="index" class="channel-card animate-pulse">
        <div class="flex items-start gap-3">
          <div class="h-10 w-10 shrink-0 rounded-xl bg-gray-100 dark:bg-dark-700"></div>
          <div class="flex-1 space-y-2 pt-1">
            <div class="h-4 w-3/4 rounded bg-gray-100 dark:bg-dark-700"></div>
            <div class="h-3 w-1/3 rounded bg-gray-100 dark:bg-dark-700"></div>
          </div>
        </div>
        <div class="mt-6 grid grid-cols-3 gap-2">
          <div v-for="metric in 3" :key="metric" class="h-[76px] rounded-2xl bg-gray-50 dark:bg-dark-900"></div>
        </div>
        <div class="mt-auto h-5 rounded bg-gray-100 dark:bg-dark-700"></div>
      </div>
    </div>

    <EmptyState
      v-else-if="rows.length === 0"
      :title="t('channelMonitorV2.matrix.emptyTitle')"
      :description="t('channelMonitorV2.empty.description')"
    />

    <div v-else class="space-y-7">
      <section v-for="group in platformGroups" :key="group.platform" class="space-y-3">
        <h2 class="flex min-w-0 items-center gap-3 text-lg font-bold text-gray-900 dark:text-gray-100">
          <span class="provider-icon" :class="providerGradient(group.platform)">
            <ProviderIcon :provider="group.platform" :size="21" />
          </span>
          <span class="truncate">{{ providerLabel(group.platform) }}</span>
          <span class="platform-count">{{ group.entries.length }}</span>
        </h2>

        <div class="channel-grid">
          <article v-for="entry in group.entries" :key="rowKey(entry.row)" class="channel-card">
            <div class="flex min-w-0 items-start gap-3">
              <span class="provider-icon shrink-0" :class="providerGradient(entry.row.platform)">
                <ProviderIcon :provider="entry.row.platform" :size="20" />
              </span>
              <div class="min-w-0 flex-1">
                <h3 class="truncate text-base font-semibold text-gray-900 dark:text-gray-100" :title="cardName(entry.row)">
                  {{ cardName(entry.row) }}
                </h3>
                <div class="mt-1 flex min-w-0 flex-wrap items-center gap-1.5">
                  <span class="provider-badge" :class="providerBadgeClass(entry.row.platform)">
                    {{ providerLabel(entry.row.platform) }}
                  </span>
                  <span
                    v-if="cardModel(entry.row)"
                    class="min-w-0 truncate font-mono text-[10px] text-gray-500 dark:text-gray-400"
                    :title="cardModel(entry.row)"
                  >
                    {{ cardModel(entry.row) }}
                  </span>
                </div>
              </div>
              <span class="health-badge shrink-0" :class="healthBadgeClass(entry.row.health.overall)">
                {{ healthLabel(entry.row.health.overall) }}
              </span>
            </div>

            <dl class="channel-metrics" :title="t('channelMonitorV2.cards.metricsHint')">
              <div class="channel-metric">
                <dt>{{ t('channelMonitorV2.metrics.cacheRate') }}</dt>
                <dd>{{ hasMetrics(entry.row) ? formatMonitorPercent(entry.row.metrics.cache_rate) : '-' }}</dd>
              </div>
              <div class="channel-metric">
                <dt>{{ t('channelMonitorV2.cards.availability') }}</dt>
                <dd :class="metricTextClass(entry.row.health.error_rate)">
                  {{ hasMetrics(entry.row) ? formatMonitorSuccessRateFromError(entry.row.metrics.error_rate) : '-' }}
                </dd>
              </div>
              <div class="channel-metric" :title="latencyDetail(entry.row.metrics.ttft)">
                <dt>{{ t('channelMonitorV2.metrics.ttftP50') }}</dt>
                <dd :class="metricTextClass(ttftDisplayState(entry.row.health.ttft, entry.row.metrics.ttft))">
                  {{ formatMonitorMs(entry.row.metrics.ttft.p50_ms) }}
                </dd>
              </div>
            </dl>

            <div v-if="showThroughput && hasMetrics(entry.row)" class="mt-3 flex flex-wrap justify-between gap-2 text-[11px] text-gray-500 dark:text-gray-400">
              <span>{{ t('channelMonitorV2.metrics.tpsValue', { value: formatMonitorTokensPerSecond(entry.row.metrics.tpm) }) }}</span>
              <span>{{ t('channelMonitorV2.metrics.rpmValue', { value: formatMonitorThroughput(entry.row.metrics.rpm) }) }}</span>
            </div>

            <div class="channel-history">
              <div class="mb-2 flex flex-wrap items-center justify-between gap-1 text-[10px] font-medium text-gray-400 dark:text-gray-500">
                <span>{{ t('channelMonitorV2.cards.intervals', { count: bucketStarts.length }) }}</span>
                <span>{{ bucketLabel }}</span>
              </div>
              <div class="timeline-track" :style="{ gridTemplateColumns: `repeat(${Math.max(1, bucketStarts.length)}, minmax(0, 1fr))` }">
                <span
                  v-for="slot in entry.slots"
                  :key="slot.start"
                  class="timeline-bucket"
                  :class="slot.bucket ? healthScoreClass(slot.bucket.health, healthMode, slot.bucket.metrics.request_count) : 'health-unknown'"
                  tabindex="0"
                  role="img"
                  :title="slot.tooltip"
                  :aria-label="slot.tooltip"
                >
                  <span class="bucket-tooltip" aria-hidden="true">{{ slot.tooltip }}</span>
                </span>
              </div>
              <div class="mt-1.5 flex justify-between text-[9px] tracking-wider text-gray-400 dark:text-gray-500">
                <span>{{ t('channelMonitorV2.cards.past') }}</span>
                <span>{{ t('channelMonitorV2.cards.now') }}</span>
              </div>
            </div>
          </article>
        </div>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type {
  HealthState,
  LatencyMetric,
  MonitorCoverage,
  MonitorMatrixBucket,
  MonitorMatrixRow,
} from '@/api/channelMonitorV2'
import EmptyState from '@/components/common/EmptyState.vue'
import ProviderIcon from '@/components/user/monitor/ProviderIcon.vue'
import { providerGradient, useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'
import {
  formatLatencyPrivacy,
  formatMonitorMs,
  formatMonitorPercent,
  formatMonitorSuccessRateFromError,
  formatMonitorThroughput,
  formatMonitorTokensPerSecond,
  healthModeScore,
  healthScoreClass,
  ttftDisplayState,
} from './monitorFormat'

const props = withDefaults(defineProps<{
  rows: MonitorMatrixRow[]
  coverage: MonitorCoverage
  healthMode: 'overall' | 'success' | 'ttft' | 'cache'
  showThroughput?: boolean
  loading?: boolean
}>(), {
  showThroughput: true,
  loading: false,
})

const { t, locale } = useI18n()
const { providerLabel, providerBadgeClass } = useChannelMonitorFormat()

interface TimelineSlot {
  start: number
  bucket?: MonitorMatrixBucket
  tooltip: string
}
interface CardEntry {
  row: MonitorMatrixRow
  slots: TimelineSlot[]
}

const bucketStep = computed(() => Math.max(60, Number(props.coverage.bucket_seconds) || 60) * 1000)
const bucketStarts = computed(() => {
  const start = Date.parse(props.coverage.requested_start)
  const requestedEnd = Date.parse(props.coverage.requested_end || '')
  const end = Number.isFinite(requestedEnd) && requestedEnd > start
    ? requestedEnd
    : Date.parse(props.coverage.data_through)
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return []
  const starts: number[] = []
  // Keep the complete selected window, including uncovered intervals, as in V2's matrix.
  for (let cursor = Math.floor(start / bucketStep.value) * bucketStep.value; cursor < end; cursor += bucketStep.value) {
    starts.push(cursor)
  }
  return starts
})

const platformGroups = computed(() => {
  const groups = new Map<string, { platform: string; entries: CardEntry[] }>()
  for (const row of props.rows) {
    let group = groups.get(row.platform)
    if (!group) {
      group = { platform: row.platform, entries: [] }
      groups.set(row.platform, group)
    }
    const buckets = new Map((row.buckets || []).map(bucket => [Date.parse(bucket.bucket_start), bucket]))
    group.entries.push({
      row,
      slots: bucketStarts.value.map(start => {
        const bucket = buckets.get(start)
        return { start, bucket, tooltip: bucketTooltip(start, bucket) }
      }),
    })
  }
  return [...groups.values()]
})

const bucketLabel = computed(() => {
  const minutes = bucketStep.value / 60000
  if (minutes < 60) return t('channelMonitorV2.bucket.minutes', { count: minutes })
  if (minutes < 1440) return t('channelMonitorV2.bucket.hours', { count: minutes / 60 })
  return t('channelMonitorV2.bucket.days', { count: minutes / 1440 })
})

function rowKey(row: MonitorMatrixRow) {
  return [row.platform, row.group_id || 0, row.model || ''].join(':')
}
function hasMetrics(row: MonitorMatrixRow) {
  // Absolute counts are redacted for users; actual buckets remain available.
  return Boolean(row.buckets?.length)
}
function modelLabel(model: string) {
  return model === '__other__' ? t('channelMonitorV2.otherModels') : model
}
function groupName(row: MonitorMatrixRow) {
  return row.group_name || (row.group_id ? t('channelMonitorV2.cards.nameFallback', { id: row.group_id }) : '')
}
function cardName(row: MonitorMatrixRow) {
  return groupName(row) || (row.model ? modelLabel(row.model) : providerLabel(row.platform))
}
function cardModel(row: MonitorMatrixRow) {
  return groupName(row) && row.model ? modelLabel(row.model) : ''
}
function healthLabel(state: HealthState) {
  return t(`channelMonitorV2.cards.${state === 'healthy' || state === 'warning' || state === 'critical' ? state : 'unknown'}`)
}
function healthBadgeClass(state: HealthState) {
  if (state === 'healthy') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
  if (state === 'warning') return 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
  if (state === 'critical') return 'bg-red-100 text-red-700 dark:bg-red-500/15 dark:text-red-300'
  return 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-gray-400'
}
function metricTextClass(state?: HealthState) {
  if (state === 'healthy') return 'text-emerald-700 dark:text-emerald-400'
  if (state === 'warning') return 'text-amber-700 dark:text-amber-400'
  if (state === 'critical') return 'text-red-600 dark:text-red-400'
  return 'text-gray-900 dark:text-gray-100'
}
function latencyDetail(metric: LatencyMetric) {
  return formatLatencyPrivacy(metric.p50_ms, metric.p90_ms, metric.avg_ms, metric.p95_ms)
}
function formatTime(timestamp: number) {
  return new Intl.DateTimeFormat(locale.value || undefined, {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
  }).format(new Date(timestamp))
}
function bucketTooltip(start: number, bucket?: MonitorMatrixBucket) {
  const time = `${formatTime(start)} – ${formatTime(start + bucketStep.value)}`
  if (!bucket) return t('channelMonitorV2.cards.noDataAt', { time })
  const score = healthModeScore(bucket.health, props.healthMode)
  const lines = [
    time,
    t('channelMonitorV2.matrix.scoreLine', { score: score == null ? '—' : Math.round(score) }),
    t('channelMonitorV2.metrics.successRateValue', { value: formatMonitorSuccessRateFromError(bucket.metrics.error_rate) }),
    t('channelMonitorV2.metrics.ttftValue', { value: latencyDetail(bucket.metrics.ttft) }),
    t('channelMonitorV2.metrics.cacheRateValue', { value: formatMonitorPercent(bucket.metrics.cache_rate) }),
    t('channelMonitorV2.metrics.errorRateValue', { value: formatMonitorPercent(bucket.metrics.error_rate) }),
  ]
  if (props.showThroughput) {
    lines.push(
      t('channelMonitorV2.metrics.tpsValue', { value: formatMonitorTokensPerSecond(bucket.metrics.tpm) }),
      t('channelMonitorV2.metrics.rpmValue', { value: formatMonitorThroughput(bucket.metrics.rpm) }),
    )
  }
  lines.push(t('channelMonitorV2.metrics.durationValue', { value: latencyDetail(bucket.metrics.duration) }))
  return lines.join('\n')
}
</script>

<style scoped>
.channel-cards { container-type: inline-size; }
.channel-grid { display: grid; grid-template-columns: minmax(0, 1fr); gap: 1.25rem; }
.channel-card {
  display: flex;
  min-width: 0;
  min-height: 280px;
  flex-direction: column;
  border: 1px solid #e5e5e5;
  border-radius: 24px;
  background: #fff;
  padding: 1.25rem;
  box-shadow: 0 8px 24px rgb(0 0 0 / 3%);
}
.provider-icon {
  display: inline-grid;
  width: 2.5rem;
  height: 2.5rem;
  flex: none;
  place-items: center;
  border-radius: 0.875rem;
  color: #404040;
  box-shadow: inset 0 0 0 1px rgb(0 0 0 / 4%);
}
.platform-count {
  display: inline-grid;
  min-width: 1.75rem;
  height: 1.5rem;
  flex: none;
  place-items: center;
  border: 1px solid #e5e5e5;
  border-radius: 9999px;
  background: #fff;
  padding: 0 0.375rem;
  color: #737373;
  font-size: 0.75rem;
  font-weight: 600;
}
.provider-badge { border-radius: 5px; padding: 2px 6px; font-size: 10px; font-weight: 500; }
.health-badge { border-radius: 9999px; padding: 4px 9px; font-size: 11px; font-weight: 600; }
.channel-metrics { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.5rem; margin-top: 1.5rem; }
.channel-metric { min-width: 0; border: 1px solid #e5e5e5; border-radius: 16px; background: #fafafa; padding: 0.75rem; }
.channel-metric dt { overflow: hidden; color: #a3a3a3; font-size: 10px; line-height: 1.5; text-overflow: ellipsis; white-space: nowrap; }
.channel-metric dd { margin-top: 0.5rem; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; font-size: clamp(0.875rem, 1.25vw, 1.125rem); font-weight: 700; font-variant-numeric: tabular-nums; }
.channel-history { margin-top: auto; padding-top: 1.5rem; }
.timeline-track { position: relative; display: grid; height: 20px; gap: 3px; }
.timeline-bucket { min-width: 0; border-radius: 3px; outline-offset: 3px; }
.timeline-bucket:focus-visible { z-index: 10; outline: 2px solid #737373; }
.bucket-tooltip { display: none; position: absolute; z-index: 30; bottom: calc(100% + 10px); right: 0; left: 0; border-radius: 10px; background: #171717; padding: 10px 12px; color: #fff; font-size: 11px; line-height: 1.6; overflow-wrap: anywhere; white-space: pre-line; pointer-events: none; box-shadow: 0 6px 20px rgb(0 0 0 / 15%); }
.timeline-bucket:focus-visible .bucket-tooltip { display: block; }
.health-score10 { background: #16a34a; }
.health-score9 { background: #22c55e; }
.health-score8 { background: #4ade80; }
.health-score7 { background: #a3e635; }
.health-score6 { background: #facc15; }
.health-score5 { background: #fbbf24; }
.health-score4 { background: #f59e0b; }
.health-score3 { background: #f97316; }
.health-score2 { background: #fb7185; }
.health-score1 { background: #f87171; }
.health-score0 { background: rgb(239 67 67); }
.health-healthy { background: #22c55e; }
.health-warning { background: #f59e0b; }
.health-critical { background: #ef4444; }
.health-unknown { background: #d4d4d4; }
.dark .channel-card { border-color: #404040; background: #262626; box-shadow: 0 8px 24px rgb(0 0 0 / 10%); }
.dark .provider-icon { color: #e5e5e5; box-shadow: inset 0 0 0 1px rgb(255 255 255 / 6%); }
.dark .platform-count { border-color: #404040; background: #262626; color: #a3a3a3; }
.dark .channel-metric { border-color: #404040; background: #202020; }
.dark .channel-metric dt { color: #a3a3a3; }
.dark .health-unknown { background: #525252; }
@container (min-width: 640px) { .channel-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@container (min-width: 1040px) { .channel-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); } }
@media (min-width: 1600px) { @container (min-width: 1280px) { .channel-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); } } }
@media (max-width: 380px) { .channel-card { padding: 1rem; } .channel-metric { padding: 0.625rem; } }
</style>
