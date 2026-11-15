<template>
  <AppLayout>
    <div class="space-y-5">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 class="text-2xl font-semibold">{{ t('operations.registrationSources') }}</h1>
          <p class="mt-1 text-sm text-muted-foreground">{{ t('operations.registrationDescription') }}</p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button type="button" class="btn btn-secondary gap-2" @click="showCampaignLink = true">
            <Icon name="link" size="sm" aria-hidden="true" />
            {{ t('operations.campaignLink.title') }}
          </button>
          <button type="button" class="btn btn-secondary" :disabled="exporting || loading" @click="downloadCSV">
            {{ t('operations.exportCSV') }}
          </button>
        </div>
      </div>

      <div class="card flex flex-wrap items-end gap-4 p-4">
        <div>
          <label for="source-range" class="input-label">{{ t('admin.dashboard.timeRange') }}</label>
          <select id="source-range" v-model="range" class="input" @change="reload">
            <option value="today">{{ t('dates.today') }}</option>
            <option value="7d">{{ t('dates.last7Days') }}</option>
            <option value="30d">{{ t('dates.last30Days') }}</option>
            <option value="custom">{{ t('operations.customRange') }}</option>
          </select>
        </div>
        <DateRangePicker v-if="range === 'custom'" v-model:start-date="startDate" v-model:end-date="endDate" @change="reload" />
        <div>
          <label for="source-dimension" class="input-label">{{ t('operations.registrationSources') }}</label>
          <select id="source-dimension" v-model="dimension" class="input" @change="reload">
            <option value="selected">{{ t('operations.selectedSource') }}</option>
            <option value="utm">{{ t('operations.utmSource') }}</option>
          </select>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-3 xl:grid-cols-4">
        <div v-for="metric in metrics" :key="metric.label" class="card min-w-0 p-4">
          <div class="text-sm text-muted-foreground">{{ t(metric.label) }}</div>
          <div class="mt-2 text-2xl font-semibold tabular-nums">{{ metric.value == null ? '—' : number(metric.value) }}</div>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div class="card min-w-0 p-4 lg:col-span-2">
          <h2 class="mb-3 font-medium">{{ t('operations.signupTrend') }}</h2>
          <div v-if="trend.labels.length" class="h-64">
            <Line :data="trend" :options="trendOptions" :aria-label="t('operations.signupTrend')" />
          </div>
          <p v-else class="py-10 text-center text-sm text-muted-foreground">{{ loading ? t('common.loading') : t('empty.noData') }}</p>
        </div>
        <div class="card min-w-0 p-4">
          <h2 class="mb-3 font-medium">{{ t('operations.sourceShare') }}</h2>
          <div v-if="rankings.length" class="mx-auto h-64 max-w-72">
            <Doughnut :data="distribution" :options="distributionOptions" :aria-label="t('operations.sourceShare')" />
          </div>
          <p v-else class="py-10 text-center text-sm text-muted-foreground">{{ loading ? t('common.loading') : t('empty.noData') }}</p>
        </div>
      </div>

      <div class="space-y-2">
        <h2 class="font-medium">{{ t('operations.rankings') }}</h2>
        <DataTable :columns="rankColumns" :data="rankings" row-key="source" :loading="loading">
          <template #cell-source="{ value }">
            <button type="button" class="font-medium text-primary-600 hover:underline dark:text-primary-400" @click="selectSource(value)">{{ sourceLabel(value) }}</button>
          </template>
          <template #cell-count="{ value }">{{ number(value) }}</template>
          <template #cell-percentage="{ value }">{{ number(value, 2) }}%</template>
        </DataTable>
      </div>

      <section ref="usersSection" class="space-y-3">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h2 class="font-medium">{{ t('operations.usersInRange') }}</h2>
          <label class="min-w-48">
            <span class="sr-only">{{ t('operations.allSources') }}</span>
            <select v-model="source" class="input" @change="setPage(1)">
              <option value="">{{ t('operations.allSources') }}</option>
              <option v-for="item in rankings" :key="item.source" :value="item.source">{{ sourceLabel(item.source) }}</option>
            </select>
          </label>
        </div>
        <DataTable :columns="userColumns" :data="userRows" :loading="usersLoading">
          <template #cell-selected_source="{ row }">
            {{ selectedLabel(row.selected_source) }}
            <div v-if="row.source_note" class="mt-0.5 break-words text-xs text-muted-foreground">{{ row.source_note }}</div>
          </template>
          <template #cell-utm_source="{ row }">
            {{ row.utm_source || '—' }}
            <div v-if="row.utm_campaign" class="mt-0.5 break-words text-xs text-muted-foreground">{{ row.utm_campaign }}</div>
          </template>
          <template #cell-created_at="{ value }">{{ formatDateTime(value) }}</template>
        </DataTable>
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
import RegistrationCampaignLinkDialog from '@/components/admin/registration/RegistrationCampaignLinkDialog.vue'
import Icon from '@/components/icons/Icon.vue'
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
  { label: 'operations.totalRegistrations', value: summary.value?.total_registrations },
  { label: 'operations.todayRegistrations', value: summary.value?.today_registrations },
  { label: 'operations.rangeRegistrations', value: summary.value?.range_registrations },
  { label: 'operations.unreported', value: summary.value?.unreported_count },
])
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
