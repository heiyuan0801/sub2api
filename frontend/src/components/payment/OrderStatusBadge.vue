<template>
  <StatusBadge :label="statusLabel" :tone="statusTone" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { OrderStatus } from '@/types/payment'
import StatusBadge from '@/components/page/StatusBadge.vue'
import type { PageTone } from '@/components/page/tones'

const props = defineProps<{
  status: OrderStatus
}>()

const { t } = useI18n()

const statusMap: Record<OrderStatus, { key: string; tone: PageTone }> = {
  PENDING: { key: 'payment.status.pending', tone: 'warning' },
  PAID: { key: 'payment.status.paid', tone: 'info' },
  RECHARGING: { key: 'payment.status.recharging', tone: 'info' },
  COMPLETED: { key: 'payment.status.completed', tone: 'success' },
  EXPIRED: { key: 'payment.status.expired', tone: 'neutral' },
  CANCELLED: { key: 'payment.status.cancelled', tone: 'neutral' },
  FAILED: { key: 'payment.status.failed', tone: 'danger' },
  REFUND_REQUESTED: { key: 'payment.status.refund_requested', tone: 'warning' },
  REFUNDING: { key: 'payment.status.refunding', tone: 'warning' },
  REFUND_PENDING: { key: 'payment.status.refund_pending', tone: 'warning' },
  REFUNDED: { key: 'payment.status.refunded', tone: 'info' },
  PARTIALLY_REFUNDED: { key: 'payment.status.partially_refunded', tone: 'info' },
  REFUND_FAILED: { key: 'payment.status.refund_failed', tone: 'danger' },
}

const statusLabel = computed(() => {
  const entry = statusMap[props.status]
  return entry ? t(entry.key) : props.status
})

const statusTone = computed(() => statusMap[props.status]?.tone ?? 'neutral')
</script>
