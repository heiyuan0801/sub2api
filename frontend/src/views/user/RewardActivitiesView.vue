<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-5">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <p class="text-sm text-muted-foreground">{{ t('rewards.dailyRules') }}</p>
        <Button variant="outline" :disabled="loading || busy !== null" @click="load">{{ t('rewards.refresh') }}</Button>
      </div>
      <p v-if="loading && !status" class="card p-6 text-sm text-muted-foreground" role="status">{{ t('rewards.loading') }}</p>
      <template v-if="status">
        <div v-if="status.config.check_in.enabled || status.config.wheel.enabled" class="card flex flex-wrap items-center justify-between gap-3 p-5">
          <p class="text-sm text-muted-foreground">{{ t('rewards.todaySpend') }} · {{ status.day }}</p>
          <p class="text-2xl font-semibold tabular-nums">{{ amount(status.spent_cents) }}</p>
        </div>
        <div v-else class="card p-6 text-muted-foreground">{{ t('rewards.closed') }}</div>
        <div class="grid items-start gap-5 lg:grid-cols-2">
          <section v-if="status.config.check_in.enabled" class="card space-y-4 p-5">
            <h2 class="text-lg font-semibold">{{ t('rewards.checkIn') }}</h2>
            <p class="text-sm text-muted-foreground">
              {{ t('rewards.checkInRule', {
                threshold: amount(status.config.check_in.threshold_cents),
                min: amount(status.config.check_in.min_reward_cents),
                max: amount(status.config.check_in.max_reward_cents),
              }) }}
            </p>
            <progress class="h-2 w-full overflow-hidden rounded-full accent-emerald-600" :aria-label="t('rewards.checkIn')"
              :value="Math.min(status.spent_cents, status.config.check_in.threshold_cents)" :max="status.config.check_in.threshold_cents" />
            <p v-if="!status.checked_in && checkInShortfall > 0" class="text-sm text-muted-foreground">
              {{ t('rewards.needSpend', { amount: amount(checkInShortfall) }) }}
            </p>
            <Button class="w-full" :disabled="busy !== null || loading || status.checked_in || checkInShortfall > 0" @click="participate('check_in')">
              {{ status.checked_in ? t('rewards.checkedIn') : busy === 'check_in' ? t('rewards.processing') : t('rewards.checkInNow') }}
            </Button>
            <p v-if="result?.kind === 'check_in'" class="text-sm font-medium text-emerald-600 dark:text-emerald-400" role="status">
              {{ t('rewards.credited', { amount: amount(result.reward_cents) }) }}
            </p>
          </section>
          <section v-if="status.config.wheel.enabled" class="card space-y-4 p-5">
            <h2 class="text-lg font-semibold">{{ t('rewards.wheel') }}</h2>
            <p class="text-sm text-muted-foreground">
              {{ t('rewards.wheelRule', { threshold: amount(status.config.wheel.threshold_cents), limit: status.config.wheel.daily_limit }) }}
            </p>
            <div class="reward-wheel-shell" aria-hidden="true">
              <div class="reward-wheel-pointer" />
              <div class="reward-wheel" :style="{ background: wheelGradient, transform: `rotate(${rotation}deg)` }">
                <span v-for="(_, index) in prizes" :key="index" class="reward-wheel-label"
                  :style="{ transform: `rotate(${(index + 0.5) * 360 / prizes.length}deg)` }">
                  <span>{{ index + 1 }}</span>
                </span>
              </div>
              <span class="reward-wheel-center" />
            </div>
            <ol class="grid grid-cols-1 gap-2 text-sm sm:grid-cols-2">
              <li v-for="(prize, index) in prizes" :key="index" class="rounded-lg border px-2.5 py-1.5">
                <span class="min-w-0 break-words">{{ index + 1 }}. {{ prize.name || prizeLabel(prize.reward_cents) }}</span>
              </li>
            </ol>
            <div class="flex flex-wrap items-center justify-between gap-2 text-sm">
              <span>{{ t('rewards.drawsAvailable', { count: status.draws_available }) }}</span>
              <span v-if="status.draws_used >= status.config.wheel.daily_limit" class="text-muted-foreground">{{ t('rewards.limitReached') }}</span>
              <span v-else-if="status.draws_available === 0" class="text-muted-foreground">{{ t('rewards.needSpend', { amount: amount(wheelShortfall) }) }}</span>
            </div>
            <Button class="w-full" :disabled="busy !== null || loading || status.draws_available === 0" @click="participate('wheel')">
              {{ busy === 'wheel' ? t('rewards.drawing') : t('rewards.drawNow') }}
            </Button>
            <p v-if="result?.kind === 'wheel'" class="text-sm font-medium" role="status">
              <span v-if="result.prize_name">{{ result.prize_name }} · </span>{{ result.reward_cents > 0 ? t('rewards.credited', { amount: amount(result.reward_cents) }) : t('rewards.noPrize') }}
            </p>
          </section>
        </div>
        <RewardClaimHistory v-if="status.history.length || status.config.check_in.enabled || status.config.wheel.enabled" :claims="status.history" />
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { Button } from '@/components/ui/button'
import RewardClaimHistory from '@/components/rewards/RewardClaimHistory.vue'
import { useAppStore, useAuthStore } from '@/stores'
import { claimReward, getRewardActivityStatus, type RewardActivityClaim, type RewardActivityKind, type RewardActivityStatus } from '@/api/rewardActivities'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const status = ref<RewardActivityStatus>()
const loading = ref(false)
const busy = ref<RewardActivityKind | null>(null)
const result = ref<RewardActivityClaim>()
const rotation = ref(0)
const pending: Partial<Record<RewardActivityKind, string>> = {}
const amount = (cents: number) => (cents / 100).toFixed(2)
const prizeLabel = (cents: number) => cents > 0 ? t('rewards.prizeAmount', { amount: amount(cents) }) : t('rewards.noPrize')
const prizes = computed(() => status.value?.config.wheel.prizes ?? [])
const checkInShortfall = computed(() => status.value ? Math.max(0, status.value.config.check_in.threshold_cents - status.value.spent_cents) : 0)
const wheelShortfall = computed(() => status.value
  ? Math.max(0, status.value.config.wheel.threshold_cents - Math.max(0, status.value.spent_cents - status.value.draw_spend_used_cents))
  : 0)
const wheelGradient = computed(() => {
  const colors = ['#1e5f50', '#2c4c82', '#7f4a37', '#5a4381', '#256676', '#576b34']
  return `conic-gradient(${prizes.value.map((_, index) => {
    const size = 360 / prizes.value.length
    return `${colors[index % colors.length]} ${index * size}deg ${(index + 1) * size}deg`
  }).join(', ')})`
})

async function load() {
  loading.value = true
  try {
    status.value = await getRewardActivityStatus()
  } catch {
    appStore.showError(t('rewards.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function participate(kind: RewardActivityKind) {
  if (busy.value || loading.value) return
  busy.value = kind
  result.value = undefined
  try {
    const requestId = pending[kind] ??= crypto.randomUUID()
    const claim = await claimReward(kind, requestId)
    delete pending[kind]
    const prize = prizes.value[claim.prize_index]
    if (kind === 'wheel' && prize?.name === claim.prize_name && prize.reward_cents === claim.reward_cents) {
      const target = (360 - (claim.prize_index + 0.5) * 360 / prizes.value.length) % 360
      rotation.value += 1440 + (target - rotation.value % 360 + 360) % 360
      if (!window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
        await new Promise(resolve => window.setTimeout(resolve, 1300))
      }
    }
    result.value = claim
    if (claim.reward_cents > 0) void authStore.refreshUser().catch(() => {})
    await load()
  } catch (error) {
    const failure = error as { status?: number; reason?: string }
    if (failure.status && failure.status >= 400 && failure.status < 500) delete pending[kind]
    const messages: Record<string, string> = {
      REWARD_ACTIVITY_DISABLED: 'closed',
      REWARD_ALREADY_CHECKED_IN: 'checkedIn',
      REWARD_SPEND_REQUIRED: 'spendRequired',
      REWARD_DRAW_LIMIT: 'limitReached',
    }
    appStore.showError(t(`rewards.${messages[failure.reason ?? ''] ?? 'claimFailed'}`))
    await load()
  } finally {
    busy.value = null
  }
}

onMounted(load)
</script>

<style scoped>
.reward-wheel-shell {
  position: relative;
  width: min(100%, 220px);
  aspect-ratio: 1;
  margin: 1.25rem auto;
}
.reward-wheel {
  position: absolute;
  inset: 0;
  overflow: hidden;
  border: 4px solid hsl(var(--background));
  border-radius: 50%;
  box-shadow: 0 0 0 1px hsl(var(--border));
  transition: transform 1.25s cubic-bezier(.12, .65, .16, 1);
}
.reward-wheel-label {
  position: absolute;
  inset: 0;
  display: flex;
  justify-content: center;
  padding-top: 12%;
  color: white;
  font-weight: 600;
}
.reward-wheel-center {
  position: absolute;
  inset: 43%;
  border: 3px solid hsl(var(--background));
  border-radius: 50%;
  background: #e3ca8c;
}
.reward-wheel-pointer {
  position: absolute;
  z-index: 1;
  top: -4px;
  left: calc(50% - 8px);
  border-top: 17px solid #e3ca8c;
  border-inline: 8px solid transparent;
}
@media (prefers-reduced-motion: reduce) {
  .reward-wheel { transition: none; }
}
</style>
