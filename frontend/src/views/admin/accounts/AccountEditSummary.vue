<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { StatusBadge } from '@/components/page'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import type { Account, WindowStats } from '@/types'

const props = defineProps<{ account: Account; stats?: WindowStats | null }>()
const { t, locale } = useI18n()
const status = computed(() => props.account.status)
const statusTone = computed(() => status.value === 'active' ? 'success' : status.value === 'error' ? 'danger' : 'neutral')
const numberFormat = computed(() => new Intl.NumberFormat(locale.value))
const decimalFormat = computed(() => new Intl.NumberFormat(locale.value, { maximumFractionDigits: 1 }))
const dollarFormat = computed(() => new Intl.NumberFormat(locale.value, { style: 'currency', currency: 'USD' }))
const missing = '—'

const cache = computed(() => {
  const read = props.stats?.cache_read_tokens
  const input = props.stats?.cache_input_tokens
  if (typeof input !== 'number' || typeof read !== 'number' || !Number.isFinite(read) ||
      !Number.isFinite(input) || input <= 0 || read < 0 || read > input) return missing
  return decimalFormat.value.format(read / input * 100) + '%'
})
const first = computed(() => {
  const ms = props.stats?.average_first_token_ms
  return typeof ms === 'number' && Number.isFinite(ms) && ms > 0 ? decimalFormat.value.format(ms) + ' ms' : missing
})
const failed = computed(() => {
  const count = props.stats?.error_stats?.failed_requests
  return typeof count === 'number' && Number.isFinite(count) && count >= 0 ? numberFormat.value.format(count) : missing
})
const balance = computed(() => {
  const probe = props.account.extra?.upstream_billing_probe?.balance
  return probe?.status === 'ok' && probe.unit === 'USD' && typeof probe.amount === 'number' && Number.isFinite(probe.amount)
    ? dollarFormat.value.format(probe.amount)
    : missing
})
</script>

<template>
  <div class="shrink-0 px-4 py-3">
    <div class="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
      <span class="font-mono text-xs text-muted-foreground">#{{ account.id }}</span>
      <span class="min-w-0 break-all text-sm font-medium">{{ account.name }}</span>
      <PlatformTypeBadge :platform="account.platform" :type="account.type" />
      <StatusBadge :tone="statusTone" :label="t(`admin.accounts.status.${status}`)" />
    </div>
    <dl class="mt-3 grid grid-cols-2 overflow-hidden rounded-lg border border-border text-xs sm:grid-cols-4">
      <div class="min-w-0 border-b border-r border-border p-2.5 sm:border-b-0">
        <dt class="text-muted-foreground">{{ t('design.accountForm.cacheHitRate') }}</dt>
        <dd class="mt-1.5 font-mono text-sm">{{ cache }}</dd>
      </div>
      <div class="min-w-0 border-b border-border p-2.5 sm:border-b-0 sm:border-r">
        <dt class="text-muted-foreground">{{ t('design.accountForm.averageFirstToken') }}</dt>
        <dd class="mt-1.5 font-mono text-sm">{{ first }}</dd>
      </div>
      <div class="min-w-0 border-r border-border p-2.5">
        <dt class="text-muted-foreground">{{ t('design.accountForm.todayFailed') }}</dt>
        <dd class="mt-1.5 font-mono text-sm">{{ failed }}</dd>
      </div>
      <div class="min-w-0 p-2.5">
        <dt class="text-muted-foreground">{{ t('design.accountForm.lastUpstreamBalance') }}</dt>
        <dd class="mt-1.5 break-all font-mono text-sm">{{ balance }}</dd>
      </div>
    </dl>
  </div>
</template>
