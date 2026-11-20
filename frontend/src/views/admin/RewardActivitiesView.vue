<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <PageHeader
        :title="t('rewards.adminTitle')"
        :description="t('rewards.adminRules')"
      >
        <template #actions>
          <button
            type="button"
            class="inline-flex h-9 items-center gap-1.5 rounded-full border border-border bg-card px-3.5 text-xs font-medium text-foreground transition-colors hover:bg-muted/60 disabled:pointer-events-none disabled:opacity-50"
            :disabled="loading"
            @click="load"
          >
            <Icon
              name="refresh"
              size="xs"
              :stroke-width="1.75"
              :class="{ 'animate-spin': loading }"
            />
            <span>{{ t('rewards.refresh') }}</span>
          </button>
        </template>
      </PageHeader>

      <!-- Loading Skeleton -->
      <div v-if="loading" class="space-y-5" role="status" :aria-label="t('rewards.loading')">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div
            v-for="i in 3"
            :key="`reward-kpi-sk-${i}`"
            class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5 shadow-sm"
          >
            <div class="flex items-center justify-between gap-2">
              <Skeleton class="h-3.5 w-24" />
              <Skeleton class="size-9 rounded-full" />
            </div>
            <Skeleton class="mt-3 h-7 w-28" />
            <Skeleton class="mt-2 h-3.5 w-36" />
          </div>
        </div>
        <div
          v-for="i in 2"
          :key="`reward-sec-sk-${i}`"
          class="space-y-4 rounded-2xl border border-border/80 bg-card p-5 shadow-sm"
        >
          <div class="flex items-center justify-between">
            <Skeleton class="h-5 w-36" />
            <Skeleton class="h-6 w-20 rounded-full" />
          </div>
          <div class="grid gap-4 sm:grid-cols-3">
            <Skeleton class="h-14 rounded-lg" />
            <Skeleton class="h-14 rounded-lg" />
            <Skeleton class="h-14 rounded-lg" />
          </div>
        </div>
      </div>

      <template v-else-if="form">
        <!-- Summary KPI Cards -->
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
            <div class="flex items-center justify-between gap-2">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('rewards.checkIn') }}
              </p>
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-emerald-500 text-white shadow-sm">
                <Icon name="checkCircle" size="sm" :stroke-width="2" />
              </div>
            </div>
            <p class="mt-3 truncate text-2xl font-bold tracking-tight tabular-nums text-foreground">
              {{ form.checkIn.min.toFixed(2) }} – {{ form.checkIn.max.toFixed(2) }}
            </p>
            <div class="mt-2 flex items-center justify-between gap-2 text-xs text-muted-foreground">
              <span class="truncate tabular-nums">{{ t('rewards.checkInThreshold') }}: {{ form.checkIn.threshold.toFixed(2) }}</span>
              <span
                :class="[
                  'badge shrink-0',
                  form.checkIn.enabled ? 'badge-success' : 'badge-gray'
                ]"
              >
                {{ form.checkIn.enabled ? t('rewards.enabled') : '—' }}
              </span>
            </div>
          </div>

          <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
            <div class="flex items-center justify-between gap-2">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('rewards.wheel') }}
              </p>
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-violet-500 text-white shadow-sm">
                <Icon name="sparkles" size="sm" :stroke-width="2" />
              </div>
            </div>
            <p class="mt-3 truncate text-2xl font-bold tracking-tight tabular-nums text-foreground">
              {{ form.wheel.dailyLimit }}
            </p>
            <div class="mt-2 flex items-center justify-between gap-2 text-xs text-muted-foreground">
              <span class="truncate tabular-nums">{{ t('rewards.wheelThreshold') }}: {{ form.wheel.threshold.toFixed(2) }}</span>
              <span
                :class="[
                  'badge shrink-0',
                  form.wheel.enabled ? 'badge-success' : 'badge-gray'
                ]"
              >
                {{ form.wheel.enabled ? t('rewards.enabled') : '—' }}
              </span>
            </div>
          </div>

          <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
            <div class="flex items-center justify-between gap-2">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('rewards.history') }}
              </p>
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-amber-500 text-white shadow-sm">
                <Icon name="trophy" size="sm" :stroke-width="2" />
              </div>
            </div>
            <p class="mt-3 truncate text-2xl font-bold tracking-tight tabular-nums text-foreground">
              {{ claims.length }}
            </p>
            <p class="mt-2 truncate text-xs tabular-nums text-muted-foreground">
              {{ t('rewards.dailyLimit') }}: {{ form.wheel.dailyLimit }}
            </p>
          </div>
        </div>

        <form class="space-y-5" @submit.prevent="save">
          <section class="space-y-4 rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
            <div class="flex items-center justify-between gap-4 border-b border-border/70 pb-3.5">
              <div class="flex items-center gap-2.5">
                <div class="flex size-8 items-center justify-center rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                  <Icon name="checkCircle" size="sm" :stroke-width="2" />
                </div>
                <h2 class="text-base font-semibold tracking-tight text-foreground">{{ t('rewards.checkIn') }}</h2>
              </div>
              <label class="inline-flex cursor-pointer items-center gap-2 rounded-full border border-border bg-muted/40 px-3 py-1 text-xs font-medium text-foreground transition-colors hover:bg-muted/70">
                <input v-model="form.checkIn.enabled" type="checkbox" class="size-3.5 rounded border-border accent-primary" />
                {{ t('rewards.enabled') }}
              </label>
            </div>
            <div class="grid gap-4 sm:grid-cols-3">
              <label class="block">
                <span class="input-label">{{ t('rewards.checkInThreshold') }}</span>
                <input v-model.number="form.checkIn.threshold" type="number" class="input tabular-nums" min="0.01" max="1000000" step="0.01" required />
              </label>
              <label class="block">
                <span class="input-label">{{ t('rewards.minReward') }}</span>
                <input v-model.number="form.checkIn.min" type="number" class="input tabular-nums" min="0.01" max="1000000" step="0.01" required />
              </label>
              <label class="block">
                <span class="input-label">{{ t('rewards.maxReward') }}</span>
                <input v-model.number="form.checkIn.max" type="number" class="input tabular-nums" :min="form.checkIn.min" max="1000000" step="0.01" required />
              </label>
            </div>
          </section>

          <section class="space-y-4 rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
            <div class="flex items-center justify-between gap-4 border-b border-border/70 pb-3.5">
              <div class="flex items-center gap-2.5">
                <div class="flex size-8 items-center justify-center rounded-full bg-violet-500/10 text-violet-600 dark:text-violet-400">
                  <Icon name="sparkles" size="sm" :stroke-width="2" />
                </div>
                <h2 class="text-base font-semibold tracking-tight text-foreground">{{ t('rewards.wheel') }}</h2>
              </div>
              <label class="inline-flex cursor-pointer items-center gap-2 rounded-full border border-border bg-muted/40 px-3 py-1 text-xs font-medium text-foreground transition-colors hover:bg-muted/70">
                <input v-model="form.wheel.enabled" type="checkbox" class="size-3.5 rounded border-border accent-primary" />
                {{ t('rewards.enabled') }}
              </label>
            </div>
            <div class="grid gap-4 sm:grid-cols-2">
              <label class="block">
                <span class="input-label">{{ t('rewards.wheelThreshold') }}</span>
                <input v-model.number="form.wheel.threshold" type="number" class="input tabular-nums" min="0.01" max="1000000" step="0.01" required />
              </label>
              <label class="block">
                <span class="input-label">{{ t('rewards.dailyLimit') }}</span>
                <input v-model.number="form.wheel.dailyLimit" type="number" class="input tabular-nums" min="1" max="100" step="1" required />
              </label>
            </div>
            <p class="rounded-lg border border-border/60 bg-muted/40 px-3 py-2 text-xs text-muted-foreground">{{ t('rewards.weightHelp') }}</p>
            <div class="space-y-2.5">
              <div
                v-for="(prize, index) in form.wheel.prizes"
                :key="prize.id"
                class="grid gap-3 rounded-xl border border-border/80 bg-muted/30 p-3.5 sm:grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,1fr)_auto_auto] sm:items-end"
              >
                <label class="block">
                  <span class="input-label">{{ t('rewards.prizeName', { index: index + 1 }) }}</span>
                  <input v-model="prize.name" class="input" maxlength="40" :placeholder="prize.amount > 0 ? t('rewards.prizeAmount', { amount: prize.amount }) : t('rewards.noPrize')" />
                </label>
                <label class="block">
                  <span class="input-label">{{ t('rewards.reward') }}</span>
                  <input v-model.number="prize.amount" type="number" class="input tabular-nums" min="0" max="1000000" step="0.01" required />
                </label>
                <label class="block">
                  <span class="input-label">{{ t('rewards.weight') }}</span>
                  <input v-model.number="prize.weight" type="number" class="input tabular-nums" min="1" max="1000000" step="1" required />
                </label>
                <span class="inline-flex h-9 min-w-16 items-center justify-center rounded-md border border-border/60 bg-background px-2.5 text-xs font-medium tabular-nums text-foreground">{{ probability(prize.weight) }}</span>
                <Button type="button" variant="outline" class="h-9" :disabled="form.wheel.prizes.length <= 2" @click="form.wheel.prizes.splice(index, 1)">{{ t('rewards.removePrize') }}</Button>
              </div>
            </div>
            <div>
              <Button type="button" variant="outline" :disabled="form.wheel.prizes.length >= 12" @click="addPrize">{{ t('rewards.addPrize') }}</Button>
            </div>
          </section>

          <div class="flex justify-end">
            <Button type="submit" :disabled="saving">{{ saving ? t('rewards.processing') : t('rewards.save') }}</Button>
          </div>
        </form>
      </template>

      <div v-else class="rounded-2xl border border-border/80 bg-card p-6 text-center shadow-sm">
        <Button variant="outline" @click="load">{{ t('rewards.refresh') }}</Button>
      </div>

      <div class="space-y-3">
        <div class="flex items-center justify-end">
          <Button variant="outline" size="sm" :disabled="loadingClaims" @click="loadClaims">
            <Icon name="refresh" size="xs" :class="{ 'animate-spin': loadingClaims }" />
            {{ t('rewards.refreshHistory') }}
          </Button>
        </div>
        <RewardClaimHistory :claims="claims" show-user />
      </div>
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
import { useAppStore } from '@/stores'
import { getRewardActivityConfig, listRewardActivityClaims, saveRewardActivityConfig, type RewardActivityClaim, type RewardActivityConfig } from '@/api/rewardActivities'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const loadingClaims = ref(false)
const saving = ref(false)
const claims = ref<RewardActivityClaim[]>([])
let nextPrizeId = 0

function toForm(config: RewardActivityConfig) {
  return {
    checkIn: {
      enabled: config.check_in.enabled,
      threshold: config.check_in.threshold_cents / 100,
      min: config.check_in.min_reward_cents / 100,
      max: config.check_in.max_reward_cents / 100,
    },
    wheel: {
      enabled: config.wheel.enabled,
      threshold: config.wheel.threshold_cents / 100,
      dailyLimit: config.wheel.daily_limit,
      prizes: config.wheel.prizes.map(prize => ({
        id: nextPrizeId++, name: prize.name, amount: prize.reward_cents / 100, weight: prize.weight,
      })),
    },
  }
}

const form = ref<ReturnType<typeof toForm>>()
const totalWeight = computed(() => form.value?.wheel.prizes.reduce((sum, prize) => sum + Number(prize.weight), 0) ?? 0)
const probability = (weight: number) => totalWeight.value
  ? new Intl.NumberFormat(undefined, { style: 'percent', maximumFractionDigits: 2 }).format(weight / totalWeight.value)
  : '—'
const cents = (amount: number) => Math.round(amount * 100)

function addPrize() {
  form.value?.wheel.prizes.push({ id: nextPrizeId++, name: '', amount: 0, weight: 1 })
}

async function load() {
  loading.value = true
  try {
    form.value = toForm(await getRewardActivityConfig())
  } catch {
    appStore.showError(t('rewards.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadClaims() {
  loadingClaims.value = true
  try {
    claims.value = await listRewardActivityClaims()
  } catch {
    appStore.showError(t('rewards.loadFailed'))
  } finally {
    loadingClaims.value = false
  }
}

async function save() {
  if (!form.value || saving.value) return
  const { checkIn, wheel } = form.value
  const config: RewardActivityConfig = {
    check_in: {
      enabled: checkIn.enabled,
      threshold_cents: cents(checkIn.threshold),
      min_reward_cents: cents(checkIn.min),
      max_reward_cents: cents(checkIn.max),
    },
    wheel: {
      enabled: wheel.enabled,
      threshold_cents: cents(wheel.threshold),
      daily_limit: wheel.dailyLimit,
      prizes: wheel.prizes.map(prize => ({ name: prize.name.trim(), reward_cents: cents(prize.amount), weight: prize.weight })),
    },
  }
  saving.value = true
  try {
    form.value = toForm(await saveRewardActivityConfig(config))
    appStore.showSuccess(t('rewards.saved'))
  } catch (error) {
    const reason = (error as { reason?: string }).reason
    const messages: Record<string, string> = {
      INVALID_REWARD_THRESHOLD: 'invalidThreshold',
      INVALID_CHECK_IN_REWARD: 'invalidRange',
      INVALID_WHEEL_CONFIG: 'invalidWheel',
      INVALID_WHEEL_PRIZE: 'invalidPrize',
    }
    appStore.showError(t(`rewards.${messages[reason ?? ''] ?? 'saveFailed'}`))
  } finally {
    saving.value = false
  }
}

onMounted(() => { void Promise.all([load(), loadClaims()]) })
</script>
