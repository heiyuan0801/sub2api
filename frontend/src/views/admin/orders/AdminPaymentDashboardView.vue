<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Page Header with Day Switcher -->
      <PageHeader
        :title="t('payment.admin.dashboardTitle')"
        :description="t('payment.admin.dashboardDesc')"
      >
        <template #actions>
          <div
            class="inline-flex h-8 items-center rounded-full border border-border bg-background p-0.5"
            role="group"
          >
            <button
              v-for="d in DAYS_OPTIONS"
              :key="d"
              type="button"
              class="inline-flex h-7 items-center rounded-full px-3 text-xs font-medium tabular-nums transition-colors"
              :class="days === d
                ? 'bg-foreground text-background shadow-sm'
                : 'text-muted-foreground hover:text-foreground'"
              @click="days = d"
            >
              {{ d }}{{ t('payment.admin.daySuffix') }}
            </button>
          </div>
          <button
            type="button"
            class="inline-flex h-8 items-center gap-1.5 rounded-full border border-border bg-card px-3 text-xs font-medium text-foreground transition-colors hover:bg-muted/60 disabled:pointer-events-none disabled:opacity-50"
            :disabled="loading"
            :title="t('common.refresh')"
            @click="loadDashboard"
          >
            <Icon name="refresh" size="xs" :stroke-width="1.75" :class="loading ? 'animate-spin' : ''" />
            <span>{{ t('common.refresh') }}</span>
          </button>
        </template>
      </PageHeader>

      <!-- Skeleton Loading State -->
      <div v-if="loading" class="space-y-6">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div
            v-for="i in 4"
            :key="`payment-kpi-sk-${i}`"
            class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5 shadow-sm"
          >
            <div class="flex items-center justify-between gap-2">
              <Skeleton class="h-3.5 w-24" />
              <Skeleton class="size-9 rounded-full" />
            </div>
            <Skeleton class="mt-3 h-7 w-28" />
            <Skeleton class="mt-3 h-3.5 w-20" />
          </div>
        </div>

        <div class="rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
          <Skeleton class="mb-4 h-4 w-36" />
          <Skeleton class="h-64 w-full rounded-xl" />
        </div>

        <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div
            v-for="i in 2"
            :key="`payment-card-sk-${i}`"
            class="rounded-2xl border border-border/80 bg-card p-5 shadow-sm"
          >
            <Skeleton class="mb-4 h-4 w-36" />
            <div class="space-y-3">
              <Skeleton v-for="row in 4" :key="`payment-row-sk-${i}-${row}`" class="h-10 w-full rounded-xl" />
            </div>
          </div>
        </div>
      </div>

      <!-- Dashboard Content -->
      <template v-else-if="stats">
        <!-- KPI Cards -->
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <!-- Today Revenue -->
          <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
            <div class="flex items-center justify-between gap-2">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('payment.admin.todayRevenue') }}
              </p>
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-pink-500 text-white shadow-sm">
                <Icon name="dollar" size="sm" :stroke-width="2" />
              </div>
            </div>
            <div class="mt-3 space-y-0.5">
              <p
                v-for="[currency, amount] in sortedAmounts(stats.today_amount)"
                :key="currency"
                class="truncate text-2xl font-bold tracking-tight tabular-nums text-foreground"
              >
                {{ formatMoney(currency, amount) }}
              </p>
              <p
                v-if="sortedAmounts(stats.today_amount).length === 0"
                class="truncate text-2xl font-bold tracking-tight tabular-nums text-foreground"
              >
                0
              </p>
            </div>
            <p class="mt-3 truncate text-xs tabular-nums text-muted-foreground">
              <span class="font-medium text-foreground/80">{{ stats.today_count }}</span>
              {{ t('payment.admin.orders') }}
            </p>
          </div>

          <!-- Total Revenue -->
          <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
            <div class="flex items-center justify-between gap-2">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('payment.admin.totalRevenue') }}
              </p>
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-emerald-500 text-white shadow-sm">
                <Icon name="creditCard" size="sm" :stroke-width="2" />
              </div>
            </div>
            <div class="mt-3 space-y-0.5">
              <p
                v-for="[currency, amount] in sortedAmounts(stats.total_amount)"
                :key="currency"
                class="truncate text-2xl font-bold tracking-tight tabular-nums text-foreground"
              >
                {{ formatMoney(currency, amount) }}
              </p>
              <p
                v-if="sortedAmounts(stats.total_amount).length === 0"
                class="truncate text-2xl font-bold tracking-tight tabular-nums text-foreground"
              >
                0
              </p>
            </div>
            <p class="mt-3 truncate text-xs tabular-nums text-muted-foreground">
              <span class="font-medium text-foreground/80">{{ stats.total_count }}</span>
              {{ t('payment.admin.orders') }}
            </p>
          </div>

          <!-- Today Orders -->
          <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
            <div class="flex items-center justify-between gap-2">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('payment.admin.todayOrders') }}
              </p>
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-amber-500 text-white shadow-sm">
                <Icon name="chart" size="sm" :stroke-width="2" />
              </div>
            </div>
            <p class="mt-3 truncate text-2xl font-bold tracking-tight tabular-nums text-foreground">
              {{ stats.today_count }}
            </p>
            <p class="mt-3 truncate text-xs tabular-nums text-muted-foreground">
              {{ t('common.total') }}:
              <span class="font-medium text-foreground/80">{{ stats.total_count }}</span>
            </p>
          </div>

          <!-- Average Amount -->
          <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
            <div class="flex items-center justify-between gap-2">
              <p class="truncate text-xs font-medium text-muted-foreground">
                {{ t('payment.admin.avgAmount') }}
              </p>
              <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-violet-500 text-white shadow-sm">
                <Icon name="chart" size="sm" :stroke-width="2" />
              </div>
            </div>
            <div class="mt-3 space-y-0.5">
              <p
                v-for="[currency, amount] in sortedAmounts(stats.avg_amount)"
                :key="currency"
                class="truncate text-2xl font-bold tracking-tight tabular-nums text-foreground"
              >
                {{ formatMoney(currency, amount) }}
              </p>
              <p
                v-if="sortedAmounts(stats.avg_amount).length === 0"
                class="truncate text-2xl font-bold tracking-tight tabular-nums text-foreground"
              >
                0
              </p>
            </div>
            <p class="mt-3 truncate text-xs tabular-nums text-muted-foreground">
              {{ days }}{{ t('payment.admin.daySuffix') }}
            </p>
          </div>
        </div>

        <DailyRevenueChart :data="stats.daily_series || []" :loading="loading" />

        <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <!-- Payment Distribution -->
          <div class="rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
            <h3 class="mb-4 text-sm font-semibold tracking-tight text-foreground">
              {{ t('payment.admin.paymentDistribution') }}
            </h3>
            <div
              v-if="!stats.payment_methods?.length"
              class="flex h-32 items-center justify-center text-sm text-muted-foreground"
            >
              {{ t('payment.admin.noData') }}
            </div>
            <div v-else class="space-y-2">
              <div
                v-for="method in stats.payment_methods"
                :key="method.type"
                class="flex items-center justify-between gap-3 rounded-xl border border-border/60 bg-muted/20 px-3.5 py-2.5 transition-colors hover:bg-muted/40"
              >
                <div class="flex min-w-0 items-center gap-2.5">
                  <span :class="['inline-block size-2.5 shrink-0 rounded-full', methodColor(method.type)]"></span>
                  <span class="truncate text-sm font-medium text-foreground">
                    {{ t('payment.methods.' + method.type, method.type) }}
                  </span>
                </div>
                <div class="flex shrink-0 items-center gap-2.5 text-right">
                  <div class="space-y-0.5">
                    <span
                      v-for="[currency, amount] in sortedAmounts(method.amount)"
                      :key="currency"
                      class="block text-sm font-semibold tabular-nums text-foreground"
                    >
                      {{ formatMoney(currency, amount) }}
                    </span>
                  </div>
                  <span class="rounded-full border border-border/70 bg-background px-2 py-0.5 text-xs font-medium tabular-nums text-muted-foreground">
                    {{ method.count }}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <!-- Top Users -->
          <div class="rounded-2xl border border-border/80 bg-card p-5 shadow-sm">
            <h3 class="mb-4 text-sm font-semibold tracking-tight text-foreground">
              {{ t('payment.admin.topUsers') }}
            </h3>
            <div
              v-if="!hasTopUsers(stats.top_users)"
              class="flex h-32 items-center justify-center text-sm text-muted-foreground"
            >
              {{ t('payment.admin.noData') }}
            </div>
            <div v-else class="space-y-3">
              <div v-for="[currency, users] in sortedTopUsers(stats.top_users)" :key="currency" class="space-y-1.5">
                <p class="px-1 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  {{ currency }}
                </p>
                <div
                  v-for="(user, idx) in users"
                  :key="user.user_id"
                  class="flex items-center justify-between gap-3 rounded-xl px-3 py-2 transition-colors hover:bg-muted/50"
                >
                  <div class="flex min-w-0 items-center gap-3">
                    <span
                      :class="[
                        'flex size-6 shrink-0 items-center justify-center rounded-full text-xs font-bold tabular-nums',
                        rankClass(idx)
                      ]"
                    >
                      {{ idx + 1 }}
                    </span>
                    <span class="truncate text-sm text-foreground">{{ user.email }}</span>
                  </div>
                  <span class="shrink-0 text-sm font-semibold tabular-nums text-foreground">
                    {{ formatMoney(currency, user.amount) }}
                  </span>
                </div>
              </div>
            </div>
          </div>
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
import PageHeader from '@/components/common/PageHeader.vue'
import { Skeleton } from '@/components/ui/skeleton'
import Icon from '@/components/icons/Icon.vue'
import DailyRevenueChart from '@/components/admin/payment/DailyRevenueChart.vue'

const { t } = useI18n()
const appStore = useAppStore()

const DAYS_OPTIONS = [7, 30, 90] as const
const days = ref<number>(30)
const loading = ref(false)
const stats = ref<DashboardStats | null>(null)

function methodColor(type: string): string {
  const c: Record<string, string> = {
    alipay: 'bg-blue-500', wxpay: 'bg-green-500',
    alipay_direct: 'bg-blue-400', wxpay_direct: 'bg-green-400',
    stripe: 'bg-purple-500',
    bepusdt: 'bg-emerald-500',
  }
  return c[type] || 'bg-muted-foreground/50'
}

function rankClass(idx: number): string {
  if (idx === 0) return 'bg-amber-500/15 text-amber-700 dark:text-amber-400'
  if (idx === 1) return 'bg-muted text-foreground/80'
  if (idx === 2) return 'bg-orange-500/15 text-orange-700 dark:text-orange-400'
  return 'bg-muted/70 text-muted-foreground'
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
