<template>
  <SectionCard>
    <div class="flex items-center justify-end gap-3 flex-wrap">
      <div
        role="tablist"
        class="inline-flex p-0.5 rounded-xl bg-muted border border-border text-xs"
      >
        <button
          v-for="opt in windowOptions"
          :key="opt.value"
          type="button"
          role="tab"
          :aria-selected="window === opt.value"
          class="px-3 py-1 rounded-lg transition-colors"
          :class="window === opt.value
            ? 'bg-card shadow-sm text-foreground font-semibold'
            : 'text-muted-foreground hover:text-foreground'"
          @click="emit('update:window', opt.value)"
        >
          {{ opt.label }}
        </button>
      </div>

      <StatusBadge :label="overallLabel" :tone="overallStatus === 'operational' ? 'success' : 'warning'" />

      <button
        type="button"
        class="h-8 w-8 rounded-lg flex items-center justify-center text-muted-foreground hover:text-foreground hover:bg-muted transition-colors disabled:opacity-50"
        :disabled="loading"
        :title="t('common.refresh')"
        @click="emit('refresh')"
      >
        <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
      </button>

      <AutoRefreshButton
        v-if="autoRefresh"
        :enabled="autoRefresh.enabled.value"
        :interval-seconds="autoRefresh.intervalSeconds.value"
        :countdown="autoRefresh.countdown.value"
        :intervals="autoRefresh.intervals"
        @update:enabled="autoRefresh.setEnabled"
        @update:interval="autoRefresh.setInterval"
      />
      <WebsiteStatusLink />
    </div>
  </SectionCard>
</template>

<script setup lang="ts">
import { SectionCard, StatusBadge } from '@/components/page'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import AutoRefreshButton from '@/components/common/AutoRefreshButton.vue'
import WebsiteStatusLink from './WebsiteStatusLink.vue'
export type MonitorWindow = '7d' | '15d' | '30d'
export type OverallStatus = 'operational' | 'degraded'

const props = defineProps<{
  overallStatus: OverallStatus
  intervalSeconds: number
  window: MonitorWindow
  loading: boolean
  autoRefresh?: {
    enabled: { value: boolean }
    intervalSeconds: { value: number }
    countdown: { value: number }
    intervals: readonly number[]
    setEnabled: (v: boolean) => void
    setInterval: (v: number) => void
  }
}>()

const emit = defineEmits<{
  (e: 'update:window', value: MonitorWindow): void
  (e: 'refresh'): void
}>()

const { t } = useI18n()

const windowOptions = computed<{ value: MonitorWindow; label: string }[]>(() => [
  { value: '7d', label: t('channelStatus.windowTab.7d') },
  { value: '15d', label: t('channelStatus.windowTab.15d') },
  { value: '30d', label: t('channelStatus.windowTab.30d') },
])

const overallLabel = computed(() => t(`channelStatus.overall.${props.overallStatus}`))
</script>
