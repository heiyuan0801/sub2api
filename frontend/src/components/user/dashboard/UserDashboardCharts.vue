<template>
  <div class="space-y-6">
    <!-- Date Range Filter -->
    <div class="rounded-2xl border border-border/80 bg-card px-4 py-3">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex flex-wrap items-center gap-2.5">
          <span class="text-xs font-medium text-muted-foreground">{{ t('dashboard.timeRange') }}</span>
          <DateRangePicker
            :start-date="startDate"
            :end-date="endDate"
            @update:startDate="emit('update:startDate', $event)"
            @update:endDate="emit('update:endDate', $event)"
            @change="emit('dateRangeChange', $event)"
          />
          <button
            type="button"
            :disabled="loading"
            class="inline-flex h-8 items-center gap-1.5 rounded-full border border-border bg-background px-3 text-xs font-medium text-foreground transition-colors hover:bg-muted/60 disabled:pointer-events-none disabled:opacity-50"
            @click="emit('refresh')"
          >
            <Icon name="refresh" size="xs" :stroke-width="1.75" :class="loading ? 'animate-spin' : ''" />
            <span>{{ t('common.refresh') }}</span>
          </button>
        </div>
        <div class="flex items-center gap-2">
          <span class="text-xs font-medium text-muted-foreground">{{ t('dashboard.granularity') }}</span>
          <div
            class="inline-flex items-center rounded-full border border-border bg-background p-0.5"
            role="group"
            :aria-label="t('dashboard.granularity')"
          >
            <button
              v-for="opt in granularityOptions"
              :key="opt.value"
              type="button"
              class="rounded-full px-3 py-1 text-xs font-medium transition-colors"
              :class="
                granularity === opt.value
                  ? 'bg-foreground text-background shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              "
              :aria-pressed="granularity === opt.value"
              @click="selectGranularity(opt.value)"
            >
              {{ opt.label }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Charts Grid -->
    <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
      <!-- Model Distribution Chart -->
      <div class="relative overflow-hidden rounded-2xl border border-border/80 bg-card p-5">
        <div v-if="loading" class="absolute inset-0 z-10 flex items-center justify-center bg-background/60 backdrop-blur-[2px]">
          <LoadingSpinner size="md" />
        </div>
        <h3 class="mb-4 text-sm font-semibold text-foreground">{{ t('dashboard.modelDistribution') }}</h3>
        <div class="flex flex-col items-center gap-5 sm:flex-row">
          <div class="h-44 w-44 shrink-0">
            <Doughnut v-if="modelData" :data="modelData" :options="doughnutOptions" />
            <div v-else class="flex h-full items-center justify-center text-sm text-muted-foreground">{{ t('dashboard.noDataAvailable') }}</div>
          </div>
          <div class="max-h-48 w-full min-w-0 flex-1 overflow-auto rounded-xl border border-border/80">
            <table class="w-full text-xs">
              <thead class="sticky top-0 bg-muted/50 backdrop-blur-sm">
                <tr class="border-b border-border text-muted-foreground">
                  <th class="px-2.5 py-2 text-left font-medium">{{ t('dashboard.model') }}</th>
                  <th class="px-2 py-2 text-right font-medium">{{ t('dashboard.requests') }}</th>
                  <th class="px-2 py-2 text-right font-medium">{{ t('dashboard.tokens') }}</th>
                  <th class="px-2 py-2 text-right font-medium">{{ t('dashboard.actual') }}</th>
                  <th class="px-2.5 py-2 text-right font-medium">{{ t('dashboard.standard') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border">
                <tr
                  v-for="(model, idx) in models"
                  :key="model.model"
                  class="transition-colors hover:bg-muted/40"
                >
                  <td class="max-w-[110px] truncate px-2.5 py-1.5 font-medium text-foreground" :title="model.model">
                    <span class="inline-flex items-center gap-1.5">
                      <span
                        class="size-2 shrink-0 rounded-full"
                        :style="{ backgroundColor: CHART_COLORS[idx % CHART_COLORS.length] }"
                      />
                      <span class="truncate">{{ model.model }}</span>
                    </span>
                  </td>
                  <td class="px-2 py-1.5 text-right tabular-nums text-muted-foreground">{{ formatNumber(model.requests) }}</td>
                  <td class="px-2 py-1.5 text-right tabular-nums text-muted-foreground">{{ formatTokens(model.total_tokens) }}</td>
                  <td class="px-2 py-1.5 text-right font-medium tabular-nums text-emerald-600 dark:text-emerald-400">${{ formatCost(model.actual_cost) }}</td>
                  <td class="px-2.5 py-1.5 text-right tabular-nums text-muted-foreground">${{ formatCost(model.cost) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- Token Usage Trend Chart -->
      <TokenUsageTrend :trend-data="trend" :loading="loading" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Icon from '@/components/icons/Icon.vue'
import { Doughnut } from 'vue-chartjs'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import type { TrendDataPoint, ModelStat } from '@/types'
import { formatCostFixed as formatCost, formatNumberLocaleString as formatNumber, formatTokensK as formatTokens } from '@/utils/format'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, ArcElement, Title, Tooltip, Legend, Filler } from 'chart.js'
ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, ArcElement, Title, Tooltip, Legend, Filler)

const CHART_COLORS = [
  '#2563eb',
  '#0d9488',
  '#d97706',
  '#7c3aed',
  '#0284c7',
  '#e11d48',
  '#16a34a',
  '#64748b',
]

const props = defineProps<{ loading: boolean, startDate: string, endDate: string, granularity: string, trend: TrendDataPoint[], models: ModelStat[] }>()
const emit = defineEmits(['update:startDate', 'update:endDate', 'update:granularity', 'dateRangeChange', 'granularityChange', 'refresh'])
const { t } = useI18n()

const granularityOptions = computed(() => [
  { value: 'day', label: t('dashboard.day') },
  { value: 'hour', label: t('dashboard.hour') },
])

const selectGranularity = (value: string) => {
  if (props.granularity === value) return
  emit('update:granularity', value)
  emit('granularityChange')
}

const modelData = computed(() => !props.models?.length ? null : {
  labels: props.models.map((m: ModelStat) => m.model),
  datasets: [{
    data: props.models.map((m: ModelStat) => m.total_tokens),
    backgroundColor: props.models.map((_: ModelStat, i: number) => CHART_COLORS[i % CHART_COLORS.length]),
    borderWidth: 2,
    borderColor: 'transparent',
  }]
})

const doughnutOptions = {
  responsive: true,
  maintainAspectRatio: false,
  cutout: '64%',
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (context: any) => `${context.label}: ${formatTokens(context.parsed)} tokens`
      }
    }
  }
}
</script>
