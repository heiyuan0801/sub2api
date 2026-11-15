<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <RouterLink class="btn btn-secondary" to="/admin/orders/plans">
          {{ t('nav.paymentPlans') }}
        </RouterLink>
        <label class="flex items-center gap-2">
          <span class="text-sm">{{ t('planOps.filterPlan') }}</span>
          <select v-model="planId" class="input min-w-32" @change="changePlanFilter">
            <option :value="0">{{ t('planOps.allPlans') }}</option>
            <option v-if="planId && !plan" :value="planId">#{{ planId }}</option>
            <option v-for="item in plans" :key="item.id" :value="item.id">{{ item.name }}</option>
          </select>
        </label>
      </div>

      <section v-if="plan && (adjusting || reviewing)" class="card space-y-4 p-4 sm:p-6">
        <div>
          <h2 class="text-lg font-semibold">{{ t(reviewing ? 'planOps.review' : 'planOps.adjust') }} · {{ plan.name }}</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('planOps.orderHint') }}</p>
        </div>
        <dl class="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <div v-for="metric in metrics" :key="metric.label" class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
            <dt class="text-sm text-gray-500">{{ metric.label }}</dt>
            <dd class="mt-1 text-lg font-semibold tabular-nums">{{ metric.value }}</dd>
          </div>
        </dl>
        <form v-if="!reviewing" class="space-y-3" @submit.prevent="review">
          <div class="grid gap-3 md:grid-cols-3">
            <label>
              <span class="input-label">{{ t('planOps.newPrice') }}</span>
              <input v-model.number="newPrice" type="number" step="0.01" min="0.01" required class="input" />
            </label>
            <label>
              <span class="input-label">{{ t('planOps.effectiveAt') }}</span>
              <select v-model="scheduled" class="input">
                <option :value="false">{{ t('planOps.immediate') }}</option>
                <option :value="true">{{ t('planOps.schedule') }}</option>
              </select>
            </label>
            <label v-if="scheduled">
              <span class="input-label">{{ t('planOps.effectiveAt') }}</span>
              <input v-model="localTime" type="datetime-local" required class="input" />
            </label>
          </div>
          <label class="block">
            <span class="input-label">{{ t('planOps.reason') }}</span>
            <textarea v-model="reason" maxlength="1000" rows="2" class="input" />
          </label>
          <button class="btn btn-primary" type="submit">{{ t('planOps.review') }}</button>
        </form>
        <div v-else class="space-y-3">
          <p class="text-sm">{{ t('planOps.effectiveAt') }} · {{ effectiveAt ? formatDateTime(effectiveAt) : t('planOps.immediate') }}</p>
          <p v-if="reason.trim()" class="whitespace-pre-wrap break-words text-sm">{{ reason.trim() }}</p>
          <label class="flex items-center gap-2 text-sm">
            <input v-model="notify" type="checkbox" />
            {{ t('planOps.notifyDraft') }}
          </label>
          <div class="flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary" :disabled="saving" @click="reviewing = false">{{ t('common.back') }}</button>
            <button type="button" class="btn btn-primary" :disabled="saving" @click="submit">{{ t('common.confirm') }}</button>
          </div>
        </div>
      </section>
      <button v-else-if="plan" class="btn btn-primary" type="button" @click="openEditor">{{ t('planOps.adjust') }}</button>

      <DataTable :columns="columns" :data="history" :loading="loading">
        <template #cell-plan_name="{ row }">
          <RouterLink class="font-medium hover:underline" :to="{ query: { plan_id: row.plan_id } }">{{ row.plan_name }}</RouterLink>
          <span class="ml-1 text-xs text-gray-500">#{{ row.plan_id }}</span>
        </template>
        <template #cell-price="{ row }">
          <span class="tabular-nums">{{ money(row.old_price, row.currency) }} → {{ money(row.new_price, row.currency) }}</span>
        </template>
        <template #cell-delta="{ row }">
          <span class="tabular-nums" :class="row.delta > 0 ? 'text-amber-700 dark:text-amber-300' : 'text-emerald-700 dark:text-emerald-300'">
            {{ row.delta > 0 ? '+' : '' }}{{ row.delta.toFixed(2) }}
            <template v-if="row.percent != null">({{ row.percent > 0 ? '+' : '' }}{{ row.percent.toFixed(2) }}%)</template>
          </span>
        </template>
        <template #cell-status="{ row }">
          <span class="badge" :class="row.status === 'applied' ? 'badge-success' : row.status === 'conflict' ? 'badge-danger' : 'badge-info'">{{ t('planOps.states.' + row.status) }}</span>
        </template>
        <template #cell-time="{ row }">
          <div class="text-sm">{{ formatDateTime(row.effective_at) }}</div>
          <div v-if="row.resolved_at" class="text-xs text-gray-500">{{ t('planOps.resolvedAt') }} · {{ formatDateTime(row.resolved_at) }}</div>
        </template>
        <template #cell-reason="{ row }">
          <p class="max-w-64 whitespace-pre-wrap break-words text-sm">{{ row.reason || '—' }}</p>
        </template>
        <template #cell-actor_id="{ row }">#{{ row.actor_id }}</template>
        <template #cell-actions="{ row }">
          <button v-if="row.status === 'pending'" type="button" class="btn btn-secondary btn-sm" @click="cancelTarget = row.id">{{ t('planOps.cancel') }}</button>
          <button v-if="(row.status === 'pending' || row.status === 'applied') && plans.some(item => item.id === row.plan_id)" type="button" class="btn btn-secondary btn-sm" @click="openNotice(row)">{{ t('planOps.notify') }}</button>
        </template>
      </DataTable>
      <button v-if="hasMore" type="button" class="btn btn-secondary" :disabled="loading" @click="loadHistory(true)">{{ t('planOps.loadMore') }}</button>
    </div>
    <ConfirmDialog :show="cancelTarget != null" :title="t('planOps.cancel')" :message="t('planOps.cancelPrompt')" @confirm="cancelChange" @cancel="cancelTarget = null" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { adminPaymentAPI } from '@/api/admin/payment'
import { formatPaymentAmount } from '@/components/payment/currency'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useAppStore } from '@/stores/app'
import type { SubscriptionPlan, PlanPriceChange } from '@/types/payment'
import type { Column } from '@/components/common/types'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import AppLayout from '@/components/layout/AppLayout.vue'

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const app = useAppStore()
const plans = ref<SubscriptionPlan[]>([])
const history = ref<PlanPriceChange[]>([])
const planId = ref(0)
const plan = computed(() => plans.value.find(item => item.id === planId.value))
const loading = ref(false)
const hasMore = ref(false)
const cancelTarget = ref<number | null>(null)
const adjusting = ref(false)
const reviewing = ref(false)
const newPrice = ref(0)
const reason = ref('')
const scheduled = ref(false)
const localTime = ref('')
const effectiveAt = ref<string>()
const notify = ref(false)
const saving = ref(false)
let loadVersion = 0

const money = (value: number, currency?: string) => formatPaymentAmount(value, currency || 'USD')
const formatDateTime = (value: string) => new Date(value).toLocaleString(locale.value)
const delta = computed(() => Math.round((Number(newPrice.value) - (plan.value?.price ?? 0)) * 100) / 100)
const metrics = computed(() => {
  const old = plan.value?.price ?? 0
  const currency = plan.value?.currency
  return [
    { label: t('planOps.oldPrice'), value: money(old, currency) },
    { label: t('planOps.newPrice'), value: money(Number(newPrice.value), currency) },
    { label: t('planOps.delta'), value: (delta.value > 0 ? '+' : '') + delta.value.toFixed(2) },
    { label: t('planOps.percent'), value: old > 0 ? (delta.value > 0 ? '+' : '') + (delta.value / old * 100).toFixed(2) + '%' : '—' },
  ]
})
const columns = computed<Column[]>(() => [
  { key: 'plan_name', label: t('payment.admin.planName') },
  { key: 'price', label: t('payment.admin.price') },
  { key: 'delta', label: t('planOps.delta') },
  { key: 'status', label: t('common.status') },
  { key: 'time', label: t('planOps.effectiveAt') },
  { key: 'reason', label: t('planOps.reason') },
  { key: 'actor_id', label: t('planOps.actor') },
  { key: 'actions', label: t('common.actions') },
])

function showError(error: unknown) {
  app.showError(extractApiErrorMessage(error, t('common.error')))
}

async function loadHistory(more = false) {
  const version = ++loadVersion
  loading.value = true
  if (!more) {
    history.value = []
    hasMore.value = false
  }
  try {
    const { data } = await adminPaymentAPI.listPlanPriceChanges(planId.value || undefined, more ? history.value.at(-1)?.id : undefined)
    if (version !== loadVersion) return
    history.value = more ? [...history.value, ...data] : data
    hasMore.value = data.length === 50
  } catch (error) {
    if (version === loadVersion) showError(error)
  } finally {
    if (version === loadVersion) loading.value = false
  }
}

function changePlanFilter() {
  void router.replace({ query: planId.value ? { plan_id: planId.value } : {} })
}

function openEditor() {
  newPrice.value = plan.value?.price ?? 0
  reason.value = ''
  scheduled.value = false
  localTime.value = ''
  effectiveAt.value = undefined
  notify.value = false
  reviewing.value = false
  adjusting.value = true
}

function review() {
  if (!Number.isFinite(newPrice.value) || newPrice.value <= 0 || delta.value === 0) {
    app.showError(t('planOps.priceInvalid'))
    return
  }
  const timestamp = new Date(localTime.value).getTime()
  if (scheduled.value && (!Number.isFinite(timestamp) || timestamp <= Date.now())) {
    app.showError(t('planOps.scheduleInvalid'))
    return
  }
  effectiveAt.value = scheduled.value ? new Date(timestamp).toISOString() : undefined
  reviewing.value = true
}

function openNotice(change: PlanPriceChange) {
  void router.push({
    path: '/admin/announcements',
    query: {
      compose: '1', kind: change.delta > 0 ? 'price_increase' : 'price_decrease',
      plan_id: change.plan_id, old_price: change.old_price, new_price: change.new_price, effective_at: change.effective_at,
    },
  })
}

async function submit() {
  if (!plan.value || !reviewing.value || saving.value) return
  saving.value = true
  try {
    const current = plan.value
    const { data } = await adminPaymentAPI.changePlanPrice(current.id, {
      expected_price: current.price, new_price: newPrice.value, reason: reason.value.trim(), effective_at: effectiveAt.value,
    })
    if (data.status === 'applied') current.price = data.new_price
    reviewing.value = false
    adjusting.value = false
    app.showSuccess(t('common.saved'))
    if (notify.value) openNotice(data)
    else await loadHistory()
  } catch (error) {
    showError(error)
  } finally {
    saving.value = false
  }
}

async function cancelChange() {
  if (cancelTarget.value == null) return
  try {
    await adminPaymentAPI.cancelPlanPriceChange(cancelTarget.value)
    await loadHistory()
  } catch (error) {
    showError(error)
  } finally {
    cancelTarget.value = null
  }
}

watch(() => route.query.plan_id, value => {
  const id = Number(value)
  planId.value = Number.isSafeInteger(id) && id > 0 ? id : 0
  adjusting.value = false
  reviewing.value = false
  void loadHistory()
}, { immediate: true })

onMounted(async () => {
  try {
    plans.value = (await adminPaymentAPI.getPlans()).data
    if (plan.value && route.query.adjust === '1') openEditor()
  } catch (error) {
    showError(error)
  }
})
</script>
