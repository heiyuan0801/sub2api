<template>
  <AppLayout>
    <div class="space-y-6">
      <PageHeader :title="t('nav.myOrders')">
        <template #actions>
          <button
            type="button"
            class="btn btn-primary h-9 px-3.5 text-sm"
            @click="router.push('/purchase')"
          >
            {{ t('payment.result.backToRecharge') }}
          </button>
        </template>
      </PageHeader>

      <!-- Filters -->
      <div class="rounded-2xl border border-border/80 bg-card p-4 shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <Select v-model="currentFilter" :options="statusFilters" class="w-40" @change="handlePageChange(1)" />
          <div class="flex items-center gap-2">
            <button
              type="button"
              @click="fetchOrders"
              :disabled="loading"
              class="inline-flex h-9 items-center gap-1.5 rounded-full border border-border bg-card px-3.5 text-xs font-medium text-foreground transition-colors hover:bg-muted/60 disabled:pointer-events-none disabled:opacity-50"
              :title="t('common.refresh')"
            >
              <Icon name="refresh" size="xs" :stroke-width="1.75" :class="loading ? 'animate-spin' : ''" />
              <span>{{ t('common.refresh') }}</span>
            </button>
          </div>
        </div>
      </div>

      <!-- Table -->
      <OrderTable :orders="orders" :loading="loading">
        <template #actions="{ row }">
          <div class="flex items-center gap-2">
            <button
              v-if="row.status === 'PENDING'"
              type="button"
              @click="handleCancel(row.id)"
              class="inline-flex h-7 items-center gap-1 rounded-md px-2 text-xs font-medium text-amber-600 transition-colors hover:bg-amber-500/10 dark:text-amber-400"
            >
              <Icon name="x" size="xs" />
              <span>{{ t('payment.orders.cancel') }}</span>
            </button>
            <button
              v-if="canRequestRefund(row)"
              type="button"
              @click="openRefundDialog(row)"
              class="inline-flex h-7 items-center gap-1 rounded-md px-2 text-xs font-medium text-violet-600 transition-colors hover:bg-violet-500/10 dark:text-violet-400"
            >
              <Icon name="dollar" size="xs" />
              <span>{{ t('payment.orders.requestRefund') }}</span>
            </button>
          </div>
        </template>
      </OrderTable>

      <!-- Pagination -->
      <Pagination
        v-if="pagination.total > 0"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />
    </div>

    <!-- Cancel Confirm Dialog -->
    <BaseDialog :show="!!cancelTargetId" :title="t('payment.orders.cancel')" width="narrow" @close="cancelTargetId = null">
      <p class="text-sm text-muted-foreground">{{ t('payment.confirmCancel') }}</p>
      <template #footer>
        <div class="flex justify-end gap-2.5">
          <button type="button" class="btn btn-secondary h-9 px-3.5 text-sm" @click="cancelTargetId = null">{{ t('common.cancel') }}</button>
          <button type="button" class="btn btn-danger h-9 px-3.5 text-sm" :disabled="actionLoading" @click="confirmCancel">{{ actionLoading ? t('common.processing') : t('payment.orders.cancel') }}</button>
        </div>
      </template>
    </BaseDialog>

    <!-- Refund Dialog -->
    <BaseDialog :show="!!refundTarget" :title="t('payment.orders.requestRefund')" @close="refundTarget = null">
      <div v-if="refundTarget" class="space-y-4">
        <div class="divide-y divide-border/70 rounded-xl border border-border/80 bg-muted/40 px-4">
          <div class="flex justify-between py-3 text-sm">
            <span class="text-muted-foreground">{{ t('payment.orders.orderId') }}</span>
            <span class="font-mono font-medium tabular-nums text-foreground">#{{ refundTarget.id }}</span>
          </div>
          <div class="flex justify-between py-3 text-sm">
            <span class="text-muted-foreground">{{ t('payment.orders.amount') }}</span>
            <span class="font-semibold tabular-nums text-foreground">${{ refundTarget.amount.toFixed(2) }}</span>
          </div>
        </div>
        <div>
          <label class="mb-1.5 block text-sm font-medium text-foreground">{{ t('payment.refundReason') }}</label>
          <textarea
            v-model="refundReason"
            rows="3"
            class="w-full rounded-xl border border-border bg-background px-3.5 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/20"
            :placeholder="t('payment.refundReasonPlaceholder')"
          />
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2.5">
          <button type="button" class="btn btn-secondary h-9 px-3.5 text-sm" @click="refundTarget = null">{{ t('common.cancel') }}</button>
          <button type="button" class="btn btn-primary h-9 px-3.5 text-sm" :disabled="actionLoading || !refundReason.trim()" @click="confirmRefund">{{ actionLoading ? t('common.processing') : t('payment.orders.requestRefund') }}</button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores'
import { paymentAPI } from '@/api/payment'
import { extractI18nErrorMessage } from '@/utils/apiError'
import type { PaymentOrder } from '@/types/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import OrderTable from '@/components/payment/OrderTable.vue'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()

const loading = ref(false)
const actionLoading = ref(false)
const orders = ref<PaymentOrder[]>([])
const refundEligibleProviders = ref<Set<string>>(new Set())
const currentFilter = ref('')
const cancelTargetId = ref<number | null>(null)
const refundTarget = ref<PaymentOrder | null>(null)
const refundReason = ref('')
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

const statusFilters = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'PENDING', label: t('payment.status.pending') },
  { value: 'COMPLETED', label: t('payment.status.completed') },
  { value: 'FAILED', label: t('payment.status.failed') },
  { value: 'REFUNDED', label: t('payment.status.refunded') },
])

async function fetchOrders() {
  loading.value = true
  try {
    const res = await paymentAPI.getMyOrders({
      page: pagination.page,
      page_size: pagination.page_size,
      status: currentFilter.value || undefined,
    })
    orders.value = res.data.items || []
    pagination.total = res.data.total || 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) { pagination.page = page; fetchOrders() }
function handlePageSizeChange(size: number) { pagination.page_size = size; pagination.page = 1; fetchOrders() }

function handleCancel(orderId: number) { cancelTargetId.value = orderId }

async function confirmCancel() {
  if (!cancelTargetId.value) return
  actionLoading.value = true
  try {
    await paymentAPI.cancelOrder(cancelTargetId.value)
    appStore.showSuccess(t('common.success'))
    cancelTargetId.value = null
    await fetchOrders()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    actionLoading.value = false
  }
}

function openRefundDialog(order: PaymentOrder) { refundTarget.value = order; refundReason.value = '' }

async function confirmRefund() {
  if (!refundTarget.value || !refundReason.value.trim()) return
  actionLoading.value = true
  try {
    await paymentAPI.requestRefund(refundTarget.value.id, { reason: refundReason.value.trim() })
    appStore.showSuccess(t('common.success'))
    refundTarget.value = null
    refundReason.value = ''
    await fetchOrders()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    actionLoading.value = false
  }
}

function canRequestRefund(order: PaymentOrder): boolean {
  if (order.status !== 'COMPLETED') return false
  if (!order.provider_instance_id) return false
  return refundEligibleProviders.value.has(order.provider_instance_id)
}

async function loadRefundEligibility() {
  try {
    const res = await paymentAPI.getRefundEligibleProviders()
    refundEligibleProviders.value = new Set(res.data.provider_instance_ids || [])
  } catch { /* ignore — default to hiding refund button */ }
}

onMounted(() => { fetchOrders(); loadRefundEligibility() })
</script>
