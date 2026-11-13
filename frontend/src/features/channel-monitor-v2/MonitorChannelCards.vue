<template>
  <div class="channel-cards" :aria-label="t('channelMonitorV2.cards.aria')" :aria-busy="loading">
    <EmptyState v-if="!rows.length" :title="t('channelMonitorV2.matrix.emptyTitle')" :description="emptyDescription || t('channelMonitorV2.empty.description')" />

    <section v-for="section in sections" :key="section.platform" class="provider-section">
      <h2 class="provider-heading">
        {{ providerLabel(section.platform) }}
        <span>{{ section.entries.length }}</span>
      </h2>
      <div class="channel-grid">
        <article v-for="entry in section.entries" :key="entry.key" class="channel-card">
          <header class="channel-heading">
            <span class="provider-icon" :class="providerBadgeClass(section.platform)">
              <ProviderIcon :provider="section.platform" :size="22" />
            </span>
            <div class="channel-name">
              <h3>
                <button type="button" :aria-label="t('channelMonitorV2.availability.inspect', { group: groupName(entry.row) })" @click="emit('inspect', { row: entry.row })">
                  {{ groupName(entry.row) }}
                </button>
              </h3>
              <div class="channel-tags">
                <span :class="providerBadgeClass(section.platform)">{{ providerLabel(section.platform) }}</span>
                <span v-if="entry.rate != null" class="rate-tag">{{ t('channelMonitorV2.channels.userRate', { value: formatRate(entry.rate) }) }}</span>
              </div>
            </div>
            <div class="channel-actions">
              <button v-if="entry.row.group_id" type="button" :title="t('channelMonitorV2.availability.filterGroup')" :aria-label="t('channelMonitorV2.availability.filterGroup')" @click="emit('select-group', entry.row)"><Icon name="filter" size="xs" /></button>
              <button type="button" :aria-pressed="favoriteKeys.includes(entry.key)" :aria-label="t(favoriteKeys.includes(entry.key) ? 'channelMonitorV2.availability.unfavorite' : 'channelMonitorV2.availability.favorite')" @click="emit('toggle-favorite', entry.row)">{{ favoriteKeys.includes(entry.key) ? '★' : '☆' }}</button>
            </div>
          </header>

          <dl class="channel-metrics">
            <div>
              <dt>{{ t('channelMonitorV2.availability.cacheRate') }}</dt>
              <dd>{{ cacheRate(entry.row.metrics, entry.hasData) }}</dd>
            </div>
            <div>
              <dt>{{ t('channelMonitorV2.cards.availability') }}</dt>
              <dd :class="stale || !entry.hasData ? 'tone-unknown' : `tone-${entry.row.health.success_rate ?? entry.row.health.error_rate}`">{{ availability(entry.row.metrics, entry.hasData) }}</dd>
            </div>
            <div :title="latencyDetail(entry.row.metrics.ttft)">
              <dt>{{ t('channelMonitorV2.availability.ttft') }}</dt>
              <dd>{{ firstToken(entry.row.metrics, entry.hasData) }}</dd>
            </div>
          </dl>

          <div class="timeline">
            <div class="timeline-heading">
              <span>{{ t('channelMonitorV2.cards.intervals', { count: entry.slots.length }) }}</span>
              <span v-if="stale">{{ t('channelMonitorV2.availability.staleStatus') }}</span>
              <span v-else-if="refreshSeconds > 0">{{ t('channelMonitorV2.channels.refreshIn', { seconds: refreshSeconds }) }}</span>
            </div>
            <div
              v-if="entry.slots.length"
              class="timeline-track"
              :style="{ gridTemplateColumns: `repeat(${entry.slots.length}, minmax(0, 1fr))` }"
              role="slider"
              tabindex="0"
              :aria-label="t('channelMonitorV2.channels.timeline', { group: groupName(entry.row) })"
              :aria-describedby="`${componentId}-timeline-help`"
              :aria-valuemin="1"
              :aria-valuemax="entry.slots.length"
              :aria-valuenow="selectedIndex(entry) + 1"
              :aria-valuetext="entry.slots[selectedIndex(entry)]?.tooltip"
              @focus="activate(entry)"
              @blur="activeTimeline = null"
              @mouseleave="leaveTimeline"
              @keydown="navigateTimeline($event, entry)"
            >
              <span
                v-for="(slot, index) in entry.slots"
                :key="slot.start"
                class="timeline-bucket"
                :class="[`health-${slot.state}`, { 'bucket-selected': activeTimeline === entry.key && selectedIndex(entry) === index }]"
                aria-hidden="true"
                @mouseenter="activate(entry, index)"
                @click="inspectSlot(entry, slot)"
              ></span>
              <span v-if="activeTimeline === entry.key" class="bucket-tooltip" role="tooltip">{{ entry.slots[selectedIndex(entry)]?.tooltip }}</span>
            </div>
            <div class="timeline-axis"><span>{{ t('channelMonitorV2.cards.past') }}</span><span>{{ t('channelMonitorV2.cards.now') }}</span></div>
          </div>

          <details class="channel-history" open>
            <summary>{{ t('channelMonitorV2.channels.history', { count: entry.history.length }) }}</summary>
            <div v-if="entry.history.length" class="history-list">
              <button v-for="slot in entry.history" :key="slot.start" type="button" :title="slot.tooltip" :aria-label="slot.tooltip" @click="inspectSlot(entry, slot)">
                <span class="history-dot" :class="`health-${slot.state}`" aria-hidden="true"></span><time :datetime="new Date(slot.start).toISOString()">{{ formatTime(slot.start) }}</time>
              </button>
            </div>
            <p v-else class="history-empty">{{ t('channelMonitorV2.channels.historyEmpty') }}</p>
          </details>
        </article>
      </div>
    </section>
    <p :id="`${componentId}-timeline-help`" class="sr-only">{{ t('channelMonitorV2.availability.timelineHelp') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import type { HealthState, LatencyMetric, MonitorCoverage, MonitorMatrixBucket, MonitorMatrixRow, MonitorMetric } from '@/api/channelMonitorV2'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import ProviderIcon from '@/components/user/monitor/ProviderIcon.vue'
import { useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'
import { formatLatencyPrivacy, formatMonitorMs, formatMonitorPercent, formatMonitorThroughput, formatMonitorTokensPerSecond, ttftDisplayState } from './monitorFormat'
import { monitorRowKey } from './monitorPreferences'

const props = withDefaults(defineProps<{
  rows: MonitorMatrixRow[]
  coverage: MonitorCoverage
  healthMode: 'overall' | 'success' | 'ttft' | 'cache'
  showThroughput?: boolean
  loading?: boolean
  emptyDescription?: string
  favoriteKeys?: string[]
  stale?: boolean
  groupRates?: Record<number, number>
  refreshSeconds?: number
}>(), { showThroughput: true, favoriteKeys: () => [], groupRates: () => ({}), refreshSeconds: 0 })
const emit = defineEmits<{
  'select-group': [row: MonitorMatrixRow]
  'toggle-favorite': [row: MonitorMatrixRow]
  inspect: [selection: { row: MonitorMatrixRow; bucketStart?: string; bucketEnd?: string }]
}>()
const { t, locale } = useI18n()
const { providerLabel, providerBadgeClass } = useChannelMonitorFormat()
const componentId = useId()
const activeTimeline = ref<string | null>(null)
const activeIndex = ref(0)

interface TimelineSlot { start: number; end: number; hasData: boolean; state: HealthState; tooltip: string }
interface CardEntry { key: string; row: MonitorMatrixRow; rate?: number; hasData: boolean; slots: TimelineSlot[]; history: TimelineSlot[] }

const bucketStep = computed(() => Math.max(60, Number(props.coverage.bucket_seconds) || 60) * 1000)
const rangeStart = computed(() => Date.parse(props.coverage.requested_start))
const rangeEnd = computed(() => Date.parse(props.coverage.requested_end || props.coverage.data_through))
const bucketStarts = computed(() => {
  const start = rangeStart.value
  const end = rangeEnd.value
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return []
  const step = bucketStep.value
  const last = (Math.ceil(end / step) - 1) * step
  const first = Math.max(Math.floor(start / step) * step, last - 29 * step)
  return Array.from({ length: Math.round((last - first) / step) + 1 }, (_, index) => first + index * step)
})

const sections = computed(() => {
  const byPlatform = new Map<string, CardEntry[]>()
  for (const row of props.rows) {
    const buckets = new Map(row.buckets.map(bucket => [Date.parse(bucket.bucket_start), bucket]))
    const slots = bucketStarts.value.map(start => buildSlot(start, buckets.get(start)))
    const rate = row.group_id == null ? undefined : props.groupRates[row.group_id]
    const entry: CardEntry = {
      key: monitorRowKey(row), row, slots, rate: Number.isFinite(rate) ? rate : undefined,
      hasData: row.metrics.has_data ?? (row.metrics.request_count > 0 || slots.some(slot => slot.hasData)),
      history: slots.filter(slot => slot.hasData).reverse(),
    }
    const entries = byPlatform.get(row.platform)
    if (entries) entries.push(entry)
    else byPlatform.set(row.platform, [entry])
  }
  return [...byPlatform].map(([platform, entries]) => ({ platform, entries }))
})

function groupName(row: MonitorMatrixRow) {
  return row.group_name || t('channelMonitorV2.cards.nameFallback', { id: row.group_id })
}

function availability(metric: MonitorMetric, hasData: boolean) {
  return hasData ? formatMonitorPercent(metric.success_rate ?? 1 - metric.error_rate, locale.value) : '—'
}

function cacheRate(metric: MonitorMetric, hasData: boolean) {
  return hasData && metric.cache_rate_available !== false ? formatMonitorPercent(metric.cache_rate, locale.value) : '—'
}

function firstToken(metric: MonitorMetric, hasData: boolean) {
  return hasData && metric.ttft.p50_ms != null ? formatMonitorMs(metric.ttft.p50_ms) : '—'
}

function latencyDetail(metric: LatencyMetric) {
  const detail = formatLatencyPrivacy(metric.p50_ms, metric.p90_ms, metric.avg_ms, metric.p95_ms)
  return detail === '-' ? '—' : detail
}

function formatRate(value: number) {
  return new Intl.NumberFormat(locale.value || undefined, { minimumFractionDigits: 2, maximumFractionDigits: 4 }).format(value)
}

function formatTime(value: number, includeDate = false) {
  return new Intl.DateTimeFormat(locale.value || undefined, {
    ...(includeDate ? { month: '2-digit', day: '2-digit' } : {}),
    hour: '2-digit', minute: '2-digit', hourCycle: 'h23',
  }).format(value)
}

function bucketState(bucket: MonitorMatrixBucket): HealthState {
  if (props.healthMode === 'success') return bucket.health.success_rate ?? bucket.health.error_rate
  if (props.healthMode === 'ttft') return ttftDisplayState(bucket.health.ttft, bucket.metrics.ttft) ?? 'unknown'
  if (props.healthMode === 'cache') return bucket.metrics.cache_rate_available === false ? 'unknown' : bucket.health.cache ?? 'unknown'
  return bucket.health.overall
}

function buildSlot(bucketStart: number, bucket?: MonitorMatrixBucket): TimelineSlot {
  const start = Math.max(bucketStart, rangeStart.value)
  const end = Math.min(bucketStart + bucketStep.value, rangeEnd.value)
  const time = `${formatTime(start, true)} – ${formatTime(end, true)}`
  if (!bucket || bucket.metrics.has_data === false) {
    return { start, end, hasData: false, state: 'unknown', tooltip: t('channelMonitorV2.cards.noDataAt', { time }) }
  }
  const state = bucketState(bucket)
  const metrics = bucket.metrics
  const lines = [
    time,
    t(`channelMonitorV2.cards.${state}`),
    t('channelMonitorV2.cards.availabilityValue', { value: availability(metrics, true) }),
    t('channelMonitorV2.metrics.cacheRateValue', { value: cacheRate(metrics, true) }),
    t('channelMonitorV2.metrics.ttftValue', { value: latencyDetail(metrics.ttft) }),
  ]
  if (props.showThroughput) {
    lines.push(t('channelMonitorV2.metrics.tpsValue', { value: formatMonitorTokensPerSecond(metrics.tpm) }))
    lines.push(t('channelMonitorV2.metrics.rpmValue', { value: formatMonitorThroughput(metrics.rpm) }))
  }
  return { start, end, hasData: true, state, tooltip: lines.join('\n') }
}

function selectedIndex(entry: CardEntry) {
  return activeTimeline.value === entry.key ? Math.min(activeIndex.value, entry.slots.length - 1) : entry.slots.length - 1
}

function activate(entry: CardEntry, index = selectedIndex(entry)) {
  activeIndex.value = index
  activeTimeline.value = entry.key
}

function leaveTimeline(event: MouseEvent) {
  if (document.activeElement !== event.currentTarget) activeTimeline.value = null
}

function inspectSlot(entry: CardEntry, slot: TimelineSlot) {
  emit('inspect', { row: entry.row, bucketStart: new Date(slot.start).toISOString(), bucketEnd: new Date(slot.end).toISOString() })
}

function navigateTimeline(event: KeyboardEvent, entry: CardEntry) {
  const current = selectedIndex(entry)
  const last = entry.slots.length - 1
  switch (event.key) {
    case 'ArrowRight': activate(entry, Math.min(last, current + 1)); break
    case 'ArrowLeft': activate(entry, Math.max(0, current - 1)); break
    case 'Home': activate(entry, 0); break
    case 'End': activate(entry, last); break
    case 'Enter':
    case ' ': inspectSlot(entry, entry.slots[current]); break
    case 'Escape': activeTimeline.value = null; break
    default: return
  }
  event.preventDefault()
}
</script>

<style scoped>
.channel-cards { container-type: inline-size; min-width: 0; }
.provider-section + .provider-section { margin-top: 1.65rem; }
.provider-heading { display: flex; align-items: center; gap: .55rem; margin: 0 0 .75rem .125rem; color: var(--channel-muted); font-size: .82rem; font-weight: 650; letter-spacing: .045em; text-transform: uppercase; }
.provider-heading > span { padding: .12rem .42rem; border: 1px solid var(--channel-border); border-radius: 5px; background: var(--channel-subtle); color: var(--channel-text); font-size: .7rem; }
.channel-grid { display: grid; grid-template-columns: minmax(0, 1fr); gap: 1rem; }
.channel-card { min-width: 0; padding: 1rem; border: 1px solid var(--channel-border); border-radius: 10px; background: var(--channel-surface); box-shadow: 0 1px 3px rgb(15 23 42 / 7%); }
.channel-heading { display: flex; align-items: flex-start; gap: .7rem; min-width: 0; }
.provider-icon { display: grid; flex: none; place-items: center; width: 2.55rem; height: 2.55rem; border-radius: 7px; }
.channel-name { flex: 1; min-width: 0; }
.channel-name h3 { color: var(--channel-text); font-size: 1rem; font-weight: 700; line-height: 1.4; overflow-wrap: anywhere; }
.channel-name button { text-align: left; }
.channel-name button:hover { text-decoration: underline; text-underline-offset: 3px; }
.channel-tags { display: flex; flex-wrap: wrap; gap: .35rem; margin-top: .3rem; font-size: .69rem; }
.channel-tags > span { border-radius: 4px; padding: .18rem .35rem; }
.rate-tag { background: #fdf0f4; color: #a72650; font-variant-numeric: tabular-nums; }
.dark .rate-tag { background: #412337; color: #f9adc7; }
.channel-actions { display: flex; flex: none; gap: .15rem; color: var(--channel-muted); }
.channel-actions button { display: grid; width: 1.3rem; height: 1.55rem; place-items: center; border-radius: 4px; }
.channel-actions button:hover { background: var(--channel-subtle); color: var(--channel-text); }
.channel-actions [aria-pressed="true"] { color: #b06a0f; }
.channel-card button:focus-visible, .channel-card summary:focus-visible, .timeline-track:focus-visible { outline: 2px solid var(--channel-muted); outline-offset: 3px; }
.channel-metrics { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: .35rem; margin-top: .85rem; padding: 1.2rem .15rem; border-block: 1px solid var(--channel-border); }
.channel-metrics dt { color: var(--channel-muted); font-size: .68rem; font-weight: 600; text-transform: uppercase; }
.channel-metrics dd { margin-top: .35rem; font: 700 clamp(.9rem, 1.5vw, 1.2rem) / 1.35 ui-monospace, monospace; overflow-wrap: anywhere; }
.tone-healthy { color: #047857; } .tone-warning { color: #995a0c; } .tone-critical { color: #b91c1c; } .tone-unknown { color: var(--channel-muted); }
.dark .tone-healthy { color: #6ee7b7; } .dark .tone-warning { color: #fcd34d; } .dark .tone-critical { color: #fda4af; }
.timeline { margin-top: .8rem; }
.timeline-heading, .timeline-axis { display: flex; flex-wrap: wrap; justify-content: space-between; gap: .2rem .5rem; color: var(--channel-muted); font-size: .67rem; }
.timeline-heading { font-weight: 600; }
.timeline-axis { margin-top: .25rem; text-transform: uppercase; }
.timeline-track { position: relative; display: grid; align-items: end; gap: 2px; height: 1.35rem; margin-top: .35rem; border-radius: 2px; }
.timeline-bucket { height: 100%; border-radius: 2px; cursor: pointer; }
.health-healthy { background: #00976e; } .health-warning { background: #f9c32f; } .health-critical { background: #ef4444; } .health-unknown { background: var(--channel-gap); }
.timeline-bucket.health-warning { height: 65%; } .timeline-bucket.health-critical { height: 43%; } .timeline-bucket.health-unknown { height: 22%; }
.bucket-selected { outline: 1px solid var(--channel-text); outline-offset: 1px; }
.bucket-tooltip { position: absolute; z-index: 2; inset-inline: 0; bottom: calc(100% + .5rem); width: max-content; max-width: 100%; padding: .55rem .65rem; border: 1px solid var(--channel-border); border-radius: 6px; background: var(--channel-surface); color: var(--channel-text); box-shadow: 0 3px 12px rgb(15 23 42 / 15%); font-size: .7rem; line-height: 1.55; white-space: pre-line; overflow-wrap: anywhere; pointer-events: none; }
.channel-history { margin-top: .85rem; color: var(--channel-muted); font-size: .72rem; }
.channel-history summary { width: fit-content; cursor: pointer; font-weight: 600; }
.history-list { display: flex; flex-wrap: wrap; gap: .32rem; max-height: 6rem; margin-top: .4rem; overflow: auto; padding: .2rem; }
.history-list button { display: flex; align-items: center; gap: .3rem; padding: .14rem .38rem; border: 1px solid var(--channel-border); border-radius: 999px; background: var(--channel-subtle); font-variant-numeric: tabular-nums; }
.history-dot { width: .35rem; height: .35rem; border-radius: 50%; }
.history-empty { margin-top: .45rem; }
@container (min-width: 520px) { .channel-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@container (min-width: 800px) { .channel-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); } }
@container (min-width: 1060px) { .channel-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); } }
</style>
