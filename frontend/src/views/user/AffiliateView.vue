<template>
  <AppLayout>
    <div class="space-y-6">
      <PageHeader :title="t('affiliate.title')" :description="t('affiliate.description')" />

      <!-- Loading Skeleton -->
      <div v-if="loading" class="space-y-6">
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div
            v-for="i in 4"
            :key="`aff-kpi-sk-${i}`"
            class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5 shadow-sm"
          >
            <div class="flex items-center justify-between gap-2">
              <Skeleton class="h-3.5 w-24" />
              <Skeleton class="size-9 rounded-full" />
            </div>
            <Skeleton class="mt-3 h-7 w-28" />
            <Skeleton class="mt-2 h-3 w-32" />
          </div>
        </div>
        <div class="rounded-2xl border border-border/80 bg-card p-6 shadow-sm">
          <Skeleton class="h-5 w-40" />
          <Skeleton class="mt-2 h-4 w-72" />
          <div class="mt-5 grid gap-4 md:grid-cols-2">
            <Skeleton class="h-12 w-full rounded-xl" />
            <Skeleton class="h-12 w-full rounded-xl" />
          </div>
        </div>
      </div>

      <template v-else-if="detail">
        <!-- KPI Cards -->
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
            <div class="flex items-center justify-between gap-2">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('affiliate.stats.rebateRate') }}
              </p>
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-violet-500 text-white shadow-sm">
                <Icon name="trendingUp" size="sm" :stroke-width="2" />
              </div>
            </div>
            <p class="mt-3 text-2xl font-bold tracking-tight tabular-nums text-foreground">
              {{ formattedRebateRate }}<span class="ml-0.5 text-base font-medium text-muted-foreground">%</span>
            </p>
            <p class="mt-1.5 text-xs text-muted-foreground">
              {{ t('affiliate.stats.rebateRateHint') }}
            </p>
          </div>

          <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
            <div class="flex items-center justify-between gap-2">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('affiliate.stats.invitedUsers') }}
              </p>
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-sky-500 text-white shadow-sm">
                <Icon name="users" size="sm" :stroke-width="2" />
              </div>
            </div>
            <p class="mt-3 text-2xl font-bold tracking-tight tabular-nums text-foreground">
              {{ formatCount(detail.aff_count) }}
            </p>
          </div>

          <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
            <div class="flex items-center justify-between gap-2">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('affiliate.stats.availableQuota') }}
              </p>
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-emerald-500 text-white shadow-sm">
                <Icon name="dollar" size="sm" :stroke-width="2" />
              </div>
            </div>
            <p class="mt-3 text-2xl font-bold tracking-tight tabular-nums text-emerald-600 dark:text-emerald-400">
              {{ formatCurrency(detail.aff_quota) }}
            </p>
          </div>

          <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
            <div class="flex items-center justify-between gap-2">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('affiliate.stats.totalQuota') }}
              </p>
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-amber-500 text-white shadow-sm">
                <Icon name="trophy" size="sm" :stroke-width="2" />
              </div>
            </div>
            <p class="mt-3 text-2xl font-bold tracking-tight tabular-nums text-foreground">
              {{ formatCurrency(detail.aff_history_quota) }}
            </p>
            <p v-if="detail.aff_frozen_quota > 0" class="mt-1.5 text-xs font-medium tabular-nums text-amber-600 dark:text-amber-400">
              {{ t('affiliate.stats.frozenQuota') }}: {{ formatCurrency(detail.aff_frozen_quota) }}
            </p>
          </div>
        </div>

        <!-- Invite Code & Link Card -->
        <div class="rounded-2xl border border-border/80 bg-card p-6 shadow-sm">
          <h3 class="text-base font-semibold tracking-tight text-foreground">{{ t('affiliate.title') }}</h3>
          <p class="mt-1 text-sm text-muted-foreground">{{ t('affiliate.description') }}</p>

          <div class="mt-5 grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <p class="text-xs font-medium text-muted-foreground">{{ t('affiliate.yourCode') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-xl border border-border bg-muted/40 px-3 py-2 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm font-semibold text-foreground sm:flex-1 sm:truncate">{{ detail.aff_code }}</code>
                <button type="button" class="btn btn-secondary btn-sm h-8 w-full sm:w-auto sm:shrink-0" @click="copyCode">
                  <Icon name="copy" size="xs" />
                  <span>{{ t('affiliate.copyCode') }}</span>
                </button>
              </div>
            </div>

            <div class="space-y-2">
              <p class="text-xs font-medium text-muted-foreground">{{ t('affiliate.inviteLink') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-xl border border-border bg-muted/40 px-3 py-2 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm text-muted-foreground sm:flex-1 sm:truncate">{{ inviteLink }}</code>
                <button type="button" class="btn btn-secondary btn-sm h-8 w-full sm:w-auto sm:shrink-0" @click="copyInviteLink">
                  <Icon name="copy" size="xs" />
                  <span>{{ t('affiliate.copyLink') }}</span>
                </button>
              </div>
            </div>
          </div>

          <div class="mt-5 rounded-xl border border-border/80 bg-muted/40 p-4">
            <p class="text-sm font-semibold text-foreground">{{ t('affiliate.tips.title') }}</p>
            <ul class="mt-2 space-y-1 text-sm text-muted-foreground">
              <li>1. {{ t('affiliate.tips.line1') }}</li>
              <li>2. {{ t('affiliate.tips.line2', { rate: `${formattedRebateRate}%` }) }}</li>
              <li>3. {{ t('affiliate.tips.line3') }}</li>
              <li v-if="detail.aff_frozen_quota > 0">4. {{ t('affiliate.tips.line4') }}</li>
            </ul>
          </div>
        </div>

        <!-- Transfer Quota Card -->
        <div class="rounded-2xl border border-border/80 bg-card p-6 shadow-sm">
          <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div class="flex items-start gap-3.5">
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-emerald-500 text-white shadow-sm">
                <Icon name="swap" size="sm" :stroke-width="2" />
              </div>
              <div>
                <h3 class="text-base font-semibold tracking-tight text-foreground">{{ t('affiliate.transfer.title') }}</h3>
                <p class="mt-1 text-sm text-muted-foreground">{{ t('affiliate.transfer.description') }}</p>
              </div>
            </div>
            <button
              type="button"
              class="btn btn-primary h-9 shrink-0 px-4 text-sm"
              :disabled="transferring || detail.aff_quota <= 0"
              @click="transferQuota"
            >
              <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="dollar" size="sm" />
              <span>{{ transferring ? t('affiliate.transfer.transferring') : t('affiliate.transfer.button') }}</span>
            </button>
          </div>
          <p v-if="detail.aff_quota <= 0" class="mt-3 text-xs font-medium text-amber-600 dark:text-amber-400">
            {{ t('affiliate.transfer.empty') }}
          </p>
        </div>

        <!-- Invitees Table Card -->
        <div class="overflow-hidden rounded-2xl border border-border/80 bg-card shadow-sm">
          <div class="border-b border-border/80 px-6 py-4">
            <h3 class="text-base font-semibold tracking-tight text-foreground">{{ t('affiliate.invitees.title') }}</h3>
          </div>
          <div v-if="detail.invitees.length === 0" class="p-6">
            <div class="rounded-xl border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
              {{ t('affiliate.invitees.empty') }}
            </div>
          </div>
          <div v-else class="overflow-x-auto">
            <table class="w-full min-w-[560px] text-left text-sm">
              <thead>
                <tr class="border-b border-border bg-muted/40 text-xs font-medium text-muted-foreground">
                  <th class="px-6 py-3 font-medium">{{ t('affiliate.invitees.columns.email') }}</th>
                  <th class="px-4 py-3 font-medium">{{ t('affiliate.invitees.columns.username') }}</th>
                  <th class="px-4 py-3 text-right font-medium">{{ t('affiliate.invitees.columns.rebate') }}</th>
                  <th class="px-6 py-3 font-medium">{{ t('affiliate.invitees.columns.joinedAt') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border/60">
                <tr
                  v-for="item in detail.invitees"
                  :key="item.user_id"
                  class="transition-colors hover:bg-muted/30"
                >
                  <td class="px-6 py-3.5 font-medium text-foreground">{{ item.email || '-' }}</td>
                  <td class="px-4 py-3.5 text-muted-foreground">{{ item.username || '-' }}</td>
                  <td class="px-4 py-3.5 text-right font-semibold tabular-nums text-emerald-600 dark:text-emerald-400">{{ formatCurrency(item.total_rebate) }}</td>
                  <td class="px-6 py-3.5 tabular-nums text-muted-foreground">{{ formatDateTime(item.created_at) || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
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
import { Skeleton } from '@/components/ui/skeleton'
import userAPI from '@/api/user'
import type { UserAffiliateDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const loading = ref(true)
const transferring = ref(false)
const detail = ref<UserAffiliateDetail | null>(null)

const inviteLink = computed(() => {
  if (!detail.value) return ''
  if (typeof window === 'undefined') return `/register?aff=${encodeURIComponent(detail.value.aff_code)}`
  return `${window.location.origin}/register?aff=${encodeURIComponent(detail.value.aff_code)}`
})

// Rebate rate is a percentage in the range [0, 100]; backend already clamps it.
// We trim trailing zeros (e.g. 20.00 → "20", 12.50 → "12.5") for a cleaner UI.
const formattedRebateRate = computed(() => {
  const v = detail.value?.effective_rebate_rate_percent ?? 0
  const rounded = Math.round(v * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : rounded.toString()
})

function formatCount(value: number): string {
  return value.toLocaleString()
}

async function loadAffiliateDetail(silent = false): Promise<void> {
  if (!silent) {
    loading.value = true
  }
  try {
    detail.value = await userAPI.getAffiliateDetail()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.loadFailed')))
  } finally {
    if (!silent) {
      loading.value = false
    }
  }
}

async function copyCode(): Promise<void> {
  if (!detail.value?.aff_code) return
  await copyToClipboard(detail.value.aff_code, t('affiliate.codeCopied'))
}

async function copyInviteLink(): Promise<void> {
  if (!inviteLink.value) return
  await copyToClipboard(inviteLink.value, t('affiliate.linkCopied'))
}

async function transferQuota(): Promise<void> {
  if (!detail.value || detail.value.aff_quota <= 0 || transferring.value) return
  transferring.value = true
  try {
    const resp = await userAPI.transferAffiliateQuota()
    appStore.showSuccess(t('affiliate.transfer.success', { amount: formatCurrency(resp.transferred_quota) }))
    await Promise.all([
      loadAffiliateDetail(true),
      authStore.refreshUser().catch(() => undefined),
    ])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.transferFailed')))
  } finally {
    transferring.value = false
  }
}

onMounted(() => {
  void loadAffiliateDetail()
})
</script>
