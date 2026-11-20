<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <PageHeader :title="t('rewards.title')" :description="t('rewards.dailyRules')">
        <template #actions>
          <Button variant="outline" size="sm" class="h-9 rounded-full px-3.5" :disabled="loading || busy !== null" @click="load">
            <Icon name="refresh" size="xs" :stroke-width="1.75" :class="loading ? 'animate-spin' : ''" />
            <span>{{ t('rewards.refresh') }}</span>
          </Button>
        </template>
      </PageHeader>

      <!-- Loading Skeleton -->
      <div v-if="loading && !status" class="space-y-5" role="status">
        <span class="sr-only">{{ t('rewards.loading') }}</span>
        <div class="flex items-center justify-between gap-4 rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
          <div class="flex items-center gap-3.5">
            <Skeleton class="size-10 rounded-full" />
            <div class="space-y-2">
              <Skeleton class="h-3.5 w-36" />
              <Skeleton class="h-3 w-24" />
            </div>
          </div>
          <Skeleton class="h-8 w-24" />
        </div>
        <div class="grid items-start gap-5 lg:grid-cols-2">
          <div v-for="i in 2" :key="`reward-sk-${i}`" class="space-y-4 rounded-2xl border border-border/80 bg-card p-6 shadow-sm">
            <div class="flex items-center gap-3">
              <Skeleton class="size-9 rounded-full" />
              <Skeleton class="h-5 w-32" />
            </div>
            <Skeleton class="h-4 w-full" />
            <Skeleton class="h-2.5 w-full rounded-full" />
            <Skeleton class="h-9 w-full rounded-xl" />
          </div>
        </div>
      </div>

      <template v-if="status">
        <!-- Today's Spend KPI -->
        <div
          v-if="status.config.check_in.enabled || status.config.wheel.enabled"
          class="flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-border/80 bg-card p-5 shadow-sm"
        >
          <div class="flex items-center gap-3.5">
            <div class="flex size-10 shrink-0 items-center justify-center rounded-full bg-amber-500 text-white shadow-sm">
              <Icon name="fire" size="sm" :stroke-width="2" />
            </div>
            <div>
              <p class="text-xs font-medium text-muted-foreground">{{ t('rewards.todaySpend') }}</p>
              <p class="mt-0.5 text-xs tabular-nums text-muted-foreground">{{ status.day }}</p>
            </div>
          </div>
          <p class="text-2xl font-bold tracking-tight tabular-nums text-foreground">
            {{ amount(status.spent_cents) }}
          </p>
        </div>
        <div
          v-else
          class="rounded-2xl border border-border/80 bg-card p-8 text-center text-sm text-muted-foreground shadow-sm"
        >
          {{ t('rewards.closed') }}
        </div>

        <div class="grid items-start gap-5 lg:grid-cols-2">
          <!-- Check-in Card -->
          <section
            v-if="status.config.check_in.enabled"
            class="space-y-4 rounded-2xl border border-border/80 bg-card p-6 shadow-sm"
          >
            <div class="flex items-center gap-3">
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-emerald-500 text-white shadow-sm">
                <Icon name="calendar" size="sm" :stroke-width="2" />
              </div>
              <h2 class="text-base font-semibold tracking-tight text-foreground">{{ t('rewards.checkIn') }}</h2>
            </div>
            <p class="text-sm text-muted-foreground">
              {{ t('rewards.checkInRule', {
                threshold: amount(status.config.check_in.threshold_cents),
                min: amount(status.config.check_in.min_reward_cents),
                max: amount(status.config.check_in.max_reward_cents),
              }) }}
            </p>
            <progress
              class="h-2 w-full overflow-hidden rounded-full accent-emerald-600"
              :aria-label="t('rewards.checkIn')"
              :value="Math.min(status.spent_cents, status.config.check_in.threshold_cents)"
              :max="status.config.check_in.threshold_cents"
            />
            <p v-if="!status.checked_in && checkInShortfall > 0" class="text-sm tabular-nums text-muted-foreground">
              {{ t('rewards.needSpend', { amount: amount(checkInShortfall) }) }}
            </p>
            <Button class="h-9 w-full" :disabled="busy !== null || loading || status.checked_in || checkInShortfall > 0" @click="participate('check_in')">
              {{ status.checked_in ? t('rewards.checkedIn') : busy === 'check_in' ? t('rewards.processing') : t('rewards.checkInNow') }}
            </Button>
            <p
              v-if="result?.kind === 'check_in'"
              class="rounded-xl border border-emerald-500/30 bg-emerald-500/10 px-3.5 py-2.5 text-sm font-medium tabular-nums text-emerald-700 dark:text-emerald-300"
              role="status"
            >
              {{ t('rewards.credited', { amount: amount(result.reward_cents) }) }}
            </p>
          </section>

          <!-- Wheel Card -->
          <section
            v-if="status.config.wheel.enabled"
            class="space-y-4 rounded-2xl border border-border/80 bg-card p-6 shadow-sm"
          >
            <div class="flex items-center gap-3">
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-violet-500 text-white shadow-sm">
                <Icon name="sparkles" size="sm" :stroke-width="2" />
              </div>
              <h2 class="text-base font-semibold tracking-tight text-foreground">{{ t('rewards.wheel') }}</h2>
            </div>
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
              <li
                v-for="(prize, index) in prizes"
                :key="index"
                class="rounded-xl border border-border/80 bg-muted/40 px-3 py-2 text-foreground"
              >
                <span class="min-w-0 break-words tabular-nums">{{ index + 1 }}. {{ prize.name || prizeLabel(prize.reward_cents) }}</span>
              </li>
            </ol>
            <div class="flex flex-wrap items-center justify-between gap-2 text-sm tabular-nums">
              <span class="font-medium text-foreground">{{ t('rewards.drawsAvailable', { count: status.draws_available }) }}</span>
              <span v-if="status.draws_used >= status.config.wheel.daily_limit" class="text-muted-foreground">{{ t('rewards.limitReached') }}</span>
              <span v-else-if="status.draws_available === 0" class="text-muted-foreground">{{ t('rewards.needSpend', { amount: amount(wheelShortfall) }) }}</span>
            </div>
            <Button class="h-9 w-full" :disabled="busy !== null || loading || status.draws_available === 0" @click="participate('wheel')">
              {{ busy === 'wheel' ? t('rewards.drawing') : t('rewards.drawNow') }}
            </Button>
            <p
              v-if="result?.kind === 'wheel'"
              class="rounded-xl border border-border/80 bg-muted/40 px-3.5 py-2.5 text-sm font-medium tabular-nums text-foreground"
              role="status"
            >
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
import PageHeader from '@/components/common/PageHeader.vue'
import Icon from '@/components/icons/Icon.vue'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
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
