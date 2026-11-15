<template>
  <!-- 用量页"用户排行"tab 内容：无卡片外观，依赖父级统一卡片；筛选/时间范围复用页面级筛选栏 -->
  <div>
    <!-- Toolbar -->
    <Toolbar class="border-b border-border px-4 py-2.5">
      <p class="text-xs text-muted-foreground">{{ t('admin.usage.tokenRanking.subtitle') }}</p>
      <template #actions>
        <span v-if="!loading && items.length > 0" class="text-xs text-muted-foreground">
          {{ t('admin.usage.tokenRanking.userCount', { count: items.length }) }}
        </span>
        <div class="w-36 md:hidden">
          <Select :model-value="sortBy" :options="sortOptions" @update:model-value="onSortSelection" />
        </div>
        <div class="w-28">
          <Select v-model="limit" :options="limitOptions" @change="load" />
        </div>
      </template>
    </Toolbar>

    <!-- Table -->
    <DataTable :columns="columns" :data="items" :loading="loading" row-key="user_id" clickable-rows @row-click="row => selectUser(row.user_id, row.email)">
      <template v-for="col in sortableColumns" :key="col.key" #[`header-${col.key}`]>
        <button type="button" class="font-medium hover:text-brand" :class="sortBy === col.key ? 'text-brand' : 'text-muted-foreground'" :aria-pressed="sortBy === col.key" @click="setSort(col.key)">
          {{ t(col.label) }} <span v-if="sortBy === col.key" aria-hidden="true">↓</span>
        </button>
      </template>
      <template #cell-rank="{ row }">
        <span :class="rankClass(row.user_id)" class="inline-flex size-6 items-center justify-center rounded-full text-xs font-semibold tabular-nums">{{ ranks.get(row.user_id) }}</span>
      </template>
      <template #cell-user="{ row }">
        <button type="button" class="block max-w-[260px] truncate text-left text-sm font-medium hover:text-brand" :title="t('admin.usage.tokenRanking.rowHint')" @click.stop="selectUser(row.user_id, row.email)">
          {{ row.email || `User #${row.user_id}` }}
          <span class="ml-1 font-normal text-muted-foreground">#{{ row.user_id }}</span>
        </button>
      </template>
      <template #cell-requests="{ row }"><span class="tabular-nums">{{ row.requests.toLocaleString() }}</span></template>
      <template #cell-input_tokens="{ row }"><span class="tabular-nums">{{ fmtTokens(row.input_tokens) }}</span></template>
      <template #cell-output_tokens="{ row }"><span class="tabular-nums">{{ fmtTokens(row.output_tokens) }}</span></template>
      <template #cell-cache_tokens="{ row }"><span class="tabular-nums">{{ fmtTokens(row.cache_tokens) }}</span></template>
      <template #cell-total_tokens="{ row }"><span class="font-medium tabular-nums">{{ fmtTokens(row.total_tokens) }}</span></template>
      <template #cell-actual_cost="{ row }"><span class="font-medium tabular-nums text-success">${{ fmtCost(row.actual_cost) }}</span></template>
      <template #empty><EmptyState :title="t('admin.dashboard.noDataAvailable')" /></template>
    </DataTable>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getUserBreakdown, type UserBreakdownParams } from '@/api/admin/dashboard'
import { formatCompactNumber, formatCostFixed } from '@/utils/format'
import type { UserBreakdownItem } from '@/types'
import Select from '@/components/common/Select.vue'
import DataTable from '@/components/page/AdminDataTable.vue'
import { EmptyState, Toolbar } from '@/components/page'

const props = defineProps<{
  startDate: string
  endDate: string
  filters: Record<string, unknown>
  model?: string
}>()

const emit = defineEmits<{ (e: 'select-user', userId: number, email: string): void }>()

const { t } = useI18n()

type SortKey = NonNullable<UserBreakdownParams['sort_by']>
const sortableColumns: { key: SortKey; label: string }[] = [
  { key: 'requests', label: 'admin.usage.tokenRanking.columns.requests' },
  { key: 'input_tokens', label: 'admin.usage.tokenRanking.columns.inputTokens' },
  { key: 'output_tokens', label: 'admin.usage.tokenRanking.columns.outputTokens' },
  { key: 'cache_tokens', label: 'admin.usage.tokenRanking.columns.cacheTokens' },
  { key: 'total_tokens', label: 'admin.usage.tokenRanking.columns.totalTokens' },
  { key: 'actual_cost', label: 'admin.usage.tokenRanking.columns.cost' },
]

const columns = computed(() => [
  { key: 'rank', label: '#', class: 'w-16' },
  { key: 'user', label: t('admin.usage.tokenRanking.columns.user') },
  ...sortableColumns.map(col => ({ key: col.key, label: t(col.label), class: 'text-right' }))
])
const sortOptions = computed(() => sortableColumns.map(col => ({ value: col.key, label: t(col.label) })))

const limitOptions = [
  { value: 20, label: 'Top 20' },
  { value: 50, label: 'Top 50' },
  { value: 100, label: 'Top 100' },
  { value: 200, label: 'Top 200' },
]

// 前三名金/银/铜徽章
const RANK_BADGE_CLASSES = [
  'bg-warning/10 text-warning',
  'bg-muted text-muted-foreground',
  'bg-warning/5 text-warning',
]

const items = ref<UserBreakdownItem[]>([])
const ranks = computed(() => new Map(items.value.map((item, index) => [item.user_id, index + 1])))
const rankClass = (id: number) => RANK_BADGE_CLASSES[(ranks.value.get(id) ?? 0) - 1] ?? 'text-muted-foreground'
function selectUser(userId: number, email: string) {
  emit('select-user', userId, email)
}
const loading = ref(false)
const sortBy = ref<SortKey>('total_tokens')
const limit = ref(50)
let reqSeq = 0

const fmtTokens = (v: number) => formatCompactNumber(v)
const fmtCost = (v: number) => formatCostFixed(v, 4)

const setSort = (key: SortKey) => {
  if (sortBy.value === key) return
  sortBy.value = key
  load()
}

function onSortSelection(value: unknown) {
  const column = sortableColumns.find(col => col.key === value)
  if (column) setSort(column.key)
}

const load = async () => {
  const seq = ++reqSeq
  loading.value = true
  try {
    const params: UserBreakdownParams = {
      ...props.filters,
      start_date: props.startDate,
      end_date: props.endDate,
      sort_by: sortBy.value,
      limit: limit.value,
    }
    if (props.model) params.model = props.model
    const res = await getUserBreakdown(params)
    if (seq !== reqSeq) return
    items.value = res.users || []
  } catch {
    if (seq !== reqSeq) return
    items.value = []
  } finally {
    if (seq === reqSeq) loading.value = false
  }
}

// Reload when the shared filters / date range / model change.
watch(
  () => [props.startDate, props.endDate, props.model, JSON.stringify(props.filters)],
  () => load(),
  { immediate: true }
)

defineExpose({ reload: load })
</script>
