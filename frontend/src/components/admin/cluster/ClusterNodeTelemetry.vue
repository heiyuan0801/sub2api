<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Activity, CheckCircle2, CircleHelp, Clock3, XCircle } from '@lucide/vue'
import type { ClusterNode } from '@/api/admin/cluster'
import ClusterPerformance from './ClusterPerformance.vue'
import ClusterResourceMeter from './ClusterResourceMeter.vue'
import { createClusterText } from './clusterText'

const props = defineProps<{ node: ClusterNode; haEnabled: boolean; now: number }>()
const { t } = useI18n()
const clusterText = createClusterText(t)
const resources = computed(() => props.node.stats.resources)
const age = (value?: string) => {
  const seconds = value ? (props.now - Date.parse(value)) / 1000 : Number.NaN
  return seconds < -5 ? Number.NaN : Math.max(0, seconds)
}
const heartbeatAge = computed(() => age(props.node.heartbeat_at))
const fresh = computed(() => props.node.online && heartbeatAge.value < 20)
const resourceFresh = computed(() => fresh.value && age(resources.value?.sampled_at) < 20)
const noData = computed(() => t('cluster.noData'))
const bytes = (value?: number | null) => {
  if (value == null || !Number.isFinite(value) || value < 0) return noData.value
  if (value >= 1024 ** 3) return `${(value / 1024 ** 3).toFixed(1)} GiB`
  if (value >= 1024 ** 2) return `${(value / 1024 ** 2).toFixed(1)} MiB`
  if (value >= 1024) return `${(value / 1024).toFixed(1)} KiB`
  return `${value} B`
}
const count = (value?: number) => fresh.value && value != null ? value.toLocaleString() : '—'
function duration(seconds?: number | null) {
  if (seconds == null || !Number.isFinite(seconds)) return noData.value
  const value = Math.max(0, Math.floor(seconds))
  if (value < 60) return t('cluster.seconds', { value })
  if (value < 3600) return t('cluster.minutes', { value: Math.floor(value / 60) })
  if (value < 86400) return t('cluster.hours', { value: Math.floor(value / 3600) })
  return t('cluster.daysHours', { days: Math.floor(value / 86400), hours: Math.floor(value % 86400 / 3600) })
}
const cpuDetail = computed(() => resources.value ? t('cluster.cpuCores', { value: resources.value.cpu_cores }) : '')
const memoryDetail = computed(() => resourceFresh.value && resources.value?.memory_total_bytes ? `${bytes(resources.value.memory_used_bytes)} / ${bytes(resources.value.memory_total_bytes)}` : '')
const diskPercent = computed(() => fresh.value && props.node.stats.disk_total_bytes > 0 ? 100 * (1 - props.node.stats.disk_free_bytes / props.node.stats.disk_total_bytes) : null)
const diskDetail = computed(() => diskPercent.value !== null ? t('cluster.diskAvailable', { free: bytes(props.node.stats.disk_free_bytes), total: bytes(props.node.stats.disk_total_bytes) }) : '')
const loadText = computed(() => resourceFresh.value ? [resources.value?.load_1, resources.value?.load_5, resources.value?.load_15].map(v => v == null ? '—' : v.toFixed(2)).join(' / ') : '—')
type LinkState = 'connected' | 'disconnected' | 'unconfirmed' | 'notApplicable'
const state = (ok?: boolean): LinkState => !fresh.value || ok == null ? 'unconfirmed' : ok ? 'connected' : 'disconnected'
const latency = (value?: number) => fresh.value && value != null ? `${value.toFixed(1)} ms` : ''
const links = computed(() => [
  { name: 'heartbeat', state: !Number.isFinite(heartbeatAge.value) ? 'unconfirmed' : fresh.value ? 'connected' : 'disconnected', detail: Number.isFinite(heartbeatAge.value) ? t('cluster.ago', { value: duration(heartbeatAge.value) }) : noData.value },
  { name: 'database', state: state(props.node.stats.database_ok), detail: latency(props.node.stats.database_latency_ms) },
  { name: 'redis', state: state(props.node.stats.redis_ok), detail: latency(props.node.stats.redis_latency_ms) },
  { name: 'redisReplication', state: props.node.stats.redis_role === 'master' && !props.haEnabled && !props.node.stats.redis_replicas ? 'notApplicable' : state(props.node.stats.redis_ok ? (props.node.stats.redis_role === 'master' ? props.node.stats.redis_replicas > 0 : props.node.stats.redis_link_ok) : undefined), detail: fresh.value && props.node.stats.redis_role ? `${clusterText.label('redisRoles', props.node.stats.redis_role)} · ${t('cluster.redisReplicaCount', { value: props.node.stats.redis_replicas })}` : '' },
  { name: 'lease', state: props.haEnabled ? state(props.node.stats.lease_ok) : 'notApplicable', detail: '' },
  { name: 'ingress', state: props.haEnabled ? state(props.node.stats.ingress_ok) : 'notApplicable', detail: '' }
])
const statusClass = (value: string) => value === 'connected' ? 'text-emerald-700 dark:text-emerald-400' : value === 'disconnected' ? 'text-red-600 dark:text-red-400' : 'text-muted-foreground'
</script>

<template>
  <section :aria-label="t('cluster.resources')" class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
      <span class="flex min-w-0 items-center gap-2"><Activity :size="14" class="shrink-0" /><span class="break-all">{{ resources?.hostname || t('cluster.resources') }}</span><span v-if="resources" class="shrink-0">{{ resources.os }} / {{ resources.arch }}</span></span>
      <span v-if="!fresh" class="font-medium text-amber-700 dark:text-amber-400" role="status">{{ t('cluster.telemetryStale') }}</span>
      <span v-else-if="resources && !resourceFresh" role="status">{{ t('cluster.resourceStale') }}</span>
      <span v-else>{{ t('cluster.samplingInterval') }}</span>
    </div>
    <div class="grid gap-3 sm:grid-cols-3">
      <ClusterResourceMeter :label="t('cluster.cpuUsage')" :value="resourceFresh ? resources?.cpu_percent : null" :detail="cpuDetail" />
      <ClusterResourceMeter :label="t('cluster.systemMemory')" :value="resourceFresh ? resources?.memory_percent : null" :detail="memoryDetail" />
      <ClusterResourceMeter :label="t('cluster.diskUsage')" :value="diskPercent" :detail="diskDetail" />
    </div>
    <div class="flex flex-wrap justify-between gap-2 text-xs text-muted-foreground">
      <span>{{ t('cluster.loadAverage') }} <span class="font-mono text-foreground">{{ loadText }}</span></span>
      <span :title="resources?.sampled_at ? new Date(resources.sampled_at).toLocaleString() : undefined">{{ t('cluster.sampleAge') }} {{ duration(age(resources?.sampled_at)) }}</span>
    </div>
    <div class="rounded-lg border border-border">
      <h3 class="border-b border-border px-3 py-2.5 text-xs font-medium">{{ t('cluster.connections') }}</h3>
      <dl class="grid gap-3 p-3 sm:grid-cols-2">
        <div v-for="link in links" :key="link.name" class="flex items-center justify-between gap-2 text-xs" :data-connection="link.name">
          <dt class="text-muted-foreground">{{ t(`cluster.${link.name}`) }}</dt>
          <dd class="text-right" :class="statusClass(link.state)">
            <span class="inline-flex items-center gap-1"><CheckCircle2 v-if="link.state === 'connected'" :size="12" /><XCircle v-else-if="link.state === 'disconnected'" :size="12" /><CircleHelp v-else :size="12" />{{ t(`cluster.${link.state}`) }}</span>
            <span v-if="link.detail" class="ml-1.5 text-muted-foreground">{{ link.detail }}</span>
          </dd>
        </div>
      </dl>
    </div>
    <dl class="grid grid-cols-2 gap-x-5 gap-y-2 text-xs">
      <dt class="text-muted-foreground">{{ t('cluster.processMemory') }}</dt><dd>{{ resourceFresh ? bytes(resources?.process_rss_bytes) : '—' }} <span class="text-muted-foreground">/ {{ t('cluster.heap') }} {{ fresh ? bytes(node.stats.heap_bytes) : '—' }}</span></dd>
      <dt class="text-muted-foreground">{{ t('cluster.goroutines') }}</dt><dd>{{ count(node.stats.goroutines) }}</dd>
      <dt class="text-muted-foreground">{{ t('cluster.requestCounters') }}</dt><dd>{{ count(node.stats.requests) }} / {{ count(node.stats.errors) }} / {{ count(node.stats.rejected) }}</dd>
      <dt class="text-muted-foreground">{{ t('cluster.replicas') }}</dt><dd>{{ fresh && node.stats.replication_known ? `${node.stats.sync_replicas} / ${node.stats.replication_peers}` : '—' }}</dd>
      <dt class="text-muted-foreground">{{ t('cluster.lag') }}</dt><dd>{{ fresh && node.stats.replication_known ? bytes(node.stats.replication_lag_bytes) : '—' }}</dd>
      <dt class="text-muted-foreground">{{ t('cluster.systemUptime') }}</dt><dd>{{ resourceFresh ? duration(resources?.uptime_seconds) : '—' }}</dd>
      <dt class="text-muted-foreground">{{ t('cluster.appUptime') }}</dt><dd class="flex items-center gap-1"><Clock3 :size="12" />{{ fresh ? duration(age(node.started_at)) : '—' }}</dd>
      <dt class="text-muted-foreground">{{ t('cluster.version') }}</dt><dd class="break-all">{{ node.version }}</dd>
    </dl>
    <ClusterPerformance :node="node" :now="now" />
    <p class="text-[11px] leading-5 text-muted-foreground">{{ t('cluster.resourceScope') }}</p>
  </section>
</template>
