<template>
  <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
    <StatCard :label="t('payment.admin.todayRevenue')" :icon="CircleDollarSign" :description="`${stats.today_count} ${t('payment.admin.orders')}`">
      <template #value>
        <span v-for="[currency, amount] in sortedAmounts(stats.today_amount)" :key="currency" class="block">
          {{ formatMoney(currency, amount) }}
        </span>
      </template>
    </StatCard>

    <StatCard :label="t('payment.admin.totalRevenue')" :icon="CreditCard" :description="`${stats.total_count} ${t('payment.admin.orders')}`">
      <template #value>
        <span v-for="[currency, amount] in sortedAmounts(stats.total_amount)" :key="currency" class="block">
          {{ formatMoney(currency, amount) }}
        </span>
      </template>
    </StatCard>

    <StatCard :label="t('payment.admin.todayOrders')" :icon="ReceiptText" :value="stats.today_count" />
    <StatCard :label="t('payment.admin.avgAmount')" :icon="ChartNoAxesColumn">
      <template #value>
        <span v-for="[currency, amount] in sortedAmounts(stats.avg_amount)" :key="currency" class="block">
          {{ formatMoney(currency, amount) }}
        </span>
      </template>
    </StatCard>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ChartNoAxesColumn, CircleDollarSign, CreditCard, ReceiptText } from '@lucide/vue'
import { StatCard } from '@/components/page'
import type { CurrencyAmounts, DashboardStats } from '@/types/payment'

const { t } = useI18n()

defineProps<{
  stats: DashboardStats
}>()

function sortedAmounts(amounts: CurrencyAmounts): [string, number][] {
  return Object.entries(amounts).sort(([left], [right]) => left.localeCompare(right))
}

function formatMoney(currency: string, amount: number): string {
  return new Intl.NumberFormat(undefined, { style: 'currency', currency }).format(amount)
}
</script>
