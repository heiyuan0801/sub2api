<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Greeting Header -->
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="min-w-0 space-y-1.5">
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-xl leading-none" aria-hidden="true">👋</span>
            <h1 class="truncate text-xl font-semibold tracking-tight text-foreground">
              {{ user?.username ? `${t('auth.welcomeBack')}, ${user.username}` : t('dashboard.title') }}
            </h1>
            <span
              v-if="user?.id != null"
              class="inline-flex items-center rounded-md border border-border bg-muted/50 px-2 py-0.5 text-xs text-muted-foreground tabular-nums"
            >
              ID: {{ user.id }}
            </span>
            <span
              v-if="user?.email"
              class="inline-flex max-w-[240px] items-center truncate rounded-md border border-border bg-muted/50 px-2 py-0.5 text-xs text-muted-foreground tabular-nums"
              :title="user.email"
            >
              {{ user.email }}
            </span>
          </div>
          <p class="text-sm text-muted-foreground">
            {{ t('dashboard.welcomeMessage') }}
          </p>
        </div>
        <div class="flex shrink-0 flex-wrap items-center gap-2">
          <router-link
            to="/keys"
            class="inline-flex h-8 items-center gap-1.5 rounded-full border border-border bg-card px-3 text-xs font-medium text-foreground transition-colors hover:bg-muted/60"
          >
            <Icon name="key" size="xs" :stroke-width="1.75" />
            <span>{{ t('dashboard.apiKeys') }}</span>
          </router-link>
          <button
            type="button"
            class="inline-flex h-8 items-center gap-1.5 rounded-full border border-border bg-card px-3 text-xs font-medium text-foreground transition-colors hover:bg-muted/60 disabled:pointer-events-none disabled:opacity-50"
            :disabled="loading || loadingCharts"
            @click="refreshAll"
          >
            <Icon name="refresh" size="xs" :stroke-width="1.75" :class="loading || loadingCharts ? 'animate-spin' : ''" />
            <span>{{ t('common.refresh') }}</span>
          </button>
        </div>
      </div>

      <div v-if="loading && !stats" class="space-y-6">
        <!-- Row 1 KPI Skeletons -->
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div
            v-for="i in 4"
            :key="`kpi-r1-${i}`"
            class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5"
          >
            <div class="flex items-center justify-between gap-2">
              <Skeleton class="h-3.5 w-20" />
              <Skeleton class="size-9 rounded-full" />
            </div>
            <Skeleton class="mt-3 h-7 w-28" />
            <div class="mt-3 flex items-center justify-between gap-2">
              <Skeleton class="h-3.5 w-24" />
              <Skeleton class="h-5 w-14 rounded-full" />
            </div>
          </div>
        </div>
        <!-- Row 2 Compact Horizontal Skeletons -->
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div
            v-for="i in 4"
            :key="`kpi-r2-${i}`"
            class="flex items-center gap-3.5 rounded-2xl border border-border/80 bg-card p-4"
          >
            <Skeleton class="size-10 shrink-0 rounded-full" />
            <div class="min-w-0 flex-1 space-y-1.5">
              <Skeleton class="h-3 w-20" />
              <Skeleton class="h-5 w-24" />
              <Skeleton class="h-3 w-32" />
            </div>
          </div>
        </div>
        <!-- Charts Skeleton -->
        <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div v-for="i in 2" :key="`chart-sk-${i}`" class="rounded-2xl border border-border/80 bg-card p-5">
            <Skeleton class="mb-4 h-4 w-36" />
            <Skeleton class="h-48 w-full rounded-xl" />
          </div>
        </div>
      </div>

      <template v-else-if="stats">
        <UserDashboardStats :stats="stats" :balance="user?.balance || 0" :is-simple="authStore.isSimpleMode" :platform-quotas="platformQuotas" />
        <UserDashboardCharts v-model:startDate="startDate" v-model:endDate="endDate" v-model:granularity="granularity" :loading="loadingCharts" :trend="trendData" :models="modelStats" @dateRangeChange="loadCharts" @granularityChange="loadCharts" @refresh="refreshAll" />
        <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div class="lg:col-span-2"><UserDashboardRecentUsage :data="recentUsage" :loading="loadingUsage" /></div>
          <div class="lg:col-span-1"><UserDashboardQuickActions /></div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { Skeleton } from '@/components/ui/skeleton'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardRecentUsage from '@/components/user/dashboard/UserDashboardRecentUsage.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import type { UsageLog, TrendDataPoint, ModelStat, PlatformQuotaItem } from '@/types'
import { getMyPlatformQuotas } from '@/api/user'
import { formatDateLocalInput } from '@/utils/format'

const { t } = useI18n()
const authStore = useAuthStore()
const user = computed(() => authStore.user)
const stats = ref<UserStatsType | null>(null)
const loading = ref(false)
const loadingUsage = ref(false)
const loadingCharts = ref(false)
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const recentUsage = ref<UsageLog[]>([])
const platformQuotas = ref<PlatformQuotaItem[] | null>(null)

const startDate = ref(formatDateLocalInput(new Date(Date.now() - 6 * 86400000)))
const endDate = ref(formatDateLocalInput(new Date()))
const granularity = ref('day')

const loadStats = async () => {
  loading.value = true
  try {
    await authStore.refreshUser()
    stats.value = await usageAPI.getDashboardStats()
  } catch (error) {
    console.error('Failed to load dashboard stats:', error)
  } finally {
    loading.value = false
  }
}

const loadCharts = async () => {
  loadingCharts.value = true
  try {
    const res = await Promise.all([
      usageAPI.getDashboardTrend({ start_date: startDate.value, end_date: endDate.value, granularity: granularity.value as any }),
      usageAPI.getDashboardModels({ start_date: startDate.value, end_date: endDate.value }),
    ])
    trendData.value = res[0].trend || []
    modelStats.value = res[1].models || []
  } catch (error) {
    console.error('Failed to load charts:', error)
  } finally {
    loadingCharts.value = false
  }
}

const loadRecent = async () => {
  loadingUsage.value = true
  try {
    const res = await usageAPI.getByDateRange(startDate.value, endDate.value)
    recentUsage.value = res.items.slice(0, 5)
  } catch (error) {
    console.error('Failed to load recent usage:', error)
  } finally {
    loadingUsage.value = false
  }
}

const loadPlatformQuotas = async () => {
  try {
    const data = await getMyPlatformQuotas()
    platformQuotas.value = data.platform_quotas ?? []
  } catch (error) {
    console.warn('Failed to load platform quotas:', error)
    platformQuotas.value = []
  }
}

const refreshAll = () => {
  loadStats()
  loadCharts()
  loadRecent()
  loadPlatformQuotas()
}

onMounted(() => {
  refreshAll()
})
</script>
