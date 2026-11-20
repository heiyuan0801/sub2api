<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-6">
      <PageHeader :title="t('operations.notifications')">
        <template #badge>
          <span class="inline-flex items-center rounded-full border border-border bg-muted/50 px-2.5 py-0.5 text-xs font-medium tabular-nums text-muted-foreground">
            {{ t('operations.unreadCount') }} · {{ store.unreadCount }}
          </span>
        </template>
        <template #actions>
          <button
            type="button"
            class="btn btn-secondary h-9 px-3.5 text-xs"
            @click="showPreferences = true"
          >
            <Icon name="cog" size="xs" />
            <span>{{ t('operations.notificationPreferences') }}</span>
          </button>
          <button
            type="button"
            class="btn btn-secondary h-9 px-3.5 text-xs"
            :disabled="!store.unreadCount || markingAll"
            @click="markAll"
          >
            <Icon name="check" size="xs" />
            <span>{{ t('announcements.markAllRead') }}</span>
          </button>
        </template>
      </PageHeader>

      <div class="overflow-hidden rounded-2xl border border-border/80 bg-card shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-border/80 bg-muted/30 px-5 py-3.5">
          <h2 class="text-sm font-semibold text-foreground">{{ t('operations.notificationHistory') }}</h2>
          <label class="inline-flex cursor-pointer items-center gap-2 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground">
            <input
              v-model="unreadOnly"
              type="checkbox"
              class="size-4 rounded border-border accent-foreground"
              @change="resetPage"
            />
            <span>{{ t('operations.unreadOnly') }}</span>
          </label>
        </div>

        <!-- Loading Skeleton -->
        <div v-if="loading" class="divide-y divide-border/70">
          <div v-for="i in 4" :key="`notif-sk-${i}`" class="flex items-center gap-3.5 p-4 sm:p-5">
            <Skeleton class="size-2 shrink-0 rounded-full" />
            <div class="min-w-0 flex-1 space-y-2">
              <div class="flex items-center gap-2">
                <Skeleton class="h-5 w-16 rounded-full" />
                <Skeleton class="h-4 w-56" />
              </div>
              <Skeleton class="h-3 w-32" />
            </div>
            <Skeleton class="size-4 shrink-0 rounded" />
          </div>
        </div>

        <!-- Empty State -->
        <div v-else-if="!items.length" class="flex flex-col items-center justify-center px-6 py-14 text-center">
          <div class="mb-3 flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
            <Icon name="bell" size="lg" />
          </div>
          <p class="text-sm text-muted-foreground">{{ t('operations.noNotifications') }}</p>
        </div>

        <!-- Notification List -->
        <ul v-else class="divide-y divide-border/70">
          <li v-for="item in items" :key="item.id">
            <button
              type="button"
              class="flex w-full items-start gap-3.5 p-4 text-left transition-colors hover:bg-muted/40 sm:items-center sm:px-5 sm:py-4"
              :class="{ 'bg-muted/25': !item.read_at }"
              @click="openDetail(item.id)"
            >
              <span
                class="mt-1.5 size-2 shrink-0 rounded-full sm:mt-0"
                :class="item.read_at ? 'bg-transparent' : 'bg-sky-500'"
                aria-hidden="true"
              />
              <div class="min-w-0 flex-1 space-y-1.5">
                <div class="flex flex-wrap items-center gap-2">
                  <NotificationKindBadge :kind="item.kind" />
                  <span v-if="!item.read_at" class="sr-only">{{ t('announcements.unread') }}</span>
                  <h3 class="break-words text-sm font-medium text-foreground">{{ item.title }}</h3>
                </div>
                <time
                  class="block text-xs tabular-nums text-muted-foreground"
                  :datetime="item.starts_at || item.created_at"
                >
                  {{ formatDateTime(item.starts_at || item.created_at) }}
                </time>
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
          <time class="tabular-nums">{{ formatDateTime(detail.starts_at || detail.created_at) }}</time>
        </div>
        <div class="markdown-body prose prose-sm max-w-none break-words dark:prose-invert" v-html="html" />
      </template>
      <template #footer>
        <div class="flex flex-wrap justify-end gap-2.5">
          <button type="button" class="btn btn-secondary h-9 px-3.5 text-sm" @click="closeDetail">{{ t('common.close') }}</button>
          <button v-if="canOpenTarget" type="button" class="btn btn-primary h-9 px-3.5 text-sm" :disabled="openingTarget" @click="openTarget">
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
import PageHeader from '@/components/common/PageHeader.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import NotificationKindBadge from '@/components/common/NotificationKindBadge.vue'
import NotificationPreferencesDialog from '@/components/user/NotificationPreferencesDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import { Skeleton } from '@/components/ui/skeleton'
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
