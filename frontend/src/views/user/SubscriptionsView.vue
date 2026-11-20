<template>
  <AppLayout>
    <div class="space-y-6">
      <PageHeader
        :title="t('userSubscriptions.title')"
        :description="t('userSubscriptions.description')"
      >
        <template #actions>
          <button
            type="button"
            class="inline-flex h-9 items-center gap-1.5 rounded-full border border-border bg-card px-3.5 text-xs font-medium text-foreground transition-colors hover:bg-muted/60 disabled:pointer-events-none disabled:opacity-50"
            :disabled="loading"
            @click="loadSubscriptions"
          >
            <Icon name="refresh" size="xs" :stroke-width="1.75" :class="loading ? 'animate-spin' : ''" />
            <span>{{ t('common.refresh') }}</span>
          </button>
        </template>
      </PageHeader>

      <!-- Loading State -->
      <div v-if="loading" class="grid gap-6 lg:grid-cols-2">
        <div
          v-for="i in 2"
          :key="`sub-sk-${i}`"
          class="overflow-hidden rounded-2xl border border-border/80 bg-card shadow-sm"
        >
          <div class="flex items-center justify-between gap-4 border-b border-border/80 p-5">
            <div class="flex items-center gap-3">
              <Skeleton class="size-9 shrink-0 rounded-full" />
              <div class="space-y-2">
                <Skeleton class="h-4 w-36" />
                <Skeleton class="h-3 w-48" />
              </div>
            </div>
            <Skeleton class="h-7 w-20 rounded-full" />
          </div>
          <div class="space-y-4 p-5">
            <div class="flex items-center justify-between">
              <Skeleton class="h-3.5 w-20" />
              <Skeleton class="h-3.5 w-32" />
            </div>
            <div v-for="j in 2" :key="`bar-sk-${i}-${j}`" class="space-y-2">
              <div class="flex items-center justify-between">
                <Skeleton class="h-3.5 w-16" />
                <Skeleton class="h-3.5 w-24" />
              </div>
              <Skeleton class="h-2 w-full rounded-full" />
              <Skeleton class="h-3 w-28" />
            </div>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div
        v-else-if="subscriptions.length === 0"
        class="rounded-2xl border border-border/80 bg-card p-12 text-center shadow-sm"
      >
        <div
          class="mx-auto mb-4 flex size-14 items-center justify-center rounded-full bg-muted text-muted-foreground"
        >
          <Icon name="creditCard" size="lg" />
        </div>
        <h3 class="mb-1.5 text-base font-semibold text-foreground">
          {{ t('userSubscriptions.noActiveSubscriptions') }}
        </h3>
        <p class="mx-auto max-w-md text-sm text-muted-foreground">
          {{ t('userSubscriptions.noActiveSubscriptionsDesc') }}
        </p>
      </div>

      <!-- Subscriptions Grid -->
      <div v-else class="grid gap-6 lg:grid-cols-2">
        <div
          v-for="subscription in subscriptions"
          :key="subscription.id"
          class="overflow-hidden rounded-2xl border border-border/80 bg-card shadow-sm transition-colors hover:border-border"
        >
          <!-- Header -->
          <div
            class="flex flex-wrap items-start justify-between gap-3 border-b border-border/80 p-5"
          >
            <div class="flex min-w-0 items-start gap-3">
              <div
                :class="[
                  'mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-full text-white shadow-sm',
                  platformAccentDotClass(subscription.group?.platform || '')
                ]"
              >
                <Icon name="creditCard" size="sm" :stroke-width="2" />
              </div>
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <h3 class="truncate text-base font-semibold tracking-tight text-foreground">
                    {{ subscription.group?.name || `Group #${subscription.group_id}` }}
                  </h3>
                  <span
                    :class="[
                      'rounded-md border px-2 py-0.5 text-[11px] font-medium',
                      platformBadgeClass(subscription.group?.platform || '')
                    ]"
                  >
                    {{ platformLabel(subscription.group?.platform || '') }}
                  </span>
                </div>
                <p
                  v-if="subscription.group?.description"
                  class="mt-1 text-xs text-muted-foreground"
                >
                  {{ subscription.group.description }}
                </p>
                <div class="mt-1.5 flex flex-wrap gap-x-3 gap-y-1 text-[11px] tabular-nums text-muted-foreground">
                  <span>{{ t('payment.planCard.rate') }}: ×{{ subscription.group?.rate_multiplier ?? 1 }}</span>
                  <span v-if="subscriptionHasPeakRate(subscription)" class="font-medium text-amber-600 dark:text-amber-400">
                    {{ t('payment.planCard.peakRate') }}: {{ subscriptionPeakRateLabel(subscription) }}
                  </span>
                </div>
              </div>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <span
                :class="[
                  'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
                  subscription.status === 'active'
                    ? 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
                    : subscription.status === 'expired'
                      ? 'bg-muted text-muted-foreground'
                      : 'bg-red-500/10 text-red-700 dark:text-red-400'
                ]"
              >
                {{ t(`userSubscriptions.status.${subscription.status}`) }}
              </span>
              <button
                v-if="subscription.status === 'active'"
                type="button"
                :class="[
                  'inline-flex h-8 items-center rounded-lg px-3 text-xs font-semibold text-white transition-colors',
                  platformButtonClass(subscription.group?.platform || '')
                ]"
                @click="router.push({ path: '/purchase', query: { tab: 'subscription', group: String(subscription.group_id) } })"
              >
                {{ t('payment.renewNow') }}
              </button>
            </div>
          </div>

          <!-- Usage Progress -->
          <div class="space-y-4 p-5">
            <!-- Expiration Info -->
            <div
              v-if="subscription.expires_at"
              class="flex items-center justify-between gap-3 rounded-xl border border-border/60 bg-muted/40 px-3.5 py-2.5 text-sm"
            >
              <span class="text-muted-foreground">{{
                t('userSubscriptions.expires')
              }}</span>
              <span :class="['tabular-nums', getExpirationClass(subscription.expires_at)]">
                {{ formatExpirationDate(subscription.expires_at) }}
              </span>
            </div>
            <div
              v-else
              class="flex items-center justify-between gap-3 rounded-xl border border-border/60 bg-muted/40 px-3.5 py-2.5 text-sm"
            >
              <span class="text-muted-foreground">{{
                t('userSubscriptions.expires')
              }}</span>
              <span class="font-medium text-foreground">{{
                t('userSubscriptions.noExpiration')
              }}</span>
            </div>

            <!-- Daily Usage -->
            <div v-if="subscription.group?.daily_limit_usd" class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="text-sm font-medium text-foreground">
                  {{ t('userSubscriptions.daily') }}
                </span>
                <span class="text-sm tabular-nums text-muted-foreground">
                  <span class="font-medium text-foreground">${{ (subscription.daily_usage_usd || 0).toFixed(2) }}</span>
                  / ${{ subscription.group.daily_limit_usd.toFixed(2) }}
                </span>
              </div>
              <div class="relative h-2 overflow-hidden rounded-full bg-muted">
                <div
                  class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
                  :class="
                    getProgressBarClass(
                      subscription.daily_usage_usd,
                      subscription.group.daily_limit_usd
                    )
                  "
                  :style="{
                    width: getProgressWidth(
                      subscription.daily_usage_usd,
                      subscription.group.daily_limit_usd
                    )
                  }"
                ></div>
              </div>
              <p
                v-if="subscription.daily_window_start"
                class="text-xs tabular-nums text-muted-foreground"
              >
                {{ formatDailyUsageWindow(subscription) }}
              </p>
            </div>

            <!-- Weekly Usage -->
            <div v-if="subscription.group?.weekly_limit_usd" class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="text-sm font-medium text-foreground">
                  {{ t('userSubscriptions.weekly') }}
                </span>
                <span class="text-sm tabular-nums text-muted-foreground">
                  <span class="font-medium text-foreground">${{ (subscription.weekly_usage_usd || 0).toFixed(2) }}</span>
                  / ${{ subscription.group.weekly_limit_usd.toFixed(2) }}
                </span>
              </div>
              <div class="relative h-2 overflow-hidden rounded-full bg-muted">
                <div
                  class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
                  :class="
                    getProgressBarClass(
                      subscription.weekly_usage_usd,
                      subscription.group.weekly_limit_usd
                    )
                  "
                  :style="{
                    width: getProgressWidth(
                      subscription.weekly_usage_usd,
                      subscription.group.weekly_limit_usd
                    )
                  }"
                ></div>
              </div>
              <p
                v-if="subscription.weekly_window_start"
                class="text-xs tabular-nums text-muted-foreground"
              >
                {{
                  t('userSubscriptions.resetIn', {
                    time: formatResetTime(subscription.weekly_window_start, 168)
                  })
                }}
              </p>
            </div>

            <!-- Monthly Usage -->
            <div v-if="subscription.group?.monthly_limit_usd" class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="text-sm font-medium text-foreground">
                  {{ t('userSubscriptions.monthly') }}
                </span>
                <span class="text-sm tabular-nums text-muted-foreground">
                  <span class="font-medium text-foreground">${{ (subscription.monthly_usage_usd || 0).toFixed(2) }}</span>
                  / ${{ subscription.group.monthly_limit_usd.toFixed(2) }}
                </span>
              </div>
              <div class="relative h-2 overflow-hidden rounded-full bg-muted">
                <div
                  class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
                  :class="
                    getProgressBarClass(
                      subscription.monthly_usage_usd,
                      subscription.group.monthly_limit_usd
                    )
                  "
                  :style="{
                    width: getProgressWidth(
                      subscription.monthly_usage_usd,
                      subscription.group.monthly_limit_usd
                    )
                  }"
                ></div>
              </div>
              <p
                v-if="subscription.monthly_window_start"
                class="text-xs tabular-nums text-muted-foreground"
              >
                {{
                  t('userSubscriptions.resetIn', {
                    time: formatResetTime(subscription.monthly_window_start, 720)
                  })
                }}
              </p>
            </div>

            <!-- No limits configured - Unlimited badge -->
            <div
              v-if="
                !subscription.group?.daily_limit_usd &&
                !subscription.group?.weekly_limit_usd &&
                !subscription.group?.monthly_limit_usd
              "
              class="flex items-center justify-center rounded-xl border border-emerald-500/20 bg-emerald-500/10 py-6"
            >
              <div class="flex items-center gap-3">
                <span class="text-4xl leading-none text-emerald-600 dark:text-emerald-400">∞</span>
                <div>
                  <p class="text-sm font-semibold text-emerald-700 dark:text-emerald-300">
                    {{ t('userSubscriptions.unlimited') }}
                  </p>
                  <p class="text-xs text-emerald-600/80 dark:text-emerald-400/80">
                    {{ t('userSubscriptions.unlimitedDesc') }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import subscriptionsAPI from '@/api/subscriptions'
import type { UserSubscription } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import Icon from '@/components/icons/Icon.vue'
import { Skeleton } from '@/components/ui/skeleton'
import { formatDateTimeToMinute } from '@/utils/format'
import { hasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'
import { platformBadgeClass, platformButtonClass, platformLabel } from '@/utils/platformColors'
import {
  getExpirationDateRelation,
  getRemainingDurationParts,
  isOneTimeDailyQuota,
  type RemainingDurationParts
} from '@/utils/subscriptionQuota'

function platformAccentDotClass(p: string): string {
  switch (p) {
    case 'anthropic': return 'bg-orange-500'
    case 'openai': return 'bg-emerald-500'
    case 'antigravity': return 'bg-purple-500'
    case 'gemini': return 'bg-blue-500'
    default: return 'bg-zinc-500'
  }
}

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()

const subscriptions = ref<UserSubscription[]>([])
const loading = ref(true)

function subscriptionHasPeakRate(subscription: UserSubscription): boolean {
  return hasPeakRate(subscription.group)
}

function subscriptionPeakRateLabel(subscription: UserSubscription): string {
  return formatPeakRateWindow(subscription.group, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset))
}

async function loadSubscriptions() {
  try {
    loading.value = true
    subscriptions.value = await subscriptionsAPI.getMySubscriptions()
  } catch (error) {
    console.error('Failed to load subscriptions:', error)
    appStore.showError(t('userSubscriptions.failedToLoad'))
  } finally {
    loading.value = false
  }
}

function getProgressWidth(used: number | undefined, limit: number | null | undefined): string {
  if (!limit || limit === 0) return '0%'
  const percentage = Math.min(((used || 0) / limit) * 100, 100)
  return `${percentage}%`
}

function getProgressBarClass(used: number | undefined, limit: number | null | undefined): string {
  if (!limit || limit === 0) return 'bg-muted-foreground/40'
  const percentage = ((used || 0) / limit) * 100
  if (percentage >= 90) return 'bg-red-500'
  if (percentage >= 70) return 'bg-orange-500'
  return 'bg-emerald-500'
}

function formatExpirationDate(expiresAt: string): string {
  const now = new Date()
  const expires = new Date(expiresAt)
  const diff = expires.getTime() - now.getTime()
  const days = Math.ceil(diff / (1000 * 60 * 60 * 24))
  const relation = getExpirationDateRelation(expires, now)

  if (relation === null) return ''

  if (relation === 'expired') {
    return t('userSubscriptions.status.expired')
  }

  const dateStr = formatDateTimeToMinute(expires)

  if (relation === 'today') {
    return `${dateStr} (${t('common.today')})`
  }
  if (relation === 'tomorrow') {
    return `${dateStr} (${t('common.tomorrow')})`
  }

  return t('userSubscriptions.daysRemaining', { days }) + ` (${dateStr})`
}

function getExpirationClass(expiresAt: string): string {
  const now = new Date()
  const expires = new Date(expiresAt)
  const diff = expires.getTime() - now.getTime()
  const days = Math.ceil(diff / (1000 * 60 * 60 * 24))

  if (diff <= 0) return 'text-red-600 dark:text-red-400 font-medium'
  if (days <= 3) return 'text-red-600 dark:text-red-400 font-medium'
  if (days <= 7) return 'text-orange-600 dark:text-orange-400 font-medium'
  return 'text-foreground font-medium'
}

function formatDurationParts(parts: RemainingDurationParts): string {
  if (parts.days > 0) {
    return `${parts.days}d ${parts.hours}h`
  }

  if (parts.hours > 0) {
    return `${parts.hours}h ${parts.minutes}m`
  }

  return `${parts.minutes}m`
}

function formatDailyUsageWindow(subscription: UserSubscription): string {
  if (isOneTimeDailyQuota(subscription) && subscription.expires_at) {
    const parts = getRemainingDurationParts(subscription.expires_at)
    if (!parts) return t('userSubscriptions.windowNotActive')
    return t('userSubscriptions.quotaEndsIn', { time: formatDurationParts(parts) })
  }

  return t('userSubscriptions.resetIn', {
    time: formatResetTime(subscription.daily_window_start, 24)
  })
}

function formatResetTime(windowStart: string | null, windowHours: number): string {
  if (!windowStart) return t('userSubscriptions.windowNotActive')

  const start = new Date(windowStart)
  const end = new Date(start.getTime() + windowHours * 60 * 60 * 1000)
  const parts = getRemainingDurationParts(end)

  return parts ? formatDurationParts(parts) : t('userSubscriptions.windowNotActive')
}

onMounted(() => {
  loadSubscriptions()
})
</script>
