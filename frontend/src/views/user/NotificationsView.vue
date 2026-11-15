<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-4">
      <header class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 class="text-2xl font-semibold">{{ t('operations.notifications') }}</h1>
          <p class="mt-1 text-sm text-muted-foreground">{{ t('operations.unreadCount') }} · {{ store.unreadCount }}</p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button type="button" class="btn btn-secondary" @click="showPreferences = true">{{ t('operations.notificationPreferences') }}</button>
          <button type="button" class="btn btn-secondary" :disabled="!store.unreadCount || markingAll" @click="markAll">
            {{ t('announcements.markAllRead') }}
          </button>
        </div>
      </header>
      <div class="card overflow-hidden">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-border p-4">
          <h2 class="font-medium">{{ t('operations.notificationHistory') }}</h2>
          <label class="flex items-center gap-2 text-sm">
            <input v-model="unreadOnly" type="checkbox" class="h-4 w-4" @change="resetPage" />
            {{ t('operations.unreadOnly') }}
          </label>
        </div>
        <div v-if="loading" class="grid min-h-40 place-items-center"><LoadingSpinner /></div>
        <p v-else-if="!items.length" class="p-12 text-center text-sm text-muted-foreground">{{ t('operations.noNotifications') }}</p>
        <ul v-else class="divide-y divide-border">
          <li v-for="item in items" :key="item.id">
            <button
              type="button"
              class="flex w-full items-start gap-3 p-4 text-left transition-colors hover:bg-muted/50 sm:items-center sm:p-5"
              :class="{ 'bg-muted/30': !item.read_at }"
              @click="openDetail(item.id)"
            >
              <span class="mt-1.5 h-2 w-2 shrink-0 rounded-full sm:mt-0" :class="item.read_at ? 'bg-transparent' : 'bg-primary-500'" aria-hidden="true" />
              <div class="min-w-0 flex-1 space-y-1.5">
                <div class="flex flex-wrap items-center gap-2">
                  <NotificationKindBadge :kind="item.kind" />
                  <span v-if="!item.read_at" class="sr-only">{{ t('announcements.unread') }}</span>
                  <h3 class="break-words font-medium">{{ item.title }}</h3>
                </div>
                <time class="text-xs text-muted-foreground" :datetime="item.starts_at || item.created_at">{{ formatDateTime(item.starts_at || item.created_at) }}</time>
              </div>
              <Icon name="chevronRight" size="sm" class="shrink-0 text-muted-foreground" />
            </button>
          </li>
        </ul>
      </div>
      <Pagination
        v-if="total"
        :page="page"
        :total="total"
        :page-size="pageSize"
        @update:page="setPage"
        @update:page-size="setPageSize"
      />
    </div>
    <NotificationPreferencesDialog :show="showPreferences" @close="showPreferences = false" />
    <BaseDialog :show="Boolean(detail)" :title="detail?.title || ''" width="wide" @close="closeDetail">
      <template v-if="detail">
        <div class="mb-4 flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
          <NotificationKindBadge :kind="detail.kind" />
          <span v-if="detail.group_name && groupTarget" class="inline-flex items-center gap-1.5">
            <Icon name="creditCard" size="xs" aria-hidden="true" />
            {{ detail.group_name }}
          </span>
          <time>{{ formatDateTime(detail.starts_at || detail.created_at) }}</time>
        </div>
        <div class="markdown-body prose prose-sm max-w-none break-words dark:prose-invert" v-html="html" />
      </template>
      <template #footer>
        <div class="flex flex-wrap justify-end gap-2">
          <button type="button" class="btn btn-secondary" @click="closeDetail">{{ t('common.close') }}</button>
          <button v-if="canOpenTarget" type="button" class="btn btn-primary" :disabled="openingTarget" @click="openTarget">
            {{ t(groupTarget ? 'operations.openGroup' : 'operations.openPlan') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import NotificationKindBadge from '@/components/common/NotificationKindBadge.vue'
import NotificationPreferencesDialog from '@/components/user/NotificationPreferencesDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { useAnnouncementStore } from '@/stores/announcements'
import announcementsAPI from '@/api/announcements'
import { formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import type { UserAnnouncement } from '@/types'
import '@/styles/announcement-markdown.css'

const { t } = useI18n()
const app = useAppStore()
const store = useAnnouncementStore()
const route = useRoute()
const router = useRouter()
const unreadOnly = ref(false)
const showPreferences = ref(false)
const loading = ref(false)
const markingAll = ref(false)
const openingTarget = ref(false)
const items = ref<UserAnnouncement[]>([])
const detail = ref<UserAnnouncement>()
const groupTarget = computed(() => Boolean(detail.value?.group_id && !detail.value.plan_id))
const canOpenTarget = computed(() => Boolean(detail.value?.action_url) &&
  (!groupTarget.value || isFeatureFlagEnabled(FeatureFlags.availableChannels)))
const html = computed(() => DOMPurify.sanitize(marked.parse(detail.value?.content ?? '', { async: false, gfm: true, breaks: true })))
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
let listFetch: AbortController | undefined
let detailFetch: AbortController | undefined

async function load() {
  listFetch?.abort()
  const controller = new AbortController()
  listFetch = controller
  loading.value = true
  try {
    const data = await announcementsAPI.history(page.value, pageSize.value, unreadOnly.value, controller.signal)
    if (controller.signal.aborted) return
    if (page.value > data.pages && page.value > 1) {
      page.value = Math.max(1, data.pages)
      void load()
      return
    }
    items.value = data.items
    total.value = data.total
  } catch (error) {
    if (!controller.signal.aborted) app.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    if (!controller.signal.aborted) loading.value = false
  }
}

function setPage(value: number) {
  page.value = value
  void load()
}
function resetPage() { setPage(1) }
function setPageSize(value: number) {
  pageSize.value = value
  resetPage()
}
function openDetail(id: number) {
  void router.replace({ query: { ...route.query, id } })
}
function closeDetail() {
  void router.replace({ query: { ...route.query, id: undefined } })
}
async function loadDetail(value: unknown) {
  detailFetch?.abort()
  detail.value = undefined
  const id = typeof value === 'string' && /^[1-9][0-9]*$/.test(value) ? Number(value) : 0
  if (!Number.isSafeInteger(id) || id <= 0) return
  const controller = new AbortController()
  detailFetch = controller
  try {
    const data = await announcementsAPI.detail(id, controller.signal)
    if (controller.signal.aborted) return
    detail.value = data
    if (!data.read_at && await store.markAsRead(id) && !controller.signal.aborted) {
      detail.value = { ...data, read_at: new Date().toISOString() }
      void load()
    }
  } catch (error) {
    if (!controller.signal.aborted) app.showError(extractApiErrorMessage(error, t('common.error')))
  }
}
async function markAll() {
  markingAll.value = true
  try {
    await store.markAllAsRead()
    resetPage()
  } catch (error) {
    app.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    markingAll.value = false
  }
}
async function openTarget() {
  const item = detail.value
  if (!item || !canOpenTarget.value || openingTarget.value) return
  const group = groupTarget.value
  const unavailable = t(group ? 'operations.groupUnavailable' : 'operations.planUnavailable')
  const target = group
    ? /^\/available-channels\?group_id=[1-9][0-9]*$/
    : /^\/purchase\?plan_id=[1-9][0-9]*$/
  openingTarget.value = true
  try {
    const { action_url } = await announcementsAPI.click(item.id)
    if (!target.test(action_url) || action_url !== item.action_url) throw new Error(unavailable)
    if (detail.value?.id === item.id) await router.push(action_url)
  } catch (error) {
    app.showError(extractApiErrorMessage(error, unavailable))
  } finally {
    openingTarget.value = false
  }
}
watch(() => route.query.id, loadDetail, { immediate: true })
onMounted(load)
onUnmounted(() => {
  listFetch?.abort()
  detailFetch?.abort()
})
</script>
