<script setup lang="ts">
import { computed } from 'vue'
import { Server, Waypoints } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import type { ClusterNode, ClusterSnapshot } from '@/api/admin/cluster'
import { EmptyState, SectionCard, StatusBadge } from '@/components/page'
import { createClusterText } from '@/components/admin/cluster/clusterText'
import ClusterDashboardNode from './ClusterDashboardNode.vue'
import { formatMetric, nodeRates } from './dashboardMetrics'

const props = defineProps<{ snapshot: ClusterSnapshot; now: number }>()
const emit = defineEmits<{ manage: [node: ClusterNode] }>()
const { t } = useI18n()
const text = createClusterText(t)
const roles = computed(() => {
  const tally = new Map<string, number>()
  for (const node of props.snapshot.nodes) tally.set(node.role, (tally.get(node.role) ?? 0) + 1)
  return Array.from(tally, ([role, count]) => ({ role, count }))
})
const rates = computed(() => nodeRates(props.snapshot.nodes, props.now))
const total = computed(() => rates.value.reduce((sum, node) => sum + node.rps, 0))
const swatches = ['bg-brand', 'bg-info/70', 'bg-foreground/55', 'bg-brand/55', 'bg-foreground/30']
const traffic = computed(() => rates.value.map((node, index) => ({
  ...node,
  percent: total.value ? node.rps / total.value * 100 : 0,
  color: node.maintenance ? 'bg-warning' : swatches[index % swatches.length]
})))
</script>

<template>
  <SectionCard
    :title="t('cluster.dashboard.topology')"
    :description="t('cluster.dashboard.nodeCount', { count: snapshot.nodes.length })"
    flush
  >
    <template v-if="roles.length" #actions>
      <StatusBadge
        v-for="group in roles" :key="group.role"
        :label="t('cluster.dashboard.roleCount', { role: text.label('roles', group.role), count: group.count })"
        :tone="group.role === 'primary' ? 'brand' : 'neutral'"
        soft
      />
    </template>
    <EmptyState v-if="!snapshot.nodes.length" :icon="Server" :title="t('cluster.dashboard.noNodes')" />
    <template v-else>
      <div class="flex flex-wrap items-start gap-3 border-b border-border bg-muted/20 px-3.5 py-3">
        <div class="inline-flex items-center gap-2 rounded-md border border-dashed border-border px-2.5 py-2 text-xs font-medium">
          <Waypoints aria-hidden="true" class="size-4 text-muted-foreground" />
          {{ t('cluster.dashboard.traffic') }}
        </div>
        <div class="basis-full space-y-1.5 sm:min-w-36 sm:flex-1 sm:basis-0">
          <div class="flex items-center justify-between gap-2 text-xs text-muted-foreground">
            <p>{{ t('cluster.dashboard.trafficScope') }}</p>
            <span v-if="rates.length" class="shrink-0 font-mono tabular-nums">{{ formatMetric(total, 1) }} {{ t('cluster.dashboard.requestsPerSecond') }}</span>
          </div>
          <div v-if="total > 0" aria-hidden="true" class="flex h-2 w-full overflow-hidden rounded-full bg-muted">
            <div
              v-for="node in traffic.filter(item => item.percent > 0)" :key="node.id"
              :class="node.color" :style="{ width: `${node.percent}%` }"
            />
          </div>
          <ul v-if="traffic.length" class="flex flex-wrap gap-x-3 gap-y-1 text-xs">
            <li v-for="node in traffic" :key="node.id" class="flex min-w-0 items-center gap-1.5">
              <span class="size-2 shrink-0 rounded-sm" :class="node.color" aria-hidden="true" />
              <span class="break-all text-muted-foreground">{{ node.id }}</span>
              <span class="shrink-0 font-mono tabular-nums">{{ total > 0 ? `${formatMetric(node.percent, 1)}%` : '—' }}</span>
            </li>
          </ul>
          <p v-else class="text-xs text-muted-foreground">{{ t('cluster.noData') }}</p>
        </div>
      </div>
      <div class="grid gap-2.5 p-3.5 sm:grid-cols-2 xl:grid-cols-[repeat(auto-fit,minmax(14rem,1fr))]">
        <ClusterDashboardNode
          v-for="node in snapshot.nodes" :key="node.id"
          :node="node" :ha-enabled="snapshot.ha_enabled" :now="now"
          @manage="emit('manage', $event)"
        />
      </div>
    </template>
  </SectionCard>
</template>
