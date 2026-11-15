<template>
  <AppLayout>
    <Tabs :model-value="tab" class="gap-3.5" @update:model-value="selectTab($event as Tab)">
      <PageHeader :title="t('cluster.title')" :description="t('cluster.description')">
        <template #actions>
          <Button type="button" size="sm" variant="outline" class="order-last" :disabled="busy" @click="refresh">
            <RefreshCw class="size-3.5" :class="{ 'animate-spin': busy }" aria-hidden="true" />{{ t('cluster.refresh') }}
          </Button>
          <TabsList v-if="snapshot?.enabled" class="h-auto max-w-full flex-wrap justify-start" :aria-label="t('cluster.title')">
            <TabsTrigger v-for="item in tabs" :key="item" :value="item" class="min-h-7 px-2.5 text-xs">
              {{ t(`cluster.${item}`) }}
            </TabsTrigger>
          </TabsList>
        </template>
      </PageHeader>

      <p v-if="error" role="alert" class="flex items-start gap-2 rounded-lg border border-danger/25 bg-danger/10 px-3 py-2.5 text-sm text-danger">
        <CircleAlert class="mt-0.5 size-4 shrink-0" aria-hidden="true" />{{ error }}
      </p>
      <SectionCard v-if="snapshot && !snapshot.enabled">
        <EmptyState :icon="Server" :title="t('cluster.disabled')" />
      </SectionCard>
      <SectionCard v-else-if="busy && !snapshot">
        <p role="status" class="py-8 text-center text-sm text-muted-foreground">{{ t('common.loading') }}</p>
      </SectionCard>

      <template v-if="snapshot?.enabled">
        <ClusterDashboardSummary :snapshot="snapshot" :now="snapshotNow" />

        <TabsContent value="nodes" class="space-y-3.5">
          <div class="grid items-start gap-3.5 xl:grid-cols-[minmax(0,1fr)_minmax(17rem,26%)]">
            <ClusterDashboardTopology :snapshot="snapshot" :now="snapshotNow" @manage="edit" />
            <div class="min-w-0 space-y-3.5">
              <ClusterDashboardLog :alerts="snapshot.alerts" compact>
                <template v-if="snapshot.alerts.length" #actions>
                  <Button type="button" variant="ghost" size="xs" @click="selectTab('alerts')">{{ t('cluster.dashboard.viewAlerts') }}</Button>
                </template>
              </ClusterDashboardLog>
              <ClusterDashboardLog :events="snapshot.events" compact>
                <template v-if="snapshot.events.length" #actions>
                  <Button type="button" variant="ghost" size="xs" @click="selectTab('events')">{{ t('cluster.dashboard.viewEvents') }}</Button>
                </template>
              </ClusterDashboardLog>
            </div>
          </div>
          <SectionCard v-if="snapshot.nodes.length" :title="t('cluster.history')" flush>
            <template #actions>
              <Button type="button" variant="ghost" size="xs" @click="selectTab('history')">{{ t('cluster.dashboard.viewHistory') }}</Button>
            </template>
            <p class="px-3.5 py-3 text-xs text-muted-foreground">{{ t('cluster.historyView.hint') }}</p>
          </SectionCard>
        </TabsContent>

        <TabsContent value="history">
          <ClusterHistory v-if="tab === 'history' && snapshot.nodes.length" :nodes="snapshot.nodes" />
          <SectionCard v-else-if="tab === 'history'" :title="t('cluster.history')">
            <EmptyState :icon="Server" :title="t('cluster.dashboard.noNodes')" />
          </SectionCard>
        </TabsContent>

        <TabsContent value="alerts"><ClusterDashboardLog :alerts="snapshot.alerts" /></TabsContent>

        <TabsContent value="requests" class="space-y-3.5">
          <SectionCard
            :title="t('cluster.requests')"
            :description="`${t('cluster.reviewCount')}: ${snapshot.review_count}${snapshot.review_count >= 100 ? '+' : ''}`"
            flush
          >
            <template #actions>
              <label class="flex items-center gap-2 text-xs">
                <input v-model="onlyReview" type="checkbox" class="size-4 accent-brand" @change="loadRequests()" />
                {{ t('cluster.onlyReview') }}
              </label>
              <Button type="button" variant="outline" size="xs" :disabled="requestBusy" @click="loadRequests()">{{ t('cluster.first') }}</Button>
            </template>
            <div v-if="requests.length" class="overflow-x-auto">
              <table class="w-full text-left text-sm">
                <thead class="border-b border-border bg-muted/40 text-xs text-muted-foreground">
                  <tr>
                    <th v-for="key in ['requestID','user','state','billing','expected','recorded']" :key="key" scope="col" class="whitespace-nowrap px-3 py-2 font-medium">
                      {{ t(`cluster.${key}`) }}
                    </th>
                    <th scope="col"><span class="sr-only">{{ t('cluster.review') }}</span></th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-border">
                  <tr v-for="request in requests" :key="request.id" class="hover:bg-muted/35">
                    <td class="max-w-xs break-all px-3 py-2">
                      <span class="font-mono text-xs">{{ request.request_id }}</span>
                      <p class="mt-0.5 text-xs text-muted-foreground">{{ request.node_id }} · {{ date(request.started_at) }}</p>
                    </td>
                    <td class="px-3 py-2">{{ request.user_id }}</td>
                    <td class="px-3 py-2">{{ clusterText.label('requests', request.state) }} · {{ request.http_status || '—' }}</td>
                    <td class="px-3 py-2"><StatusBadge :label="clusterText.label('billing', request.billing_state)" :tone="request.billing_state === 'settled' ? 'success' : request.billing_state === 'cost_mismatch' ? 'warning' : 'neutral'" soft /></td>
                    <td class="px-3 py-2 font-mono tabular-nums">{{ request.expected_cost }}</td>
                    <td class="px-3 py-2 font-mono tabular-nums">{{ request.recorded_cost }}</td>
                    <td class="px-3 py-2">
                      <Button v-if="request.state !== 'running'" type="button" variant="outline" size="xs" @click="reviewing = request; reviewNote = request.review_note">
                        {{ t('cluster.review') }}
                      </Button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p v-else-if="requestBusy" role="status" class="py-8 text-center text-sm text-muted-foreground">{{ t('common.loading') }}</p>
            <EmptyState v-else :title="t('cluster.empty')" />
            <template v-if="requests.length === 100" #footer>
              <Button type="button" variant="outline" size="sm" :disabled="requestBusy" @click="loadRequests(requests[requests.length - 1]?.started_at)">{{ t('cluster.more') }}</Button>
            </template>
          </SectionCard>

          <SectionCard :title="t('cluster.balance')" :description="t('cluster.balanceHint')">
            <form class="flex flex-wrap gap-2.5" @submit.prevent="inspectBalance">
              <input v-model.number="userID" type="number" min="1" step="1" class="input max-w-xs" :aria-label="t('cluster.userID')" :placeholder="t('cluster.userID')" required />
              <Button variant="outline" type="submit" :disabled="mutating">{{ t('cluster.inspect') }}</Button>
            </form>
            <div v-if="balance" class="mt-3 space-y-3 text-sm">
              <p>{{ t('cluster.database') }}: {{ balance.database_balance }} · {{ t('cluster.redis') }}: {{ balance.cache_balance ?? '—' }} · {{ t('cluster.inflight') }}: {{ balance.inflight }}</p>
              <Button type="button" variant="outline" size="sm" :disabled="mutating" @click="invalidateBalance">{{ t('cluster.invalidate') }}</Button>
            </div>
          </SectionCard>
        </TabsContent>

        <TabsContent value="preflight">
          <SectionCard :title="t('cluster.preflight')" flush>
            <p class="m-3.5 rounded-lg border border-warning/30 bg-warning/10 px-3 py-2.5 text-sm text-foreground">{{ t('cluster.productionHint') }}</p>
            <EmptyState v-if="preflight && !preflight.checks.length" :title="t('cluster.empty')" />
            <ul v-else class="divide-y divide-border">
              <li v-for="check in preflight?.checks ?? []" :key="check.code" class="space-y-2 px-3.5 py-3">
                <div class="flex flex-wrap items-center gap-2">
                  <p class="min-w-0 text-sm font-medium">{{ clusterText.label('checks', check.code) }}</p>
                  <StatusBadge :label="clusterText.label('statuses', check.status)" :tone="check.status === 'pass' ? 'success' : check.status === 'fail' ? 'danger' : check.status === 'warning' || check.status === 'manual' ? 'warning' : 'neutral'" />
                </div>
                <p class="text-sm text-muted-foreground">{{ clusterText.message(check.detail) }}</p>
                <ClusterLogDetails :code="`${check.code} · ${check.status}`" :message="check.detail" />
              </li>
            </ul>
          </SectionCard>
        </TabsContent>

        <TabsContent value="events"><ClusterDashboardLog :events="snapshot.events" /></TabsContent>

        <p class="flex flex-wrap justify-end gap-x-2 text-xs text-muted-foreground">
          <span>{{ t('cluster.dashboard.refreshInterval') }}</span>
          <span>{{ t('cluster.updated') }} <time :datetime="snapshot.now">{{ date(snapshot.now) }}</time></span>
        </p>
      </template>
    </Tabs>
    <BaseDialog :show="!!editing" :title="`${t('cluster.manage')} · ${editing?.id ?? ''}`" :show-close-button="!mutating" :close-on-escape="!mutating" @close="editing = null">
      <form v-if="editing" class="space-y-4" @submit.prevent="saveControl"><p class="text-sm text-muted-foreground">{{ t('cluster.drainHint') }}</p><label class="flex items-center gap-2"><input v-model="control.maintenance" type="checkbox" :disabled="editing.role === 'primary' && !snapshot?.ha_enabled" />{{ t('cluster.maintenance') }}</label><label class="block">{{ t('cluster.capacity') }}<input v-model.number="control.max_concurrency" class="input mt-2 w-full" type="number" min="1" max="20000" required /></label><label class="block">{{ t('cluster.weight') }}<input v-model.number="control.weight" class="input mt-2 w-full" type="number" min="1" max="256" required /></label><label class="block">{{ t('cluster.reason') }}<textarea v-model="control.reason" class="input mt-2 w-full" minlength="3" maxlength="500" required /></label><Button variant="default" type="submit" :disabled="mutating">{{ t('cluster.save') }}</Button></form>
    </BaseDialog>
    <BaseDialog :show="!!reviewing" :title="t('cluster.review')" :show-close-button="!mutating" :close-on-escape="!mutating" @close="reviewing = null"><form class="space-y-4" @submit.prevent="saveReview"><p class="text-sm text-muted-foreground">{{ t('cluster.reviewHint') }}</p><label class="block">{{ t('cluster.note') }}<textarea v-model="reviewNote" class="input mt-2 w-full" minlength="3" maxlength="1000" required /></label><Button variant="default" type="submit" :disabled="mutating">{{ t('cluster.save') }}</Button></form></BaseDialog>
  </AppLayout>
</template>
<script setup lang="ts">
import { EmptyState, PageHeader, SectionCard, StatusBadge } from '@/components/page'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { CircleAlert, RefreshCw, Server } from '@lucide/vue'

import { Button } from '@/components/ui/button'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import ClusterHistory from '@/components/admin/cluster/ClusterHistory.vue'
import ClusterLogDetails from '@/components/admin/cluster/ClusterLogDetails.vue'
import { createClusterText } from '@/components/admin/cluster/clusterText'
import ClusterDashboardSummary from './cluster/ClusterDashboardSummary.vue'
import ClusterDashboardTopology from './cluster/ClusterDashboardTopology.vue'
import ClusterDashboardLog from './cluster/ClusterDashboardLog.vue'
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
let clockTimer: ReturnType<typeof setInterval> | undefined
let timer: ReturnType<typeof setInterval> | undefined
let controller: AbortController | undefined
const date = (value: string) => new Date(value).toLocaleString()
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
