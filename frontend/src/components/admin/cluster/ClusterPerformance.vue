<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ClusterNode } from '@/api/admin/cluster'
const props = defineProps<{ node: ClusterNode; now: number }>()
const { t } = useI18n()
const recent = (date: string | undefined, seconds: number) => {
  const delta = props.now - Date.parse(date || '')
  return Number.isFinite(delta) && delta >= -5000 && delta < seconds * 1000
}
const alive = computed(() => props.node.online && recent(props.node.heartbeat_at, 20))
const performance = computed(() => alive.value && recent(props.node.stats.performance?.sampled_at, 20) ? props.node.stats.performance : undefined)
const traffic = computed(() => alive.value && recent(props.node.stats.ingress_traffic?.sampled_at, 20) ? props.node.stats.ingress_traffic : undefined)
const archive = computed(() => alive.value && recent(props.node.stats.archive?.sampled_at, 90) && props.node.stats.archive?.known ? props.node.stats.archive : undefined)
const proof = computed(() => props.node.stats.verification)
const proofFresh = computed(() => alive.value && recent(proof.value?.checked_at, 300))
const value = (v?: number | null, suffix = '') => v == null || !Number.isFinite(v) || v < 0 ? '—' : `${v.toFixed(1)}${suffix}`
const count = (v?: number | null) => v == null ? '—' : v.toLocaleString()
const mib = (v?: number | null) => v == null ? '—' : value(v / 1024 ** 2, ' MiB')
const date = (v?: string | null) => v && Number.isFinite(Date.parse(v)) ? new Date(v).toLocaleString() : '—'
const certificateWarning = (v?: string | null) => v && Date.parse(v) - props.now < 14 * 86400000
</script>
<template>
  <div class="space-y-3 rounded-lg border border-border p-3 text-xs" data-testid="cluster-performance">
    <h3 class="font-medium">{{ t('cluster.performance.title') }}</h3>
    <dl class="grid grid-cols-2 gap-x-4 gap-y-2">
      <dt class="text-muted-foreground">{{ t('cluster.performance.rps') }}</dt><dd>{{ value(performance?.requests_per_second, ' /s') }}</dd>
      <dt class="text-muted-foreground">{{ t('cluster.performance.ttft') }}</dt><dd>{{ performance?.first_token_overflow ? '> 600000 ms' : value(performance?.first_token_p95_ms, ' ms') }} · {{ count(performance?.first_token_samples) }} {{ t('cluster.performance.samples') }}</dd>
      <dt class="text-muted-foreground">{{ t('cluster.performance.cache') }}</dt><dd>{{ value(performance?.cache_read_percent, '%') }}</dd>
      <dt class="text-muted-foreground">{{ t('cluster.performance.dbPool') }}</dt><dd>{{ count(performance?.db_in_use) }} / {{ count(performance?.db_open) }} / {{ count(performance?.db_limit) }}</dd>
      <dt class="text-muted-foreground">{{ t('cluster.performance.dbWait') }}</dt><dd>{{ count(performance?.db_waits) }} / {{ value(performance?.db_wait_mean_ms, ' ms') }}</dd>
      <dt class="text-muted-foreground">{{ t('cluster.performance.redisPool') }}</dt><dd>{{ count(performance?.redis_connections) }} / {{ count(performance?.redis_idle) }} / {{ count(performance?.redis_pool_timeouts) }}</dd>
      <dt class="text-muted-foreground">{{ t('cluster.performance.queue') }}</dt><dd>{{ count(traffic?.queued) }} / {{ count(traffic?.active) }}</dd>
      <dt class="text-muted-foreground">{{ t('cluster.performance.traffic') }}</dt><dd>{{ mib(traffic?.bytes_in) }} / {{ mib(traffic?.bytes_out) }}</dd>
      <dt class="text-muted-foreground">{{ t('cluster.performance.crossNode') }}</dt><dd>{{ mib(traffic?.cross_node_bytes) }}</dd>
    </dl>
    <dl class="grid grid-cols-2 gap-x-4 gap-y-2 border-t pt-2">
      <dt>{{ t('cluster.performance.contract') }}</dt><dd>{{ t(`cluster.log.statuses.${!alive || node.stats.runtime_config_ok == null ? 'unknown' : node.stats.runtime_config_ok ? 'pass' : 'fail'}`) }}</dd>
      <dt>{{ t('cluster.performance.duration') }}</dt><dd>{{ value(performance?.duration_mean_ms, ' ms') }} · {{ count(performance?.completed_samples) }}</dd>
      <dt>{{ t('cluster.performance.phases') }}</dt><dd>{{ value(performance?.auth_mean_ms) }} / {{ value(performance?.routing_mean_ms) }} / {{ value(performance?.upstream_mean_ms) }} ms</dd>
      <dt>{{ t('cluster.performance.archive') }}</dt><dd>{{ count(archive?.pending) }} · {{ t(archive == null ? 'cluster.unconfirmed' : archive.enabled ? 'cluster.log.enabled' : 'cluster.log.disabled') }}</dd>
    </dl>
    <p class="text-muted-foreground leading-5">{{ t('cluster.performance.timingScope') }}</p>
    <p class="text-muted-foreground leading-5">{{ t('cluster.performance.scope') }}</p>
    <h3 class="border-t border-border pt-3 font-medium">{{ t('cluster.performance.verification') }}</h3>
    <p v-if="!proof" class="text-muted-foreground">{{ t('cluster.performance.noProof') }}</p>
    <template v-else>
      <p class="text-muted-foreground">{{ date(proof.checked_at) }} · {{ proof.scope === 'local_lab' ? t('cluster.performance.lab') : t('cluster.performance.probe') }}</p>
      <p v-if="!proofFresh" class="text-amber-700 dark:text-amber-400">{{ t('cluster.performance.staleProof') }}</p>
      <div v-for="endpoint in proof.endpoints" :key="endpoint.id" class="rounded border border-border p-2" data-testid="ingress-proof">
        <div class="flex justify-between gap-3"><span>{{ endpoint.id }}</span><span :class="proofFresh && endpoint.status === 'fail' ? 'text-red-600' : ''">{{ t(`cluster.log.statuses.${proofFresh ? endpoint.status : 'unknown'}`) }} · {{ proofFresh ? value(endpoint.latency_ms, ' ms') : '—' }}</span></div>
        <p v-if="endpoint.reason && proofFresh" class="mt-1 text-muted-foreground">{{ t(`cluster.performance.${['tls_failed', 'connection_failed', 'unexpected_response'].includes(endpoint.reason) ? endpoint.reason : 'unexpected_response'}`) }}</p>
        <p class="mt-1" :class="certificateWarning(endpoint.certificate_expires_at) ? 'text-amber-700 dark:text-amber-400' : 'text-muted-foreground'">{{ t('cluster.performance.certificate') }} {{ date(endpoint.certificate_expires_at) }}</p>
      </div>
      <p>{{ t('cluster.performance.artifacts') }}：{{ t(`cluster.log.statuses.${!proofFresh || proof.artifacts_match == null ? 'unknown' : proof.artifacts_match ? 'pass' : 'fail'}`) }}</p>
      <p>{{ t('cluster.log.checks.shared_storage') }}：{{ t(`cluster.log.statuses.${!proofFresh || !recent(proof.storage?.checked_at, 86400) ? 'unknown' : proof.storage?.status ?? 'unknown'}`) }} · {{ date(proof.storage?.checked_at) }}<span v-if="proof.storage?.scope === 'local_lab'"> · {{ t('cluster.performance.lab') }}</span></p>
      <p>{{ t('cluster.performance.restore') }}：{{ date(proof.restore_checked_at) }} <span v-if="proof.restore_scope === 'local_lab'">· {{ t('cluster.performance.lab') }}</span></p>
    </template>
  </div>
</template>
