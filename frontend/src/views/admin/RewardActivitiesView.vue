<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-5">
      <p class="text-sm text-muted-foreground">{{ t('rewards.adminRules') }}</p>
      <p v-if="loading" class="card p-6 text-sm text-muted-foreground" role="status">{{ t('rewards.loading') }}</p>
      <form v-else-if="form" class="space-y-5" @submit.prevent="save">
        <section class="card space-y-4 p-5">
          <div class="flex items-center justify-between gap-4">
            <h2 class="text-base font-semibold">{{ t('rewards.checkIn') }}</h2>
            <label class="flex items-center gap-2 text-sm">
              <input v-model="form.checkIn.enabled" type="checkbox" class="size-4 accent-emerald-600" />
              {{ t('rewards.enabled') }}
            </label>
          </div>
          <div class="grid gap-4 sm:grid-cols-3">
            <label class="space-y-1 text-sm">
              <span>{{ t('rewards.checkInThreshold') }}</span>
              <input v-model.number="form.checkIn.threshold" type="number" class="input" min="0.01" max="1000000" step="0.01" required />
            </label>
            <label class="space-y-1 text-sm">
              <span>{{ t('rewards.minReward') }}</span>
              <input v-model.number="form.checkIn.min" type="number" class="input" min="0.01" max="1000000" step="0.01" required />
            </label>
            <label class="space-y-1 text-sm">
              <span>{{ t('rewards.maxReward') }}</span>
              <input v-model.number="form.checkIn.max" type="number" class="input" :min="form.checkIn.min" max="1000000" step="0.01" required />
            </label>
          </div>
        </section>

        <section class="card space-y-4 p-5">
          <div class="flex items-center justify-between gap-4">
            <h2 class="text-base font-semibold">{{ t('rewards.wheel') }}</h2>
            <label class="flex items-center gap-2 text-sm">
              <input v-model="form.wheel.enabled" type="checkbox" class="size-4 accent-emerald-600" />
              {{ t('rewards.enabled') }}
            </label>
          </div>
          <div class="grid gap-4 sm:grid-cols-2">
            <label class="space-y-1 text-sm">
              <span>{{ t('rewards.wheelThreshold') }}</span>
              <input v-model.number="form.wheel.threshold" type="number" class="input" min="0.01" max="1000000" step="0.01" required />
            </label>
            <label class="space-y-1 text-sm">
              <span>{{ t('rewards.dailyLimit') }}</span>
              <input v-model.number="form.wheel.dailyLimit" type="number" class="input" min="1" max="100" step="1" required />
            </label>
          </div>
          <p class="text-sm text-muted-foreground">{{ t('rewards.weightHelp') }}</p>
          <div v-for="(prize, index) in form.wheel.prizes" :key="prize.id"
            class="grid gap-3 rounded-lg border p-3 sm:grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,1fr)_auto_auto] sm:items-end">
            <label class="space-y-1 text-sm">
              <span>{{ t('rewards.prizeName', { index: index + 1 }) }}</span>
              <input v-model="prize.name" class="input" maxlength="40" :placeholder="prize.amount > 0 ? t('rewards.prizeAmount', { amount: prize.amount }) : t('rewards.noPrize')" />
            </label>
            <label class="space-y-1 text-sm">
              <span>{{ t('rewards.reward') }}</span>
              <input v-model.number="prize.amount" type="number" class="input" min="0" max="1000000" step="0.01" required />
            </label>
            <label class="space-y-1 text-sm">
              <span>{{ t('rewards.weight') }}</span>
              <input v-model.number="prize.weight" type="number" class="input" min="1" max="1000000" step="1" required />
            </label>
            <span class="min-w-14 self-center text-sm tabular-nums text-muted-foreground">{{ probability(prize.weight) }}</span>
            <Button type="button" variant="outline" :disabled="form.wheel.prizes.length <= 2" @click="form.wheel.prizes.splice(index, 1)">{{ t('rewards.removePrize') }}</Button>
          </div>
          <Button type="button" variant="outline" :disabled="form.wheel.prizes.length >= 12" @click="addPrize">{{ t('rewards.addPrize') }}</Button>
        </section>

        <div class="flex justify-end">
          <Button type="submit" :disabled="saving">{{ saving ? t('rewards.processing') : t('rewards.save') }}</Button>
        </div>
      </form>
      <Button v-else variant="outline" @click="load">{{ t('rewards.refresh') }}</Button>

      <div class="space-y-2">
        <div class="flex justify-end">
          <Button variant="outline" :disabled="loadingClaims" @click="loadClaims">{{ t('rewards.refreshHistory') }}</Button>
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
import { Button } from '@/components/ui/button'
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
