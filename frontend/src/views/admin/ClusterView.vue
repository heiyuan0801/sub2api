<template>
  <AppLayout>
    <div class="space-y-6">
      <header class="flex flex-wrap items-center justify-between gap-4">
        <div><h1 class="text-2xl font-semibold">{{ t('cluster.title') }}</h1><p class="mt-2 text-sm text-gray-500">{{ t('cluster.description') }}</p></div>
        <button class="btn btn-primary" :disabled="busy" @click="refresh">{{ t('cluster.refresh') }}</button>
      </header>
      <p v-if="error" role="alert" class="rounded-xl bg-red-50 p-4 text-sm text-red-700 dark:bg-red-950/30">{{ error }}</p>
      <p v-if="snapshot && !snapshot.enabled" class="card p-6">{{ t('cluster.disabled') }}</p>
      <template v-if="snapshot?.enabled">
        <div class="grid gap-4 sm:grid-cols-3">
          <div class="card p-5"><p class="text-sm text-gray-500">{{ t('cluster.nodes') }}</p><strong class="mt-2 block text-3xl">{{ snapshot.nodes.filter(n => n.online).length }} / {{ snapshot.nodes.length }}</strong></div>
          <div class="card p-5"><p class="text-sm text-gray-500">{{ t('cluster.alerts') }}</p><strong class="mt-2 block text-3xl">{{ snapshot.alerts.length }}</strong></div>
          <div class="card p-5"><p class="text-sm text-gray-500">{{ t('cluster.reviewCount') }}</p><strong class="mt-2 block text-3xl">{{ snapshot.review_count }}{{ snapshot.review_count >= 100 ? '+' : '' }}</strong></div>
        </div>
        <nav class="flex flex-wrap gap-2" :aria-label="t('cluster.title')"><button v-for="item in tabs" :key="item" class="btn" :class="tab === item ? 'btn-primary' : 'btn-secondary'" :aria-pressed="tab === item" @click="selectTab(item)">{{ t(`cluster.${item}`) }}</button></nav>
        <section v-if="tab === 'nodes'" class="grid gap-5 xl:grid-cols-2">
          <article v-for="node in snapshot.nodes" :key="node.id" class="card p-6">
            <div class="flex items-start justify-between gap-3"><div><h2 class="text-lg font-semibold">{{ node.id }} <span class="text-sm font-normal text-gray-500">{{ node.role }}</span></h2><p class="mt-1 text-sm" :class="node.online ? 'text-emerald-600' : 'text-red-500'">{{ t(node.online ? 'cluster.online' : 'cluster.offline') }} · {{ t(node.control.maintenance ? 'cluster.maintenance' : 'cluster.serving') }}</p></div><button class="btn btn-secondary" @click="edit(node)">{{ t('cluster.manage') }}</button></div>
            <div class="my-5 grid grid-cols-3 gap-3"><div v-for="[label,value] in [['active',node.stats.active_requests],['capacity',node.control.max_concurrency],['weight',node.control.weight]]" :key="label" class="rounded-xl bg-gray-50 p-3 dark:bg-dark-800"><p class="text-xs text-gray-500">{{ t(`cluster.${label}`) }}</p><strong class="mt-2 block text-xl">{{ value }}</strong></div></div>
            <dl class="grid grid-cols-2 gap-3 text-sm">
              <template v-for="[label,value] in [['database',node.stats.database_ok],['redis',node.stats.redis_ok],['lease',node.stats.lease_ok]]" :key="label"><dt class="text-gray-500">{{ t(`cluster.${label}`) }}</dt><dd :class="value ? 'text-emerald-600' : 'text-red-500'">{{ t(value ? 'cluster.ok' : 'cluster.failed') }}</dd></template>
              <dt class="text-gray-500">{{ t('cluster.version') }}</dt><dd class="break-all">{{ node.version }}</dd>
              <dt class="text-gray-500">{{ t('cluster.replicas') }}</dt><dd>{{ node.stats.replication_known ? `${node.stats.sync_replicas} / ${node.stats.replication_peers}` : '—' }}</dd>
              <dt class="text-gray-500">{{ t('cluster.lag') }}</dt><dd>{{ node.stats.replication_known ? node.stats.replication_lag_bytes : '—' }}</dd>
              <dt class="text-gray-500">{{ t('cluster.disk') }}</dt><dd>{{ node.stats.disk_total_bytes ? (node.stats.disk_free_bytes / 1073741824).toFixed(1) : '—' }}</dd>
              <dt class="text-gray-500">{{ t('cluster.memory') }}</dt><dd>{{ (node.stats.heap_bytes / 1048576).toFixed(1) }}</dd>
            </dl>
            <p class="mt-4 text-xs text-gray-500">{{ t(applied(node) ? 'cluster.applied' : 'cluster.pending') }} · r{{ node.control.revision }}</p>
          </article>
        </section>
        <section v-if="tab === 'alerts'" class="card divide-y divide-gray-100 dark:divide-dark-700">
          <p v-if="!snapshot.alerts.length" class="p-6 text-gray-500">{{ t('cluster.healthyAlerts') }}</p>
          <article v-for="alert in snapshot.alerts" :key="alert.key" class="p-5"><strong :class="alert.severity === 'critical' ? 'text-red-600' : 'text-amber-600'">{{ alert.node_id || 'Cluster' }} · {{ alert.key }}</strong><p class="mt-2">{{ alert.message }}</p><time class="text-xs text-gray-500">{{ date(alert.raised_at) }}</time></article>
        </section>
        <section v-if="tab === 'requests'" class="space-y-5">
          <div class="card overflow-hidden"><div class="flex flex-wrap items-center gap-3 p-4"><label class="flex items-center gap-2"><input v-model="onlyReview" type="checkbox" @change="loadRequests()" />{{ t('cluster.onlyReview') }}</label><button class="btn btn-secondary" :disabled="requestBusy" @click="loadRequests()">{{ t('cluster.first') }}</button></div>
            <div class="overflow-x-auto"><table class="w-full text-left text-sm"><thead class="bg-gray-50 text-xs text-gray-500 dark:bg-dark-800"><tr><th v-for="key in ['requestID','user','state','billing','expected','recorded']" :key="key" class="p-4">{{ t(`cluster.${key}`) }}</th><th /></tr></thead><tbody><tr v-for="request in requests" :key="request.id" class="border-t border-gray-100 dark:border-dark-700"><td class="max-w-xs break-all p-4"><span class="font-mono text-xs">{{ request.request_id }}</span><p class="mt-1 text-xs text-gray-500">{{ request.node_id }} · {{ date(request.started_at) }}</p></td><td class="p-4">{{ request.user_id }}</td><td class="p-4">{{ request.state }} · {{ request.http_status }}</td><td class="p-4">{{ t(`cluster.${request.billing_state}`) }}</td><td class="p-4 font-mono">{{ request.expected_cost }}</td><td class="p-4 font-mono">{{ request.recorded_cost }}</td><td class="p-4"><button v-if="request.state !== 'running'" class="btn btn-secondary whitespace-nowrap" @click="reviewing = request; reviewNote = request.review_note">{{ t('cluster.review') }}</button></td></tr></tbody></table></div><p v-if="!requests.length" class="p-6 text-gray-500">{{ t('cluster.empty') }}</p><div v-if="requests.length === 100" class="p-4"><button class="btn btn-secondary" :disabled="requestBusy" @click="loadRequests(requests[requests.length - 1]?.started_at)">{{ t('cluster.more') }}</button></div>
          </div>
          <div class="card space-y-4 p-5"><h2 class="font-semibold">{{ t('cluster.balance') }}</h2><p class="text-sm text-gray-500">{{ t('cluster.balanceHint') }}</p><form class="flex flex-wrap gap-3" @submit.prevent="inspectBalance"><input v-model.number="userID" type="number" min="1" step="1" class="input max-w-xs" :aria-label="t('cluster.userID')" :placeholder="t('cluster.userID')" required /><button class="btn btn-secondary" :disabled="mutating">{{ t('cluster.inspect') }}</button></form><div v-if="balance" class="space-y-3 text-sm"><p>{{ t('cluster.database') }}: {{ balance.database_balance }} · {{ t('cluster.redis') }}: {{ balance.cache_balance ?? '—' }} · {{ t('cluster.inflight') }}: {{ balance.inflight }}</p><button class="btn btn-secondary" :disabled="mutating" @click="invalidateBalance">{{ t('cluster.invalidate') }}</button></div></div>
        </section>
        <section v-if="tab === 'preflight'" class="card p-5"><p class="mb-5 rounded-xl bg-amber-50 p-4 text-sm text-amber-800 dark:bg-amber-950/30 dark:text-amber-300">{{ t('cluster.productionHint') }}</p><div v-for="check in preflight?.checks ?? []" :key="check.code" class="border-t border-gray-100 py-4 dark:border-dark-700"><p class="font-medium">{{ check.code }} <span class="ml-3 text-sm" :class="check.status === 'pass' ? 'text-emerald-600' : check.status === 'fail' ? 'text-red-500' : 'text-amber-600'">{{ check.status }}</span></p><p class="mt-1 text-sm text-gray-500">{{ check.detail }}</p></div></section>
        <section v-if="tab === 'events'" class="card divide-y divide-gray-100 dark:divide-dark-700"><article v-for="event in snapshot.events" :key="event.id" class="p-5"><div class="flex flex-wrap justify-between gap-2 text-sm"><strong>{{ event.node_id }} {{ event.kind }}</strong><time class="text-gray-500">{{ date(event.created_at) }}</time></div><p class="mt-2 break-all text-sm">{{ event.message }}</p></article></section>
        <p class="text-right text-xs text-gray-500">{{ t('cluster.updated') }} {{ date(snapshot.now) }}</p>
      </template>
    </div>
    <BaseDialog :show="!!editing" :title="`${t('cluster.manage')} · ${editing?.id ?? ''}`" :show-close-button="!mutating" :close-on-escape="!mutating" @close="editing = null">
      <form v-if="editing" class="space-y-4" @submit.prevent="saveControl"><p class="text-sm text-gray-500">{{ t('cluster.drainHint') }}</p><label class="flex items-center gap-2"><input v-model="control.maintenance" type="checkbox" :disabled="editing.role === 'primary' && !snapshot?.ha_enabled" />{{ t('cluster.maintenance') }}</label><label class="block">{{ t('cluster.capacity') }}<input v-model.number="control.max_concurrency" class="input mt-2 w-full" type="number" min="1" max="20000" required /></label><label class="block">{{ t('cluster.weight') }}<input v-model.number="control.weight" class="input mt-2 w-full" type="number" min="1" max="256" required /></label><label class="block">{{ t('cluster.reason') }}<textarea v-model="control.reason" class="input mt-2 w-full" minlength="3" maxlength="500" required /></label><button class="btn btn-primary" :disabled="mutating">{{ t('cluster.save') }}</button></form>
    </BaseDialog>
    <BaseDialog :show="!!reviewing" :title="t('cluster.review')" :show-close-button="!mutating" :close-on-escape="!mutating" @close="reviewing = null"><form class="space-y-4" @submit.prevent="saveReview"><p class="text-sm text-gray-500">{{ t('cluster.reviewHint') }}</p><label class="block">{{ t('cluster.note') }}<textarea v-model="reviewNote" class="input mt-2 w-full" minlength="3" maxlength="1000" required /></label><button class="btn btn-primary" :disabled="mutating">{{ t('cluster.save') }}</button></form></BaseDialog>
  </AppLayout>
</template>
<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useAppStore } from '@/stores/app'
import { clusterApi, type ClusterSnapshot, type ClusterNode, type Control, type ClusterRequest, type ClusterPreflight, type ClusterBalance } from '@/api/admin/cluster'
const { t } = useI18n()
const app = useAppStore()
const tabs = ['nodes', 'alerts', 'requests', 'preflight', 'events'] as const
type Tab = typeof tabs[number]
const tab = ref<Tab>('nodes')
const snapshot = ref<ClusterSnapshot | null>(null), preflight = ref<ClusterPreflight | null>(null)
const busy = ref(false), requestBusy = ref(false), mutating = ref(false), error = ref(''), onlyReview = ref(true)
const requests = ref<ClusterRequest[]>([]), editing = ref<ClusterNode | null>(null), reviewing = ref<ClusterRequest | null>(null), reviewNote = ref('')
const control = ref<Control>({ maintenance: false, max_concurrency: 1000, weight: 100, revision: 0, reason: '' })
const userID = ref<number | undefined>(), balance = ref<ClusterBalance | null>(null)
let timer: ReturnType<typeof setInterval> | undefined
let controller: AbortController | undefined
const date = (value: string) => new Date(value).toLocaleString()
const applied = (node: ClusterNode) => node.stats.control_revision >= node.control.revision && (!snapshot.value?.ha_enabled || node.stats.ingress_revision >= node.control.revision)
async function refresh() {
 if (busy.value) return
 busy.value = true
 controller = new AbortController()
 try { snapshot.value = await clusterApi.snapshot(controller.signal); error.value = '' }
 catch { if (!controller.signal.aborted) error.value = t(snapshot.value ? 'cluster.refreshError' : 'cluster.error') }
 finally { busy.value = false }
}
async function loadRequests(before?: string) {
 if (requestBusy.value) return
 requestBusy.value = true
 try { requests.value = await clusterApi.requests(onlyReview.value, before) } catch { app.showError(t('cluster.error')) } finally { requestBusy.value = false }
}
async function selectTab(value: Tab) { tab.value = value; if (value === 'requests') await loadRequests(); if (value === 'preflight') { try { preflight.value = await clusterApi.preflight() } catch { app.showError(t('cluster.error')) } } }
function edit(node: ClusterNode) { editing.value = node; control.value = { ...node.control, reason: '' } }
async function mutate(work: () => Promise<void>, message = t('cluster.done')) { if (mutating.value) return; mutating.value = true; try { await work(); if (message) app.showSuccess(message); await refresh() } catch { app.showError(t('cluster.error')) } finally { mutating.value = false } }
async function saveControl() { const node = editing.value; if (node) await mutate(async () => { await clusterApi.update(node.id, control.value); editing.value = null }, t('cluster.saved')) }
async function saveReview() { const request = reviewing.value; if (request) await mutate(async () => { await clusterApi.review(request.id, reviewNote.value); reviewing.value = null; await loadRequests() }) }
async function inspectBalance() { const id = userID.value; if (!id || !Number.isSafeInteger(id) || id < 1) return; balance.value = null; await mutate(async () => { balance.value = await clusterApi.balance(id) }, '') }
async function invalidateBalance() { const id = balance.value?.user_id; if (id) await mutate(async () => { await clusterApi.invalidate(id); balance.value = await clusterApi.balance(id) }) }
onMounted(() => { void refresh(); timer = setInterval(() => { if (!document.hidden && !editing.value && !reviewing.value && !mutating.value) void refresh() }, 10000) })
onUnmounted(() => { if (timer) clearInterval(timer); controller?.abort() })
</script>
