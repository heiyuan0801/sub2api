<template>
  <SectionCard :title="t('rewards.history')">
    <DataTable :columns="columns" :data="claims" row-key="id">
      <template #cell-kind="{ row }">
        {{ t(row.kind === 'check_in' ? 'rewards.checkIn' : 'rewards.wheel') }}
      </template>
      <template #cell-reward_cents="{ row }">
        <span v-if="row.reward_cents > 0" class="font-medium text-success">+{{ amount(row.reward_cents) }}</span>
        <span v-else class="text-muted-foreground">{{ t('rewards.noPrize') }}</span>
        <span v-if="row.prize_name" class="ml-2 text-xs text-muted-foreground">{{ row.prize_name }}</span>
      </template>
      <template #empty><EmptyState :title="t('rewards.noHistory')" /></template>
    </DataTable>
  </SectionCard>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { EmptyState, SectionCard } from '@/components/page'
import DataTable from '@/components/page/AdminDataTable.vue'
import type { Column } from '@/components/common/types'
import type { RewardActivityClaim } from '@/api/rewardActivities'
import { formatDateTime } from '@/utils/format'

const props = defineProps<{ claims: RewardActivityClaim[]; showUser?: boolean }>()
const { t } = useI18n()
const columns = computed<Column[]>(() => [
  { key: 'created_at', label: t('rewards.date'), formatter: value => formatDateTime(value) },
  ...(props.showUser ? [{ key: 'user_id', label: t('rewards.userId'), class: 'tabular-nums' }] : []),
  { key: 'kind', label: t('rewards.activity') },
  { key: 'reward_cents', label: t('rewards.reward'), class: 'text-right tabular-nums' }
])
const amount = (cents: number) => (cents / 100).toFixed(2)
</script>
