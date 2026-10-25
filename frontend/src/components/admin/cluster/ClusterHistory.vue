<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { clusterApi, type ClusterNode, type ClusterHistoryPoint } from '@/api/admin/cluster'
const props = defineProps<{ nodes: ClusterNode[] }>()
const { t } = useI18n()
const node = ref(props.nodes[0]?.id || '')
const range = ref<'24h' | '7d'>('24h')
const metric = ref('rps'), points = ref<ClusterHistoryPoint[]>([]), error = ref(false), busy = ref(false)
const metrics = ['rps', 'ttft', 'cache', 'duration', 'cpu', 'memory', 'active', 'queued'] as const
let controller: AbortController | undefined
async function load() {
  controller?.abort()
  const current = new AbortController(); controller = current
  busy.value = true; error.value = false; points.value = []
  try { const data = await clusterApi.history(node.value, range.value, current.signal); if (!current.signal.aborted) points.value = data.points }
  catch { if (!current.signal.aborted) error.value = true }
  finally { if (!current.signal.aborted) busy.value = false }
}
watch([node, range], () => { if (node.value) void load() }, { immediate: true })
onUnmounted(() => controller?.abort())
function value(p: ClusterHistoryPoint): number | null | undefined {
  switch (metric.value) {
    case 'rps': return p.performance?.requests_per_second
    case 'ttft': return p.performance?.first_token_p95_ms
    case 'cache': return p.performance?.cache_read_percent
    case 'duration': return p.performance?.duration_mean_ms
    case 'cpu': return p.cpu
    case 'memory': return p.memory
    case 'active': return p.active
    default: return p.queued
  }
}
const series = computed(() => points.value.map(p => ({ at: Date.parse(p.at), value: value(p) })))
const maximum = computed(() => Math.max(1, ...series.value.map(p => p.value != null && Number.isFinite(p.value) ? p.value : 0)))
const end = computed(() => Math.max(Date.now(), ...series.value.map(p => p.at)))
const span = computed(() => (range.value === '24h' ? 1 : 7) * 86400000)
const x = (at: number) => 48 + Math.max(0, (at - (end.value - span.value)) / span.value) * 904
const y = (v: number) => 174 - v / maximum.value * 150
// Split at missing samples or gaps; do not draw a false healthy bridge.
const paths = computed(() => {
  let previous = 0, current = ''; const parts: string[] = []
  for (const p of series.value) {
    if (p.value == null || !Number.isFinite(p.value) || (previous && p.at - previous > 360000)) { if (current) parts.push(current); current = '' }
    if (p.value != null && Number.isFinite(p.value)) current += `${current ? ' L' : 'M'}${x(p.at)},${y(p.value)}`
    previous = p.at
  }
  if (current) parts.push(current)
  return parts
})
const date = (at: number) => new Date(at).toLocaleString()
</script>
<template>
  <section class="card space-y-4 p-5">
    <div class="flex flex-wrap gap-3">
      <label>{{ t('cluster.historyView.node') }}<select v-model="node" :aria-label="t('cluster.historyView.node')" class="input ml-2"><option v-for="n in nodes" :key="n.id" :value="n.id">{{ n.id }}</option></select></label>
      <label>{{ t('cluster.historyView.range') }}<select v-model="range" :aria-label="t('cluster.historyView.range')" class="input ml-2"><option value="24h">24h</option><option value="7d">7d</option></select></label>
      <label>{{ t('cluster.historyView.metric') }}<select v-model="metric" :aria-label="t('cluster.historyView.metric')" class="input ml-2"><option v-for="m in metrics" :key="m" :value="m">{{ t(`cluster.historyView.${m}`) }}</option></select></label>
      <button class="btn btn-secondary" :disabled="busy" @click="load">{{ t('cluster.refresh') }}</button>
    </div>
    <p class="text-sm text-muted-foreground">{{ t('cluster.historyView.hint') }}</p>
    <p v-if="error" role="alert" class="text-red-600">{{ t('cluster.historyView.failed') }}</p>
    <p v-else-if="!points.length && !busy">{{ t('cluster.historyView.empty') }}</p>
    <template v-else-if="points.length">
      <svg viewBox="0 0 1000 210" role="img" :aria-label="t(`cluster.historyView.${metric}`)" class="w-full min-w-0">
        <path d="M48,24 V174 H952" fill="none" stroke="currentColor" opacity="0.25" />
        <text x="0" y="28" fill="currentColor" font-size="12">{{ maximum.toFixed(1) }}</text><text x="24" y="174" fill="currentColor" font-size="12">0</text>
        <path v-for="(path, index) in paths" :key="index" :d="path" fill="none" stroke="currentColor" stroke-width="2" />
        <template v-for="(p,index) in series" :key="index"><circle v-if="p.value != null && Number.isFinite(p.value)" :cx="x(p.at)" :cy="y(p.value)" r="2.5" fill="currentColor"><title>{{ date(p.at) }} · {{ p.value.toFixed(2) }}</title></circle></template>
        <text x="48" y="200" fill="currentColor" font-size="12">{{ date(end - span) }}</text><text x="952" y="200" text-anchor="end" fill="currentColor" font-size="12">{{ date(end) }}</text>
      </svg>
      <details><summary>{{ points.length }} {{ t('cluster.performance.samples') }}</summary><div class="max-h-64 overflow-auto"><table class="w-full text-sm"><tbody><tr v-for="p in [...series].reverse()" :key="p.at"><td>{{ date(p.at) }}</td><td>{{ p.value == null ? '—' : p.value.toFixed(2) }}</td></tr></tbody></table></div></details>
    </template>
  </section>
</template>
