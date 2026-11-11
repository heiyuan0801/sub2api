<template>
  <section class="card p-5">
    <h2 class="text-base font-semibold">{{ t('rewards.history') }}</h2>
    <p v-if="claims.length === 0" class="mt-4 text-sm text-muted-foreground">{{ t('rewards.noHistory') }}</p>
    <div v-else class="mt-3 overflow-x-auto">
      <table class="w-full text-left text-sm">
        <thead class="text-muted-foreground">
          <tr class="border-b">
            <th scope="col" class="py-3 pr-4 font-medium">{{ t('rewards.date') }}</th>
            <th v-if="showUser" scope="col" class="py-3 pr-4 font-medium">{{ t('rewards.userId') }}</th>
            <th scope="col" class="py-3 pr-4 font-medium">{{ t('rewards.activity') }}</th>
            <th scope="col" class="py-3 text-right font-medium">{{ t('rewards.reward') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="claim in claims" :key="claim.id" class="border-b last:border-0">
            <td class="whitespace-nowrap py-3 pr-4">{{ formatDateTime(claim.created_at) }}</td>
            <td v-if="showUser" class="py-3 pr-4 tabular-nums">{{ claim.user_id }}</td>
            <td class="py-3 pr-4">{{ t(claim.kind === 'check_in' ? 'rewards.checkIn' : 'rewards.wheel') }}</td>
            <td class="py-3 text-right tabular-nums">
              <span v-if="claim.reward_cents > 0" class="font-medium text-emerald-600 dark:text-emerald-400">+{{ amount(claim.reward_cents) }}</span>
              <span v-else class="text-muted-foreground">{{ t('rewards.noPrize') }}</span>
              <span v-if="claim.prize_name" class="ml-2 text-xs text-muted-foreground">{{ claim.prize_name }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { RewardActivityClaim } from '@/api/rewardActivities'
import { formatDateTime } from '@/utils/format'

defineProps<{ claims: RewardActivityClaim[]; showUser?: boolean }>()
const { t } = useI18n()
const amount = (cents: number) => (cents / 100).toFixed(2)
</script>
