<template>
  <SectionCard :title="t('payment.admin.dailyRevenue')">
    <div class="h-64">
      <div v-if="loading" class="flex h-full items-center justify-center">
        <LoadingSpinner size="md" />
      </div>
      <Line v-else-if="chartData" :data="chartData" :options="chartOptions" />
      <EmptyState v-else :title="t('payment.admin.noData')" />
    </div>
  </SectionCard>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { EmptyState, SectionCard } from '@/components/page'
import { useChartColors } from '@/design/useChartColors'
import type { DailyPaymentStats } from '@/types/payment'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)

const { t } = useI18n()

const props = defineProps<{
  data: DailyPaymentStats[]
  loading?: boolean
}>()

const { color } = useChartColors()
const currencyColors = ['brand', 'chart-violet', 'warning', 'danger'] as const

const chartData = computed(() => {
  if (!props.data || props.data.length === 0) return null
  const currencies = [...new Set(props.data.flatMap(day => Object.keys(day.amount)))].sort()
  return {
    labels: props.data.map(d => d.date),
    datasets: [
      ...currencies.map((currency, index) => {
        const token = currencyColors[index % currencyColors.length]!
        return {
          label: `${currency} ${t('payment.admin.revenue')}`,
          data: props.data.map(day => day.amount[currency] || 0),
          borderColor: color(token),
          backgroundColor: color(token, 0.1),
          fill: true,
          tension: 0.3,
          pointRadius: 3,
          pointHoverRadius: 5,
        }
      }),
      {
        label: t('payment.admin.orderCount'),
        data: props.data.map(d => d.count),
        borderColor: color('success'),
        backgroundColor: color('success', 0.1),
        fill: false,
        tension: 0.3,
        pointRadius: 3,
        pointHoverRadius: 5,
        yAxisID: 'y1',
      }
    ]
  }
})

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { mode: 'index' as const, intersect: false },
  scales: {
    x: {
      ticks: { color: color('muted-foreground') },
      grid: { color: color('border') },
    },
    y: {
      type: 'linear' as const,
      display: true,
      position: 'left' as const,
      title: { display: true, text: t('payment.admin.revenue'), color: color('muted-foreground') },
      ticks: { color: color('muted-foreground') },
      grid: { color: color('border') },
    },
    y1: {
      type: 'linear' as const,
      display: true,
      position: 'right' as const,
      title: { display: true, text: t('payment.admin.orderCount'), color: color('muted-foreground') },
      ticks: { color: color('muted-foreground') },
      grid: { drawOnChartArea: false },
    }
  },
  plugins: {
    legend: { position: 'top' as const, labels: { color: color('muted-foreground') } },
    tooltip: {
      backgroundColor: color('popover'),
      titleColor: color('foreground'),
      bodyColor: color('foreground'),
      borderColor: color('border'),
      borderWidth: 1,
    },
  }
}))
</script>
