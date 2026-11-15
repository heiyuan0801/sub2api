<template>
  <AppLayout>
    <div class="space-y-4">
      <PageHeader :title="t('affiliate.title')" :description="t('affiliate.description')" />
      <div v-if="loading" class="flex justify-center py-12">
        <div
          class="h-8 w-8 animate-spin rounded-full border-2 border-brand border-t-transparent"
        ></div>
      </div>

      <template v-else-if="detail">
        <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard :label="t('affiliate.stats.rebateRate')" :value="formattedRebateRate" unit="%" :description="t('affiliate.stats.rebateRateHint')" />
          <StatCard :label="t('affiliate.stats.invitedUsers')" :value="formatCount(detail.aff_count)" />
          <StatCard :label="t('affiliate.stats.availableQuota')">
            <template #value><span class="text-success">{{ formatCurrency(detail.aff_quota) }}</span></template>
          </StatCard>
          <StatCard :label="t('affiliate.stats.totalQuota')" :value="formatCurrency(detail.aff_history_quota)">
            <template v-if="detail.aff_frozen_quota > 0" #footer>
              <span class="text-warning">{{ t('affiliate.stats.frozenQuota') }}: {{ formatCurrency(detail.aff_frozen_quota) }}</span>
            </template>
          </StatCard>
        </div>
        <SectionCard>
          <div class="grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <p class="text-sm font-medium text-muted-foreground">{{ t('affiliate.yourCode') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-xl border border-border bg-muted/40 px-3 py-2 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm font-semibold text-foreground sm:flex-1 sm:truncate">{{ detail.aff_code }}</code>
                <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyCode">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyCode') }}</span>
                </button>
              </div>
            </div>

            <div class="space-y-2">
              <p class="text-sm font-medium text-muted-foreground">{{ t('affiliate.inviteLink') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-xl border border-border bg-muted/40 px-3 py-2 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm text-muted-foreground sm:flex-1 sm:truncate">{{ inviteLink }}</code>
                <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyInviteLink">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyLink') }}</span>
                </button>
              </div>
            </div>
          </div>

          <div class="mt-5 rounded-xl border border-brand/25 bg-brand/5 p-4">
            <p class="text-sm font-medium text-foreground">{{ t('affiliate.tips.title') }}</p>
            <ul class="mt-2 space-y-1 text-sm text-brand">
              <li>1. {{ t('affiliate.tips.line1') }}</li>
              <li>2. {{ t('affiliate.tips.line2', { rate: `${formattedRebateRate}%` }) }}</li>
              <li>3. {{ t('affiliate.tips.line3') }}</li>
              <li v-if="detail.aff_frozen_quota > 0">4. {{ t('affiliate.tips.line4') }}</li>
            </ul>
          </div>
        </SectionCard>

        <SectionCard>
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-foreground">{{ t('affiliate.transfer.title') }}</h3>
              <p class="mt-1 text-sm text-muted-foreground">{{ t('affiliate.transfer.description') }}</p>
            </div>
            <button
              class="btn btn-primary"
              :disabled="transferring || detail.aff_quota <= 0"
              @click="transferQuota"
            >
              <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="dollar" size="sm" />
              <span>{{ transferring ? t('affiliate.transfer.transferring') : t('affiliate.transfer.button') }}</span>
            </button>
          </div>
          <p v-if="detail.aff_quota <= 0" class="mt-3 text-sm text-warning">
            {{ t('affiliate.transfer.empty') }}
          </p>
        </SectionCard>

        <SectionCard :title="t('affiliate.invitees.title')">
          <DataTable :data="detail.invitees" :columns="inviteeColumns" row-key="user_id">
            <template #empty><EmptyState :title="t('affiliate.invitees.empty')" /></template>
          </DataTable>
        </SectionCard>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { EmptyState, PageHeader, SectionCard, StatCard } from '@/components/page'
import DataTable from '@/components/page/AdminDataTable.vue'
import type { Column } from '@/components/common/types'
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
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

const inviteeColumns = computed<Column[]>(() => [
  { key: 'email', label: t('affiliate.invitees.columns.email'), formatter: value => value || '-' },
  { key: 'username', label: t('affiliate.invitees.columns.username'), formatter: value => value || '-' },
  { key: 'total_rebate', label: t('affiliate.invitees.columns.rebate'), class: 'text-right font-medium tabular-nums text-success', formatter: value => formatCurrency(value) },
  { key: 'created_at', label: t('affiliate.invitees.columns.joinedAt'), formatter: value => formatDateTime(value) || '-' }
])

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
