<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-2.5">
          <!-- Left: Search + Filters -->
          <div class="relative flex-1 sm:max-w-64">
            <Icon
              name="search"
              size="sm"
              class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
            />
            <input
              v-model="searchQuery"
              type="text"
              :placeholder="t('admin.announcements.searchAnnouncements')"
              class="input pl-9"
              @input="handleSearch"
            />
          </div>
          <Select
            v-model="filters.status"
            :options="statusFilterOptions"
            class="w-40"
            @change="handleStatusChange"
          />

          <!-- Right: Action buttons -->
          <div class="flex flex-1 flex-wrap items-center justify-end gap-2">
            <button
              @click="loadAnnouncements"
              :disabled="loading"
              class="btn btn-secondary px-2.5"
              :title="t('common.refresh')"
            >
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button @click="openCreateDialog" class="btn btn-primary">
              <Icon name="plus" size="sm" />
              {{ t('admin.announcements.createAnnouncement') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="announcements"
          :loading="loading"
          :server-side-sort="true"
          default-sort-key="created_at"
          default-sort-order="desc"
          @sort="handleSort"
        >
          <template #cell-title="{ value, row }">
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="truncate font-medium text-foreground">{{ value }}</span>
              </div>
              <div class="mt-1 flex items-center gap-1.5 text-xs tabular-nums text-muted-foreground">
                <span class="font-mono">#{{ row.id }}</span>
                <span class="text-border">·</span>
                <span>{{ formatDateTime(row.created_at) }}</span>
              </div>
              <div v-if="row.group_id && !row.plan_id" class="mt-1 flex items-center gap-1 text-xs text-muted-foreground">
                <Icon name="creditCard" size="xs" aria-hidden="true" />
                <span class="truncate">{{ row.group_name || `${t('operations.standardBalanceGroup')} #${row.group_id}` }}</span>
              </div>
            </div>
          </template>

          <template #cell-kind="{ row }">
            <NotificationKindBadge :kind="row.kind" />
          </template>

          <template #cell-status="{ value, row }">
            <span
              :class="[
                'badge',
                value === 'active'
                  ? 'badge-success'
                  : value === 'draft'
                    ? 'badge-gray'
                    : 'badge-warning'
              ]"
            >
              {{ row.delivery_status ? t(`operations.deliveryStates.${row.delivery_status}`, row.delivery_status) : statusLabel(value) }}
            </span>
          </template>

          <template #cell-notify_mode="{ row }">
            <span
              :class="[
                'badge',
                row.notify_mode === 'popup'
                  ? 'badge-warning'
                  : 'badge-gray'
              ]"
            >
              {{ row.notify_mode === 'popup' ? t('admin.announcements.notifyModeLabels.popup') : t('admin.announcements.notifyModeLabels.silent') }}
            </span>
          </template>

          <template #cell-targeting="{ row }">
            <span class="text-sm text-muted-foreground">
              {{ row.audience ? t(`operations.audienceModes.${row.audience.mode}`) : targetingSummary(row.targeting) }}
            </span>
          </template>

          <template #cell-timeRange="{ row }">
            <div class="space-y-0.5 text-xs tabular-nums text-muted-foreground">
              <div>
                <span class="font-medium text-foreground">{{ t('admin.announcements.form.startsAt') }}:</span>
                <span class="ml-1">{{ row.starts_at ? formatDateTime(row.starts_at) : t('admin.announcements.timeImmediate') }}</span>
              </div>
              <div>
                <span class="font-medium text-foreground">{{ t('admin.announcements.form.endsAt') }}:</span>
                <span class="ml-1">{{ row.ends_at ? formatDateTime(row.ends_at) : t('admin.announcements.timeNever') }}</span>
              </div>
            </div>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm tabular-nums text-muted-foreground">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button
                @click="openPreview(row)"
                class="inline-flex items-center justify-center rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                :title="t('admin.announcements.preview')"
              >
                <Icon name="eye" size="sm" />
              </button>
              <button
                @click="openReadStatus(row)"
                class="inline-flex items-center justify-center rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                :title="t('admin.announcements.readStatus')"
              >
                <Icon name="chartBar" size="sm" />
              </button>
              <button
                v-if="row.kind && row.delivery_status === 'draft'"
                type="button"
                class="btn btn-secondary btn-sm"
                :disabled="actionBusy"
                @click="openPublish(row)"
              >{{ t('operations.publish') }}</button>
              <button
                v-if="row.kind && ['queued', 'published', 'failed'].includes(row.delivery_status)"
                type="button"
                class="btn btn-secondary btn-sm"
                :disabled="actionBusy"
                @click="withdrawingNotice = row"
              >{{ t('operations.withdraw') }}</button>
              <button
                v-if="row.kind && row.delivery_status === 'failed'"
                type="button"
                class="btn btn-secondary btn-sm"
                :disabled="actionBusy"
                @click="retryDelivery(row)"
              >{{ t('operations.retryDelivery') }}</button>
              <button
                v-if="row.delivery_status !== 'withdrawn'"
                @click="openEditDialog(row)"
                class="inline-flex items-center justify-center rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                :title="t('common.edit')"
              >
                <Icon name="edit" size="sm" />
              </button>
              <button
                @click="handleDelete(row)"
                class="inline-flex items-center justify-center rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
                :title="t('common.delete')"
              >
                <Icon name="trash" size="sm" />
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('empty.noData')"
              :description="t('admin.announcements.createFirstAnnouncement')"
              :action-text="t('admin.announcements.createAnnouncement')"
              @action="openCreateDialog"
            />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <!-- Create/Edit Dialog -->
    <BaseDialog
      :show="showEditDialog"
      :title="isEditing ? t('admin.announcements.editAnnouncement') : t('admin.announcements.createAnnouncement')"
      width="wide"
      @close="closeEdit"
    >
      <form id="announcement-form" @submit.prevent="handleSave" class="space-y-4">
        <div>
          <label class="input-label">{{ t('admin.announcements.form.title') }}</label>
          <input v-model="form.title" type="text" class="input" required />
        </div>

        <div>
          <label class="input-label">{{ t('admin.announcements.form.content') }}</label>
          <textarea v-model="form.content" rows="6" class="input h-auto py-2" required></textarea>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div v-if="form.kind">
            <label for="notice-kind" class="input-label">{{ t('operations.notificationKind') }}</label>
            <select id="notice-kind" v-model="form.kind" class="input" :disabled="publishedFieldsLocked" @change="resetIrrelevantTarget">
              <option v-for="kind in kinds" :key="kind" :value="kind">{{ t(`operations.kind.${kind}`) }}</option>
            </select>
          </div>
          <div v-else>
            <label class="input-label">{{ t('admin.announcements.form.status') }}</label>
            <Select v-model="form.status" :options="statusOptions" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.announcements.form.notifyMode') }}</label>
            <Select v-model="form.notify_mode" :options="notifyModeOptions" />
            <p class="input-hint">{{ t('admin.announcements.form.notifyModeHint') }}</p>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div v-if="!form.kind">
            <label class="input-label">{{ t('admin.announcements.form.startsAt') }}</label>
            <input v-model="form.starts_at_str" type="datetime-local" max="9999-12-31T23:59" class="input" />
            <p class="input-hint">{{ t('admin.announcements.form.startsAtHint') }}</p>
          </div>
          <div>
            <label class="input-label">{{ t('admin.announcements.form.endsAt') }}</label>
            <input v-model="form.ends_at_str" type="datetime-local" max="9999-12-31T23:59" class="input" />
            <p class="input-hint">{{ t('admin.announcements.form.endsAtHint') }}</p>
          </div>
        </div>

        <template v-if="form.kind">
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div>
              <label for="notice-target" class="input-label">{{ t('operations.linkedTarget') }}</label>
              <select id="notice-target" v-model="linkedTarget" class="input" :required="requiresStandardGroup" :disabled="publishedFieldsLocked">
                <option value="" :disabled="requiresStandardGroup">{{ t(requiresStandardGroup ? 'operations.selectStandardGroup' : 'operations.noLinkedTarget') }}</option>
                <optgroup v-if="canLinkStandardGroup && standardGroups.length" :label="t('operations.standardBalanceGroup')">
                  <option v-for="group in standardGroups" :key="group.id" :value="`group:${group.id}`">{{ group.name }} · #{{ group.id }}</option>
                </optgroup>
                <optgroup v-if="!requiresStandardGroup && plans.length" :label="t('operations.linkedPlan')">
                  <option v-for="plan in plans" :key="plan.id" :value="`plan:${plan.id}`">{{ plan.name }} · #{{ plan.id }}</option>
                </optgroup>
                <option v-if="missingLinkedTarget" :value="linkedTarget">{{ missingLinkedTarget }}</option>
              </select>
            </div>
            <div>
              <label for="notice-audience" class="input-label">{{ t('operations.audience') }}</label>
              <select id="notice-audience" v-model="form.audience_mode" class="input" :disabled="publishedFieldsLocked">
                <option value="all">{{ t('operations.audienceModes.all') }}</option>
                <option value="groups">{{ t('operations.audienceModes.groups') }}</option>
                <option value="users">{{ t('operations.audienceModes.users') }}</option>
              </select>
            </div>
          </div>
          <fieldset v-if="form.audience_mode === 'groups'" :disabled="publishedFieldsLocked" class="rounded-xl border border-border/80 bg-muted/30 p-3.5">
            <legend class="px-1.5 text-xs font-medium text-muted-foreground">{{ t('operations.selectGroups') }}</legend>
            <div class="grid max-h-36 gap-2 overflow-y-auto sm:grid-cols-2">
              <label v-for="group in audienceGroups" :key="group.id" class="flex cursor-pointer items-start gap-2 rounded-md px-1.5 py-1 text-sm text-foreground transition-colors hover:bg-muted/60">
                <input v-model="form.group_ids" type="checkbox" :value="group.id" class="mt-0.5 shrink-0 rounded border-border accent-primary" />
                <span class="min-w-0 break-words">
                  {{ group.name }}
                  <span class="ml-1 text-xs text-muted-foreground">{{ t(group.subscription_type === 'standard' ? 'operations.standardBalanceGroup' : 'operations.subscriptionGroup') }}</span>
                </span>
              </label>
            </div>
          </fieldset>
          <div v-else-if="form.audience_mode === 'users'">
            <label for="notice-user-ids" class="input-label">{{ t('operations.userIDs') }}</label>
            <textarea id="notice-user-ids" v-model="form.user_ids" rows="2" class="input h-auto py-2 font-mono text-xs" :disabled="publishedFieldsLocked" aria-describedby="notice-user-ids-hint" required />
            <p id="notice-user-ids-hint" class="input-hint">{{ t('operations.userIDsHint') }}</p>
          </div>
        </template>
        <AnnouncementTargetingEditor v-else v-model="form.targeting" :groups="subscriptionGroups" />
      </form>

      <template #footer>
        <div class="flex justify-end gap-2.5">
          <button type="button" @click="closeEdit" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" form="announcement-form" :disabled="saving" class="btn btn-primary">
            {{ saving ? t('common.saving') : t(form.kind && form.status === 'draft' ? 'operations.saveDraft' : 'common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="Boolean(publishingNotice)" :title="t('operations.publishNotice')" @close="publishingNotice = null">
      <form id="publish-notice" class="space-y-4" @submit.prevent="publishNotice">
        <p v-if="publishingNotice" class="break-words text-sm font-medium text-foreground">{{ publishingNotice.title }}</p>
        <fieldset class="grid grid-cols-1 gap-2.5 sm:grid-cols-2">
          <label
            :class="[
              'flex cursor-pointer items-center gap-2.5 rounded-xl border px-3.5 py-2.5 text-sm font-medium transition-colors',
              publishTiming === 'now'
                ? 'border-foreground bg-muted/60 text-foreground'
                : 'border-border/80 bg-card text-muted-foreground hover:bg-muted/40 hover:text-foreground'
            ]"
          >
            <input v-model="publishTiming" type="radio" value="now" class="accent-primary" />
            {{ t('operations.publishNow') }}
          </label>
          <label
            :class="[
              'flex cursor-pointer items-center gap-2.5 rounded-xl border px-3.5 py-2.5 text-sm font-medium transition-colors',
              publishTiming === 'later'
                ? 'border-foreground bg-muted/60 text-foreground'
                : 'border-border/80 bg-card text-muted-foreground hover:bg-muted/40 hover:text-foreground'
            ]"
          >
            <input v-model="publishTiming" type="radio" value="later" class="accent-primary" />
            {{ t('operations.publishLater') }}
          </label>
        </fieldset>
        <div v-if="publishTiming === 'later'">
          <label for="notice-publish-at" class="input-label">{{ t('operations.publishAt') }}</label>
          <input id="notice-publish-at" v-model="publishAt" type="datetime-local" max="9999-12-31T23:59" class="input" required />
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-2.5">
          <button type="button" class="btn btn-secondary" :disabled="actionBusy" @click="publishingNotice = null">{{ t('common.cancel') }}</button>
          <button type="submit" form="publish-notice" class="btn btn-primary" :disabled="actionBusy">{{ t('operations.publish') }}</button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="Boolean(withdrawingNotice)"
      :title="t('operations.withdraw')"
      :message="t('operations.withdrawConfirm')"
      :confirm-text="t('operations.withdraw')"
      danger
      @confirm="withdrawNotice"
      @cancel="withdrawingNotice = null"
    />

    <!-- Delete Confirmation -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('admin.announcements.deleteAnnouncement')"
      :message="t('admin.announcements.deleteConfirm')"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      danger
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />

    <!-- Read Status Dialog -->
    <AnnouncementReadStatusDialog
      :show="showReadStatusDialog"
      :announcement-id="readStatusAnnouncementId"
      @close="showReadStatusDialog = false"
    />

    <NotificationDeliveryDialog :notification="deliveryNotice" @close="deliveryNotice = null" />

    <AnnouncementPopup
      :announcement="previewAnnouncement"
      preview
      @close="previewAnnouncement = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { adminAPI } from '@/api/admin'
import { formatDateTime, formatDateTimeLocalInput, parseDateTimeLocalInput } from '@/utils/format'
import type { AdminGroup, Announcement, AnnouncementTargeting, CreateAnnouncementRequest, UpdateAnnouncementRequest } from '@/types'
import type { NotificationAudience, NotificationAudienceMode, NotificationKind } from '@/types/operations'
import type { SubscriptionPlan } from '@/types/payment'
import { adminPaymentAPI } from '@/api/admin/payment'
import { formatPaymentAmount } from '@/components/payment/currency'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { Column } from '@/components/common/types'

import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Select from '@/components/common/Select.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'

import AnnouncementTargetingEditor from '@/components/admin/announcements/AnnouncementTargetingEditor.vue'
import AnnouncementReadStatusDialog from '@/components/admin/announcements/AnnouncementReadStatusDialog.vue'
import AnnouncementPopup from '@/components/common/AnnouncementPopup.vue'
import NotificationKindBadge from '@/components/common/NotificationKindBadge.vue'
import NotificationDeliveryDialog from '@/components/admin/announcements/NotificationDeliveryDialog.vue'

const { t } = useI18n()
const appStore = useAppStore()
const route = useRoute()
const router = useRouter()

const announcements = ref<Announcement[]>([])
const loading = ref(false)

const filters = reactive({
  status: '',
})
const searchQuery = ref('')

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})

const sortState = reactive({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc'
})

const statusFilterOptions = computed(() => [
  { value: '', label: t('admin.announcements.allStatus') },
  { value: 'draft', label: t('admin.announcements.statusLabels.draft') },
  { value: 'active', label: t('admin.announcements.statusLabels.active') },
  { value: 'archived', label: t('admin.announcements.statusLabels.archived') }
])

const statusOptions = computed(() => [
  { value: 'draft', label: t('admin.announcements.statusLabels.draft') },
  { value: 'active', label: t('admin.announcements.statusLabels.active') },
  { value: 'archived', label: t('admin.announcements.statusLabels.archived') }
])

const notifyModeOptions = computed(() => [
  { value: 'silent', label: t('admin.announcements.notifyModeLabels.silent') },
  { value: 'popup', label: t('admin.announcements.notifyModeLabels.popup') }
])

const columns = computed<Column[]>(() => [
  { key: 'title', label: t('admin.announcements.columns.title'), sortable: true },
  { key: 'kind', label: t('operations.notificationKind') },
  { key: 'status', label: t('admin.announcements.columns.status'), sortable: true },
  { key: 'notify_mode', label: t('admin.announcements.columns.notifyMode'), sortable: true },
  { key: 'targeting', label: t('admin.announcements.columns.targeting') },
  { key: 'timeRange', label: t('admin.announcements.columns.timeRange') },
  { key: 'created_at', label: t('admin.announcements.columns.createdAt'), sortable: true },
  { key: 'actions', label: t('admin.announcements.columns.actions') }
])

const statusLabel = (status: string) => {
  if (status === 'draft') return t('admin.announcements.statusLabels.draft')
  if (status === 'active') return t('admin.announcements.statusLabels.active')
  if (status === 'archived') return t('admin.announcements.statusLabels.archived')
  return status
}

const targetingSummary = (targeting: AnnouncementTargeting) => {
  const anyOf = targeting?.any_of ?? []
  if (!anyOf || anyOf.length === 0) return t('admin.announcements.targetingSummaryAll')
  return t('admin.announcements.targetingSummaryCustom', { groups: anyOf.length })
}

// ===== CRUD / list =====
let currentController: AbortController | null = null

async function loadAnnouncements() {
  currentController?.abort()
  const requestController = new AbortController()
  currentController = requestController
  const { signal } = requestController

  try {
    loading.value = true
    const res = await adminAPI.announcements.list(pagination.page, pagination.page_size, {
      status: filters.status || undefined,
      search: searchQuery.value || undefined,
      sort_by: sortState.sort_by,
      sort_order: sortState.sort_order
    }, { signal })

    if (signal.aborted || currentController !== requestController) return

    announcements.value = res.items
    pagination.total = res.total
    pagination.pages = res.pages
    pagination.page = res.page
    pagination.page_size = res.page_size
  } catch (error: any) {
    if (
      signal.aborted ||
      currentController !== requestController ||
      error?.name === 'AbortError' ||
      error?.code === 'ERR_CANCELED'
    ) {
      return
    }
    console.error('Error loading announcements:', error)
    appStore.showError(error.response?.data?.detail || t('admin.announcements.failedToLoad'))
  } finally {
    if (currentController === requestController) {
      loading.value = false
      currentController = null
    }
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  loadAnnouncements()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  loadAnnouncements()
}

function handleStatusChange() {
  pagination.page = 1
  loadAnnouncements()
}

function handleSort(key: string, order: 'asc' | 'desc') {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  loadAnnouncements()
}

let searchDebounceTimer: number | null = null
function handleSearch() {
  if (searchDebounceTimer) window.clearTimeout(searchDebounceTimer)
  searchDebounceTimer = window.setTimeout(() => {
    pagination.page = 1
    loadAnnouncements()
  }, 300)
}

// ===== Create/Edit dialog =====
const showEditDialog = ref(false)
const saving = ref(false)
const editingAnnouncement = ref<Announcement | null>(null)

const isEditing = computed(() => !!editingAnnouncement.value)

const form = reactive({
  title: '',
  content: '',
  kind: 'system' as NotificationKind | '',
  audience_mode: 'all' as NotificationAudienceMode,
  group_ids: [] as number[],
  user_ids: '',
  plan_id: 0,
  group_id: 0,
  status: 'draft',
  notify_mode: 'silent',
  starts_at_str: '',
  ends_at_str: '',
  targeting: { any_of: [] } as AnnouncementTargeting
})

const kinds: NotificationKind[] = ['system', 'activity', 'new_plan', 'plan_unlisted', 'new_group', 'group_unlisted', 'price_increase', 'price_decrease']
const publishedFieldsLocked = computed(() => editingAnnouncement.value?.delivery_status === 'published')
const requiresStandardGroup = computed(() => form.kind === 'new_group' || form.kind === 'group_unlisted')
const canLinkStandardGroup = computed(() => requiresStandardGroup.value || form.kind === 'price_increase' || form.kind === 'price_decrease')
const audienceGroups = ref<AdminGroup[]>([])
const subscriptionGroups = computed(() => audienceGroups.value.filter(group => group.subscription_type === 'subscription' && group.status === 'active'))
const standardGroups = computed(() => audienceGroups.value.filter(group => group.subscription_type === 'standard'))
const plans = ref<SubscriptionPlan[]>([])
const publishSuggestion = ref<number>()
let planRequest: Promise<void> | undefined
let groupRequest: Promise<void> | undefined

const linkedTarget = computed({
  get: () => form.plan_id ? `plan:${form.plan_id}` : form.group_id ? `group:${form.group_id}` : '',
  set: (value: string) => {
    const [type, id] = value.split(':')
    form.plan_id = type === 'plan' ? Number(id) : 0
    form.group_id = type === 'group' ? Number(id) : 0
  },
})

const missingLinkedTarget = computed(() => {
  if (form.group_id && !standardGroups.value.some(group => group.id === form.group_id)) {
    return editingAnnouncement.value?.group_name || `${t('operations.standardBalanceGroup')} #${form.group_id}`
  }
  if (form.plan_id && !plans.value.some(plan => plan.id === form.plan_id)) {
    return `${t('operations.linkedPlan')} #${form.plan_id}`
  }
  return ''
})

function resetIrrelevantTarget() {
  if (requiresStandardGroup.value) form.plan_id = 0
  if (!canLinkStandardGroup.value) form.group_id = 0
}

function loadPlans() {
  if (planRequest) return planRequest
  planRequest = adminPaymentAPI.getPlans()
    .then(({ data }) => { plans.value = data })
    .catch(error => { appStore.showError(extractApiErrorMessage(error, t('common.error'))) })
    .finally(() => { planRequest = undefined })
  return planRequest
}

function loadGroups() {
  if (groupRequest) return groupRequest
  groupRequest = adminAPI.groups.getAllIncludingInactive()
    .then(groups => { audienceGroups.value = groups })
    .catch(error => { appStore.showError(extractApiErrorMessage(error, t('common.error'))) })
    .finally(() => { groupRequest = undefined })
  return groupRequest
}

function resetForm() {
  form.title = ''
  form.content = ''
  form.kind = 'system'
  form.audience_mode = 'all'
  form.group_ids = []
  form.user_ids = ''
  form.plan_id = 0
  form.group_id = 0
  publishSuggestion.value = undefined
  form.status = 'draft'
  form.notify_mode = 'silent'
  form.starts_at_str = ''
  form.ends_at_str = ''
  form.targeting = { any_of: [] }
}

function fillFormFromAnnouncement(a: Announcement) {
  form.title = a.title
  form.content = a.content
  form.kind = a.kind || ''
  form.audience_mode = a.audience?.mode ?? 'all'
  form.group_ids = [...(a.audience?.group_ids ?? [])]
  form.user_ids = (a.audience?.user_ids ?? []).join(', ')
  form.plan_id = a.plan_id ?? 0
  form.group_id = a.plan_id ? 0 : a.group_id ?? 0
  publishSuggestion.value = undefined
  form.status = a.status
  form.notify_mode = a.notify_mode || 'silent'

  // Backend returns RFC3339 strings
  form.starts_at_str = a.starts_at ? formatDateTimeLocalInput(Math.floor(new Date(a.starts_at).getTime() / 1000)) : ''
  form.ends_at_str = a.ends_at ? formatDateTimeLocalInput(Math.floor(new Date(a.ends_at).getTime() / 1000)) : ''

  form.targeting = a.targeting ?? { any_of: [] }
}

function openCreateDialog() {
  editingAnnouncement.value = null
  resetForm()
  showEditDialog.value = true
  void loadPlans()
}

function openEditDialog(row: Announcement) {
  editingAnnouncement.value = row
  fillFormFromAnnouncement(row)
  showEditDialog.value = true
  if (row.kind) void loadPlans()
}

function closeEdit() {
  showEditDialog.value = false
  editingAnnouncement.value = null
}

function noticeAudience(): NotificationAudience | undefined {
  if (form.audience_mode === 'all') return { mode: 'all' }
  if (form.audience_mode === 'groups') {
    return form.group_ids.length ? { mode: 'groups', group_ids: form.group_ids } : undefined
  }
  const tokens = form.user_ids.trim().split(/[\s,，]+/u)
  if (!tokens.every(token => /^[1-9][0-9]*$/.test(token) && Number.isSafeInteger(Number(token)))) return
  return { mode: 'users', user_ids: [...new Set(tokens.map(Number))] }
}

function buildCreatePayload(audience: NotificationAudience | undefined): CreateAnnouncementRequest {
  const startsAt = parseDateTimeLocalInput(form.starts_at_str)
  const endsAt = parseDateTimeLocalInput(form.ends_at_str)

  return {
    title: form.title,
    content: form.content,
    status: form.kind ? 'draft' : form.status as Announcement['status'],
    notify_mode: form.notify_mode as Announcement['notify_mode'],
    targeting: form.targeting,
    starts_at: !form.kind && startsAt != null ? startsAt : undefined,
    ends_at: endsAt ?? undefined,
    ...(form.kind ? { kind: form.kind, audience, plan_id: form.plan_id || undefined, group_id: form.group_id || undefined } : {}),
  }
}

function buildUpdatePayload(original: Announcement, audience: NotificationAudience | undefined): UpdateAnnouncementRequest {
  const payload: UpdateAnnouncementRequest = {}

  if (form.title !== original.title) payload.title = form.title
  if (form.content !== original.content) payload.content = form.content
  if (!form.kind && form.status !== original.status) payload.status = form.status as Announcement['status']
  if (form.notify_mode !== (original.notify_mode || 'silent')) payload.notify_mode = form.notify_mode as Announcement['notify_mode']
  if (form.kind && !publishedFieldsLocked.value) {
    if (form.kind !== original.kind) payload.kind = form.kind
    if (form.plan_id !== (original.plan_id ?? 0)) payload.plan_id = form.plan_id
    if (form.group_id !== (original.plan_id ? 0 : original.group_id ?? 0)) payload.group_id = form.group_id
    if (JSON.stringify(audience) !== JSON.stringify(original.audience)) payload.audience = audience
  }

  // starts_at / ends_at: distinguish unchanged vs clear(0) vs set
  const originalStarts = original.starts_at ? Math.floor(new Date(original.starts_at).getTime() / 1000) : null
  const originalEnds = original.ends_at ? Math.floor(new Date(original.ends_at).getTime() / 1000) : null

  const newStarts = parseDateTimeLocalInput(form.starts_at_str)
  const newEnds = parseDateTimeLocalInput(form.ends_at_str)

  if (!form.kind && newStarts !== originalStarts) {
    payload.starts_at = newStarts === null ? 0 : newStarts
  }
  if (newEnds !== originalEnds) {
    payload.ends_at = newEnds === null ? 0 : newEnds
  }

  // targeting: do shallow compare by JSON
  if (!form.kind && JSON.stringify(form.targeting ?? {}) !== JSON.stringify(original.targeting ?? {})) {
    payload.targeting = form.targeting
  }

  return payload
}

async function handleSave() {
  const audience = form.kind && !publishedFieldsLocked.value ? noticeAudience() : undefined
  if (form.kind && !publishedFieldsLocked.value && !audience) {
    appStore.showError(t('operations.invalidAudience'))
    return
  }
  // Frontend validation for targeting (to avoid ANNOUNCEMENT_INVALID_TARGET)
  const anyOf = form.targeting?.any_of ?? []
  if (anyOf.length > 50) {
    appStore.showError(t('admin.announcements.failedToCreate'))
    return
  }
  for (const g of anyOf) {
    const allOf = g?.all_of ?? []
    if (allOf.length > 50) {
      appStore.showError(t('admin.announcements.failedToCreate'))
      return
    }
  }

  saving.value = true
  try {
    if (!editingAnnouncement.value) {
      const payload = buildCreatePayload(audience)
      const draft = await adminAPI.announcements.create(payload)
      const suggestedTime = publishSuggestion.value
      if (draft.kind) openPublish(draft, suggestedTime)
      appStore.showSuccess(t('common.success'))
      showEditDialog.value = false
      await loadAnnouncements()
      return
    }

    const original = editingAnnouncement.value
    const payload = buildUpdatePayload(original, audience)
    await adminAPI.announcements.update(original.id, payload)
    appStore.showSuccess(t('common.success'))
    showEditDialog.value = false
    editingAnnouncement.value = null
    await loadAnnouncements()
  } catch (error: any) {
    console.error('Failed to save announcement:', error)
    appStore.showError(error.response?.data?.detail || (editingAnnouncement.value ? t('admin.announcements.failedToUpdate') : t('admin.announcements.failedToCreate')))
  } finally {
    saving.value = false
  }
}

// ===== Delete =====
const showDeleteDialog = ref(false)
const deletingAnnouncement = ref<Announcement | null>(null)

function handleDelete(row: Announcement) {
  deletingAnnouncement.value = row
  showDeleteDialog.value = true
}

async function confirmDelete() {
  if (!deletingAnnouncement.value) return

  try {
    await adminAPI.announcements.delete(deletingAnnouncement.value.id)
    appStore.showSuccess(t('common.success'))
    showDeleteDialog.value = false
    deletingAnnouncement.value = null
    await loadAnnouncements()
  } catch (error: any) {
    console.error('Failed to delete announcement:', error)
    appStore.showError(error.response?.data?.detail || t('admin.announcements.failedToDelete'))
  }
}

// ===== Read status =====
const showReadStatusDialog = ref(false)
const readStatusAnnouncementId = ref<number | null>(null)
const previewAnnouncement = ref<Announcement | null>(null)
const deliveryNotice = ref<Announcement | null>(null)

function openPreview(row: Announcement) {
  previewAnnouncement.value = row
}

function openReadStatus(row: Announcement) {
  if (row.kind) {
    deliveryNotice.value = row
    return
  }
  readStatusAnnouncementId.value = row.id
  showReadStatusDialog.value = true
}

const publishingNotice = ref<Announcement | null>(null)
const withdrawingNotice = ref<Announcement | null>(null)
const publishTiming = ref<'now' | 'later'>('now')
const publishAt = ref('')
const actionBusy = ref(false)
function openPublish(notice: Announcement, suggestedTime?: number) {
  publishingNotice.value = notice
  publishTiming.value = suggestedTime ? 'later' : 'now'
  publishAt.value = suggestedTime ? formatDateTimeLocalInput(suggestedTime) : ''
}
async function publishNotice() {
  if (!publishingNotice.value || actionBusy.value) return
  const startsAt = publishTiming.value === 'later' ? parseDateTimeLocalInput(publishAt.value) : undefined
  if (publishTiming.value === 'later' && (!startsAt || startsAt * 1000 <= Date.now())) {
    appStore.showError(t('operations.scheduleInvalid'))
    return
  }
  actionBusy.value = true
  try {
    await adminAPI.announcements.publish(publishingNotice.value.id, startsAt ?? undefined)
    publishingNotice.value = null
    appStore.showSuccess(t('common.success'))
    await loadAnnouncements()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    actionBusy.value = false
  }
}
async function withdrawNotice() {
  if (!withdrawingNotice.value || actionBusy.value) return
  actionBusy.value = true
  try {
    await adminAPI.announcements.withdraw(withdrawingNotice.value.id)
    withdrawingNotice.value = null
    appStore.showSuccess(t('common.success'))
    await loadAnnouncements()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    actionBusy.value = false
  }
}
async function retryDelivery(notice: Announcement) {
  if (actionBusy.value) return
  actionBusy.value = true
  try {
    await adminAPI.announcements.retry(notice.id)
    appStore.showSuccess(t('common.success'))
    await loadAnnouncements()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    actionBusy.value = false
  }
}

function queryPrice(value: unknown) {
  if (typeof value !== 'string' || !/^\d+(\.\d+)?(e[+-]?\d+)?$/i.test(value)) return undefined
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : undefined
}

function queryID(value: unknown) {
  if (typeof value !== 'string' || !/^[1-9][0-9]*$/.test(value)) return 0
  const id = Number(value)
  return Number.isSafeInteger(id) ? id : 0
}

watch(() => route.query.compose, async compose => {
  if (compose !== '1') return
  const { compose: _compose, kind, plan_id, group_id, old_price, new_price, old_rate, new_rate, effective_at, ...query } = route.query
  const planID = queryID(plan_id)
  const groupID = queryID(group_id)
  const noticeKind = kinds.find(value => value === kind)
  void router.replace({ query })
  if (Boolean(planID) === Boolean(groupID) || !noticeKind) return

  if (groupID) {
    await loadGroups()
    const group = standardGroups.value.find(item => item.id === groupID)
    if (!group) {
      appStore.showError(t('operations.groupUnavailable'))
      return
    }
    openCreateDialog()
    form.kind = noticeKind
    form.group_id = groupID
    if (noticeKind !== 'new_group' || group.is_exclusive) {
      form.audience_mode = 'groups'
      form.group_ids = [groupID]
    }
    form.title = `${t(`operations.kind.${noticeKind}`)} · ${group.name}`
    const oldRate = queryPrice(old_rate)
    const newRate = queryPrice(new_rate)
    const rateText = oldRate !== undefined && newRate !== undefined
      ? `${oldRate}× → ${newRate}×`
      : `${group.rate_multiplier}×`
    form.content = [group.description, `${t('operations.standardRateMultiplier')} · ${rateText}`].filter(Boolean).join('\n\n')
    return
  }

  await loadPlans()
  const plan = plans.value.find(item => item.id === planID)
  if (!plan) {
    appStore.showError(t('operations.planUnavailable'))
    return
  }
  openCreateDialog()
  form.kind = noticeKind
  form.plan_id = planID
  form.title = `${t(`operations.kind.${noticeKind}`)} · ${plan.name}`
  const oldPrice = queryPrice(old_price)
  const newPrice = queryPrice(new_price)
  const currency = plan.currency || 'USD'
  const money = (price: number) => formatPaymentAmount(price, currency)
  const priceText = oldPrice !== undefined && newPrice !== undefined
    ? `${money(oldPrice)} → ${money(newPrice)}`
    : money(plan.price)
  const features = Array.isArray(plan.features) ? plan.features : String(plan.features || '').split('\n')
  const time = typeof effective_at === 'string'
    ? (/^[0-9]+$/.test(effective_at) ? Number(effective_at) : Date.parse(effective_at) / 1000)
    : NaN
  const effective = new Date(time * 1000)
  const effectiveText = Number.isFinite(effective.valueOf()) ? `${t('operations.effectiveAt')} · ${formatDateTime(effective)}` : ''
  form.content = [
    plan.description,
    `${t('payment.admin.price')} · ${priceText}`,
    effectiveText,
    features.filter(Boolean).map(feature => `- ${feature}`).join('\n'),
  ].filter(Boolean).join('\n\n')
  if (effective.valueOf() > Date.now()) publishSuggestion.value = Math.floor(time)
}, { immediate: true })

onMounted(() => {
  void loadGroups()
  void loadAnnouncements()
})

onUnmounted(() => {
  if (searchDebounceTimer) window.clearTimeout(searchDebounceTimer)
  currentController?.abort()
})
</script>
