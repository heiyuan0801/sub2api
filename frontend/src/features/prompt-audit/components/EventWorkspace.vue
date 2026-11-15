<template>
  <section :aria-label="t('admin.promptAudit.events.title')" class="py-6">
    <PageHeader
      as="h2"
      :title="t('admin.promptAudit.events.title')"
      :description="t('admin.promptAudit.events.description')"
    >
      <template #actions>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="selectedIds.length === 0" @click="$emit('batch-delete')">
          {{ t('admin.promptAudit.events.deleteSelected', { count: selectedIds.length }) }}
        </button>
        <button type="button" class="btn btn-danger btn-sm" data-test="filter-delete" @click="$emit('preview-delete')">
          {{ t('admin.promptAudit.events.deleteByFilter') }}
        </button>
      </template>
    </PageHeader>

    <form class="mt-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-4 xl:grid-cols-5" @submit.prevent="applyFilters">
      <label class="text-xs text-muted-foreground">
        <span>{{ t('admin.promptAudit.events.decision') }}</span>
        <select v-model="localFilters.decision" class="input mt-1 w-full" :aria-label="t('admin.promptAudit.events.decision')" @change="filtersChanged">
          <option value="">{{ t('common.all') }}</option>
          <option value="pass">{{ t('admin.promptAudit.decisions.pass') }}</option>
          <option value="flag">{{ t('admin.promptAudit.decisions.flag') }}</option>
          <option value="critical">{{ t('admin.promptAudit.decisions.critical') }}</option>
        </select>
      </label>
      <label class="text-xs text-muted-foreground">
        <span>{{ t('admin.promptAudit.events.risk') }}</span>
        <select v-model="localFilters.risk_level" class="input mt-1 w-full" :aria-label="t('admin.promptAudit.events.risk')" @change="filtersChanged">
          <option value="">{{ t('common.all') }}</option>
          <option value="low">{{ t('admin.promptAudit.riskLevels.low') }}</option>
          <option value="medium">{{ t('admin.promptAudit.riskLevels.medium') }}</option>
          <option value="high">{{ t('admin.promptAudit.riskLevels.high') }}</option>
          <option value="critical">{{ t('admin.promptAudit.riskLevels.critical') }}</option>
        </select>
      </label>
      <FilterInput v-model="localFilters.endpoint" :label="t('admin.promptAudit.events.endpoint')" @change="filtersChanged" />
      <FilterInput v-model="localFilters.group_id" :label="t('admin.promptAudit.events.groupId')" type="number" @change="filtersChanged" />
      <FilterInput v-model="localFilters.user_id" :label="t('admin.promptAudit.events.userId')" type="number" @change="filtersChanged" />
      <FilterInput v-model="localFilters.api_key_id" :label="t('admin.promptAudit.events.apiKeyId')" type="number" @change="filtersChanged" />
      <FilterInput v-model="localFilters.request_id" :label="t('admin.promptAudit.events.requestId')" @change="filtersChanged" />
      <FilterInput v-model="localFilters.prompt_hash" :label="t('admin.promptAudit.events.promptHash')" @change="filtersChanged" />
      <FilterInput v-model="localFilters.keyword" :label="t('admin.promptAudit.events.keyword')" @change="filtersChanged" />
      <label class="text-xs text-muted-foreground">
        <span>{{ t('admin.promptAudit.events.startAt') }}</span>
        <input v-model="localFilters.start_at" type="datetime-local" class="input mt-1 w-full" :aria-label="t('admin.promptAudit.events.startAt')" @change="filtersChanged" />
      </label>
      <label class="text-xs text-muted-foreground">
        <span>{{ t('admin.promptAudit.events.endAt') }}</span>
        <input v-model="localFilters.end_at" type="datetime-local" class="input mt-1 w-full" :aria-label="t('admin.promptAudit.events.endAt')" @change="filtersChanged" />
      </label>
      <div class="flex items-end gap-2 sm:col-span-2">
        <button type="submit" class="btn btn-primary btn-sm">{{ t('common.search') }}</button>
        <button type="button" class="btn btn-ghost btn-sm" @click="resetFilters">{{ t('common.reset') }}</button>
      </div>
    </form>
    <div v-if="error" role="alert" class="mt-4 rounded-lg border border-danger/30 bg-danger/5 px-4 py-3 text-sm text-danger">{{ error }}</div>
    <div class="mt-5 min-w-0 rounded-lg border border-border">
      <div v-if="events.length && !loading" class="border-b border-border px-3 py-2 md:hidden">
        <label class="inline-flex items-center gap-2 text-xs font-medium text-muted-foreground">
          <input type="checkbox" class="size-4 accent-brand" :checked="allSelected" @change="toggleAll" />
          {{ t('admin.promptAudit.events.selectAll') }}
        </label>
      </div>
      <DataTable :data="events" :columns="columns" :loading="loading" row-key="id" :sticky-first-column="false">
        <template #header-selection>
          <input type="checkbox" class="size-4 accent-brand" :checked="allSelected" :aria-label="t('admin.promptAudit.events.selectAll')" @change="toggleAll" />
        </template>
        <template #cell-selection="{ row }">
          <input type="checkbox" class="size-4 accent-brand" :checked="selectedIds.includes(row.id)" :aria-label="t('admin.promptAudit.events.selectEvent', { id: row.id })" @change="toggleOne(row.id)" />
        </template>
        <template #cell-time="{ row }">
          <span class="text-xs tabular-nums text-muted-foreground">{{ formatDate(row.created_at) }}</span>
        </template>
        <template #cell-identity="{ row }">
          <CopyLine :label="t('admin.promptAudit.events.user')" :value="row.snapshot.username" />
          <CopyLine :label="t('admin.promptAudit.events.email')" :value="row.snapshot.user_email" />
          <CopyLine :label="t('admin.promptAudit.events.apiKey')" :value="row.snapshot.api_key_name" />
        </template>
        <template #cell-group="{ row }">{{ row.snapshot.group_name || '—' }}</template>
        <template #cell-route="{ row }">
          <p class="font-medium text-foreground">{{ row.snapshot.endpoint }}</p>
          <p class="mt-1 text-xs text-muted-foreground">{{ row.snapshot.model }} · {{ row.snapshot.protocol }} · {{ row.snapshot.stage || 'http' }}</p>
        </template>
        <template #cell-result="{ row }">
          <StatusBadge :label="formatDecisionRisk(row.decision, row.risk_level)" :tone="decisionTone(row.decision)" />
          <p class="mt-2 max-w-48 truncate text-xs text-muted-foreground" :title="formatCategories(row.categories)">{{ formatCategories(row.categories) }}</p>
        </template>
        <template #cell-preview="{ row }">
          <p class="line-clamp-2 max-w-xs whitespace-normal break-words text-muted-foreground">{{ row.snapshot.redacted_preview || '—' }}</p>
        </template>
        <template #cell-actions="{ row }">
          <button type="button" class="btn btn-ghost btn-sm" @click="$emit('view', row.id)">{{ t('common.view') }}</button>
          <button type="button" class="btn btn-ghost btn-sm text-danger" @click="$emit('delete', row.id)">{{ t('common.delete') }}</button>
        </template>
        <template #empty><EmptyState :title="t('admin.promptAudit.events.empty')" /></template>
      </DataTable>
      <Pagination :total="total" :page="page" :page-size="pageSize" @update:page="$emit('page', $event)" @update:page-size="$emit('page-size', $event)" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Pagination from '@/components/common/Pagination.vue'
import DataTable from '@/components/page/AdminDataTable.vue'
import { EmptyState, PageHeader, StatusBadge } from '@/components/page'
import type { PromptAuditEvent, PromptEventFilters } from '../types'
import { cloneData, emptyEventFilters, SCANNER_CATALOG } from '../viewModel'

const props = defineProps<{
  events: PromptAuditEvent[]; total: number; page: number; pageSize: number
  filters: PromptEventFilters; selectedIds: number[]; loading: boolean; error: string
}>()
const emit = defineEmits<{
  (event: 'filters-change', value: PromptEventFilters): void
  (event: 'search', value: PromptEventFilters): void
  (event: 'selection', value: number[]): void
  (event: 'page', value: number): void
  (event: 'page-size', value: number): void
  (event: 'view', id: number): void
  (event: 'delete', id: number): void
  (event: 'batch-delete'): void
  (event: 'preview-delete'): void
}>()
const { t, locale } = useI18n()
const columns = computed(() => [
  { key: 'selection', label: t('admin.promptAudit.events.selectAll'), class: 'w-10' },
  { key: 'time', label: t('admin.promptAudit.events.time') },
  { key: 'identity', label: t('admin.promptAudit.events.identity') },
  { key: 'group', label: t('admin.promptAudit.events.group') },
  { key: 'route', label: t('admin.promptAudit.events.route') },
  { key: 'result', label: t('admin.promptAudit.events.result') },
  { key: 'preview', label: t('admin.promptAudit.events.preview') },
  { key: 'actions', label: t('admin.promptAudit.common.actions'), class: 'text-right' }
])
const localFilters = reactive<PromptEventFilters>(cloneData(props.filters))
watch(() => props.filters, (value) => Object.assign(localFilters, cloneData(value)), { deep: true })
const allSelected = computed(() => props.events.length > 0 && props.events.every((event) => props.selectedIds.includes(event.id)))

const FilterInput = defineComponent({
  props: { modelValue: { type: String, required: true }, label: { type: String, required: true }, type: { type: String, default: 'text' } },
  emits: ['update:modelValue', 'change'],
  setup(componentProps, { emit: componentEmit }) {
    return () => h('label', { class: 'text-xs text-muted-foreground' }, [
      h('span', componentProps.label),
      h('input', {
        value: componentProps.modelValue, type: componentProps.type, class: 'input mt-1 w-full', 'aria-label': componentProps.label,
        onInput: (event: Event) => componentEmit('update:modelValue', (event.target as HTMLInputElement).value),
        onChange: () => componentEmit('change'),
      }),
    ])
  },
})

const CopyLine = defineComponent({
  props: { label: { type: String, required: true }, value: { type: String, default: '' } },
  setup(componentProps) {
    return () => h('div', { class: 'flex max-w-56 items-center gap-1 text-xs' }, [
      h('span', { class: 'w-16 flex-none text-muted-foreground' }, componentProps.label),
      h('span', { class: 'min-w-0 flex-1 truncate text-foreground' }, componentProps.value || '—'),
      componentProps.value ? h('button', {
        type: 'button', class: 'text-brand hover:underline', 'aria-label': `${t('common.copy')} ${componentProps.label}`,
        onClick: () => navigator.clipboard?.writeText(componentProps.value),
      }, t('common.copy')) : null,
    ])
  },
})

function filtersChanged() {
  emit('filters-change', cloneData(localFilters))
}
function applyFilters() {
  const value = cloneData(localFilters)
  emit('filters-change', value)
  emit('search', value)
}
function resetFilters() {
  Object.assign(localFilters, emptyEventFilters())
  applyFilters()
}
function toggleOne(id: number) {
  const selected = new Set(props.selectedIds)
  if (selected.has(id)) selected.delete(id)
  else selected.add(id)
  emit('selection', [...selected])
}
function toggleAll() {
  emit('selection', allSelected.value ? [] : props.events.map((event) => event.id))
}
function formatDate(value: string): string {
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'short', timeStyle: 'medium' }).format(new Date(value))
}
function decisionTone(decision: string) {
  if (decision === 'critical') return 'danger'
  if (decision === 'flag') return 'warning'
  return 'success'
}
const DECISIONS = new Set(['pass', 'flag', 'critical'])
const RISK_LEVELS = new Set(['low', 'medium', 'high', 'critical'])

function translateDecision(decision: string): string {
  return DECISIONS.has(decision) ? t(`admin.promptAudit.decisions.${decision}`) : decision
}
function translateRiskLevel(riskLevel: string): string {
  return RISK_LEVELS.has(riskLevel) ? t(`admin.promptAudit.riskLevels.${riskLevel}`) : riskLevel
}
function translateCategory(category: string): string {
  return SCANNER_CATALOG.some((scanner) => scanner.id === category)
    ? t(`admin.promptAudit.scanners.${category}`)
    : category
}
function formatDecisionRisk(decision: string, riskLevel: string): string {
  return `${translateDecision(decision)} · ${translateRiskLevel(riskLevel)}`
}
function formatCategories(categories: string[]): string {
  if (!categories.length) return '—'
  return categories.map(translateCategory).join(', ')
}
</script>
