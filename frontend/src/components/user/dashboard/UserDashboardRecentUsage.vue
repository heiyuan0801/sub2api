<template>
  <div class="card overflow-hidden">
    <div class="flex items-center justify-between border-b border-border px-5 py-3.5">
      <h2 class="text-sm font-semibold text-foreground">{{ t('dashboard.recentUsage') }}</h2>
      <span class="badge badge-gray">{{ t('dashboard.last7Days') }}</span>
    </div>

    <div v-if="loading" class="divide-y divide-border">
      <div v-for="i in 4" :key="`recent-sk-${i}`" class="flex items-center justify-between gap-3 px-5 py-3">
        <div class="flex items-center gap-3">
          <Skeleton class="size-8 rounded-md" />
          <div class="space-y-1.5">
            <Skeleton class="h-3.5 w-32" />
            <Skeleton class="h-3 w-24" />
          </div>
        </div>
        <div class="space-y-1.5 text-right">
          <Skeleton class="ml-auto h-3.5 w-24" />
          <Skeleton class="ml-auto h-3 w-16" />
        </div>
      </div>
    </div>

    <div v-else-if="data.length === 0" class="p-6">
      <EmptyState :title="t('dashboard.noUsageRecords')" :description="t('dashboard.startUsingApi')" />
    </div>

    <div v-else>
      <div class="divide-y divide-border">
        <div
          v-for="log in data"
          :key="log.id"
          class="flex items-center justify-between gap-3 px-5 py-3 transition-colors hover:bg-muted/40"
        >
          <div class="flex min-w-0 items-center gap-3">
            <div class="flex size-8 shrink-0 items-center justify-center rounded-md border border-border bg-muted/50 text-muted-foreground">
              <Icon name="beaker" size="sm" />
            </div>
            <div class="min-w-0">
              <p class="truncate text-sm font-medium text-foreground">{{ log.model }}</p>
              <p class="text-xs tabular-nums text-muted-foreground">{{ formatDateTime(log.created_at) }}</p>
            </div>
          </div>
          <div class="shrink-0 text-right tabular-nums">
            <p class="text-sm font-medium">
              <span class="text-emerald-600 dark:text-emerald-400" :title="t('dashboard.actual')">${{ formatCost(log.actual_cost) }}</span>
              <span class="font-normal text-muted-foreground" :title="t('dashboard.standard')"> / ${{ formatCost(log.total_cost) }}</span>
            </p>
            <p class="text-xs text-muted-foreground">{{ (log.input_tokens + log.output_tokens).toLocaleString() }} tokens</p>
          </div>
        </div>
      </div>

      <div class="border-t border-border px-5 py-3">
        <router-link
          to="/usage"
          class="flex items-center justify-center gap-1.5 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground"
        >
          {{ t('dashboard.viewAllUsage') }}
          <Icon name="arrowRight" size="xs" />
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import { Skeleton } from '@/components/ui/skeleton'
import { formatDateTime } from '@/utils/format'
import type { UsageLog } from '@/types'

defineProps<{
  data: UsageLog[]
  loading: boolean
}>()
const { t } = useI18n()
const formatCost = (c: number) => c.toFixed(4)
</script>
