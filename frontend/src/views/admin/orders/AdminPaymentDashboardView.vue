<template>
  <AppLayout>
    <div class="space-y-4">
      <PageHeader as="h2" :title="t('nav.paymentDashboard')">
        <template #actions>
          <div class="flex rounded-lg border border-border bg-muted/40 p-0.5">
            <button
              v-for="d in DAYS_OPTIONS"
              :key="d"
              type="button"
              class="rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
              :aria-pressed="days === d"
              :class="days === d ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'"
              @click="days = d"
            >
              {{ d }}{{ t('payment.admin.daySuffix') }}
            </button>
          </div>
          <button type="button" @click="loadDashboard" :disabled="loading" class="btn btn-secondary" :aria-label="t('common.refresh')" :title="t('common.refresh')">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
        </template>
      </PageHeader>

      <!-- Dashboard Content -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>
      <template v-else-if="stats">
        <OrderStatsCards :stats="stats" />
        <DailyRevenueChart :data="stats.daily_series || []" :loading="loading" />
        <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <SectionCard :title="t('payment.admin.paymentDistribution')">
            <EmptyState v-if="!stats.payment_methods?.length" :title="t('payment.admin.noData')" />
            <div v-else class="space-y-3">
              <div v-for="method in stats.payment_methods" :key="method.type" class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <span :class="['inline-block h-3 w-3 rounded-full', methodColor(method.type)]"></span>
                  <span class="text-sm text-foreground">{{ t('payment.methods.' + method.type, method.type) }}</span>
                </div>
                <div class="space-y-1 text-right">
                  <span v-for="[currency, amount] in sortedAmounts(method.amount)" :key="currency" class="block text-sm font-medium text-foreground">{{ formatMoney(currency, amount) }}</span>
                  <span class="ml-2 text-xs text-muted-foreground">({{ method.count }})</span>
                </div>
              </div>
            </div>
          </SectionCard>
          <SectionCard :title="t('payment.admin.topUsers')">
            <EmptyState v-if="!hasTopUsers(stats.top_users)" :title="t('payment.admin.noData')" />
            <div v-else class="space-y-2">
              <div v-for="[currency, users] in sortedTopUsers(stats.top_users)" :key="currency" class="space-y-2">
                <p class="text-xs font-semibold text-muted-foreground">{{ currency }}</p>
                <div v-for="(user, idx) in users" :key="user.user_id" class="flex items-center justify-between rounded-lg px-3 py-2 hover:bg-muted">
                  <div class="flex items-center gap-3">
                    <span :class="['flex h-6 w-6 items-center justify-center rounded-full text-xs font-bold', rankClass(idx)]">{{ idx + 1 }}</span>
                    <span class="text-sm text-foreground">{{ user.email }}</span>
                  </div>
                  <span class="text-sm font-medium text-foreground">{{ formatMoney(currency, user.amount) }}</span>
                </div>
              </div>
            </div>
          </SectionCard>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminPaymentAPI } from '@/api/admin/payment'
import { extractI18nErrorMessage } from '@/utils/apiError'
import type { CurrencyAmounts, DashboardStats, TopUserPaymentStats } from '@/types/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import { EmptyState, PageHeader, SectionCard } from '@/components/page'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import OrderStatsCards from '@/components/admin/payment/OrderStatsCards.vue'
import DailyRevenueChart from '@/components/admin/payment/DailyRevenueChart.vue'

const { t } = useI18n()
const appStore = useAppStore()

const DAYS_OPTIONS = [7, 30, 90] as const
const days = ref<number>(30)
const loading = ref(false)
const stats = ref<DashboardStats | null>(null)

function methodColor(type: string): string {
  const c: Record<string, string> = {
    alipay: 'bg-brand', wxpay: 'bg-success',
    alipay_direct: 'bg-brand/65', wxpay_direct: 'bg-success/65',
    stripe: 'bg-info',
    bepusdt: 'bg-warning',
  }
  return c[type] || 'bg-muted-foreground'
}

function rankClass(idx: number): string {
  if (idx === 0) return 'bg-warning/15 text-warning'
  if (idx === 1) return 'bg-info/15 text-info'
  if (idx === 2) return 'bg-brand/10 text-brand'
  return 'bg-muted text-muted-foreground'
}

function sortedAmounts(amounts: CurrencyAmounts): [string, number][] {
  return Object.entries(amounts).sort(([left], [right]) => left.localeCompare(right))
}

function sortedTopUsers(usersByCurrency: Record<string, TopUserPaymentStats[]>): [string, TopUserPaymentStats[]][] {
  return Object.entries(usersByCurrency).sort(([left], [right]) => left.localeCompare(right))
}

function hasTopUsers(usersByCurrency: Record<string, TopUserPaymentStats[]>): boolean {
  return Object.values(usersByCurrency).some(users => users.length > 0)
}

function formatMoney(currency: string, amount: number): string {
  return new Intl.NumberFormat(undefined, { style: 'currency', currency }).format(amount)
}

async function loadDashboard() {
  loading.value = true
  try {
    const res = await adminPaymentAPI.getDashboard(days.value)
    stats.value = res.data
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

watch(days, () => loadDashboard())
onMounted(() => loadDashboard())
</script>
