<script setup lang="ts" generic="T extends object">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowDown, ArrowUp, ArrowUpDown, Columns3, List, Rows3 } from '@lucide/vue'
import {
  FlexRender,
  functionalUpdate,
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useVueTable,
  type PaginationState,
  type RowSelectionState,
  type SortingState,
  type TableOptions,
  type VisibilityState
} from '@tanstack/vue-table'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu, DropdownMenuCheckboxItem, DropdownMenuContent, DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import {
  Pagination, PaginationContent, PaginationEllipsis, PaginationItem, PaginationNext, PaginationPrevious
} from '@/components/ui/pagination'
import {
  Table, TableBody, TableCaption, TableCell, TableEmpty, TableHead, TableHeader, TableRow
} from '@/components/ui/table'

const props = defineProps<{
  data: T[]
  columns: TableOptions<T>['columns']
  caption?: string
  columnLabels?: Record<string, string>
  getRowId?: TableOptions<T>['getRowId']
  selectable?: boolean
  /** Provide the result count only when data contains a single server page. */
  rowCount?: number
  manualSorting?: boolean
}>()

const sorting = defineModel<SortingState>('sorting', { default: () => [] })
const selection = defineModel<RowSelectionState>('rowSelection', { default: () => ({}) })
const visibility = defineModel<VisibilityState>('columnVisibility', { default: () => ({}) })
const pagination = defineModel<PaginationState>('pagination', { default: () => ({ pageIndex: 0, pageSize: 20 }) })
const density = defineModel<'comfortable' | 'compact'>('density', { default: 'comfortable' })

const { t } = useI18n()

const table = useVueTable<T>({
  get data() { return props.data },
  get columns() { return props.columns },
  get getRowId() { return props.getRowId },
  get enableRowSelection() { return props.selectable ?? false },
  get rowCount() { return props.rowCount },
  get manualPagination() { return props.rowCount !== undefined },
  get manualSorting() { return props.manualSorting ?? false },
  state: {
    get sorting() { return sorting.value },
    get rowSelection() { return selection.value },
    get columnVisibility() { return visibility.value },
    get pagination() { return pagination.value }
  },
  onSortingChange: update => { sorting.value = functionalUpdate(update, sorting.value) },
  onRowSelectionChange: update => { selection.value = functionalUpdate(update, selection.value) },
  onColumnVisibilityChange: update => { visibility.value = functionalUpdate(update, visibility.value) },
  onPaginationChange: update => { pagination.value = functionalUpdate(update, pagination.value) },
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel(),
  getPaginationRowModel: getPaginationRowModel(),
  autoResetPageIndex: false
})

const rows = computed(() => table.getRowModel().rows)
const pageCount = computed(() => table.getPageCount())
const hideableColumns = computed(() => table.getAllLeafColumns().filter(column => column.getCanHide()))
const selectableRows = computed(() => rows.value.filter(row => row.getCanSelect()))
const selectedOnPage = computed(() => selectableRows.value.filter(row => row.getIsSelected()).length)
const allOnPageSelected = computed(() => selectedOnPage.value > 0 && selectedOnPage.value === selectableRows.value.length)
const totalSelected = computed(() => Object.values(selection.value).filter(Boolean).length)

watch([pageCount, () => pagination.value.pageIndex], ([count, index]) => {
  // A server request may briefly receive an empty page before its total arrives.
  if (props.rowCount === undefined || props.rowCount > 0) {
    const clamped = Math.max(0, Math.min(index, count - 1))
    if (clamped !== index) table.setPageIndex(clamped)
  }
}, { immediate: true })

function labelForColumn(column: (typeof hideableColumns.value)[number]) {
  const header = column.columnDef.header
  return props.columnLabels?.[column.id] ?? (typeof header === 'string' ? header : column.id)
}

function ariaSort(direction: false | 'asc' | 'desc') {
  return direction === 'asc' ? 'ascending' : direction === 'desc' ? 'descending' : 'none'
}

defineExpose({ table })
</script>

<template>
  <div class="min-w-0 overflow-hidden rounded-lg border border-border bg-card text-card-foreground">
    <div class="flex flex-wrap items-center justify-between gap-2 border-b border-border px-3 py-2">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <slot name="toolbar" :table="table" />
        <span v-if="totalSelected > 0" aria-live="polite" class="text-xs text-muted-foreground">
          {{ t('design.table.selected', { count: totalSelected }) }}
        </span>
      </div>
      <div class="ml-auto flex flex-wrap items-center gap-2">
        <Button
          variant="outline" size="sm" type="button" :aria-pressed="density === 'compact'"
          @click="density = density === 'compact' ? 'comfortable' : 'compact'"
        >
          <List v-if="density === 'compact'" aria-hidden="true" />
          <Rows3 v-else aria-hidden="true" />
          {{ t('design.table.compactRows') }}
        </Button>
        <DropdownMenu v-if="hideableColumns.length">
          <DropdownMenuTrigger as-child>
            <Button variant="outline" size="sm" type="button">
              <Columns3 aria-hidden="true" />
              {{ t('design.table.columns') }}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuCheckboxItem
              v-for="column in hideableColumns" :key="column.id"
              :model-value="column.getIsVisible()"
              :disabled="column.getIsVisible() && table.getVisibleLeafColumns().length === 1"
              @update:model-value="column.toggleVisibility($event === true)"
              @select.prevent
            >
              {{ labelForColumn(column) }}
            </DropdownMenuCheckboxItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
    <Table :data-density="density">
      <TableCaption v-if="caption" class="sr-only">{{ caption }}</TableCaption>
      <TableHeader class="bg-muted/50">
        <TableRow v-for="(headerGroup, index) in table.getHeaderGroups()" :key="headerGroup.id">
          <TableHead
            v-if="selectable && index === 0"
            :rowspan="table.getHeaderGroups().length"
            scope="col"
            class="w-10 px-3"
          >
            <input
              type="checkbox" class="size-4 align-middle accent-brand"
              :aria-label="t('design.table.selectPage')"
              :checked="allOnPageSelected"
              :indeterminate.prop="selectedOnPage > 0 && !allOnPageSelected"
              :disabled="!selectableRows.length"
              @change="table.getToggleAllPageRowsSelectedHandler()($event)"
            />
          </TableHead>
          <TableHead
            v-for="header in headerGroup.headers" :key="header.id"
            :colspan="header.colSpan"
            :scope="header.isPlaceholder ? undefined : header.subHeaders.length ? 'colgroup' : 'col'"
            :aria-sort="!header.isPlaceholder && header.column.getCanSort() ? ariaSort(header.column.getIsSorted()) : undefined"
            class="px-3 text-xs text-muted-foreground"
          >
            <template v-if="!header.isPlaceholder">
              <button
                v-if="header.column.getCanSort()" type="button"
                class="inline-flex items-center gap-1.5 rounded-sm text-left hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand"
                @click="header.column.getToggleSortingHandler()?.($event)"
              >
                <FlexRender :render="header.column.columnDef.header" :props="header.getContext()" />
                <ArrowUp v-if="header.column.getIsSorted() === 'asc'" class="size-3.5" aria-hidden="true" />
                <ArrowDown v-else-if="header.column.getIsSorted() === 'desc'" class="size-3.5" aria-hidden="true" />
                <ArrowUpDown v-else class="size-3.5 opacity-60" aria-hidden="true" />
              </button>
              <FlexRender v-else :render="header.column.columnDef.header" :props="header.getContext()" />
            </template>
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <template v-if="rows.length">
          <TableRow v-for="row in rows" :key="row.id" :data-state="row.getIsSelected() ? 'selected' : undefined">
            <TableCell v-if="selectable" class="px-3" :class="density === 'compact' ? 'py-1' : 'py-2.5'">
              <input
                type="checkbox" class="size-4 align-middle accent-brand"
                :aria-label="t('design.table.selectRow', { row: row.id })"
                :checked="row.getIsSelected()"
                :disabled="!row.getCanSelect()"
                @change="row.getToggleSelectedHandler()($event)"
              />
            </TableCell>
            <TableCell
              v-for="cell in row.getVisibleCells()" :key="cell.id"
              class="px-3" :class="density === 'compact' ? 'py-1' : 'py-2.5'"
            >
              <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
            </TableCell>
          </TableRow>
        </template>
        <TableEmpty v-else :colspan="Math.max(1, table.getVisibleLeafColumns().length + Number(!!selectable))">
          <slot name="empty">{{ t('design.table.empty') }}</slot>
        </TableEmpty>
      </TableBody>
    </Table>
    <div v-if="pageCount > 1 || $slots.footer" class="flex flex-wrap items-center justify-between gap-2 border-t border-border px-3 py-2">
      <slot name="footer" :table="table" />
      <Pagination
        v-if="pageCount > 1"
        :aria-label="t('design.pagination.label')"
        class="ml-auto mr-0 w-auto"
        :page="pagination.pageIndex + 1"
        :total="table.getRowCount()"
        :items-per-page="pagination.pageSize"
        :sibling-count="0"
        show-edges
        @update:page="table.setPageIndex($event - 1)"
      >
        <PaginationContent v-slot="{ items }" class="flex-wrap">
          <PaginationPrevious size="sm" />
          <template v-for="(item, index) in items" :key="index">
            <PaginationItem v-if="item.type === 'page'" :value="item.value" :is-active="item.value === pagination.pageIndex + 1">
              {{ item.value }}
            </PaginationItem>
            <PaginationEllipsis v-else />
          </template>
          <PaginationNext size="sm" />
        </PaginationContent>
      </Pagination>
    </div>
  </div>
</template>
