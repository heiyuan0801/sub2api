<script setup lang="ts">
import { CheckCircle2, History } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { EmptyState, SectionCard, StatusBadge, type PageTone } from '@/components/page'
import ClusterLogDetails from '@/components/admin/cluster/ClusterLogDetails.vue'
import { createClusterText } from '@/components/admin/cluster/clusterText'
import type { ClusterAlert, ClusterEvent } from '@/api/admin/cluster'

const props = defineProps<{
  alerts?: readonly ClusterAlert[]
  events?: readonly ClusterEvent[]
  compact?: boolean
}>()
const { t } = useI18n()
const text = createClusterText(t)
const alertTone = (severity: string): PageTone => severity === 'critical' || severity === 'error' ? 'danger' : severity === 'warning' ? 'warning' : 'info'
const eventTone = (kind: string): PageTone => kind === 'alert_raised' ? 'warning' : kind === 'alert_resolved' || kind === 'node_started' ? 'success' : 'info'
const displayDate = (value: string) => new Date(value).toLocaleString()
const firstItems = <T,>(items: readonly T[]) => props.compact ? items.slice(0, 4) : items
</script>

<template>
  <SectionCard
    :title="t(alerts ? 'cluster.alerts' : 'cluster.dashboard.timeline')"
    :description="alerts?.length.toLocaleString()"
    flush
  >
    <template v-if="$slots.actions" #actions><slot name="actions" /></template>
    <template v-if="alerts">
      <EmptyState v-if="!alerts.length" :icon="CheckCircle2" :title="t('cluster.healthyAlerts')" class="py-7" />
      <ul v-else class="divide-y divide-border">
        <li v-for="alert in firstItems(alerts)" :key="alert.key" class="space-y-1.5 px-3.5 py-3">
          <div class="flex flex-wrap items-center justify-between gap-1.5">
            <p class="min-w-0 break-words text-sm font-medium">{{ text.alertName(alert.key) }}</p>
            <StatusBadge :label="text.label('severities', alert.severity)" :tone="alertTone(alert.severity)" />
          </div>
          <p class="break-words text-xs text-muted-foreground">
            {{ text.scope(alert.node_id, alert.key) }} · <time :datetime="alert.raised_at">{{ displayDate(alert.raised_at) }}</time>
          </p>
          <p class="break-words text-xs text-muted-foreground" :class="compact && 'line-clamp-2'">{{ text.message(alert.message) }}</p>
          <ClusterLogDetails v-if="!compact" :code="alert.key" :message="alert.message" />
        </li>
      </ul>
    </template>

    <template v-else>
      <EmptyState v-if="!events?.length" :icon="History" :title="t('cluster.empty')" class="py-7" />
      <ol v-else class="divide-y divide-border">
        <li v-for="event in firstItems(events ?? [])" :key="event.id" class="space-y-1.5 px-3.5 py-3">
          <div class="flex flex-wrap items-center justify-between gap-1.5">
            <StatusBadge :label="text.label('kinds', event.kind)" :tone="eventTone(event.kind)" soft />
            <time class="text-xs text-muted-foreground" :datetime="event.created_at">{{ displayDate(event.created_at) }}</time>
          </div>
          <p class="break-words text-xs text-muted-foreground" :class="compact ? 'line-clamp-2' : 'whitespace-pre-wrap'">
            {{ text.scope(event.node_id, event.kind === 'alert_resolved' ? event.message : undefined) }} · {{ text.event(event) }}
          </p>
          <ClusterLogDetails v-if="!compact" :code="event.kind" :message="event.message" />
        </li>
      </ol>
    </template>
  </SectionCard>
</template>
