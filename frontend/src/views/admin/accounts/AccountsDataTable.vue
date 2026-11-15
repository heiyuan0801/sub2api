<script setup lang="ts">
import { computed, ref, type VNode } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ColumnDef, RowSelectionState, SortingState } from '@tanstack/vue-table'
import type { Virtualizer } from '@tanstack/vue-virtual'
import LegacyTable from '@/components/common/DataTable.vue'
import type { Column } from '@/components/common/types'
import { DataTable as PrimitiveDataTable } from '@/components/ui/data-table'
import { EmptyState } from '@/components/page'
import type { AccountListItem } from '@/types'

defineOptions({ name: 'DataTable' })

type Order = 'asc' | 'desc'
type Density = 'compact' | 'comfortable'
type TableApi = {
  virtualizer: Virtualizer<HTMLElement, Element>
  sortedData: AccountListItem[]
}

const props = defineProps<{
  columns: Column[]
  data: AccountListItem[]
  loading: boolean
  desktop: boolean
  density: Density
  sortKey: string
  sortOrder: Order
  selectedIds: number[]
}>()
const emit = defineEmits<{ sort: [key: string, order: Order] }>()
const slots = defineSlots<{
  [key: `header-${string}`]: (props: { column: Column; sortKey: string; sortOrder: Order }) => VNode[]
  // Legacy cell slots use a mixture of API fields and presentation-only columns.
  [key: `cell-${string}`]: (props: { row: AccountListItem; value: any; expanded: boolean }) => VNode[]
}>()
const legacySlotNames = Object.keys(slots).filter(name => name.startsWith('cell-') || name.startsWith('header-')) as (`header-${string}` | `cell-${string}`)[]
const { t } = useI18n()
const desktopTable = ref<TableApi | null>(null)

const definitions = computed<ColumnDef<AccountListItem>[]>(() => props.columns.map(column => ({
  id: column.key,
  accessorFn: row => row[column.key as keyof AccountListItem],
  header: column.label,
  enableSorting: column.sortable === true
})))
const headerOverrides = computed(() => props.columns
  .map(column => column.key)
  .filter(key => Boolean(slots[`header-${key}`])))
const columnsByKey = computed(() => new Map(props.columns.map(column => [column.key, column])))
const columnForKey = (key: string): Column => columnsByKey.value.get(key) ?? { key, label: key }
const rowId = (row: AccountListItem) => String(row.id)

const sorting = computed<SortingState>(() => [{ id: props.sortKey, desc: props.sortOrder === 'desc' }])
const rowSelection = computed<RowSelectionState>(() => Object.fromEntries(props.selectedIds.map(id => [String(id), true])))
function setSorting(next: SortingState) {
  const sort = next[0]
  if (sort) emit('sort', sort.id, sort.desc ? 'desc' : 'asc')
}

defineExpose({
  virtualizer: computed(() => desktopTable.value?.virtualizer ?? null),
  sortedData: computed(() => desktopTable.value?.sortedData ?? props.data)
})
</script>

<template>
  <PrimitiveDataTable
    v-if="desktop && (data.length || loading)"
    ref="desktopTable"
    class="account-data-table flex h-full min-h-0 flex-col"
    :columns="definitions"
    :data="data"
    :caption="t('admin.accounts.title')"
    :get-row-id="rowId"
    :loading="loading"
    :density="density"
    :sorting="sorting"
    :row-selection="rowSelection"
    hide-controls
    ascending-first
    :custom-header-columns="headerOverrides"
    :virtualize-threshold="50"
    :estimate-row-height="density === 'compact' ? 100 : 120"
    scroll-class="account-data-scroll flex-1 min-h-0"
    table-class="min-w-max text-xs"
    external-pagination
    manual-sorting
    single-sort
    keep-sort
    @update:sorting="setSorting"
  >
    <template #column-header="{ header }">
      <slot
        :name="`header-${header.column.id}`"
        :column="columnForKey(header.column.id)"
        :sort-key="sortKey"
        :sort-order="sortOrder"
      />
    </template>
    <template #cell="{ cell }">
      <slot
        :name="`cell-${cell.column.id}`"
        :row="cell.row.original"
        :value="cell.getValue()"
        :expanded="false"
      >
        {{ cell.getValue() }}
      </slot>
    </template>
  </PrimitiveDataTable>
  <div v-else-if="desktop" class="grid min-h-44 flex-1 place-items-center rounded-lg border border-border bg-card">
    <EmptyState :title="t('empty.noData')" />
  </div>
  <!-- The established small-screen cards keep controls usable without wide horizontal scrolling. -->
  <div v-else class="space-y-2">
    <label v-if="data.length && !loading" class="flex items-center gap-2 px-1 text-sm text-muted-foreground">
      <slot name="header-select" :column="columnForKey('select')" :sort-key="sortKey" :sort-order="sortOrder" />
      {{ t('design.table.selectPage') }}
    </label>
    <LegacyTable
      :columns="columns"
      :data="data"
      :loading="loading"
      row-key="id"
      server-side-sort
      :default-sort-key="sortKey"
      :default-sort-order="sortOrder"
    >
      <template v-for="name in legacySlotNames" :key="name" #[name]="slotProps">
        <slot :name="name" v-bind="slotProps" />
      </template>
      <template #empty><EmptyState :title="t('empty.noData')" /></template>
    </LegacyTable>
  </div>
</template>

<style scoped>
.account-data-table :deep(.account-data-scroll) {
  scrollbar-gutter: stable;
}

.account-data-table :deep(thead) {
  @apply sticky top-0 z-20 bg-muted;
}

.account-data-table :deep(th[data-column-id]) {
  @apply h-8 px-2 py-1 text-xs;
}

.account-data-table :deep(td[data-column-id]) {
  @apply px-2;
}

.account-data-table :deep([data-column-id="select"]) {
  @apply sticky left-0 z-10 w-9 min-w-9 max-w-9 text-center;
}

.account-data-table :deep([data-column-id="name"]) {
  @apply sticky left-9 z-10 min-w-36;
}

.account-data-table :deep([data-column-id="actions"]) {
  @apply sticky right-0 z-10;
}

.account-data-table :deep(th[data-column-id="select"]),
.account-data-table :deep(th[data-column-id="name"]),
.account-data-table :deep(th[data-column-id="actions"]) {
  @apply bg-muted;
}

.account-data-table :deep(td[data-column-id="select"]),
.account-data-table :deep(td[data-column-id="name"]),
.account-data-table :deep(td[data-column-id="actions"]) {
  @apply bg-card;
}

.account-data-table :deep(tr[data-state="selected"] td) {
  background-color: hsl(var(--brand) / 0.07);
}

.account-data-table :deep(tr[data-state="selected"] td[data-column-id="select"]),
.account-data-table :deep(tr[data-state="selected"] td[data-column-id="name"]),
.account-data-table :deep(tr[data-state="selected"] td[data-column-id="actions"]) {
  background: linear-gradient(hsl(var(--brand) / 0.07), hsl(var(--brand) / 0.07)), hsl(var(--card));
}
</style>
