<template>
  <div class="min-w-[7rem] space-y-1 text-xs" :title="t('admin.accounts.callMetrics.hint')">
    <span v-if="loading && !stats" class="text-gray-400">{{ t('common.loading') }}</span>
    <span v-else-if="error" class="text-red-500">{{ t('admin.accounts.callMetrics.failed') }}</span>
    <template v-else-if="stats && stats.requests > 0">
      <div>{{ t('admin.accounts.callMetrics.cache') }} <span class="font-mono" :class="cacheRate != null && cacheRate >= 70 ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'">{{ cacheRate == null ? '—' : `${cacheRate.toFixed(1)}%` }}</span></div>
      <div>{{ t('admin.accounts.callMetrics.firstToken') }} <span class="font-mono" :class="firstToken != null && firstToken <= 5000 ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'">{{ firstToken == null ? '—' : `${(firstToken / 1000).toFixed(1)}s` }}</span></div>
      <div class="text-gray-400">{{ t('admin.accounts.callMetrics.today', { count: stats.requests }) }}</div>
    </template>
    <span v-else-if="!loading" class="text-gray-400">{{ t(stats?.error_stats?.abnormal_requests ? 'admin.accounts.callMetrics.noUsage' : 'admin.accounts.callMetrics.unused') }}</span>
    <div v-if="stats && !error" :title="t('admin.accounts.callMetrics.errorsHint')" class="space-y-1">
      <template v-if="stats.error_stats">
        <div :class="stats.error_stats.abnormal_requests > 0 ? 'text-red-500' : 'text-gray-400'">{{ t('admin.accounts.callMetrics.errors', { count: stats.error_stats.abnormal_requests }) }}</div>
        <div v-if="stats.error_stats.abnormal_requests > 0" class="text-gray-500">
          <span class="text-emerald-600">{{ t('admin.accounts.callMetrics.recovered', { count: stats.error_stats.recovered_requests }) }}</span>
          · <span class="text-red-500">{{ t('admin.accounts.callMetrics.finalFailed', { count: stats.error_stats.failed_requests }) }}</span>
        </div>
      </template>
      <span v-else class="text-gray-400">{{ t('admin.accounts.callMetrics.errorsUnavailable') }}</span>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { WindowStats } from '@/types'
const props = defineProps<{ stats?: WindowStats | null; loading?: boolean; error?: string | null }>()
const { t } = useI18n()
const cacheRate = computed(() => {
  const read = props.stats?.cache_read_tokens
  const input = props.stats?.cache_input_tokens
  return typeof read === 'number' && typeof input === 'number' && Number.isFinite(read) && Number.isFinite(input) && input > 0 && read >= 0 && read <= input ? read / input * 100 : null
})
const firstToken = computed(() => {
  const value = props.stats?.average_first_token_ms
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? value : null
})
</script>
