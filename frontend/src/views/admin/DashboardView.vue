<template>
  <AppLayout>
    <div class="space-y-6">
      <PageHeader
        :title="t('admin.dashboard.title')"
        :description="t('admin.dashboard.description')"
      >
        <template #actions>
          <button
            type="button"
            class="inline-flex h-8 items-center gap-1.5 rounded-full border border-border bg-card px-3 text-xs font-medium text-foreground transition-colors hover:bg-muted/60 disabled:pointer-events-none disabled:opacity-50"
            :disabled="loading || chartsLoading"
            @click="loadDashboardStats"
          >
            <Icon
              name="refresh"
              size="xs"
              :stroke-width="1.75"
              :class="{ 'animate-spin': loading || chartsLoading }"
            />
            <span>{{ t('common.refresh') }}</span>
          </button>
        </template>
      </PageHeader>

      <!-- Loading State -->
      <div v-if="loading" class="space-y-6">
        <!-- Row 1 KPI Skeletons -->
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div
            v-for="i in 4"
            :key="`kpi-r1-${i}`"
            class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5"
          >
            <div class="flex items-center justify-between gap-2">
              <Skeleton class="h-3.5 w-20" />
              <Skeleton class="size-9 rounded-full" />
            </div>
            <Skeleton class="mt-3 h-7 w-24" />
            <div class="mt-3 flex items-center justify-between gap-2">
              <Skeleton class="h-3.5 w-28" />
              <Skeleton class="h-5 w-14 rounded-full" />
            </div>
          </div>
        </div>

        <!-- Row 2 KPI Skeletons -->
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div
            v-for="i in 4"
            :key="`kpi-r2-${i}`"
            class="flex items-center gap-3.5 rounded-2xl border border-border/80 bg-card p-4"
          >
            <Skeleton class="size-10 shrink-0 rounded-full" />
            <div class="min-w-0 flex-1 space-y-1.5">
              <Skeleton class="h-3 w-20" />
              <Skeleton class="h-5 w-24" />
              <Skeleton class="h-3 w-32" />
            </div>
          </div>
        </div>

        <!-- Quick Actions Skeleton -->
        <div class="rounded-2xl border border-border/80 bg-card p-5">
          <Skeleton class="mb-3 h-4 w-28" />
          <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
            <Skeleton class="h-16 rounded-xl" />
            <Skeleton class="h-16 rounded-xl" />
          </div>
        </div>

        <!-- Analytics Toolbar & Charts Skeleton -->
        <Skeleton class="h-14 rounded-2xl" />
        <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div
            v-for="i in 2"
            :key="`chart-sk-${i}`"
            class="rounded-2xl border border-border/80 bg-card p-5"
          >
            <Skeleton class="mb-4 h-4 w-36" />
            <Skeleton class="h-64 w-full rounded-xl" />
          </div>
        </div>
      </div>

      <template v-else-if="stats">
        <!-- Row 1: Core Stats -->
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <!-- Total API Keys -->
          <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5">
            <div class="flex items-center justify-between gap-2">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('admin.dashboard.apiKeys') }}
              </p>
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-pink-500 text-white shadow-sm">
                <Icon name="key" size="sm" :stroke-width="2" />
              </div>
            </div>
            <p class="mt-3 truncate text-2xl font-bold tracking-tight tabular-nums text-foreground">
              {{ formatNumber(stats.total_api_keys) }}
            </p>
            <div class="mt-3 flex items-center justify-between gap-2">
              <p class="flex min-w-0 items-center gap-1.5 truncate text-xs tabular-nums">
                <span class="size-1.5 shrink-0 rounded-full bg-emerald-500"></span>
                <span class="truncate font-medium text-emerald-600 dark:text-emerald-400">
                  {{ stats.active_api_keys }} {{ t('common.active') }}
                </span>
              </p>
            </div>
          </div>

          <!-- Service Accounts -->
          <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5">
            <div class="flex items-center justify-between gap-2">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('admin.dashboard.accounts') }}
              </p>
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-emerald-500 text-white shadow-sm">
                <Icon name="server" size="sm" :stroke-width="2" />
              </div>
            </div>
            <p class="mt-3 truncate text-2xl font-bold tracking-tight tabular-nums text-foreground">
              {{ formatNumber(stats.total_accounts) }}
            </p>
            <div class="mt-3 flex items-center justify-between gap-2">
              <p class="flex min-w-0 flex-wrap items-center gap-x-2.5 gap-y-0.5 text-xs tabular-nums">
                <span class="inline-flex items-center gap-1.5 font-medium text-emerald-600 dark:text-emerald-400">
                  <span class="size-1.5 shrink-0 rounded-full bg-emerald-500"></span>
                  {{ stats.normal_accounts }} {{ t('common.active') }}
                </span>
                <span
                  v-if="stats.error_accounts > 0"
                  class="inline-flex items-center gap-1.5 font-medium text-red-600 dark:text-red-400"
                >
                  <span class="size-1.5 shrink-0 rounded-full bg-red-500"></span>
                  {{ stats.error_accounts }} {{ t('common.error') }}
                </span>
              </p>
              <button
                type="button"
                class="inline-flex shrink-0 items-center gap-1 rounded-full border border-border bg-muted/40 px-2.5 py-0.5 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                @click="router.push('/admin/accounts')"
              >
                <span>{{ t('common.view') }}</span>
                <span aria-hidden="true">→</span>
              </button>
            </div>
          </div>

          <!-- Today Requests -->
          <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5">
            <div class="flex items-center justify-between gap-2">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('admin.dashboard.todayRequests') }}
              </p>
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-amber-500 text-white shadow-sm">
                <Icon name="chart" size="sm" :stroke-width="2" />
              </div>
            </div>
            <p class="mt-3 truncate text-2xl font-bold tracking-tight tabular-nums text-foreground">
              {{ formatNumber(stats.today_requests) }}
            </p>
            <div class="mt-3 flex items-center justify-between gap-2">
              <p class="truncate text-xs text-muted-foreground tabular-nums">
                {{ t('common.total') }}:
                <span class="font-medium text-foreground/80">{{ formatNumber(stats.total_requests) }}</span>
              </p>
              <button
                type="button"
                class="inline-flex shrink-0 items-center gap-1 rounded-full border border-border bg-muted/40 px-2.5 py-0.5 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                @click="router.push('/admin/usage')"
              >
                <span>{{ t('common.view') }}</span>
                <span aria-hidden="true">→</span>
              </button>
            </div>
          </div>

          <!-- New Users Today -->
          <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5">
            <div class="flex items-center justify-between gap-2">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('admin.dashboard.users') }}
              </p>
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-violet-500 text-white shadow-sm">
                <Icon name="userPlus" size="sm" :stroke-width="2" />
              </div>
            </div>
            <p class="mt-3 truncate text-2xl font-bold tracking-tight tabular-nums text-foreground">
              +{{ formatNumber(stats.today_new_users) }}
            </p>
            <div class="mt-3 flex items-center justify-between gap-2">
              <p class="truncate text-xs text-muted-foreground tabular-nums">
                {{ t('common.total') }}:
                <span class="font-medium text-foreground/80">{{ formatNumber(stats.total_users) }}</span>
              </p>
              <button
                type="button"
                class="inline-flex shrink-0 items-center gap-1 rounded-full border border-border bg-muted/40 px-2.5 py-0.5 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                @click="router.push('/admin/users')"
              >
                <span>{{ t('common.view') }}</span>
                <span aria-hidden="true">→</span>
              </button>
            </div>
          </div>
        </div>

        <!-- Row 2: Compact Horizontal Stat Cards -->
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <!-- Today Tokens -->
          <div class="flex items-center gap-3.5 rounded-2xl border border-border/80 bg-card p-4">
            <div class="flex size-10 shrink-0 items-center justify-center rounded-full bg-pink-500 text-white shadow-sm">
              <Icon name="cube" size="sm" :stroke-width="2" />
            </div>
            <div class="min-w-0 flex-1">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('admin.dashboard.todayTokens') }}
              </p>
              <p class="mt-0.5 truncate text-lg font-bold tracking-tight tabular-nums text-foreground">
                {{ formatTokens(stats.today_tokens) }}
              </p>
              <p class="mt-0.5 truncate text-xs tabular-nums">
                <span
                  class="font-medium text-emerald-600 dark:text-emerald-400"
                  :title="t('admin.dashboard.actual')"
                  >${{ formatCost(stats.today_actual_cost) }}</span
                >
                <span class="text-muted-foreground/50"> / </span>
                <span
                  class="font-medium text-amber-600 dark:text-amber-400"
                  :title="t('admin.dashboard.accountCost')"
                  >${{ formatCost(stats.today_account_cost) }}</span
                >
                <span class="text-muted-foreground/50"> / </span>
                <span
                  class="text-muted-foreground"
                  :title="t('admin.dashboard.standard')"
                  >${{ formatCost(stats.today_cost) }}</span
                >
              </p>
            </div>
          </div>

          <!-- Total Tokens -->
          <div class="flex items-center gap-3.5 rounded-2xl border border-border/80 bg-card p-4">
            <div class="flex size-10 shrink-0 items-center justify-center rounded-full bg-emerald-500 text-white shadow-sm">
              <Icon name="database" size="sm" :stroke-width="2" />
            </div>
            <div class="min-w-0 flex-1">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('admin.dashboard.totalTokens') }}
              </p>
              <p class="mt-0.5 truncate text-lg font-bold tracking-tight tabular-nums text-foreground">
                {{ formatTokens(stats.total_tokens) }}
              </p>
              <p class="mt-0.5 truncate text-xs tabular-nums">
                <span
                  class="font-medium text-emerald-600 dark:text-emerald-400"
                  :title="t('admin.dashboard.actual')"
                  >${{ formatCost(stats.total_actual_cost) }}</span
                >
                <span class="text-muted-foreground/50"> / </span>
                <span
                  class="font-medium text-amber-600 dark:text-amber-400"
                  :title="t('admin.dashboard.accountCost')"
                  >${{ formatCost(stats.total_account_cost) }}</span
                >
                <span class="text-muted-foreground/50"> / </span>
                <span
                  class="text-muted-foreground"
                  :title="t('admin.dashboard.standard')"
                  >${{ formatCost(stats.total_cost) }}</span
                >
              </p>
            </div>
          </div>

          <!-- Performance (RPM/TPM) -->
          <div class="flex items-center gap-3.5 rounded-2xl border border-border/80 bg-card p-4">
            <div class="flex size-10 shrink-0 items-center justify-center rounded-full bg-amber-500 text-white shadow-sm">
              <Icon name="bolt" size="sm" :stroke-width="2" />
            </div>
            <div class="min-w-0 flex-1">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('admin.dashboard.performance') }}
              </p>
              <div class="mt-0.5 flex items-baseline gap-1.5">
                <p class="truncate text-lg font-bold tracking-tight tabular-nums text-foreground">
                  {{ formatTokens(stats.rpm) }}
                </p>
                <span class="text-xs font-medium text-muted-foreground">RPM</span>
              </div>
              <div class="mt-0.5 flex items-baseline gap-1.5 truncate text-xs text-muted-foreground">
                <span class="font-medium tabular-nums text-foreground/80">
                  {{ formatTokens(stats.tpm) }}
                </span>
                <span>TPM</span>
              </div>
            </div>
          </div>

          <!-- Avg Response Time -->
          <div class="flex items-center gap-3.5 rounded-2xl border border-border/80 bg-card p-4">
            <div class="flex size-10 shrink-0 items-center justify-center rounded-full bg-sky-500 text-white shadow-sm">
              <Icon name="clock" size="sm" :stroke-width="2" />
            </div>
            <div class="min-w-0 flex-1">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('admin.dashboard.avgResponse') }}
              </p>
              <p class="mt-0.5 truncate text-lg font-bold tracking-tight tabular-nums text-foreground">
                {{ formatDuration(stats.average_duration_ms) }}
              </p>
              <p class="mt-0.5 truncate text-xs text-muted-foreground tabular-nums">
                {{ stats.active_users }} {{ t('admin.dashboard.activeUsers') }}
              </p>
            </div>
          </div>
        </div>

        <!-- Quick Actions -->
        <div class="rounded-2xl border border-border/80 bg-card p-5">
          <div class="mb-3 flex items-center justify-between">
            <h3 class="text-sm font-semibold tracking-tight text-foreground">
              {{ t('admin.dashboard.quickActions') }}
            </h3>
          </div>
          <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
            <button
              v-if="canUseBatchImage"
              type="button"
              class="group flex items-center gap-3 rounded-xl border border-border/80 bg-card p-3 text-left transition-colors hover:bg-muted/40"
              @click="router.push('/batch-image')"
            >
              <span class="flex size-9 shrink-0 items-center justify-center rounded-full bg-muted text-foreground transition-colors group-hover:bg-background">
                <Icon name="sparkles" size="sm" :stroke-width="1.75" />
              </span>
              <span class="min-w-0 flex-1">
                <span class="block text-sm font-medium text-foreground">
                  {{ t('admin.dashboard.batchImage') }}
                </span>
                <span class="block truncate text-xs text-muted-foreground">
                  {{ t('admin.dashboard.batchImageDesc') }}
                </span>
              </span>
              <Icon
                name="chevronRight"
                size="sm"
                class="shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5 group-hover:text-foreground"
              />
            </button>
            <button
              type="button"
              class="group flex items-center gap-3 rounded-xl border border-border/80 bg-card p-3 text-left transition-colors hover:bg-muted/40"
              @click="router.push('/admin/groups')"
            >
              <span class="flex size-9 shrink-0 items-center justify-center rounded-full bg-muted text-foreground transition-colors group-hover:bg-background">
                <Icon name="grid" size="sm" :stroke-width="1.75" />
              </span>
              <span class="min-w-0 flex-1">
                <span class="block text-sm font-medium text-foreground">
                  {{ t('admin.dashboard.groupPricing') }}
                </span>
                <span class="block truncate text-xs text-muted-foreground">
                  {{ t('admin.dashboard.groupPricingDesc') }}
                </span>
              </span>
              <Icon
                name="chevronRight"
                size="sm"
                class="shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5 group-hover:text-foreground"
              />
            </button>
          </div>
        </div>

        <!-- Charts Section -->
        <div class="space-y-6">
          <!-- Date Range Filter -->
          <div class="rounded-2xl border border-border/80 bg-card px-4 py-3">
            <div class="flex flex-wrap items-center gap-3">
              <div class="flex items-center gap-2">
                <span class="text-xs font-medium text-muted-foreground">
                  {{ t('admin.dashboard.timeRange') }}:
                </span>
                <DateRangePicker
                  v-model:start-date="startDate"
                  v-model:end-date="endDate"
                  @change="onDateRangeChange"
                />
              </div>
              <div class="ml-auto flex items-center gap-2">
                <span class="text-xs font-medium text-muted-foreground">
                  {{ t('admin.dashboard.granularity') }}:
                </span>
                <div
                  class="inline-flex items-center rounded-full border border-border bg-background p-0.5"
                  role="group"
                  :aria-label="t('admin.dashboard.granularity')"
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
                    @click="setGranularity(opt.value)"
                  >
                    {{ opt.label }}
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- Charts Grid -->
          <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <ModelDistributionChart
              :model-stats="modelStats"
              :enable-ranking-view="true"
              :ranking-items="rankingItems"
              :ranking-total-actual-cost="rankingTotalActualCost"
              :ranking-total-requests="rankingTotalRequests"
              :ranking-total-tokens="rankingTotalTokens"
              :loading="chartsLoading"
              :ranking-loading="rankingLoading"
              :ranking-error="rankingError"
              :start-date="startDate"
              :end-date="endDate"
              @ranking-click="goToUserUsage"
            />
            <TokenUsageTrend :trend-data="trendData" :loading="chartsLoading" />
          </div>

          <!-- User Usage Trend (Full Width) -->
          <div class="rounded-2xl border border-border/80 bg-card p-5">
            <div class="mb-4 flex items-center justify-between gap-3">
              <h3 class="text-sm font-semibold tracking-tight text-foreground">
                {{ t('admin.dashboard.recentUsage') }} (Top 12)
              </h3>
              <div
                class="inline-flex items-center rounded-full border border-border bg-background p-0.5"
                role="group"
                :aria-label="t('admin.dashboard.recentUsage')"
              >
                <button
                  v-for="metric in (['tokens', 'actual_cost'] as const)"
                  :key="metric"
                  type="button"
                  class="rounded-full px-3 py-1 text-xs font-medium transition-colors"
                  :class="
                    userTrendMetric === metric
                      ? 'bg-foreground text-background shadow-sm'
                      : 'text-muted-foreground hover:text-foreground'
                  "
                  :aria-pressed="userTrendMetric === metric"
                  @click="setUserTrendMetric(metric)"
                >
                  {{ t(metric === 'tokens' ? 'admin.dashboard.tokens' : 'admin.dashboard.actualSpending') }}
                </button>
              </div>
            </div>
            <div class="h-64">
              <div v-if="userTrendLoading" class="flex h-full items-center justify-center">
                <LoadingSpinner size="md" />
              </div>
              <Line v-else-if="userTrendChartData" :data="userTrendChartData" :options="lineOptions" />
              <div
                v-else
                class="flex h-full items-center justify-center text-sm text-muted-foreground"
              >
                {{ t('admin.dashboard.noDataAvailable') }}
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
import { adminAPI } from '@/api/admin'
import type {
  DashboardStats,
  TrendDataPoint,
  ModelStat,
  UserUsageTrendPoint,
  UserSpendingRankingItem
} from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import { Skeleton } from '@/components/ui/skeleton'
import Icon from '@/components/icons/Icon.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'

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

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
)

const appStore = useAppStore()
const router = useRouter()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
const stats = ref<DashboardStats | null>(null)
const loading = ref(false)
const chartsLoading = ref(false)
const userTrendLoading = ref(false)
const rankingLoading = ref(false)
const rankingError = ref(false)

// Chart data
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const userTrend = ref<UserUsageTrendPoint[]>([])
const userTrendMetric = ref<'tokens' | 'actual_cost'>('tokens')
const rankingItems = ref<UserSpendingRankingItem[]>([])
const rankingTotalActualCost = ref(0)
const rankingTotalRequests = ref(0)
const rankingTotalTokens = ref(0)
let chartLoadSeq = 0
let usersTrendLoadSeq = 0
let rankingLoadSeq = 0
const rankingLimit = 12

// Helper function to format date in local timezone
const formatLocalDate = (date: Date): string => {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

const getLast24HoursRangeDates = (): { start: string; end: string } => {
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  return {
    start: formatLocalDate(start),
    end: formatLocalDate(end)
  }
}

// Date range
const granularity = ref<'day' | 'hour'>('hour')
const defaultRange = getLast24HoursRangeDates()
const startDate = ref(defaultRange.start)
const endDate = ref(defaultRange.end)

// Granularity options for segmented toggle
const granularityOptions = computed<Array<{ value: 'day' | 'hour'; label: string }>>(() => [
  { value: 'day', label: t('admin.dashboard.day') },
  { value: 'hour', label: t('admin.dashboard.hour') }
])

const setGranularity = (value: 'day' | 'hour') => {
  if (granularity.value === value) return
  granularity.value = value
  loadChartData()
}

// Dark mode detection
const isDarkMode = computed(() => {
  return document.documentElement.classList.contains('dark')
})

// Chart colors
const chartColors = computed(() => ({
  text: isDarkMode.value ? '#a3a3a3' : '#525252',
  grid: isDarkMode.value ? 'rgba(255, 255, 255, 0.08)' : 'rgba(0, 0, 0, 0.06)'
}))

// Line chart options (for user trend chart)
const lineOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    intersect: false,
    mode: 'index' as const
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: chartColors.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 15,
        font: {
          size: 11
        }
      }
    },
    tooltip: {
      itemSort: (a: any, b: any) => {
        const aValue = typeof a?.raw === 'number' ? a.raw : Number(a?.parsed?.y ?? 0)
        const bValue = typeof b?.raw === 'number' ? b.raw : Number(b?.parsed?.y ?? 0)
        return bValue - aValue
      },
      callbacks: {
        label: (context: any) => {
          return `${context.dataset.label}: ${formatUserTrendValue(Number(context.raw))}`
        }
      }
    }
  },
  scales: {
    x: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        }
      }
    },
    y: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        },
        callback: (value: string | number) => formatUserTrendValue(Number(value))
      }
    }
  }
}))

// User trend chart data
const userTrendChartData = computed(() => {
  if (!userTrend.value?.length) return null

  const getDisplayName = (point: UserUsageTrendPoint): string => {
    const username = point.username?.trim()
    if (username) {
      return username
    }

    const email = point.email?.trim()
    if (email) {
      return email
    }

    return t('admin.redeem.userPrefix', { id: point.user_id })
  }

  // Group by user_id to avoid merging different users with the same display name
  const userGroups = new Map<number, { name: string; data: Map<string, number> }>()
  const allDates = new Set<string>()

  userTrend.value.forEach((point) => {
    allDates.add(point.date)
    const key = point.user_id
    if (!userGroups.has(key)) {
      userGroups.set(key, { name: getDisplayName(point), data: new Map() })
    }
    userGroups.get(key)!.data.set(point.date, userTrendMetric.value === 'tokens' ? point.tokens : point.actual_cost)
  })

  const sortedDates = Array.from(allDates).sort()
  const colors = [
    '#3b82f6',
    '#10b981',
    '#f59e0b',
    '#ef4444',
    '#8b5cf6',
    '#ec4899',
    '#14b8a6',
    '#f97316',
    '#6366f1',
    '#84cc16',
    '#06b6d4',
    '#a855f7'
  ]

  const datasets = Array.from(userGroups.values()).map((group, idx) => ({
    label: group.name,
    data: sortedDates.map((date) => group.data.get(date) || 0),
    borderColor: colors[idx % colors.length],
    backgroundColor: `${colors[idx % colors.length]}20`,
    fill: false,
    tension: 0.3
  }))

  return {
    labels: sortedDates,
    datasets
  }
})

// Format helpers
const formatUserTrendValue = (value: number): string =>
  userTrendMetric.value === 'tokens' ? formatTokens(value) : `$${formatCost(value)}`

const formatTokens = (value: number | undefined): string => {
  if (value === undefined || value === null) return '0'
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2)}B`
  } else if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2)}M`
  } else if (value >= 1_000) {
    return `${(value / 1_000).toFixed(2)}K`
  }
  return value.toLocaleString()
}

const toFiniteNumber = (value: unknown): number => {
  const numberValue = Number(value)
  return Number.isFinite(numberValue) ? numberValue : 0
}

const formatNumber = (value: number | null | undefined): string => {
  return toFiniteNumber(value).toLocaleString()
}

const formatCost = (value: number | null | undefined): string => {
  const safeValue = toFiniteNumber(value)
  if (safeValue >= 1000) {
    return (safeValue / 1000).toFixed(2) + 'K'
  } else if (safeValue >= 1) {
    return safeValue.toFixed(2)
  } else if (safeValue >= 0.01) {
    return safeValue.toFixed(3)
  }
  return safeValue.toFixed(4)
}

const formatDuration = (ms: number): string => {
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(2)}s`
  }
  return `${Math.round(ms)}ms`
}

const goToUserUsage = (item: UserSpendingRankingItem) => {
  void router.push({
    path: '/admin/usage',
    query: {
      user_id: String(item.user_id),
      start_date: startDate.value,
      end_date: endDate.value
    }
  })
}

// Date range change handler
const onDateRangeChange = (range: {
  startDate: string
  endDate: string
  preset: string | null
}) => {
  // Auto-select granularity based on date range
  const start = new Date(range.startDate)
  const end = new Date(range.endDate)
  const daysDiff = Math.ceil((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24))

  // If range is 1 day, use hourly granularity
  if (daysDiff <= 1) {
    granularity.value = 'hour'
  } else {
    granularity.value = 'day'
  }

  loadChartData()
}

// Load data
const loadDashboardSnapshot = async (includeStats: boolean) => {
  const currentSeq = ++chartLoadSeq
  if (includeStats && !stats.value) {
    loading.value = true
  }
  chartsLoading.value = true
  try {
    const response = await adminAPI.dashboard.getSnapshotV2({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      include_stats: includeStats,
      include_trend: true,
      include_model_stats: true,
      include_group_stats: false,
      include_users_trend: false
    })
    if (currentSeq !== chartLoadSeq) return
    if (includeStats && response.stats) {
      stats.value = response.stats
    }
    trendData.value = response.trend || []
    modelStats.value = response.models || []
  } catch (error) {
    if (currentSeq !== chartLoadSeq) return
    appStore.showError(t('admin.dashboard.failedToLoad'))
    console.error('Error loading dashboard snapshot:', error)
  } finally {
    if (currentSeq === chartLoadSeq) {
      loading.value = false
      chartsLoading.value = false
    }
  }
}

const setUserTrendMetric = (metric: 'tokens' | 'actual_cost') => {
  if (userTrendMetric.value === metric) return
  userTrendMetric.value = metric
  userTrend.value = []
  loadUsersTrend()
}

const loadUsersTrend = async () => {
  const currentSeq = ++usersTrendLoadSeq
  userTrendLoading.value = true
  try {
    const response = await adminAPI.dashboard.getUserUsageTrend({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      limit: 12,
      metric: userTrendMetric.value
    })
    if (currentSeq !== usersTrendLoadSeq) return
    userTrend.value = response.trend || []
  } catch (error) {
    if (currentSeq !== usersTrendLoadSeq) return
    console.error('Error loading users trend:', error)
    userTrend.value = []
  } finally {
    if (currentSeq === usersTrendLoadSeq) {
      userTrendLoading.value = false
    }
  }
}

const loadUserSpendingRanking = async () => {
  const currentSeq = ++rankingLoadSeq
  rankingLoading.value = true
  rankingError.value = false
  try {
    const response = await adminAPI.dashboard.getUserSpendingRanking({
      start_date: startDate.value,
      end_date: endDate.value,
      limit: rankingLimit
    })
    if (currentSeq !== rankingLoadSeq) return
    rankingItems.value = response.ranking || []
    rankingTotalActualCost.value = response.total_actual_cost || 0
    rankingTotalRequests.value = response.total_requests || 0
    rankingTotalTokens.value = response.total_tokens || 0
  } catch (error) {
    if (currentSeq !== rankingLoadSeq) return
    console.error('Error loading user spending ranking:', error)
    rankingItems.value = []
    rankingTotalActualCost.value = 0
    rankingTotalRequests.value = 0
    rankingTotalTokens.value = 0
    rankingError.value = true
  } finally {
    if (currentSeq === rankingLoadSeq) {
      rankingLoading.value = false
    }
  }
}

const loadDashboardStats = async () => {
  await Promise.all([
    loadDashboardSnapshot(true),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

const loadChartData = async () => {
  await Promise.all([
    loadDashboardSnapshot(false),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

onMounted(() => {
  void refreshBatchImageAccess()
  loadDashboardStats()
})
</script>

<style scoped>
</style>
