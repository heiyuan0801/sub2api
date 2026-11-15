<template>
  <BaseDialog :show="Boolean(notification)" :title="t('operations.deliveryRecords')" width="wide" @close="emit('close')">
    <div class="space-y-4">
      <h3 v-if="notification" class="break-words font-medium">{{ notification.title }}</h3>
      <dl v-if="stats" class="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <div v-for="metric in metrics" :key="metric.label" class="rounded-lg border border-border p-3">
          <dt class="text-xs text-muted-foreground">{{ t(metric.label) }}</dt>
          <dd class="mt-1 text-xl font-semibold tabular-nums">{{ metric.value }}</dd>
        </div>
      </dl>
      <div v-if="stats" class="rounded-lg border border-border p-3 text-sm">
        <dl class="grid grid-cols-1 gap-2 sm:grid-cols-3">
          <div><dt class="text-muted-foreground">{{ t('operations.deliveryStatus') }}</dt><dd>{{ t(`operations.deliveryStates.${stats.delivery_status}`, stats.delivery_status) }}</dd></div>
          <div><dt class="text-muted-foreground">{{ t('operations.attempts') }}</dt><dd>{{ stats.attempts }}</dd></div>
          <div v-if="stats.next_attempt_at"><dt class="text-muted-foreground">{{ t('operations.nextAttempt') }}</dt><dd>{{ formatDateTime(stats.next_attempt_at) }}</dd></div>
        </dl>
        <p v-if="stats.last_error" class="mt-3 break-words text-red-600 dark:text-red-400">{{ stats.last_error }}</p>
      </div>
      <DataTable :columns="columns" :data="deliveries" :loading="loading" row-key="user_id">
        <template #cell-delivered_at="{ value }">{{ formatDateTime(value) }}</template>
        <template #cell-read_at="{ value }">{{ value ? formatDateTime(value) : '—' }}</template>
      </DataTable>
      <Pagination v-if="total" :total="total" :page="page" :page-size="pageSize" @update:page="changePage" @update:page-size="changePageSize" />
    </div>
    <template #footer>
      <div class="flex justify-end gap-2">
        <button type="button" class="btn btn-secondary" :disabled="loading" @click="load">{{ t('common.refresh') }}</button>
        <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.close') }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import announcementsAPI from '@/api/admin/announcements'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import { formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { Announcement } from '@/types'
import type { NotificationDelivery, NotificationStats } from '@/types/operations'

const props = defineProps<{ notification: Announcement | null }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const app = useAppStore()
const stats = ref<NotificationStats>()
const deliveries = ref<NotificationDelivery[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
let request: AbortController | undefined

const columns = computed(() => [
  { key: 'user_id', label: t('operations.userIDs') },
  { key: 'email', label: t('common.email') },
  { key: 'delivered_at', label: t('operations.deliveredAt') },
  { key: 'read_at', label: t('announcements.read') },
  { key: 'click_count', label: t('operations.clickCount') },
])
const metrics = computed(() => [
  { label: 'operations.audienceCount', value: stats.value?.target_count },
  { label: 'operations.readCount', value: stats.value?.read_count },
  { label: 'operations.unreadCount', value: stats.value?.unread_count },
  { label: 'operations.clickCount', value: stats.value?.click_count },
])

async function load() {
  request?.abort()
  const id = props.notification?.id
  if (!id) return
  const controller = new AbortController()
  request = controller
  loading.value = true
  try {
    const [newStats, newDeliveries] = await Promise.all([
      announcementsAPI.stats(id, controller.signal),
      announcementsAPI.deliveries(id, page.value, pageSize.value, controller.signal),
    ])
    if (controller.signal.aborted) return
    stats.value = newStats
    deliveries.value = newDeliveries.items
    total.value = newDeliveries.total
  } catch (error) {
    if (!controller.signal.aborted) app.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    if (!controller.signal.aborted) loading.value = false
  }
}
function changePage(value: number) {
  page.value = value
  void load()
}
function changePageSize(value: number) {
  pageSize.value = value
  changePage(1)
}
watch(() => props.notification?.id, () => {
  stats.value = undefined
  deliveries.value = []
  total.value = 0
  changePage(1)
}, { immediate: true })
onUnmounted(() => request?.abort())
</script>
