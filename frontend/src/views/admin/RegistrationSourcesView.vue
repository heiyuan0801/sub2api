<template>
  <AppLayout>
    <div class="space-y-6">
      <PageHeader
        :title="t('operations.registrationSources')"
        :description="t('operations.registrationDescription')"
      >
        <template #actions>
          <button type="button" class="btn btn-secondary" @click="showCampaignLink = true">
            <Icon name="link" size="sm" aria-hidden="true" />
            {{ t('operations.campaignLink.title') }}
          </button>
          <button type="button" class="btn btn-secondary" :disabled="exporting || loading" @click="downloadCSV">
            <Icon name="download" size="sm" aria-hidden="true" />
            {{ t('operations.exportCSV') }}
          </button>
        </template>
      </PageHeader>

      <!-- Filter Toolbar -->
      <div class="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border/80 bg-card p-4 shadow-sm">
        <div class="flex flex-wrap items-center gap-3">
          <div class="inline-flex flex-wrap items-center rounded-full border border-border bg-background p-0.5">
            <button
              v-for="opt in rangeOptions"
              :key="opt.value"
              type="button"
              :class="[
                'rounded-full px-3 py-1 text-xs font-medium transition-colors',
                range === opt.value
                  ? 'bg-foreground text-background shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              ]"
              @click="selectRange(opt.value)"
            >
              {{ t(opt.label) }}
            </button>
          </div>
          <DateRangePicker
            v-if="range === 'custom'"
            v-model:start-date="startDate"
            v-model:end-date="endDate"
            @change="reload"
          />
        </div>

        <div class="inline-flex items-center rounded-full border border-border bg-background p-0.5">
          <button
            v-for="opt in dimensionOptions"
            :key="opt.value"
            type="button"
            :class="[
              'rounded-full px-3 py-1 text-xs font-medium transition-colors',
              dimension === opt.value
                ? 'bg-foreground text-background shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            ]"
            @click="selectDimension(opt.value)"
          >
            {{ t(opt.label) }}
          </button>
        </div>
      </div>

      <!-- KPI Metric Cards -->
      <div v-if="loading && !summary" class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <div
          v-for="i in 4"
          :key="`reg-kpi-sk-${i}`"
          class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5 shadow-sm"
        >
          <div class="flex items-center justify-between gap-2">
            <Skeleton class="h-3.5 w-24" />
            <Skeleton class="size-9 rounded-full" />
          </div>
          <Skeleton class="mt-3 h-7 w-20" />
        </div>
      </div>
      <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <div
          v-for="metric in metrics"
          :key="metric.label"
          class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5 shadow-sm"
        >
          <div class="flex items-center justify-between gap-2">
            <p class="truncate text-xs font-medium text-muted-foreground">{{ t(metric.label) }}</p>
            <div
              :class="[
                'flex size-9 shrink-0 items-center justify-center rounded-full text-white shadow-sm',
                metric.iconBg
              ]"
            >
              <Icon :name="metric.icon" size="sm" :stroke-width="2" />
            </div>
          </div>
          <p class="mt-3 truncate text-2xl font-bold tracking-tight tabular-nums text-foreground">
            {{ metric.value == null ? '—' : number(metric.value) }}
          </p>
        </div>
      </div>

      <!-- Charts -->
      <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div class="min-w-0 rounded-2xl border border-border/80 bg-card p-5 shadow-sm lg:col-span-2">
          <h2 class="mb-4 text-sm font-semibold tracking-tight text-foreground">{{ t('operations.signupTrend') }}</h2>
          <Skeleton v-if="loading && !trend.labels.length" class="h-64 w-full rounded-xl" />
          <div v-else-if="trend.labels.length" class="h-64">
            <Line :data="trend" :options="trendOptions" :aria-label="t('operations.signupTrend')" />
          </div>
          <p v-else class="flex h-64 items-center justify-center text-sm text-muted-foreground">
            {{ t('empty.noData') }}
          </p>
        </div>
        <div class="min-w-0 rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
          <h2 class="mb-4 text-sm font-semibold tracking-tight text-foreground">{{ t('operations.sourceShare') }}</h2>
          <Skeleton v-if="loading && !rankings.length" class="mx-auto size-56 rounded-full" />
          <div v-else-if="rankings.length" class="mx-auto h-64 max-w-72">
            <Doughnut :data="distribution" :options="distributionOptions" :aria-label="t('operations.sourceShare')" />
          </div>
          <p v-else class="flex h-64 items-center justify-center text-sm text-muted-foreground">
            {{ t('empty.noData') }}
          </p>
        </div>
      </div>

      <!-- Rankings Table -->
      <section class="space-y-3">
        <h2 class="text-sm font-semibold tracking-tight text-foreground">{{ t('operations.rankings') }}</h2>
        <div class="overflow-hidden rounded-2xl border border-border/80 bg-card shadow-sm">
          <DataTable :columns="rankColumns" :data="rankings" row-key="source" :loading="loading">
            <template #cell-source="{ value }">
              <button
                type="button"
                class="font-medium text-foreground underline-offset-4 transition-colors hover:underline"
                @click="selectSource(value)"
              >
                {{ sourceLabel(value) }}
              </button>
            </template>
            <template #cell-count="{ value }">
              <span class="font-medium tabular-nums text-foreground">{{ number(value) }}</span>
            </template>
            <template #cell-percentage="{ value }">
              <span class="tabular-nums text-muted-foreground">{{ number(value, 2) }}%</span>
            </template>
          </DataTable>
        </div>
      </section>

      <!-- Users Table -->
      <section ref="usersSection" class="space-y-3">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h2 class="text-sm font-semibold tracking-tight text-foreground">{{ t('operations.usersInRange') }}</h2>
          <label class="min-w-48">
            <span class="sr-only">{{ t('operations.allSources') }}</span>
            <select v-model="source" class="input" @change="setPage(1)">
              <option value="">{{ t('operations.allSources') }}</option>
              <option v-for="item in rankings" :key="item.source" :value="item.source">{{ sourceLabel(item.source) }}</option>
            </select>
          </label>
        </div>
        <div class="overflow-hidden rounded-2xl border border-border/80 bg-card shadow-sm">
          <DataTable :columns="userColumns" :data="userRows" :loading="usersLoading">
            <template #cell-id="{ value }">
              <span class="font-mono text-xs tabular-nums text-muted-foreground">{{ value }}</span>
            </template>
            <template #cell-selected_source="{ row }">
              <span class="text-foreground">{{ selectedLabel(row.selected_source) }}</span>
              <div v-if="row.source_note" class="mt-0.5 break-words text-xs text-muted-foreground">{{ row.source_note }}</div>
            </template>
            <template #cell-utm_source="{ row }">
              <span class="text-foreground">{{ row.utm_source || '—' }}</span>
              <div v-if="row.utm_campaign" class="mt-0.5 break-words text-xs text-muted-foreground">{{ row.utm_campaign }}</div>
            </template>
            <template #cell-promo_code="{ value }">
              <code
                v-if="value"
                class="rounded border border-border/60 bg-muted px-2 py-0.5 font-mono text-xs tabular-nums text-foreground"
              >{{ value }}</code>
              <span v-else class="text-muted-foreground">—</span>
            </template>
            <template #cell-created_at="{ value }">
              <span class="tabular-nums text-muted-foreground">{{ formatDateTime(value) }}</span>
            </template>
          </DataTable>
        </div>
        <Pagination
          v-if="total"
          :page="page"
          :total="total"
          :page-size="pageSize"
          @update:page="setPage"
          @update:page-size="setPageSize"
        />
      </section>
    </div>
    <RegistrationCampaignLinkDialog :show="showCampaignLink" @close="showCampaignLink = false" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArcElement, CategoryScale, Chart as ChartJS, Legend, LinearScale, LineElement, PointElement, Tooltip
} from 'chart.js'
import { Doughnut, Line } from 'vue-chartjs'
import { getRegistrationSources, listRegistrationSourceUsers, exportRegistrationSources } from '@/api/admin/registrationSources'
import AppLayout from '@/components/layout/AppLayout.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import RegistrationCampaignLinkDialog from '@/components/admin/registration/RegistrationCampaignLinkDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { Skeleton } from '@/components/ui/skeleton'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import { useAppStore } from '@/stores/app'
import type { RegistrationSourceQuery, RegistrationSourceSummary, RegistrationSourceUser } from '@/types/operations'
import { formatDateLocalInput, formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

ChartJS.register(ArcElement, CategoryScale, Legend, LinearScale, LineElement, PointElement, Tooltip)

const { t, locale } = useI18n()
const app = useAppStore()
const range = ref<RegistrationSourceQuery['range']>('7d')
const dimension = ref<RegistrationSourceQuery['dimension']>('selected')
const rangeOptions: Array<{ value: RegistrationSourceQuery['range']; label: string }> = [
  { value: 'today', label: 'dates.today' },
  { value: '7d', label: 'dates.last7Days' },
  { value: '30d', label: 'dates.last30Days' },
  { value: 'custom', label: 'operations.customRange' },
]
const dimensionOptions: Array<{ value: RegistrationSourceQuery['dimension']; label: string }> = [
  { value: 'selected', label: 'operations.selectedSource' },
  { value: 'utm', label: 'operations.utmSource' },
]
const today = new Date()
const weekStart = new Date(today)
weekStart.setDate(today.getDate() - 6)
const startDate = ref(formatDateLocalInput(weekStart))
const endDate = ref(formatDateLocalInput(today))
const source = ref('')
const summary = ref<RegistrationSourceSummary>()
const userRows = ref<RegistrationSourceUser[]>([])
const loading = ref(false)
const usersLoading = ref(false)
const exporting = ref(false)
const showCampaignLink = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const usersSection = ref<HTMLElement>()
let summaryRequest: AbortController | undefined
let usersRequest: AbortController | undefined

const metrics = computed(() => [
  { label: 'operations.totalRegistrations', value: summary.value?.total_registrations, icon: 'users' as const, iconBg: 'bg-blue-500' },
  { label: 'operations.todayRegistrations', value: summary.value?.today_registrations, icon: 'userPlus' as const, iconBg: 'bg-emerald-500' },
  { label: 'operations.rangeRegistrations', value: summary.value?.range_registrations, icon: 'trendingUp' as const, iconBg: 'bg-violet-500' },
  { label: 'operations.unreported', value: summary.value?.unreported_count, icon: 'questionCircle' as const, iconBg: 'bg-amber-500' },
])

function selectRange(value: RegistrationSourceQuery['range']) {
  if (range.value === value) return
  range.value = value
  void reload()
}

function selectDimension(value: RegistrationSourceQuery['dimension']) {
  if (dimension.value === value) return
  dimension.value = value
  void reload()
}
const rankings = computed(() => [...(summary.value?.channels ?? [])].sort((a, b) => b.count - a.count || a.source.localeCompare(b.source)))
const rankColumns = computed(() => [
  { key: 'source', label: t('operations.registrationSources') },
  { key: 'count', label: t('operations.users') },
  { key: 'percentage', label: t('operations.percentage') },
])
const userColumns = computed(() => [
  { key: 'id', label: 'ID' }, { key: 'email', label: t('common.email') },
  { key: 'username', label: t('admin.users.columns.username') }, { key: 'signup_source', label: t('operations.signupMethod') },
  { key: 'selected_source', label: t('operations.selectedSource') },
  { key: 'utm_source', label: t('operations.utmSource') },
  { key: 'inviter_id', label: t('operations.inviter') }, { key: 'promo_code', label: t('operations.promo') },
  { key: 'created_at', label: t('operations.date') },
])
const palette = ['#2563eb', '#059669', '#d97706', '#7c3aed', '#dc2626', '#0891b2', '#475569', '#db2777']
const distribution = computed(() => ({
  labels: rankings.value.map(row => sourceLabel(row.source)),
  datasets: [{ data: rankings.value.map(row => row.count), backgroundColor: rankings.value.map((_, index) => palette[index % palette.length]), borderWidth: 0 }],
}))
const trend = computed(() => {
  const daily = summary.value?.daily ?? []
  const labels = [...new Set(daily.map(row => row.date))].sort()
  const values = new Map(daily.map(row => [`${row.date}|${row.source}`, row.count]))
  return {
    labels,
    datasets: rankings.value.map((row, index) => ({
      label: sourceLabel(row.source),
      data: labels.map(date => values.get(`${date}|${row.source}`) ?? 0),
      borderColor: palette[index % palette.length],
      backgroundColor: palette[index % palette.length],
      tension: 0.2, pointRadius: 2,
    })),
  }
})
const distributionOptions = { responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } } }
const trendOptions = {
  responsive: true, maintainAspectRatio: false,
  scales: { y: { beginAtZero: true, ticks: { precision: 0 } } },
  plugins: { legend: { position: 'bottom' as const } },
}

function selectedLabel(value?: string) {
  const key = value || 'unknown'
  return t(`operations.sourceLabels.${key}`, key)
}
function sourceLabel(value: string) {
  return dimension.value === 'selected' || value === 'unknown' ? selectedLabel(value) : value
}
function number(value: number, decimals = 0) {
  return new Intl.NumberFormat(locale.value, { maximumFractionDigits: decimals }).format(value)
}
function query(): RegistrationSourceQuery {
  return {
    range: range.value, dimension: dimension.value,
    ...(range.value === 'custom' ? { start_date: startDate.value, end_date: endDate.value } : {}),
  }
}
function validRange() {
  if (range.value !== 'custom' || (startDate.value && endDate.value && startDate.value <= endDate.value)) return true
  app.showError(t('operations.invalidDateRange'))
  return false
}
async function reload() {
  if (!validRange()) return
  source.value = ''
  page.value = 1
  summaryRequest?.abort()
  const request = new AbortController()
  summaryRequest = request
  loading.value = true
  const usersLoad = loadUsers()
  try {
    const result = await getRegistrationSources(query(), request.signal)
    if (!request.signal.aborted) summary.value = result
  } catch (error) {
    if (!request.signal.aborted) {
      summary.value = undefined
      app.showError(extractApiErrorMessage(error, t('common.error')))
    }
  } finally {
    if (!request.signal.aborted) loading.value = false
    await usersLoad
  }
}
async function loadUsers() {
  if (!validRange()) return
  usersRequest?.abort()
  const request = new AbortController()
  usersRequest = request
  usersLoading.value = true
  try {
    const result = await listRegistrationSourceUsers({
      ...query(), source: source.value || undefined, page: page.value, page_size: pageSize.value,
    }, request.signal)
    if (request.signal.aborted) return
    userRows.value = result.items
    total.value = result.total
    page.value = result.page
    pageSize.value = result.page_size
  } catch (error) {
    if (!request.signal.aborted) {
      userRows.value = []
      total.value = 0
      app.showError(extractApiErrorMessage(error, t('common.error')))
    }
  } finally {
    if (!request.signal.aborted) usersLoading.value = false
  }
}
function setPage(value: number) {
  page.value = value
  void loadUsers()
}
function setPageSize(value: number) {
  pageSize.value = value
  setPage(1)
}
function selectSource(value: string) {
  source.value = value
  setPage(1)
  usersSection.value?.scrollIntoView({ block: 'start' })
}
async function downloadCSV() {
  if (!validRange()) return
  exporting.value = true
  try {
    const data = await exportRegistrationSources({ ...query(), source: source.value || undefined })
    const url = URL.createObjectURL(data)
    try {
      const link = document.createElement('a')
      link.href = url
      link.download = `registration-sources-${formatDateLocalInput(new Date())}.csv`
      link.click()
    } finally {
      window.setTimeout(() => URL.revokeObjectURL(url), 1000)
    }
  } catch (error) {
    app.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    exporting.value = false
  }
}
onMounted(reload)
onUnmounted(() => {
  summaryRequest?.abort()
  usersRequest?.abort()
})
</script>
