<template>
  <AppLayout>
    <TablePageLayout>
      <!-- Filters -->
      <template #filters>
        <div class="rounded-2xl border border-border/80 bg-card p-4 shadow-sm sm:p-5">
          <div class="flex flex-wrap items-end justify-between gap-3">
            <!-- Left: filter fields -->
            <div class="flex flex-1 flex-wrap items-end gap-3">
              <div class="w-full sm:w-auto sm:min-w-[240px]">
                <label class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('admin.audit.filters.q') }}</label>
                <div class="relative">
                  <Icon
                    name="search"
                    size="sm"
                    class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
                  />
                  <input
                    v-model.trim="filters.q"
                    type="text"
                    class="input h-9 pl-9"
                    :placeholder="t('admin.audit.filters.qPlaceholder')"
                    @keyup.enter="search"
                  />
                </div>
              </div>

              <div class="w-full sm:w-auto sm:min-w-[200px]">
                <label class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('admin.audit.filters.actorEmail') }}</label>
                <input v-model.trim="filters.actor_email" type="text" class="input h-9" @keyup.enter="search" />
              </div>

              <div class="w-full sm:w-auto sm:min-w-[180px]">
                <label class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('admin.audit.filters.action') }}</label>
                <input v-model.trim="filters.action" type="text" class="input h-9" @keyup.enter="search" />
              </div>

              <div class="w-full sm:w-auto sm:min-w-[160px]">
                <label class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('admin.audit.filters.clientIp') }}</label>
                <input v-model.trim="filters.client_ip" type="text" class="input h-9 font-mono" @keyup.enter="search" />
              </div>

              <div class="w-full sm:w-auto sm:min-w-[140px]">
                <label class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('admin.audit.filters.method') }}</label>
                <Select v-model="filters.method" :options="methodOptions" @change="search" />
              </div>

              <div class="w-full sm:w-auto sm:min-w-[170px]">
                <label class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('admin.audit.filters.authMethod') }}</label>
                <Select v-model="filters.auth_method" :options="authMethodOptions" @change="search" />
              </div>

              <div class="w-full sm:w-auto sm:min-w-[140px]">
                <label class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('admin.audit.filters.result') }}</label>
                <Select v-model="filters.success" :options="resultOptions" @change="search" />
              </div>

              <div class="w-full sm:w-auto sm:min-w-[170px]">
                <label class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('admin.dashboard.timeRange') }}</label>
                <Select
                  :model-value="timeRange"
                  :options="timeRangeOptions"
                  @update:model-value="handleTimeRangeChange"
                />
              </div>
            </div>

            <!-- Right: actions -->
            <div class="flex w-full flex-wrap items-center justify-end gap-2 sm:w-auto">
              <button type="button" class="btn btn-primary h-9 px-3.5 text-sm" :disabled="loading" @click="search">
                {{ t('common.search') }}
              </button>
              <button type="button" class="btn btn-secondary h-9 px-3.5 text-sm" :disabled="loading" @click="resetFilters">
                {{ t('common.reset') }}
              </button>
              <button type="button" class="btn btn-danger h-9 px-3.5 text-sm" @click="openClearDialog">
                <Icon name="trash" size="sm" />
                {{ t('admin.audit.clearAll') }}
              </button>
            </div>
          </div>
        </div>
      </template>

      <!-- Table -->
      <template #table>
        <DataTable :columns="columns" :data="logs" :loading="loading" row-key="id">
          <template #cell-created_at="{ value }">
            <span class="whitespace-nowrap tabular-nums text-muted-foreground">{{ formatTime(value) }}</span>
          </template>

          <template #cell-actor="{ row }">
            <div class="min-w-0 max-w-[220px]">
              <div class="truncate font-medium text-foreground" :title="row.actor_email">
                {{ row.actor_email || '—' }}
              </div>
              <div class="mt-0.5 truncate text-xs text-muted-foreground">
                {{ row.actor_role }}<span v-if="row.auth_method"> · {{ authMethodLabel(row.auth_method) }}</span>
              </div>
            </div>
          </template>

          <template #cell-action="{ row }">
            <div class="min-w-0 max-w-xs">
              <div class="truncate font-mono text-sm text-foreground" :title="row.action">
                {{ row.action }}
              </div>
              <div class="mt-0.5 truncate font-mono text-xs text-muted-foreground" :title="`${row.method} ${row.path}`">
                {{ row.method }} {{ row.path }}
              </div>
            </div>
          </template>

          <template #cell-status_code="{ row }">
            <span class="tabular-nums" :class="statusBadgeClass(row.status_code)">
              <span class="h-1.5 w-1.5 rounded-full" :class="statusDotClass(row.status_code)"></span>
              {{ row.status_code }}
            </span>
          </template>

          <template #cell-latency_ms="{ value }">
            <span class="whitespace-nowrap tabular-nums text-muted-foreground">{{ value }} ms</span>
          </template>

          <template #cell-client_ip="{ value }">
            <span class="whitespace-nowrap font-mono tabular-nums text-muted-foreground">{{ value || '—' }}</span>
          </template>

          <template #cell-actions="{ row }">
            <button
              type="button"
              class="inline-flex h-8 items-center gap-1.5 rounded-lg px-2.5 text-xs font-medium text-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
              @click="openDetail(row.id)"
            >
              <Icon name="eye" size="sm" />
              {{ t('admin.audit.columns.detail') }}
            </button>
          </template>

          <template #empty>
            <div class="flex flex-col items-center py-8">
              <Icon name="shield" size="xl" class="mb-4 h-12 w-12 text-muted-foreground/40" />
              <p class="text-sm font-medium text-muted-foreground">{{ t('admin.audit.empty') }}</p>
            </div>
          </template>
        </DataTable>
      </template>

      <!-- Pagination -->
      <template #pagination>
        <Pagination
          v-if="total > 0"
          :total="total"
          :page="page"
          :page-size="pageSize"
          @update:page="onPageChange"
          @update:pageSize="onPageSizeChange"
        />
      </template>
    </TablePageLayout>

    <!-- Detail dialog -->
    <BaseDialog
      :show="detailVisible"
      :title="t('admin.audit.detail.title')"
      width="wide"
      :close-on-click-outside="true"
      @close="detailVisible = false"
    >
      <div v-if="detailLoading" class="space-y-5 py-2">
        <div class="space-y-3 rounded-2xl border border-border/80 bg-muted/40 p-5">
          <div class="flex items-center gap-3">
            <Skeleton class="h-6 w-24 rounded-full" />
            <Skeleton class="h-5 w-52" />
          </div>
          <Skeleton class="h-9 w-full rounded-lg" />
          <div class="flex gap-4">
            <Skeleton class="h-3.5 w-32" />
            <Skeleton class="h-3.5 w-24" />
          </div>
        </div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <Skeleton v-for="i in 3" :key="`audit-meta-sk-${i}`" class="h-20 rounded-xl" />
        </div>
        <Skeleton class="h-36 w-full rounded-xl" />
      </div>

      <div v-else-if="detail" class="space-y-5 py-2">
        <!-- Hero: action + result at a glance -->
        <div class="rounded-2xl border border-border/80 bg-muted/40 p-5">
          <div class="flex flex-wrap items-center gap-3">
            <span class="tabular-nums" :class="statusBadgeClass(detail.status_code)">
              <span class="h-1.5 w-1.5 rounded-full" :class="statusDotClass(detail.status_code)"></span>
              {{ detail.status_code }} {{ statusText(detail.status_code) }}
            </span>
            <span class="break-all font-mono text-base font-semibold text-foreground">
              {{ detail.action }}
            </span>
          </div>

          <div class="mt-3 flex items-center gap-2 rounded-lg border border-border bg-card px-3 py-2">
            <span class="rounded bg-muted px-1.5 py-0.5 font-mono text-[11px] font-semibold text-foreground">
              {{ detail.method }}
            </span>
            <span class="break-all font-mono text-xs text-muted-foreground">{{ detail.path }}</span>
          </div>

          <div class="mt-3 flex flex-wrap items-center gap-x-5 gap-y-1.5 text-xs tabular-nums text-muted-foreground">
            <span class="inline-flex items-center gap-1.5">
              <Icon name="clock" size="xs" />
              {{ formatTime(detail.created_at) }}
            </span>
            <span>{{ t('admin.audit.detail.latency') }} {{ detail.latency_ms }} ms</span>
            <span v-if="detail.request_id" class="inline-flex items-center gap-1">
              {{ t('admin.audit.detail.requestId') }}
              <span class="break-all font-mono">{{ detail.request_id }}</span>
            </span>
          </div>
        </div>

        <!-- Actor / auth / source -->
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <div class="rounded-xl border border-border/70 bg-muted/40 p-4">
            <div class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
              {{ t('admin.audit.columns.actor') }}
            </div>
            <div class="mt-1 break-all text-sm font-medium text-foreground">
              {{ detail.actor_email || '—' }}
            </div>
            <div class="mt-0.5 text-xs text-muted-foreground">{{ detail.actor_role }}</div>
          </div>

          <div class="rounded-xl border border-border/70 bg-muted/40 p-4">
            <div class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
              {{ t('admin.audit.filters.authMethod') }}
            </div>
            <div class="mt-1 text-sm font-medium text-foreground">
              {{ authMethodLabel(detail.auth_method) || '—' }}
            </div>
            <div v-if="detail.credential_masked" class="mt-0.5 break-all font-mono text-xs text-muted-foreground">
              {{ detail.credential_masked }}
            </div>
          </div>

          <div class="rounded-xl border border-border/70 bg-muted/40 p-4">
            <div class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
              {{ t('admin.audit.columns.clientIp') }}
            </div>
            <div class="mt-1 break-all font-mono text-sm font-medium tabular-nums text-foreground">
              {{ detail.client_ip || '—' }}
            </div>
          </div>
        </div>

        <!-- User-Agent -->
        <section>
          <h4 class="mb-1.5 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
            {{ t('admin.audit.detail.userAgent') }}
          </h4>
          <div class="break-all rounded-xl border border-border/70 bg-muted/40 p-3 font-mono text-xs leading-relaxed text-foreground/90">
            {{ detail.user_agent || '—' }}
          </div>
        </section>

        <!-- Request body (redacted) -->
        <section v-if="detail.request_body">
          <h4 class="mb-1.5 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
            {{ t('admin.audit.detail.requestBody') }}
          </h4>
          <pre class="max-h-72 overflow-auto rounded-xl border border-border/70 bg-muted/40 p-4 font-mono text-xs leading-relaxed text-foreground/90">{{ prettyBody(detail.request_body) }}</pre>
        </section>

        <!-- Extra -->
        <section v-if="detail.extra && Object.keys(detail.extra).length">
          <h4 class="mb-1.5 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
            {{ t('admin.audit.detail.extra') }}
          </h4>
          <pre class="max-h-48 overflow-auto rounded-xl border border-border/70 bg-muted/40 p-4 font-mono text-xs leading-relaxed text-foreground/90">{{ JSON.stringify(detail.extra, null, 2) }}</pre>
        </section>
      </div>
    </BaseDialog>

    <!-- Custom time range dialog (与 /admin/ops 时间下拉一致的自定义范围，支持时分) -->
    <BaseDialog
      :show="showCustomTimeRangeDialog"
      :title="t('admin.ops.timeRange.custom')"
      width="narrow"
      @close="handleCustomTimeRangeCancel"
    >
      <div class="space-y-4 py-2">
        <div>
          <label class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('admin.ops.customTimeRange.startTime') }}</label>
          <input v-model="customStartTimeInput" type="datetime-local" class="input h-9" />
        </div>
        <div>
          <label class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('admin.ops.customTimeRange.endTime') }}</label>
          <input v-model="customEndTimeInput" type="datetime-local" class="input h-9" />
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary h-9" @click="handleCustomTimeRangeCancel">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="btn btn-primary h-9"
          :disabled="!customStartTimeInput || !customEndTimeInput"
          @click="handleCustomTimeRangeConfirm"
        >
          {{ t('common.confirm') }}
        </button>
      </template>
    </BaseDialog>

    <!-- Clear confirmation → step-up TOTP -->
    <ConfirmDialog
      :show="clearConfirmVisible"
      :title="t('admin.audit.clearConfirm.title')"
      :message="t('admin.audit.clearConfirm.message')"
      :confirm-text="t('admin.audit.clearAll')"
      :cancel-text="t('common.cancel')"
      danger
      @confirm="onClearConfirmed"
      @cancel="clearConfirmVisible = false"
    />

    <!-- TOTP prompt for the clear operation -->
    <BaseDialog
      :show="clearTotpVisible"
      :title="t('admin.audit.clearConfirm.totpTitle')"
      width="narrow"
      :z-index="60"
      @close="cancelClearTotp"
    >
      <div class="py-2">
        <p class="text-sm text-muted-foreground">{{ t('admin.audit.clearConfirm.totpHint') }}</p>
        <input
          v-model.trim="clearTotpCode"
          type="text"
          inputmode="numeric"
          maxlength="6"
          autocomplete="one-time-code"
          class="input mt-4 h-11 text-center font-mono text-lg tabular-nums tracking-[0.5em]"
          placeholder="••••••"
          @keyup.enter="submitClear"
        />
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary h-9" :disabled="clearing" @click="cancelClearTotp">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="btn btn-danger h-9"
          :disabled="clearing || clearTotpCode.length !== 6"
          @click="submitClear"
        >
          {{ clearing ? t('common.loading') : t('admin.audit.clearAll') }}
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI, type AuditLog } from '@/api/admin'
import { totpAPI } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import type { Column } from '@/components/common/types'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { Skeleton } from '@/components/ui/skeleton'
import { useAppStore } from '@/stores'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const logs = ref<AuditLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const filters = reactive({
  q: '',
  actor_email: '',
  action: '',
  client_ip: '',
  method: '',
  auth_method: '',
  success: ''
})

// 时间范围：预设窗口（同 /admin/ops 时间下拉）+ 自定义起止（datetime-local，支持时分）
const timeRange = ref('')
const customStartTime = ref('')
const customEndTime = ref('')
const showCustomTimeRangeDialog = ref(false)
const customStartTimeInput = ref('')
const customEndTimeInput = ref('')

const TIME_RANGE_MINUTES: Record<string, number> = {
  '30m': 30,
  '1h': 60,
  '6h': 6 * 60,
  '24h': 24 * 60,
  '7d': 7 * 24 * 60,
  '30d': 30 * 24 * 60
}

const timeRangeOptions = computed(() => [
  { value: '', label: t('admin.audit.filters.all') },
  { value: '30m', label: t('admin.ops.timeRange.30m') },
  { value: '1h', label: t('admin.ops.timeRange.1h') },
  { value: '6h', label: t('admin.ops.timeRange.6h') },
  { value: '24h', label: t('admin.ops.timeRange.24h') },
  { value: '7d', label: t('admin.ops.timeRange.7d') },
  { value: '30d', label: t('admin.ops.timeRange.30d') },
  {
    value: 'custom',
    label:
      timeRange.value === 'custom' && customStartTime.value && customEndTime.value
        ? `${t('admin.ops.timeRange.custom')} (${formatCustomTimeRangeLabel(customStartTime.value, customEndTime.value)})`
        : t('admin.ops.timeRange.custom')
  }
])

function formatCustomTimeRangeLabel(startTime: string, endTime: string): string {
  const fmt = (raw: string) => {
    const d = new Date(raw)
    if (Number.isNaN(d.getTime())) return raw
    const pad = (n: number) => String(n).padStart(2, '0')
    return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
  }
  return `${fmt(startTime)} ~ ${fmt(endTime)}`
}

function toDatetimeLocal(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function handleTimeRangeChange(val: string | number | boolean | null) {
  const value = String(val ?? '')
  if (value === 'custom') {
    // 预填：已有自定义值沿用，否则默认最近1小时（本地时区）
    const now = new Date()
    customStartTimeInput.value = customStartTime.value || toDatetimeLocal(new Date(now.getTime() - 60 * 60 * 1000))
    customEndTimeInput.value = customEndTime.value || toDatetimeLocal(now)
    showCustomTimeRangeDialog.value = true
    return
  }
  timeRange.value = value
  search()
}

function handleCustomTimeRangeConfirm() {
  if (!customStartTimeInput.value || !customEndTimeInput.value) return
  customStartTime.value = customStartTimeInput.value
  customEndTime.value = customEndTimeInput.value
  timeRange.value = 'custom'
  showCustomTimeRangeDialog.value = false
  search()
}

function handleCustomTimeRangeCancel() {
  // 未确认不改变当前时间范围；Select 是受控组件，展示值保持不变。
  showCustomTimeRangeDialog.value = false
}

const columns = computed<Column[]>(() => [
  { key: 'created_at', label: t('admin.audit.columns.time') },
  { key: 'actor', label: t('admin.audit.columns.actor') },
  { key: 'action', label: t('admin.audit.columns.action') },
  { key: 'status_code', label: t('admin.audit.columns.result') },
  { key: 'latency_ms', label: t('admin.audit.detail.latency') },
  { key: 'client_ip', label: t('admin.audit.columns.clientIp') },
  { key: 'actions', label: t('common.actions') }
])

const methodOptions = computed(() => [
  { value: '', label: t('admin.audit.filters.all') },
  { value: 'POST', label: 'POST' },
  { value: 'PUT', label: 'PUT' },
  { value: 'PATCH', label: 'PATCH' },
  { value: 'DELETE', label: 'DELETE' },
  { value: 'GET', label: 'GET' }
])

const authMethodOptions = computed(() => [
  { value: '', label: t('admin.audit.filters.all') },
  { value: 'jwt', label: 'JWT' },
  { value: 'admin_api_key', label: 'Admin API Key' },
  { value: 'mcp_oauth', label: 'MCP OAuth' }
])

const resultOptions = computed(() => [
  { value: '', label: t('admin.audit.filters.all') },
  { value: 'true', label: t('admin.audit.filters.resultSuccess') },
  { value: 'false', label: t('admin.audit.filters.resultFailure') }
])

function authMethodLabel(method: string): string {
  const found = authMethodOptions.value.find((o) => o.value === method)
  return found && found.value ? found.label : method
}

function toRFC3339(local: string): string | undefined {
  if (!local) return undefined
  const d = new Date(local)
  if (Number.isNaN(d.getTime())) return undefined
  return d.toISOString()
}

function buildTimeRangeQuery(): { start_time?: string; end_time?: string } {
  if (timeRange.value === 'custom') {
    return {
      start_time: toRFC3339(customStartTime.value),
      end_time: toRFC3339(customEndTime.value)
    }
  }
  const minutes = TIME_RANGE_MINUTES[timeRange.value]
  if (!minutes) return {}
  return { start_time: new Date(Date.now() - minutes * 60 * 1000).toISOString() }
}

function buildQuery() {
  return {
    page: page.value,
    page_size: pageSize.value,
    q: filters.q || undefined,
    actor_email: filters.actor_email || undefined,
    action: filters.action || undefined,
    client_ip: filters.client_ip || undefined,
    method: filters.method || undefined,
    auth_method: filters.auth_method || undefined,
    success: filters.success || undefined,
    ...buildTimeRangeQuery()
  }
}

async function fetchLogs() {
  loading.value = true
  try {
    const res = await adminAPI.audit.list(buildQuery())
    logs.value = res.items
    total.value = res.total
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.audit.loadFailed'))
  } finally {
    loading.value = false
  }
}

function search() {
  page.value = 1
  fetchLogs()
}

function resetFilters() {
  filters.q = ''
  filters.actor_email = ''
  filters.action = ''
  filters.client_ip = ''
  filters.method = ''
  filters.auth_method = ''
  filters.success = ''
  timeRange.value = ''
  customStartTime.value = ''
  customEndTime.value = ''
  search()
}

function onPageChange(p: number) {
  page.value = p
  fetchLogs()
}

function onPageSizeChange(ps: number) {
  pageSize.value = ps
  page.value = 1
  fetchLogs()
}

// Detail dialog
const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<AuditLog | null>(null)

async function openDetail(id: number) {
  detailVisible.value = true
  detailLoading.value = true
  detail.value = null
  try {
    detail.value = await adminAPI.audit.get(id)
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.audit.loadFailed'))
    detailVisible.value = false
  } finally {
    detailLoading.value = false
  }
}

function prettyBody(body: string): string {
  try {
    return JSON.stringify(JSON.parse(body), null, 2)
  } catch {
    return body
  }
}

// Clear-all flow: confirm → TOTP → clear
const clearConfirmVisible = ref(false)
const clearTotpVisible = ref(false)
const clearTotpCode = ref('')
const clearing = ref(false)
const checkingTotpStatus = ref(false)

// 与其他敏感操作一致：未启用 2FA 时直接提示去个人资料启用 TOTP，
// 而不是弹出一个无法完成的验证码输入框（后端会以 TOTP_NOT_SETUP 拒绝）。
async function openClearDialog() {
  if (checkingTotpStatus.value) return
  checkingTotpStatus.value = true
  try {
    const status = await totpAPI.getStatus()
    if (!status.enabled) {
      appStore.showError(t('stepUp.notEnabled'))
      return
    }
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.audit.loadFailed'))
    return
  } finally {
    checkingTotpStatus.value = false
  }
  clearConfirmVisible.value = true
}

function onClearConfirmed() {
  clearConfirmVisible.value = false
  clearTotpCode.value = ''
  clearTotpVisible.value = true
}

function cancelClearTotp() {
  if (clearing.value) return
  clearTotpVisible.value = false
}

async function submitClear() {
  if (clearTotpCode.value.length !== 6) return
  clearing.value = true
  try {
    const res = await adminAPI.audit.clear(clearTotpCode.value)
    clearTotpVisible.value = false
    appStore.showSuccess(t('admin.audit.clearConfirm.success', { count: res.deleted }))
    search()
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.audit.clearConfirm.failed'))
    clearTotpCode.value = ''
  } finally {
    clearing.value = false
  }
}

// Helpers
function formatTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString()
}

function statusText(status: number): string {
  return status < 400 ? t('admin.audit.filters.resultSuccess') : t('admin.audit.filters.resultFailure')
}

function statusBadgeClass(status: number): string {
  const base = 'inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-semibold '
  if (status >= 500) return base + 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  if (status >= 400) return base + 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  return base + 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
}

function statusDotClass(status: number): string {
  if (status >= 500) return 'bg-red-500'
  if (status >= 400) return 'bg-amber-500'
  return 'bg-green-500'
}

onMounted(fetchLogs)
</script>
