<template>
  <div class="flex min-h-screen items-center justify-center bg-background px-4 py-10">
    <div class="w-full max-w-md">
      <!-- Loading -->
      <div
        v-if="loading"
        class="space-y-6 rounded-2xl border border-border/80 bg-card p-6 shadow-sm sm:p-8"
      >
        <div class="flex flex-col items-center space-y-4">
          <Skeleton class="size-16 rounded-full" />
          <Skeleton class="h-6 w-40" />
        </div>
        <div class="space-y-3 rounded-xl border border-border/70 bg-muted/30 p-4">
          <div v-for="i in 4" :key="`pr-sk-${i}`" class="flex items-center justify-between gap-4">
            <Skeleton class="h-3.5 w-24" />
            <Skeleton class="h-3.5 w-28" />
          </div>
        </div>
        <div class="flex gap-3">
          <Skeleton class="h-10 flex-1 rounded-xl" />
          <Skeleton class="h-10 flex-1 rounded-xl" />
        </div>
      </div>

      <div
        v-else
        class="space-y-6 rounded-2xl border border-border/80 bg-card p-6 shadow-sm sm:p-8"
      >
        <!-- Status Icon -->
        <div class="text-center">
          <div
            v-if="isSuccess"
            class="mx-auto flex size-16 items-center justify-center rounded-full bg-emerald-500 text-white shadow-sm ring-8 ring-emerald-500/10"
          >
            <svg class="size-8" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <div
            v-else-if="isPending"
            class="mx-auto flex size-16 items-center justify-center rounded-full bg-amber-500/15 ring-8 ring-amber-500/10"
          >
            <div class="size-8 animate-spin rounded-full border-[3px] border-amber-500 border-t-transparent"></div>
          </div>
          <div
            v-else
            class="mx-auto flex size-16 items-center justify-center rounded-full bg-red-500 text-white shadow-sm ring-8 ring-red-500/10"
          >
            <svg class="size-8" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </div>
          <h2 class="mt-5 text-xl font-bold tracking-tight text-foreground">
            {{ statusTitle }}
          </h2>
          <p v-if="isPending" class="mt-2 text-sm text-muted-foreground">
            {{ t('payment.result.processingHint') }}
          </p>
        </div>

        <!-- Order Info -->
        <div
          v-if="order"
          class="divide-y divide-border/70 rounded-xl border border-border/80 bg-muted/40 px-4 text-sm"
        >
          <div v-if="hasOrderId(order)" class="flex items-center justify-between gap-4 py-3">
            <span class="text-muted-foreground">{{ t('payment.orders.orderId') }}</span>
            <span class="font-mono font-medium tabular-nums text-foreground">#{{ order.id }}</span>
          </div>
          <div v-if="order.out_trade_no" class="flex items-center justify-between gap-4 py-3">
            <span class="text-muted-foreground">{{ t('payment.orders.orderNo') }}</span>
            <span class="truncate font-mono font-medium text-foreground">{{ order.out_trade_no }}</span>
          </div>
          <div v-if="hasAmountFields(order)" class="flex items-center justify-between gap-4 py-3">
            <span class="text-muted-foreground">{{ t('payment.orders.baseAmount') }}</span>
            <span class="font-medium tabular-nums text-foreground">{{ formatGatewayAmount(baseAmount) }}</span>
          </div>
          <div v-if="hasAmountFields(order) && order.fee_rate > 0" class="flex items-center justify-between gap-4 py-3">
            <span class="text-muted-foreground">{{ t('payment.orders.fee') }} ({{ order.fee_rate }}%)</span>
            <span class="font-medium tabular-nums text-foreground">{{ formatGatewayAmount(feeAmount) }}</span>
          </div>
          <div v-if="hasAmountFields(order)" class="flex items-center justify-between gap-4 py-3">
            <span class="text-muted-foreground">{{ t('payment.orders.payAmount') }}</span>
            <span class="font-bold tabular-nums text-foreground">{{ formatGatewayAmount(order.pay_amount) }}</span>
          </div>
          <div v-if="hasAmountFields(order) && orderBonusAmount(order) > 0" class="flex items-center justify-between gap-4 py-3">
            <span class="text-muted-foreground">{{ t('payment.orders.bonusAmount') }}</span>
            <span class="font-medium tabular-nums text-amber-600 dark:text-amber-400">+${{ orderBonusAmount(order).toFixed(2) }}</span>
          </div>
          <div v-if="hasAmountFields(order) && (order.amount !== order.pay_amount || orderBonusAmount(order) > 0)" class="flex items-center justify-between gap-4 py-3">
            <span class="text-muted-foreground">{{ t('payment.orders.creditedAmount') }}</span>
            <span class="font-semibold tabular-nums text-emerald-600 dark:text-emerald-400">{{ order.order_type === 'balance' ? '$' + order.amount.toFixed(2) : formatGatewayAmount(order.amount) }}</span>
          </div>
          <div v-if="hasPaymentType(order)" class="flex items-center justify-between gap-4 py-3">
            <span class="text-muted-foreground">{{ t('payment.orders.paymentMethod') }}</span>
            <span class="font-medium text-foreground">{{ t(paymentMethodI18nKey(order.payment_type), normalizedOrderPaymentType(order.payment_type)) }}</span>
          </div>
          <div class="flex items-center justify-between gap-4 py-3">
            <span class="text-muted-foreground">{{ t('payment.orders.status') }}</span>
            <OrderStatusBadge :status="displayOrderStatus(order.status)" />
          </div>
        </div>

        <!-- EasyPay return info (when no order loaded) -->
        <div
          v-else-if="returnInfo"
          class="divide-y divide-border/70 rounded-xl border border-border/80 bg-muted/40 px-4 text-sm"
        >
          <div v-if="returnInfo.outTradeNo" class="flex items-center justify-between gap-4 py-3">
            <span class="text-muted-foreground">{{ t('payment.orders.orderId') }}</span>
            <span class="truncate font-mono font-medium text-foreground">{{ returnInfo.outTradeNo }}</span>
          </div>
          <div v-if="returnInfo.money" class="flex items-center justify-between gap-4 py-3">
            <span class="text-muted-foreground">{{ t('payment.orders.payAmount') }}</span>
            <span class="font-semibold tabular-nums text-foreground">{{ formatGatewayAmount(Number(returnInfo.money) || 0) }}</span>
          </div>
          <div v-if="returnInfo.type" class="flex items-center justify-between gap-4 py-3">
            <span class="text-muted-foreground">{{ t('payment.orders.paymentMethod') }}</span>
            <span class="font-medium text-foreground">{{ t(paymentMethodI18nKey(returnInfo.type), normalizedOrderPaymentType(returnInfo.type)) }}</span>
          </div>
        </div>

        <!-- Actions -->
        <div class="flex gap-3 pt-1">
          <button type="button" class="btn btn-secondary h-10 flex-1 text-sm" @click="router.push('/purchase')">{{ t('payment.result.backToRecharge') }}</button>
          <button type="button" class="btn btn-primary h-10 flex-1 text-sm" @click="router.push('/orders')">{{ t('payment.result.viewOrders') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onBeforeUnmount, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import OrderStatusBadge from '@/components/payment/OrderStatusBadge.vue'
import { Skeleton } from '@/components/ui/skeleton'
import {
  PAYMENT_RECOVERY_STORAGE_KEY,
  clearPaymentRecoverySnapshot,
  readPaymentRecoverySnapshot,
} from '@/components/payment/paymentFlow'
import { usePaymentStore } from '@/stores/payment'
import { useAuthStore } from '@/stores/auth'
import { paymentAPI } from '@/api/payment'
import type { PublicOrderVerifyResult } from '@/api/payment'
import type { OrderStatus, PaymentOrder } from '@/types/payment'
import { formatPaymentAmount, normalizePaymentCurrency } from '@/components/payment/currency'
import { normalizePaymentMethodForDisplay, paymentMethodI18nKey } from './paymentUx'

const i18n = useI18n()
const { t } = i18n
const route = useRoute()
const router = useRouter()
const paymentStore = usePaymentStore()
const authStore = useAuthStore()

type ResolvedOrder = PaymentOrder | PublicOrderVerifyResult

const order = ref<ResolvedOrder | null>(null)
const loading = ref(true)
const currency = ref('CNY')

interface ReturnInfo {
  outTradeNo: string
  money: string
  type: string
  tradeStatus: string
}
const returnInfo = ref<ReturnInfo | null>(null)

const SUCCESS_STATUSES = new Set(['COMPLETED', 'PAID', 'RECHARGING'])
const PENDING_STATUSES = new Set(['PENDING', 'CREATED', 'WAITING', 'PROCESSING'])
const STATUS_REFRESH_INTERVAL_MS = 2000
const STATUS_REFRESH_MAX_ATTEMPTS = 15

let statusRefreshTimer: ReturnType<typeof setTimeout> | null = null
let userBalanceRefreshStarted = false
const refreshAttempts = ref(0)

/** 充值金额 = pay_amount / (1 + fee_rate/100)，fee_rate=0 时等于 pay_amount */
const baseAmount = computed(() => {
  if (!hasAmountFields(order.value)) return 0
  const feeRate = Number(order.value.fee_rate) || 0
  if (feeRate <= 0) return order.value.pay_amount ?? 0
  return Math.round((order.value.pay_amount / (1 + feeRate / 100)) * 100) / 100
})

/** 手续费 = pay_amount - baseAmount */
const feeAmount = computed(() => {
  if (!hasAmountFields(order.value)) return 0
  const feeRate = Number(order.value.fee_rate) || 0
  if (feeRate <= 0) return 0
  return Math.round((order.value.pay_amount - baseAmount.value) * 100) / 100
})

const localeCode = computed(() => {
  const raw = i18n.locale as unknown
  if (typeof raw === 'string') return raw
  if (raw && typeof raw === 'object' && 'value' in raw) {
    return String((raw as { value?: string }).value || '')
  }
  return undefined
})

const isSuccess = computed(() => {
  return isSuccessStatus(order.value?.status)
})

const isPending = computed(() => {
  return isPendingStatus(order.value?.status)
})

const statusTitle = computed(() => {
  if (isSuccess.value) {
    return t('payment.result.success')
  }
  if (isPending.value) {
    return t('payment.result.processing')
  }
  return t('payment.result.failed')
})

function normalizedOrderPaymentType(paymentType: string): string {
  return normalizePaymentMethodForDisplay(paymentType || '') || paymentType || ''
}

function formatGatewayAmount(value: number): string {
  return formatPaymentAmount(value, currency.value, localeCode.value)
}

function setResolvedOrder(nextOrder: ResolvedOrder | null): void {
  order.value = nextOrder
  if (nextOrder && 'currency' in nextOrder && nextOrder.currency) {
    currency.value = normalizePaymentCurrency(nextOrder.currency)
  }
  refreshUserBalanceForSuccessfulOrder(nextOrder)
}

function refreshUserBalanceForSuccessfulOrder(nextOrder: ResolvedOrder | null): void {
  if (!nextOrder || userBalanceRefreshStarted || normalizeOrderStatus(nextOrder.status) !== 'COMPLETED') {
    return
  }
  if ('order_type' in nextOrder && nextOrder.order_type !== 'balance') {
    return
  }

  userBalanceRefreshStarted = true
  void authStore.refreshUser().catch(() => {
    // The order result remains authoritative even if refreshing profile data fails.
  })
}

function hasOrderId(nextOrder: ResolvedOrder | null): nextOrder is PaymentOrder {
  return !!nextOrder && 'id' in nextOrder && typeof nextOrder.id === 'number'
}

function hasAmountFields(nextOrder: ResolvedOrder | null): nextOrder is PaymentOrder {
  return !!nextOrder && 'pay_amount' in nextOrder && typeof nextOrder.pay_amount === 'number' && 'amount' in nextOrder && typeof nextOrder.amount === 'number'
}

/** 充值赠送额度（USD）；老接口/订阅订单没有该字段时视为 0 */
function orderBonusAmount(target: unknown): number {
  const value = (target as { bonus_amount?: unknown } | null | undefined)?.bonus_amount
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? value : 0
}

function hasPaymentType(nextOrder: ResolvedOrder | null): nextOrder is PaymentOrder {
  return !!nextOrder && 'payment_type' in nextOrder && typeof nextOrder.payment_type === 'string' && nextOrder.payment_type.trim() !== ''
}

function normalizeOrderStatus(status: string | null | undefined): string {
  return String(status || '').trim().toUpperCase()
}

function displayOrderStatus(status: string): OrderStatus {
  return normalizeOrderStatus(status) as OrderStatus
}

function isSuccessStatus(status: string | null | undefined): boolean {
  return SUCCESS_STATUSES.has(normalizeOrderStatus(status))
}

function isPendingStatus(status: string | null | undefined): boolean {
  return PENDING_STATUSES.has(normalizeOrderStatus(status))
}

function readRouteQueryString(key: string): string {
  const value = route.query[key]
  if (Array.isArray(value)) {
    return typeof value[0] === 'string' ? value[0] : ''
  }
  return typeof value === 'string' ? value : ''
}

function restoreRecoverySnapshot(context: {
  resumeToken: string
  routeOrderId: number
  routeOutTradeNo: string
}) {
  if (typeof window === 'undefined') {
    return null
  }

  const rawSnapshot = window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY)
  if (!rawSnapshot) {
    return null
  }

  if (context.resumeToken) {
    return readPaymentRecoverySnapshot(rawSnapshot, {
      resumeToken: context.resumeToken,
    })
  }

  if (!context.routeOrderId && !context.routeOutTradeNo) {
    return null
  }

  const restored = readPaymentRecoverySnapshot(rawSnapshot)
  if (!restored) {
    return null
  }

  if (context.routeOrderId > 0 && restored.orderId !== context.routeOrderId) {
    return null
  }

  if (context.routeOutTradeNo && restored.outTradeNo !== context.routeOutTradeNo) {
    return null
  }

  return restored
}

async function resolveOrderFromResumeToken(resumeToken: string): Promise<ResolvedOrder | null> {
  try {
    const result = await paymentAPI.resolveOrderPublicByResumeToken(resumeToken)
    return result.data
  } catch (_err: unknown) {
    return null
  }
}

async function resolveOrderFromOutTradeNo(outTradeNo: string): Promise<ResolvedOrder | null> {
  try {
    const result = await paymentAPI.verifyOrder(outTradeNo)
    return result.data
  } catch (_err: unknown) {
    try {
      const result = await paymentAPI.verifyOrderPublic(outTradeNo)
      return result.data
    } catch (_innerErr: unknown) {
      return null
    }
  }
}

function clearStatusRefreshTimer(): void {
  if (statusRefreshTimer !== null) {
    clearTimeout(statusRefreshTimer)
    statusRefreshTimer = null
  }
}

function clearRecoverySnapshot(): void {
  if (typeof window === 'undefined') return
  clearPaymentRecoverySnapshot(window.localStorage, PAYMENT_RECOVERY_STORAGE_KEY)
}

function clearRecoverySnapshotForTerminalStatus(status: string | null | undefined): void {
  if (!status) return
  if (!isPendingStatus(status)) {
    clearRecoverySnapshot()
  }
}

function scheduleStatusRefresh(refreshOrder: (() => Promise<ResolvedOrder | null>) | null): void {
  clearStatusRefreshTimer()
  if (!refreshOrder || !isPending.value || refreshAttempts.value >= STATUS_REFRESH_MAX_ATTEMPTS) {
    return
  }

  statusRefreshTimer = setTimeout(async () => {
    refreshAttempts.value += 1
    const refreshedOrder = await refreshOrder()
    if (refreshedOrder) {
      setResolvedOrder(refreshedOrder)
      clearRecoverySnapshotForTerminalStatus(refreshedOrder.status)
    }

    if (isPendingStatus(order.value?.status)) {
      scheduleStatusRefresh(refreshOrder)
    }
  }, STATUS_REFRESH_INTERVAL_MS)
}

onMounted(async () => {
  const resumeToken = readRouteQueryString('resume_token')
  const routeOrderId = Number(readRouteQueryString('order_id')) || 0
  let outTradeNo = readRouteQueryString('out_trade_no')
  let orderId = 0
  let resumeTokenLookupFailed = false

  const restored = restoreRecoverySnapshot({
    resumeToken,
    routeOrderId,
    routeOutTradeNo: outTradeNo,
  })
  if (restored?.orderId) {
    orderId = restored.orderId
  }
  if (restored?.currency) {
    currency.value = normalizePaymentCurrency(restored.currency)
  }
  if (!outTradeNo && restored?.outTradeNo) {
    outTradeNo = restored.outTradeNo
  }

  if (resumeToken) {
    const resolvedOrder = await resolveOrderFromResumeToken(resumeToken)
    if (resolvedOrder) {
      setResolvedOrder(resolvedOrder)
      if (!orderId) {
        orderId = hasOrderId(resolvedOrder) ? resolvedOrder.id : 0
      }
    } else if (routeOrderId > 0) {
      resumeTokenLookupFailed = true
      orderId = routeOrderId
    } else {
      resumeTokenLookupFailed = true
    }
  } else if (routeOrderId > 0) {
    orderId = routeOrderId
  }

  const hasLegacyFallbackContext = readRouteQueryString('trade_status').trim() !== ''
  const shouldUsePublicOutTradeNo = outTradeNo !== '' && (hasLegacyFallbackContext || routeOrderId > 0 || orderId > 0)

  if (!order.value && orderId && (!resumeToken || routeOrderId > 0)) {
    try {
      setResolvedOrder(await paymentStore.pollOrderStatus(orderId))
    } catch (_err: unknown) {
      // Order lookup failed, will try legacy fallback below when possible.
    }
  }

  if (!order.value && shouldUsePublicOutTradeNo && (!resumeToken || resumeTokenLookupFailed)) {
    const legacyOrder = await resolveOrderFromOutTradeNo(outTradeNo)
    if (legacyOrder) {
      setResolvedOrder(legacyOrder)
      if (!orderId) {
        orderId = hasOrderId(legacyOrder) ? legacyOrder.id : 0
      }
    }
  }

  if (!order.value && !orderId && outTradeNo && hasLegacyFallbackContext) {
    returnInfo.value = {
      outTradeNo,
      money: String(route.query.money || ''),
      type: String(route.query.type || ''),
      tradeStatus: String(route.query.trade_status || ''),
    }
  }

  const refreshOrder = async (): Promise<ResolvedOrder | null> => {
    if (resumeToken) {
      const resolvedOrder = await resolveOrderFromResumeToken(resumeToken)
      if (resolvedOrder) {
        return resolvedOrder
      }
    }

    if (orderId) {
      try {
        return await paymentStore.pollOrderStatus(orderId)
      } catch (_err: unknown) {
        // Fall through to legacy public verification when order polling is unavailable.
      }
    }

    if (shouldUsePublicOutTradeNo) {
      return await resolveOrderFromOutTradeNo(outTradeNo)
    }

    return null
  }

  if (isPendingStatus(order.value?.status)) {
    scheduleStatusRefresh(refreshOrder)
  } else if (order.value) {
    clearRecoverySnapshotForTerminalStatus(order.value.status)
  } else if (returnInfo.value) {
    clearRecoverySnapshot()
  }
  loading.value = false
})

onBeforeUnmount(() => {
  clearStatusRefreshTimer()
})
</script>
