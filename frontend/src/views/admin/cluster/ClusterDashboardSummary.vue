<script setup lang="ts">
import { computed } from 'vue'
import { Activity, CircleAlert, Server, Timer } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import type { ClusterSnapshot } from '@/api/admin/cluster'
import { StatCard } from '@/components/page'
import { clusterMetrics, formatMetric } from './dashboardMetrics'

const props = defineProps<{ snapshot: ClusterSnapshot; now: number }>()
const { t } = useI18n()
const metrics = computed(() => clusterMetrics(props.snapshot.nodes, props.now))
</script>

<template>
  <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
    <StatCard
      :label="t('cluster.dashboard.totalRps')"
      :value="formatMetric(metrics.totalRps, 1)"
      :unit="metrics.totalRps == null ? undefined : t('cluster.dashboard.requestsPerSecond')"
      :icon="Activity"
      :description="t('cluster.dashboard.reportingNodes', { reporting: metrics.reportingRps, online: metrics.onlineCount })"
    />
    <StatCard
      :label="t('cluster.dashboard.firstToken')"
      :value="metrics.p95Overflow ? '> 600,000' : formatMetric(metrics.slowestP95, 1)"
      :unit="metrics.p95Overflow || metrics.slowestP95 != null ? 'ms' : undefined"
      :icon="Timer"
      :description="t('cluster.dashboard.firstTokenScope')"
    />
    <StatCard
      :label="t('cluster.dashboard.errorRate')"
      :value="formatMetric(metrics.errorRate, 2)"
      :unit="metrics.errorRate == null ? undefined : '%'"
      :icon="CircleAlert"
      :description="t('cluster.dashboard.errorRateScope')"
    />
    <StatCard
      :label="t('cluster.dashboard.onlineNodes')"
      :value="metrics.onlineCount"
      :unit="`/ ${snapshot.nodes.length}`"
      :icon="Server"
      :description="t('cluster.dashboard.onlineScope', { draining: metrics.drainingCount, ha: snapshot.ha_enabled ? t('cluster.log.enabled') : t('cluster.log.disabled') })"
    />
  </div>
</template>
