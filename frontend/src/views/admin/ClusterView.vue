<template>
  <AppLayout>
    <div class="space-y-6">
      <header class="flex flex-wrap items-center justify-between gap-4">
        <div><p class="mb-2 text-xs font-medium uppercase tracking-widest text-muted-foreground">{{ t('design.clusterEyebrow') }}</p><h1 class="text-3xl font-semibold tracking-tight">{{ t('cluster.title') }}</h1><p class="mt-2 text-sm text-gray-500">{{ t('cluster.description') }}</p></div>
        <Button variant="default" :disabled="busy" @click="refresh"><RefreshCw :size="15" :class="{ 'animate-spin': busy }" />{{ t('cluster.refresh') }}</Button>
      </header>
      <p v-if="error" role="alert" class="rounded-xl bg-red-50 p-4 text-sm text-red-700 dark:bg-red-950/30"><AlertCircle class="mr-2 inline-block" :size="16" />{{ error }}</p>
      <p v-if="snapshot && !snapshot.enabled" class="card p-6">{{ t('cluster.disabled') }}</p>
      <template v-if="snapshot?.enabled">
        <div class="grid gap-4 sm:grid-cols-3">
          <Card class="card p-5"><p class="text-sm text-gray-500">{{ t('cluster.nodes') }}</p><strong class="mt-2 block text-3xl">{{ snapshot.nodes.filter(n => n.online).length }} / {{ snapshot.nodes.length }}</strong></Card>
          <Card class="card p-5"><p class="text-sm text-gray-500">{{ t('cluster.alerts') }}</p><strong class="mt-2 block text-3xl">{{ snapshot.alerts.length }}</strong></Card>
          <Card class="card p-5"><p class="text-sm text-gray-500">{{ t('cluster.reviewCount') }}</p><strong class="mt-2 block text-3xl">{{ snapshot.review_count }}{{ snapshot.review_count >= 100 ? '+' : '' }}</strong></Card>
        </div>
        <Tabs :model-value="tab" @update:model-value="selectTab($event as Tab)" class="gap-5">
          <TabsList class="h-auto w-fit max-w-full flex-wrap justify-start" :aria-label="t('cluster.title')"><TabsTrigger v-for="item in tabs" :key="item" :value="item" class="px-4 py-2">{{ t(`cluster.${item}`) }}</TabsTrigger></TabsList>
        <TabsContent value="nodes" class="grid gap-5 xl:grid-cols-2">
          <Card v-for="node in snapshot.nodes" :key="node.id" class="overflow-hidden p-6">
            <div class="flex items-start justify-between gap-3"><div><h2 class="flex items-center gap-3 text-lg font-semibold"><span class="grid size-10 place-items-center rounded-lg border bg-muted/40"><Server :size="18" /></span>{{ node.id }} <Badge variant="outline">{{ clusterText.label('roles', node.role) }}</Badge></h2><p class="mt-1 text-sm" :class="nodeCurrent(node) ? 'text-emerald-600' : 'text-red-500'">{{ t(nodeCurrent(node) ? 'cluster.online' : 'cluster.offline') }} · {{ t(node.control.maintenance ? 'cluster.maintenance' : node.stats.runtime_config_ok === false ? 'cluster.blocked' : nodeCurrent(node) ? 'cluster.serving' : 'cluster.heartbeatTimeout') }}</p></div><Button variant="outline" @click="edit(node)">{{ t('cluster.manage') }}<ArrowUpRight :size="14" /></Button></div>
            <div class="my-5 grid grid-cols-3 gap-3"><div v-for="[label,value] in [['active',node.stats.active_requests],['capacity',node.control.max_concurrency],['weight',node.control.weight]]" :key="label" class="rounded-xl bg-gray-50 p-3 dark:bg-dark-800"><p class="text-xs text-gray-500">{{ t(`cluster.${label}`) }}</p><strong class="mt-2 block text-xl">{{ label === 'active' && !nodeCurrent(node) ? '—' : value }}</strong></div></div>
            <ClusterNodeTelemetry :node="node" :ha-enabled="snapshot.ha_enabled" :now="snapshotNow" />
            <p class="mt-4 text-xs text-gray-500">{{ t(nodeCurrent(node) && applied(node) ? 'cluster.applied' : 'cluster.pending') }} · r{{ node.control.revision }}</p>
          </Card>
        </TabsContent>
        <TabsContent value="history"><ClusterHistory v-if="tab === 'history'" :nodes="snapshot.nodes" /></TabsContent>
        <TabsContent value="alerts" class="card divide-y divide-gray-100 dark:divide-dark-700">
          <p v-if="!snapshot.alerts.length" class="flex items-center gap-3 p-8 text-muted-foreground"><CheckCircle2 :size="20" class="text-emerald-600" />{{ t('cluster.healthyAlerts') }}</p>
          <article v-for="alert in snapshot.alerts" :key="alert.key" class="p-5"><strong :class="alert.severity === 'critical' ? 'text-red-600' : 'text-amber-600'">{{ clusterText.scope(alert.node_id, alert.key) }} · {{ clusterText.alertName(alert.key) }}</strong><Badge variant="outline" class="ml-2">{{ clusterText.label('severities', alert.severity) }}</Badge><p class="mt-2">{{ clusterText.message(alert.message) }}</p><ClusterLogDetails :code="alert.key" :message="alert.message" /><time class="text-xs text-gray-500">{{ date(alert.raised_at) }}</time></article>
        </TabsContent>
        <TabsContent value="requests" class="space-y-5">
          <Card class="card overflow-hidden"><div class="flex flex-wrap items-center gap-3 p-4"><label class="flex items-center gap-2"><input v-model="onlyReview" type="checkbox" @change="loadRequests()" />{{ t('cluster.onlyReview') }}</label><Button variant="outline" :disabled="requestBusy" @click="loadRequests()">{{ t('cluster.first') }}</Button></div>
            <div class="overflow-x-auto"><table class="w-full text-left text-sm"><thead class="bg-gray-50 text-xs text-gray-500 dark:bg-dark-800"><tr><th v-for="key in ['requestID','user','state','billing','expected','recorded']" :key="key" class="p-4">{{ t(`cluster.${key}`) }}</th><th /></tr></thead><tbody><tr v-for="request in requests" :key="request.id" class="border-t border-gray-100 dark:border-dark-700"><td class="max-w-xs break-all p-4"><span class="font-mono text-xs">{{ request.request_id }}</span><p class="mt-1 text-xs text-gray-500">{{ request.node_id }} · {{ date(request.started_at) }}</p></td><td class="p-4">{{ request.user_id }}</td><td class="p-4">{{ clusterText.label('requests', request.state) }} · {{ request.http_status || '—' }}</td><td class="p-4">{{ clusterText.label('billing', request.billing_state) }}</td><td class="p-4 font-mono">{{ request.expected_cost }}</td><td class="p-4 font-mono">{{ request.recorded_cost }}</td><td class="p-4"><Button variant="outline" v-if="request.state !== 'running'" class="whitespace-nowrap" @click="reviewing = request; reviewNote = request.review_note">{{ t('cluster.review') }}</Button></td></tr></tbody></table></div><p v-if="!requests.length" class="p-6 text-gray-500">{{ t('cluster.empty') }}</p><div v-if="requests.length === 100" class="p-4"><Button variant="outline" :disabled="requestBusy" @click="loadRequests(requests[requests.length - 1]?.started_at)">{{ t('cluster.more') }}</Button></div>
          </Card>
          <Card class="card space-y-4 p-5"><h2 class="font-semibold">{{ t('cluster.balance') }}</h2><p class="text-sm text-gray-500">{{ t('cluster.balanceHint') }}</p><form class="flex flex-wrap gap-3" @submit.prevent="inspectBalance"><input v-model.number="userID" type="number" min="1" step="1" class="input max-w-xs" :aria-label="t('cluster.userID')" :placeholder="t('cluster.userID')" required /><Button variant="outline" type="submit" :disabled="mutating">{{ t('cluster.inspect') }}</Button></form><div v-if="balance" class="space-y-3 text-sm"><p>{{ t('cluster.database') }}: {{ balance.database_balance }} · {{ t('cluster.redis') }}: {{ balance.cache_balance ?? '—' }} · {{ t('cluster.inflight') }}: {{ balance.inflight }}</p><Button variant="outline" :disabled="mutating" @click="invalidateBalance">{{ t('cluster.invalidate') }}</Button></div></Card>
        </TabsContent>
        <TabsContent value="preflight" class="card p-5"><p class="mb-5 rounded-xl bg-amber-50 p-4 text-sm text-amber-800 dark:bg-amber-950/30 dark:text-amber-300">{{ t('cluster.productionHint') }}</p><div v-for="check in preflight?.checks ?? []" :key="check.code" class="border-t border-gray-100 py-4 dark:border-dark-700"><p class="font-medium">{{ clusterText.label('checks', check.code) }} <span class="ml-3 text-sm" :class="check.status === 'pass' ? 'text-emerald-600' : check.status === 'fail' ? 'text-red-500' : 'text-amber-600'">{{ clusterText.label('statuses', check.status) }}</span></p><p class="mt-1 text-sm text-gray-500">{{ clusterText.message(check.detail) }}</p><ClusterLogDetails :code="`${check.code} · ${check.status}`" :message="check.detail" /></div></TabsContent>
        <TabsContent value="events" class="card divide-y divide-gray-100 dark:divide-dark-700"><p v-if="!snapshot.events.length" class="p-6 text-muted-foreground">{{ t('cluster.empty') }}</p><article v-for="event in snapshot.events" :key="event.id" class="p-5"><div class="flex flex-wrap justify-between gap-2 text-sm"><strong>{{ clusterText.scope(event.node_id, event.kind === 'alert_resolved' ? event.message : undefined) }} · {{ clusterText.label('kinds', event.kind) }}</strong><time class="text-gray-500">{{ date(event.created_at) }}</time></div><p class="mt-2 whitespace-pre-wrap break-words text-sm">{{ clusterText.event(event) }}</p><ClusterLogDetails :code="event.kind" :message="event.message" /></article></TabsContent>
        <p class="text-right text-xs text-gray-500">{{ t('cluster.updated') }} {{ date(snapshot.now) }}</p></Tabs>
      </template>
    </div>
    <BaseDialog :show="!!editing" :title="`${t('cluster.manage')} · ${editing?.id ?? ''}`" :show-close-button="!mutating" :close-on-escape="!mutating" @close="editing = null">
      <form v-if="editing" class="space-y-4" @submit.prevent="saveControl"><p class="text-sm text-gray-500">{{ t('cluster.drainHint') }}</p><label class="flex items-center gap-2"><input v-model="control.maintenance" type="checkbox" :disabled="editing.role === 'primary' && !snapshot?.ha_enabled" />{{ t('cluster.maintenance') }}</label><label class="block">{{ t('cluster.capacity') }}<input v-model.number="control.max_concurrency" class="input mt-2 w-full" type="number" min="1" max="20000" required /></label><label class="block">{{ t('cluster.weight') }}<input v-model.number="control.weight" class="input mt-2 w-full" type="number" min="1" max="256" required /></label><label class="block">{{ t('cluster.reason') }}<textarea v-model="control.reason" class="input mt-2 w-full" minlength="3" maxlength="500" required /></label><Button variant="default" type="submit" :disabled="mutating">{{ t('cluster.save') }}</Button></form>
    </BaseDialog>
    <BaseDialog :show="!!reviewing" :title="t('cluster.review')" :show-close-button="!mutating" :close-on-escape="!mutating" @close="reviewing = null"><form class="space-y-4" @submit.prevent="saveReview"><p class="text-sm text-gray-500">{{ t('cluster.reviewHint') }}</p><label class="block">{{ t('cluster.note') }}<textarea v-model="reviewNote" class="input mt-2 w-full" minlength="3" maxlength="1000" required /></label><Button variant="default" type="submit" :disabled="mutating">{{ t('cluster.save') }}</Button></form></BaseDialog>
  </AppLayout>
</template>
<script setup lang="ts">
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { Server, RefreshCw, CheckCircle2, AlertCircle, ArrowUpRight } from '@lucide/vue'

import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import ClusterNodeTelemetry from '@/components/admin/cluster/ClusterNodeTelemetry.vue'
import ClusterHistory from '@/components/admin/cluster/ClusterHistory.vue'
import ClusterLogDetails from '@/components/admin/cluster/ClusterLogDetails.vue'
import { createClusterText } from '@/components/admin/cluster/clusterText'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/UiDialog.vue'
import { useAppStore } from '@/stores/app'
import { clusterApi, type ClusterSnapshot, type ClusterNode, type Control, type ClusterRequest, type ClusterPreflight, type ClusterBalance } from '@/api/admin/cluster'
const { t } = useI18n()
const clusterText = createClusterText(t)
const app = useAppStore()
const tabs = ['nodes', 'history', 'alerts', 'requests', 'preflight', 'events'] as const
type Tab = typeof tabs[number]
const tab = ref<Tab>('nodes')
const snapshot = ref<ClusterSnapshot | null>(null), preflight = ref<ClusterPreflight | null>(null)
const busy = ref(false), requestBusy = ref(false), mutating = ref(false), error = ref(''), onlyReview = ref(true)
const requests = ref<ClusterRequest[]>([]), editing = ref<ClusterNode | null>(null), reviewing = ref<ClusterRequest | null>(null), reviewNote = ref('')
const control = ref<Control>({ maintenance: false, max_concurrency: 1000, weight: 100, revision: 0, reason: '' })
const userID = ref<number | undefined>(), balance = ref<ClusterBalance | null>(null)
const clock = ref(Date.now())
const receivedAt = ref(Date.now())
const snapshotNow = computed(() => {
 const serverNow = Date.parse(snapshot.value?.now ?? '')
 return Number.isFinite(serverNow) ? serverNow + Math.max(0, clock.value - receivedAt.value) : clock.value
})
const nodeCurrent = (node: ClusterNode) => node.online && Number.isFinite(Date.parse(node.heartbeat_at)) && snapshotNow.value - Date.parse(node.heartbeat_at) >= -5000 && snapshotNow.value - Date.parse(node.heartbeat_at) < 20000
let clockTimer: ReturnType<typeof setInterval> | undefined
let timer: ReturnType<typeof setInterval> | undefined
let controller: AbortController | undefined
const date = (value: string) => new Date(value).toLocaleString()
const applied = (node: ClusterNode) => node.stats.control_revision >= node.control.revision && (!snapshot.value?.ha_enabled || node.stats.ingress_revision >= node.control.revision)
async function refresh() {
 if (busy.value) return
 busy.value = true
 controller = new AbortController()
 try { snapshot.value = await clusterApi.snapshot(controller.signal); receivedAt.value = Date.now(); clock.value = receivedAt.value; error.value = '' }
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
onMounted(() => { clockTimer = setInterval(() => { clock.value = Date.now() }, 1000); void refresh(); timer = setInterval(() => { if (!document.hidden && !editing.value && !reviewing.value && !mutating.value) void refresh() }, 10000) })
onUnmounted(() => { if (clockTimer) clearInterval(clockTimer); if (timer) clearInterval(timer); controller?.abort() })
</script>
