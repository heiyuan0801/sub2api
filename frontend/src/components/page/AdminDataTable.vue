<script setup lang="ts" generic="Row extends Record<string, any>">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMediaQuery } from '@vueuse/core'
import type { ColumnDef, RowSelectionState, SortingState } from '@tanstack/vue-table'
import LegacyDataTable from '@/components/common/DataTable.vue'
import type { Column } from '@/components/common/types'
import { DataTable as UiDataTable } from '@/components/ui/data-table'
import EmptyState from './EmptyState.vue'

defineOptions({ name: 'DataTable' })

// Keep the existing admin list contract while the table markup moves to TanStack.
type RowKey = string | number
type SortOrder = 'asc' | 'desc'
type SortState = { key: string; order: SortOrder }

const props = withDefaults(defineProps<{
  columns: Column[]
  data: Row[]
  loading?: boolean
  stickyFirstColumn?: boolean
  stickyActionsColumn?: boolean
  expandableActions?: boolean
  actionsCount?: number
  rowKey?: string | ((row: Row) => RowKey)
  defaultSortKey?: string
  defaultSortOrder?: SortOrder
  sortStorageKey?: string
  serverSideSort?: boolean
  clickableRows?: boolean
  estimateRowHeight?: number
  overscan?: number
  virtualizeThreshold?: number
  selectable?: boolean
  selectedKeys?: RowKey[]
  selectionLabel?: string | ((row: Row) => string)
}>(), {
  stickyFirstColumn: true, stickyActionsColumn: true, expandableActions: true,
  defaultSortOrder: 'asc', selectedKeys: () => []
})

const emit = defineEmits<{
  sort: [key: string, order: SortOrder]
  rowClick: [row: Row]
  'update:selectedKeys': [keys: RowKey[]]
  selectionChange: [keys: RowKey[]]
}>()

const slots = defineSlots<{
  [name: `cell-${string}`]: (props: { row: Row; value: any; expanded: boolean }) => unknown
  [name: `header-${string}`]: (props: { column: Column; sortKey: string; sortOrder: SortOrder }) => unknown
  empty?: () => unknown
}>()
// Dynamic columns can be synthetic or nested, so cell values intentionally keep
// the legacy untyped value contract. The row itself retains the caller's type.
const headerSlots = computed(() => props.columns.map(({ key }) => `header-${key}` as const).filter(name => Boolean(slots[name])))
const cellSlots = computed(() => props.columns.map(({ key }) => `cell-${key}` as const).filter(name => Boolean(slots[name])))
const { t } = useI18n()
const desktop = useMediaQuery('(min-width: 768px)')
// The mobile cards and lists with measured expandable actions keep their existing renderer.
const usesLegacy = computed(() => !desktop.value || (props.expandableActions && props.actionsCount !== undefined))
const sort = ref<SortState | null>(null)
const columnById = computed(() => new Map(props.columns.map(column => [column.key, column])))
const defaultSort = (): SortState | null => props.defaultSortKey && columnById.value.get(props.defaultSortKey)?.sortable
  ? { key: props.defaultSortKey, order: props.defaultSortOrder }
  : null

function readInitialSort(): SortState | null {
  if (props.sortStorageKey) {
    try {
      const stored = JSON.parse(localStorage.getItem(props.sortStorageKey) ?? 'null') as Partial<SortState> | null
      if (stored?.key && columnById.value.get(stored.key)?.sortable) {
        return { key: stored.key, order: stored.order === 'desc' ? 'desc' : 'asc' }
      }
    } catch { /* Storage is optional; fall back to the configured sort. */ }
  }
  return defaultSort()
}
onMounted(() => { sort.value = readInitialSort() })

const columnSignature = computed(() => props.columns.map(column => `${column.key}:${Boolean(column.sortable)}`).join('|'))
watch(columnSignature, () => {
  if (!sort.value || !columnById.value.get(sort.value.key)?.sortable) sort.value = readInitialSort()
})

function changeSort(key: string, order: SortOrder) {
  sort.value = { key, order }
  if (props.sortStorageKey) {
    try { localStorage.setItem(props.sortStorageKey, JSON.stringify(sort.value)) }
    catch { /* Sorting still works when storage is unavailable. */ }
  }
  if (props.serverSideSort) emit('sort', key, order)
}
const sorting = computed<SortingState>(() => sort.value ? [{ id: sort.value.key, desc: sort.value.order === 'desc' }] : [])
function updateSorting(next: SortingState) {
  const column = next[0]
  if (column) changeSort(column.id, column.desc ? 'desc' : 'asc')
}

const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' })
function finiteNumber(value: unknown) {
  if (typeof value === 'boolean') return Number(value)
  if (typeof value !== 'number' && !(typeof value === 'string' && value.trim())) return null
  const number = Number(value)
  return Number.isFinite(number) ? number : null
}
function sortableString(value: unknown): string {
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') return String(value)
  try { return (value instanceof Date ? value.toISOString() : JSON.stringify(value)) ?? String(value) }
  catch { return String(value) }
}
function compareValues(a: unknown, b: unknown) {
  const aEmpty = a === null || a === undefined || a === ''
  const bEmpty = b === null || b === undefined || b === ''
  if (aEmpty || bEmpty) return Number(aEmpty) - Number(bEmpty)
  const aNumber = finiteNumber(a)
  const bNumber = finiteNumber(b)
  return aNumber !== null && bNumber !== null
    ? Math.sign(aNumber - bNumber)
    : collator.compare(sortableString(a), sortableString(b))
}

const sortedData = computed(() => {
  const state = sort.value
  if (!state || props.serverSideSort) return props.data
  const { key, order } = state
  return [...props.data].sort((a, b) => compareValues(a?.[key], b?.[key]) * (order === 'desc' ? -1 : 1))
})

const columns = computed<ColumnDef<Row>[]>(() => props.columns.map(column => ({
  id: column.key,
  accessorFn: row => row[column.key],
  header: column.label,
  enableSorting: Boolean(column.sortable)
})))
const customHeaders = computed(() => headerSlots.value.map(name => name.slice('header-'.length)))
function columnForKey(id: string): Column {
  return columnById.value.get(id) ?? { key: id, label: id }
}
function columnClass(id: string) {
  const column = columnById.value.get(id)
  if (!column) return
  const index = props.columns.indexOf(column)
  const explicitSelect = props.columns[0]?.key === 'select'
  return [
    'whitespace-nowrap',
    column.class,
    {
      'admin-sticky-left': props.stickyFirstColumn && index === 0,
      'admin-sticky-second': props.stickyFirstColumn && explicitSelect && index === 1,
      'admin-sticky-right': props.stickyActionsColumn && id === 'actions'
    }
  ]
}

function headerAlignment(id: string) {
  const classes = columnById.value.get(id)?.class
  if (classes?.includes('text-center')) return 'justify-center'
  if (classes?.includes('text-right')) return 'justify-end'
}

function resolveRowKey(row: Row, index: number): RowKey {
  const key = typeof props.rowKey === 'function'
    ? props.rowKey(row)
    : props.rowKey ? row[props.rowKey] : row.id
  return key ?? index
}
function selectionId(key: RowKey) { return `${typeof key}:${key}` }
function getRowId(row: Row, index: number) { return selectionId(resolveRowKey(row, index)) }
const rowSelection = computed<RowSelectionState>(() => Object.fromEntries(props.selectedKeys.map(key => [selectionId(key), true])))
const keysOnPage = computed(() => sortedData.value.map(resolveRowKey))
function updateSelection(next: RowSelectionState) {
  const keys = new Set(props.selectedKeys)
  for (const key of keysOnPage.value) {
    if (next[selectionId(key)]) keys.add(key)
    else keys.delete(key)
  }
  const selected = [...keys]
  emit('update:selectedKeys', selected)
  emit('selectionChange', selected)
}
function labelForSelection(row: Row, index: number) {
  return typeof props.selectionLabel === 'function'
    ? props.selectionLabel(row)
    : props.selectionLabel || `${t('common.selectOption')} ${resolveRowKey(row, index)}`
}
function cellValue(row: Row, id: string) {
  const formatter = columnById.value.get(id)?.formatter
  return formatter ? formatter(row[id], row) : row[id]
}
defineExpose({ sortedData, resolveRowKey })
</script>

<template>
  <div data-admin-data-table class="admin-data-table flex min-h-0 min-w-0 flex-col">
    <LegacyDataTable
      v-if="usesLegacy" v-bind="props"
      :data="sortedData" :server-side-sort="true"
      :default-sort-key="sort?.key ?? defaultSortKey"
      :default-sort-order="sort?.order ?? defaultSortOrder"
      @sort="changeSort" @row-click="emit('rowClick', $event)"
      @update:selected-keys="emit('update:selectedKeys', $event)" @selection-change="emit('selectionChange', $event)"
    >
      <template v-for="slotName in headerSlots" #[slotName]="slotProps">
        <slot :name="slotName" v-bind="slotProps" />
      </template>
      <template v-for="slotName in cellSlots" #[slotName]="slotProps">
        <slot :name="slotName" v-bind="slotProps" />
      </template>
      <template v-if="$slots.empty" #empty><slot name="empty" /></template>
    </LegacyDataTable>
    <UiDataTable
      v-else
      class="flex min-h-0 min-w-0 flex-1 flex-col"
      :data="sortedData" :columns="columns" :loading="loading" replace-rows-on-load
      :get-row-id="getRowId" :row-data-id="resolveRowKey" :column-class="columnClass" :header-alignment="headerAlignment"
      :selectable="selectable" :row-selection="rowSelection" :selection-label="labelForSelection" :select-page-label="t('common.selectAll')"
      :clickable-rows="clickableRows" :custom-header-columns="customHeaders"
      :sorting="sorting" manual-sorting external-pagination single-sort keep-sort ascending-first
      hide-controls density="compact"
      :virtualize-threshold="virtualizeThreshold ?? 100" :estimate-row-height="estimateRowHeight ?? 56" :overscan="overscan"
      scroll-class="min-h-0" table-class="admin-data-grid"
      @update:sorting="updateSorting" @update:row-selection="updateSelection" @row-click="emit('rowClick', $event)"
    >
      <template #column-header="{ header }">
        <slot
          :name="`header-${header.column.id}`" :column="columnForKey(header.column.id)"
          :sort-key="sort?.key ?? ''" :sort-order="sort?.order ?? 'asc'"
        />
      </template>
      <template #cell="{ cell }">
        <slot :name="`cell-${cell.column.id}`" :row="cell.row.original" :value="cell.getValue()" :expanded="true">
          {{ cellValue(cell.row.original, cell.column.id) }}
        </slot>
      </template>
      <template #empty><slot name="empty"><EmptyState :title="t('empty.noData')" /></slot></template>
    </UiDataTable>
  </div>
</template>

<style scoped>
.admin-data-table :deep(.admin-data-grid) { min-width: max-content; }
.admin-data-table :deep([data-slot='table-container']) { scrollbar-gutter: stable; }
.admin-data-table :deep(.admin-data-grid th) {
  @apply sticky top-0 z-20 h-9 bg-muted px-3 py-1.5 text-xs font-medium;
}
.admin-data-table :deep(.admin-data-grid td:not([colspan])) { @apply px-3 py-1.5; }
.admin-data-table :deep(.admin-sticky-left),
.admin-data-table :deep(.admin-sticky-second),
.admin-data-table :deep(.admin-sticky-right) { @apply sticky z-10 bg-card; }
.admin-data-table :deep(.admin-data-grid th.admin-sticky-left),
.admin-data-table :deep(.admin-data-grid th.admin-sticky-second),
.admin-data-table :deep(.admin-data-grid th.admin-sticky-right) { @apply z-30 bg-muted; }
.admin-data-table :deep(.admin-sticky-left) { @apply left-0; }
.admin-data-table :deep(.admin-sticky-second) { left: 2.75rem; }
.admin-data-table :deep(.admin-sticky-right) { @apply right-0; }
</style>
