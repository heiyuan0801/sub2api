<script setup lang="ts">
import { computed } from 'vue'
import { ChevronDown, Server } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import type { ClusterNode } from '@/api/admin/cluster'
import { StatusBadge, type PageTone } from '@/components/page'
import { Button } from '@/components/ui/button'
import ClusterNodeTelemetry from '@/components/admin/cluster/ClusterNodeTelemetry.vue'
import { createClusterText } from '@/components/admin/cluster/clusterText'
import { formatMetric, hasRecentSample, isCurrentNode, nodePerformance, validMetric } from './dashboardMetrics'

const props = defineProps<{ node: ClusterNode; haEnabled: boolean; now: number }>()
const emit = defineEmits<{ manage: [node: ClusterNode] }>()
const { t } = useI18n()
const text = createClusterText(t)
const current = computed(() => isCurrentNode(props.node, props.now))
const performance = computed(() => nodePerformance(props.node, props.now))
const applied = computed(() => current.value && props.node.stats.control_revision >= props.node.control.revision &&
  (!props.haEnabled || props.node.stats.ingress_revision >= props.node.control.revision))
const state = computed<{ label: string; tone: PageTone }>(() => {
  if (!current.value) return { label: t('cluster.offline'), tone: 'danger' }
  if (props.node.control.maintenance) return { label: t('cluster.maintenance'), tone: 'warning' }
  if (props.node.stats.runtime_config_ok === false) return { label: t('cluster.blocked'), tone: 'danger' }
  return { label: t('cluster.serving'), tone: 'success' }
})
const meters = computed(() => {
  const stats = props.node.stats
  const resources = stats.resources
  const fresh = current.value && hasRecentSample(resources?.sampled_at, props.now)
  const disk = current.value && stats.disk_total_bytes > 0 ? 100 * (1 - stats.disk_free_bytes / stats.disk_total_bytes) : null
  return [
    { label: t('cluster.cpuUsage'), value: fresh ? resources?.cpu_percent : null },
    { label: t('cluster.systemMemory'), value: fresh ? resources?.memory_percent : null },
    { label: t('cluster.diskUsage'), value: disk }
  ].map(meter => ({ ...meter, percent: validMetric(meter.value) ? Math.min(100, meter.value) : null }))
})
const appUptime = computed(() => {
  const seconds = Math.floor((props.now - Date.parse(props.node.started_at)) / 1000)
  if (!current.value || !validMetric(seconds)) return '—'
  if (seconds >= 86400) return t('cluster.daysHours', { days: Math.floor(seconds / 86400), hours: Math.floor(seconds % 86400 / 3600) })
  if (seconds >= 3600) return t('cluster.hours', { value: Math.floor(seconds / 3600) })
  if (seconds >= 60) return t('cluster.minutes', { value: Math.floor(seconds / 60) })
  return t('cluster.seconds', { value: seconds })
})
function connection(name: string, ok: boolean | undefined, latency?: number) {
  let state = 'cluster.unconfirmed'
  let tone: PageTone = 'neutral'
  if (current.value && ok !== undefined) {
    state = ok ? 'cluster.connected' : 'cluster.disconnected'
    tone = ok ? 'success' : 'danger'
  }
  const detail = current.value && validMetric(latency) ? ` · ${formatMetric(latency, 1)} ms` : ''
  return { label: `${name} · ${t(state)}${detail}`, tone }
}
const connections = computed(() => {
  const stats = props.node.stats
  return [
    connection(t('cluster.database'), stats.database_ok, stats.database_latency_ms),
    connection(t('cluster.redis'), stats.redis_ok, stats.redis_latency_ms),
    ...(props.haEnabled ? [connection(t('cluster.lease'), stats.lease_ok)] : [])
  ]
})
</script>

<template>
  <article class="min-w-0 space-y-3 rounded-lg border border-border bg-card p-3.5">
    <header class="space-y-1.5">
      <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
        <Server aria-hidden="true" class="size-4 shrink-0 text-muted-foreground" />
        <h3 class="min-w-0 break-all font-mono text-sm font-semibold">{{ node.id }}</h3>
        <StatusBadge class="ml-auto" :label="text.label('roles', node.role)" :tone="node.role === 'primary' ? 'brand' : 'neutral'" soft />
      </div>
      <div class="flex flex-wrap items-start gap-1.5">
        <StatusBadge :label="state.label" :tone="state.tone" />
        <span v-if="node.version" class="break-all pt-0.5 text-xs text-muted-foreground">{{ node.version }}</span>
      </div>
    </header>

    <dl class="grid grid-cols-3 gap-2 rounded-md bg-muted/45 p-2.5">
      <div class="min-w-0">
        <dt class="text-xs text-muted-foreground">{{ t('cluster.active') }}</dt>
        <dd class="mt-0.5 font-mono text-sm font-medium tabular-nums">
          {{ current ? formatMetric(node.stats.active_requests) : '—' }}<span class="text-xs text-muted-foreground" :title="t('cluster.capacity')"> / {{ formatMetric(node.control.max_concurrency) }}</span>
        </dd>
      </div>
      <div class="min-w-0">
        <dt class="text-xs text-muted-foreground">{{ t('cluster.dashboard.rps') }}</dt>
        <dd class="mt-0.5 font-mono text-sm font-medium tabular-nums">{{ formatMetric(performance?.requests_per_second, 1) }}</dd>
      </div>
      <div class="min-w-0">
        <dt class="text-xs text-muted-foreground">{{ t('cluster.weight') }}</dt>
        <dd class="mt-0.5 font-mono text-sm font-medium tabular-nums">{{ formatMetric(node.control.weight) }}</dd>
      </div>
    </dl>

    <ul class="space-y-2">
      <li v-for="meter in meters" :key="meter.label" class="space-y-1">
        <div class="flex justify-between gap-2 text-xs">
          <span class="text-muted-foreground">{{ meter.label }}</span>
          <span class="font-mono tabular-nums">{{ meter.percent == null ? '—' : `${formatMetric(meter.percent, 1)}%` }}</span>
        </div>
        <div aria-hidden="true" class="h-1 overflow-hidden rounded-full bg-muted">
          <div
            v-if="meter.percent != null"
            class="h-full rounded-full"
            :class="meter.percent >= 90 ? 'bg-danger' : meter.percent >= 75 ? 'bg-warning' : 'bg-foreground/60'"
            :style="{ width: `${meter.percent}%` }"
          />
        </div>
      </li>
    </ul>

    <div class="flex flex-wrap gap-1.5">
      <StatusBadge v-for="item in connections" :key="item.label" :label="item.label" :tone="item.tone" class="rounded-md" />
    </div>

    <dl class="space-y-1 text-xs">
      <div class="flex items-baseline justify-between gap-2">
        <dt class="text-muted-foreground">{{ t('cluster.appUptime') }}</dt><dd class="font-mono tabular-nums">{{ appUptime }}</dd>
      </div>
      <div class="flex items-baseline justify-between gap-2">
        <dt class="text-muted-foreground">{{ t('cluster.goroutines') }}</dt><dd class="font-mono tabular-nums">{{ current ? formatMetric(node.stats.goroutines) : '—' }}</dd>
      </div>
    </dl>

    <details class="group border-t border-border pt-2">
      <summary class="flex cursor-pointer list-none items-center gap-1.5 rounded-sm text-xs font-medium text-muted-foreground hover:text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring [&::-webkit-details-marker]:hidden">
        <ChevronDown aria-hidden="true" class="size-3.5 group-open:rotate-180" />{{ t('cluster.dashboard.nodeDetails') }}
      </summary>
      <div class="pt-3">
        <ClusterNodeTelemetry :node="node" :ha-enabled="haEnabled" :now="now" />
      </div>
    </details>

    <footer class="flex flex-wrap items-start justify-between gap-2 border-t border-border pt-2 text-xs">
      <p class="max-w-full break-words text-muted-foreground">{{ t(applied ? 'cluster.applied' : 'cluster.pending') }} · r{{ node.control.revision }}</p>
      <Button type="button" size="xs" variant="outline" class="ml-auto" @click="emit('manage', node)">{{ t('cluster.manage') }}</Button>
    </footer>
  </article>
</template>
