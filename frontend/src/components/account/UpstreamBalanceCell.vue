<template>
  <div v-if="account.type === 'apikey'" class="mt-1 space-y-0.5 text-xs">
    <div class="flex items-center gap-1">
      <span>{{ t(`admin.accounts.upstreamBalance.${balance?.kind || 'wallet'}`) }}</span>
      <span class="font-mono" :class="valid && !stale ? 'text-emerald-600 dark:text-emerald-400' : 'text-gray-400'" data-testid="upstream-balance-value">{{ valid ? `$${balance!.amount!.toFixed(2)}` : '—' }}</span>
      <button type="button" :disabled="loading" class="rounded-full border border-gray-300 px-2 py-0.5 disabled:opacity-50 dark:border-dark-500" :aria-label="t('admin.accounts.upstreamBalance.query')" @click.stop="query">{{ t(loading ? 'common.loading' : 'admin.accounts.upstreamBalance.query') }}</button>
    </div>
    <div v-if="balance" class="text-gray-400">
      <span v-if="balance.status !== 'ok'">{{ t(balance.status === 'unsupported' ? 'admin.accounts.upstreamBalance.unsupported' : 'admin.accounts.upstreamBalance.failed') }} · </span>
      <span v-else-if="stale">{{ t('admin.accounts.upstreamBalance.stale') }} · </span>
      {{ t('admin.accounts.upstreamBalance.queriedAt', { time: queriedAt }) }}
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { probeUpstreamBalance } from '@/api/admin/accounts'
import { useAppStore } from '@/stores/app'
import type { Account, UpstreamBillingProbeSnapshot } from '@/types'
const props = defineProps<{ account: Account; now: number }>()
const emit = defineEmits<{ updated: [snapshot: UpstreamBillingProbeSnapshot] }>()
const { t } = useI18n()
const app = useAppStore()
const loading = ref(false)
const balance = computed(() => props.account.extra?.upstream_billing_probe?.balance)
const valid = computed(() => balance.value?.status === 'ok' && balance.value.unit === 'USD' && typeof balance.value.amount === 'number' && Number.isFinite(balance.value.amount))
const timestamp = computed(() => Date.parse(balance.value?.queried_at ?? ''))
const stale = computed(() => !Number.isFinite(timestamp.value) || props.now - timestamp.value > 30 * 60 * 1000 || timestamp.value > props.now + 300000)
const queriedAt = computed(() => Number.isFinite(timestamp.value) ? new Date(timestamp.value).toLocaleString() : '—')
async function query() {
  if (loading.value) return
  loading.value = true
  try {
    const result = await probeUpstreamBalance(props.account.id)
    if (result.snapshot) emit('updated', result.snapshot)
  } catch {
    app.showError(t('admin.accounts.upstreamBalance.failed'))
  } finally { loading.value = false }
}
</script>
