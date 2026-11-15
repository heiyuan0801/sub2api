<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { RewardActivityClaim } from '@/api/rewardActivities'
import type { Column } from '@/components/common/types'
import DataTable from '@/components/page/AdminDataTable.vue'
import { EmptyState } from '@/components/page'
import { formatDateTime } from '@/utils/format'

defineProps<{ claims: RewardActivityClaim[] }>()

const { t } = useI18n()
const columns = computed<Column[]>(() => [
  { key: 'created_at', label: t('rewards.date') },
  { key: 'user_id', label: t('rewards.userId') },
  { key: 'kind', label: t('rewards.activity') },
  { key: 'reward_cents', label: t('rewards.reward'), class: 'text-right' },
])
</script>

<template>
  <DataTable :columns="columns" :data="claims" :sticky-first-column="false" :sticky-actions-column="false">
    <template #cell-created_at="{ value }">
      <span class="tabular-nums">{{ formatDateTime(value) }}</span>
    </template>
    <template #cell-user_id="{ value }">
      <span class="tabular-nums">{{ value }}</span>
    </template>
    <template #cell-kind="{ value }">
      {{ t(value === 'check_in' ? 'rewards.checkIn' : 'rewards.wheel') }}
    </template>
    <template #cell-reward_cents="{ value, row }">
      <span v-if="value > 0" class="font-medium tabular-nums text-success">+{{ (value / 100).toFixed(2) }}</span>
      <span v-else class="text-muted-foreground">{{ t('rewards.noPrize') }}</span>
      <span v-if="row.prize_name" class="ml-2 text-xs text-muted-foreground">{{ row.prize_name }}</span>
    </template>
    <template #empty>
      <EmptyState :title="t('rewards.noHistory')" />
    </template>
  </DataTable>
</template>
