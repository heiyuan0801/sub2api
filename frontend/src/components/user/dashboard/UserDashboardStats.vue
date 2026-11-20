<template>
  <!-- Row 1: Core Stats -->
  <div :class="['grid grid-cols-1 gap-4 sm:grid-cols-2', isSimple ? 'lg:grid-cols-3' : 'lg:grid-cols-4']">
    <!-- Balance -->
    <div v-if="!isSimple" class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5">
      <div class="flex items-center justify-between gap-2">
        <p class="truncate text-xs font-medium text-muted-foreground">{{ t('dashboard.balance') }}</p>
        <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-pink-500 text-white shadow-sm">
          <svg class="size-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z" />
          </svg>
        </div>
      </div>
      <p class="mt-3 truncate text-2xl font-bold tracking-tight tabular-nums text-foreground">
        ${{ formatBalance(balance) }}
      </p>
      <div class="mt-3 flex items-center justify-between gap-2">
        <p class="truncate text-xs text-muted-foreground">{{ t('common.available') }}</p>
        <button
          type="button"
          class="inline-flex shrink-0 items-center gap-1 rounded-full border border-border bg-muted/40 px-2.5 py-0.5 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          @click="navigateTo('/purchase')"
        >
          <span>{{ t('nav.recharge') }}</span>
          <span aria-hidden="true">→</span>
        </button>
      </div>
    </div>

    <!-- API Keys -->
    <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5">
      <div class="flex items-center justify-between gap-2">
        <p class="truncate text-xs font-medium text-muted-foreground">{{ t('dashboard.apiKeys') }}</p>
        <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-emerald-500 text-white shadow-sm">
          <Icon name="key" size="sm" :stroke-width="2" />
        </div>
      </div>
      <p class="mt-3 truncate text-2xl font-bold tracking-tight tabular-nums text-foreground">
        {{ stats?.total_api_keys || 0 }}
      </p>
      <div class="mt-3 flex items-center justify-between gap-2">
        <p class="flex min-w-0 items-center gap-1.5 truncate text-xs tabular-nums text-emerald-600 dark:text-emerald-400">
          <span class="size-1.5 shrink-0 rounded-full bg-emerald-500" />
          <span class="truncate">{{ stats?.active_api_keys || 0 }} {{ t('common.active') }}</span>
        </p>
        <button
          type="button"
          class="inline-flex shrink-0 items-center gap-1 rounded-full border border-border bg-muted/40 px-2.5 py-0.5 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          @click="navigateTo('/keys')"
        >
          <span>{{ t('common.view') }}</span>
          <span aria-hidden="true">→</span>
        </button>
      </div>
    </div>

    <!-- Today Requests -->
    <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5">
      <div class="flex items-center justify-between gap-2">
        <p class="truncate text-xs font-medium text-muted-foreground">{{ t('dashboard.todayRequests') }}</p>
        <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-amber-500 text-white shadow-sm">
          <Icon name="chart" size="sm" :stroke-width="2" />
        </div>
      </div>
      <p class="mt-3 truncate text-2xl font-bold tracking-tight tabular-nums text-foreground">
        {{ formatNumber(stats?.today_requests || 0) }}
      </p>
      <div class="mt-3 flex items-center justify-between gap-2">
        <p class="truncate text-xs tabular-nums text-muted-foreground">
          {{ t('common.total') }}: {{ formatNumber(stats?.total_requests || 0) }}
        </p>
        <button
          type="button"
          class="inline-flex shrink-0 items-center gap-1 rounded-full border border-border bg-muted/40 px-2.5 py-0.5 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          @click="navigateTo('/usage')"
        >
          <span>{{ t('common.view') }}</span>
          <span aria-hidden="true">→</span>
        </button>
      </div>
    </div>

    <!-- Today Cost -->
    <div class="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-5">
      <div class="flex items-center justify-between gap-2">
        <p class="truncate text-xs font-medium text-muted-foreground">{{ t('dashboard.todayCost') }}</p>
        <div class="flex size-9 shrink-0 items-center justify-center rounded-full bg-violet-500 text-white shadow-sm">
          <Icon name="dollar" size="sm" :stroke-width="2" />
        </div>
      </div>
      <p class="mt-3 truncate text-2xl font-bold tracking-tight tabular-nums text-foreground">
        <span :title="t('dashboard.actual')">${{ formatCost(stats?.today_actual_cost || 0) }}</span>
        <span class="text-sm font-normal text-muted-foreground" :title="t('dashboard.standard')"> / ${{ formatCost(stats?.today_cost || 0) }}</span>
      </p>
      <div class="mt-3 flex items-center justify-between gap-2">
        <p class="truncate text-xs tabular-nums text-muted-foreground">
          <span>{{ t('common.total') }}: </span>
          <span class="font-medium text-foreground" :title="t('dashboard.actual')">${{ formatCost(stats?.total_actual_cost || 0) }}</span>
          <span :title="t('dashboard.standard')"> / ${{ formatCost(stats?.total_cost || 0) }}</span>
        </p>
        <button
          type="button"
          class="inline-flex shrink-0 items-center gap-1 rounded-full border border-border bg-muted/40 px-2.5 py-0.5 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          @click="navigateTo('/usage')"
        >
          <span>{{ t('common.view') }}</span>
          <span aria-hidden="true">→</span>
        </button>
      </div>
    </div>
  </div>

  <!-- Row 2: Compact Horizontal Stat Cards -->
  <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
    <!-- Today Tokens -->
    <div class="flex items-center gap-3.5 rounded-2xl border border-border/80 bg-card p-4">
      <div class="flex size-10 shrink-0 items-center justify-center rounded-full bg-pink-500 text-white shadow-sm">
        <Icon name="cube" size="sm" :stroke-width="2" />
      </div>
      <div class="min-w-0 flex-1">
        <p class="truncate text-xs font-medium text-muted-foreground">{{ t('dashboard.todayTokens') }}</p>
        <p class="mt-0.5 truncate text-lg font-bold tracking-tight tabular-nums text-foreground">
          {{ formatTokens(stats?.today_tokens || 0) }}
        </p>
        <p class="mt-0.5 truncate text-xs tabular-nums text-muted-foreground" :title="`${t('dashboard.input')}: ${formatTokens(stats?.today_input_tokens || 0)} / ${t('dashboard.output')}: ${formatTokens(stats?.today_output_tokens || 0)} / ${t('dashboard.cache')}: ${formatTokens((stats?.today_cache_creation_tokens || 0) + (stats?.today_cache_read_tokens || 0))}`">
          {{ t('dashboard.input') }}: {{ formatTokens(stats?.today_input_tokens || 0) }} / {{ t('dashboard.output') }}: {{ formatTokens(stats?.today_output_tokens || 0) }} / {{ t('dashboard.cache') }}: {{ formatTokens((stats?.today_cache_creation_tokens || 0) + (stats?.today_cache_read_tokens || 0)) }}
        </p>
      </div>
    </div>

    <!-- Total Tokens -->
    <div class="flex items-center gap-3.5 rounded-2xl border border-border/80 bg-card p-4">
      <div class="flex size-10 shrink-0 items-center justify-center rounded-full bg-emerald-500 text-white shadow-sm">
        <Icon name="database" size="sm" :stroke-width="2" />
      </div>
      <div class="min-w-0 flex-1">
        <p class="truncate text-xs font-medium text-muted-foreground">{{ t('dashboard.totalTokens') }}</p>
        <p class="mt-0.5 truncate text-lg font-bold tracking-tight tabular-nums text-foreground">
          {{ formatTokens(stats?.total_tokens || 0) }}
        </p>
        <p class="mt-0.5 truncate text-xs tabular-nums text-muted-foreground" :title="`${t('dashboard.input')}: ${formatTokens(stats?.total_input_tokens || 0)} / ${t('dashboard.output')}: ${formatTokens(stats?.total_output_tokens || 0)} / ${t('dashboard.cache')}: ${formatTokens((stats?.total_cache_creation_tokens || 0) + (stats?.total_cache_read_tokens || 0))}`">
          {{ t('dashboard.input') }}: {{ formatTokens(stats?.total_input_tokens || 0) }} / {{ t('dashboard.output') }}: {{ formatTokens(stats?.total_output_tokens || 0) }} / {{ t('dashboard.cache') }}: {{ formatTokens((stats?.total_cache_creation_tokens || 0) + (stats?.total_cache_read_tokens || 0)) }}
        </p>
      </div>
    </div>

    <!-- Performance (RPM/TPM) -->
    <div class="flex items-center gap-3.5 rounded-2xl border border-border/80 bg-card p-4">
      <div class="flex size-10 shrink-0 items-center justify-center rounded-full bg-amber-500 text-white shadow-sm">
        <Icon name="bolt" size="sm" :stroke-width="2" />
      </div>
      <div class="min-w-0 flex-1">
        <p class="truncate text-xs font-medium text-muted-foreground">{{ t('dashboard.performance') }}</p>
        <div class="mt-0.5 flex items-baseline gap-1.5">
          <p class="truncate text-lg font-bold tracking-tight tabular-nums text-foreground">{{ formatTokens(stats?.rpm || 0) }}</p>
          <span class="text-xs font-medium text-muted-foreground">RPM</span>
        </div>
        <p class="mt-0.5 flex items-baseline gap-1.5 truncate text-xs tabular-nums text-muted-foreground">
          <span class="font-medium text-foreground">{{ formatTokens(stats?.tpm || 0) }}</span>
          <span>TPM</span>
        </p>
      </div>
    </div>

    <!-- Avg Response Time -->
    <div class="flex items-center gap-3.5 rounded-2xl border border-border/80 bg-card p-4">
      <div class="flex size-10 shrink-0 items-center justify-center rounded-full bg-sky-500 text-white shadow-sm">
        <Icon name="clock" size="sm" :stroke-width="2" />
      </div>
      <div class="min-w-0 flex-1">
        <p class="truncate text-xs font-medium text-muted-foreground">{{ t('dashboard.avgResponse') }}</p>
        <p class="mt-0.5 truncate text-lg font-bold tracking-tight tabular-nums text-foreground">
          {{ formatDuration(stats?.average_duration_ms || 0) }}
        </p>
        <p class="mt-0.5 truncate text-xs text-muted-foreground">{{ t('dashboard.averageTime') }}</p>
      </div>
    </div>
  </div>

  <!-- Row 3: Per-platform breakdown -->
  <div v-if="!isSimple && platformCards.length > 0" class="rounded-2xl border border-border/80 bg-card p-5">
    <div class="mb-3.5 flex items-center justify-between">
      <h3 class="text-sm font-semibold text-foreground">{{ t('dashboard.platformBreakdown') }}</h3>
      <span class="text-xs tabular-nums text-muted-foreground">
        {{ t('dashboard.platformCount', { count: platformCount }) }}
      </span>
    </div>
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <div
        v-for="item in platformCards"
        :key="item.platform"
        data-testid="platform-card"
        :data-platform="item.platform"
        :class="[
          'rounded-xl border p-3.5 transition-colors',
          item.isOther
            ? 'border-dashed border-border bg-muted/30'
            : 'border-border/80 bg-muted/20'
        ]"
      >
        <div class="flex items-center justify-between gap-2">
          <span class="truncate text-sm font-semibold text-foreground">
            {{ item.isOther ? t('dashboard.platformOther') : platformLabel(item.platform) }}
          </span>
          <span class="font-mono text-sm font-medium tabular-nums text-foreground" :title="t('dashboard.actual')">
            ${{ formatCost(item.total_actual_cost) }}
          </span>
        </div>
        <div class="mt-2.5 space-y-1 text-xs">
          <div class="flex items-center justify-between gap-2">
            <span class="text-muted-foreground">{{ t('dashboard.todayCost') }}</span>
            <span class="font-mono tabular-nums text-foreground">${{ formatCost(item.today_actual_cost) }}</span>
          </div>
          <div class="flex items-center justify-between gap-2">
            <span class="text-muted-foreground">{{ t('dashboard.requests') }}</span>
            <span class="font-mono tabular-nums text-muted-foreground">
              {{ item.total_requests > 0 ? formatNumber(item.total_requests) : '-' }}
            </span>
          </div>
          <div class="flex items-center justify-between gap-2">
            <span class="text-muted-foreground">{{ t('dashboard.tokens') }}</span>
            <span class="font-mono tabular-nums text-muted-foreground">
              {{ item.total_tokens > 0 ? formatTokens(item.total_tokens) : '-' }}
            </span>
          </div>
        </div>

        <!-- Quota 区：仅当 quota 配置存在、非 __other__ 且至少有一个窗口配了 limit 时显示 -->
        <div v-if="hasAnyLimit(item.quota) && !item.isOther" class="mt-3 space-y-1.5 border-t border-border pt-2.5">
          <p class="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
            {{ t('dashboard.platformQuota.title') }}
          </p>
          <template v-for="w in (['daily', 'weekly', 'monthly'] as const)" :key="w">
            <div v-if="quotaVal(item.quota, `${w}_limit_usd`) != null" class="space-y-1">
              <!-- limit=0：完全禁用 -->
              <template v-if="(quotaVal(item.quota, `${w}_limit_usd`) as number) === 0">
                <div class="flex items-center justify-between text-xs">
                  <span class="text-muted-foreground">{{ t(`dashboard.platformQuota.${w}`) }}</span>
                  <span class="font-mono text-red-600 dark:text-red-400">{{ t('dashboard.platformQuota.disabled') }}</span>
                </div>
                <div class="h-1.5 w-full overflow-hidden rounded-full bg-muted">
                  <div class="h-full w-full rounded-full bg-red-500" />
                </div>
              </template>
              <!-- limit>0：正常用量进度条 -->
              <template v-else>
                <div class="flex items-center justify-between text-xs">
                  <span class="text-muted-foreground">{{ t(`dashboard.platformQuota.${w}`) }}</span>
                  <span class="font-mono tabular-nums text-foreground">
                    ${{ formatUsd((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0) }} / ${{ formatUsd(quotaVal(item.quota, `${w}_limit_usd`) as number) }}
                  </span>
                </div>
                <div class="h-1.5 w-full overflow-hidden rounded-full bg-muted">
                  <div
                    class="h-full rounded-full transition-all"
                    :class="quotaBarClass(calcPercent((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0, quotaVal(item.quota, `${w}_limit_usd`) as number))"
                    :style="{ width: calcPercent((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0, quotaVal(item.quota, `${w}_limit_usd`) as number) + '%' }"
                  />
                </div>
                <p v-if="quotaVal(item.quota, `${w}_window_resets_at`)" class="text-[10px] tabular-nums text-muted-foreground">
                  {{ t('dashboard.platformQuota.resetsAt', { time: formatResetTime(quotaVal(item.quota, `${w}_window_resets_at`) as string) }) }}
                </p>
              </template>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, inject } from 'vue'
import { useI18n } from 'vue-i18n'
import { routerKey } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import type { PlatformDashboardStats, UserDashboardStats as UserStatsType } from '@/api/usage'
import type { PlatformQuotaItem } from '@/types'

interface FusedPlatformCard {
  platform: string
  total_actual_cost: number
  today_actual_cost: number
  total_requests: number
  total_tokens: number
  isOther?: boolean
  quota?: PlatformQuotaItem
}

const props = defineProps<{
  stats: UserStatsType
  balance: number
  isSimple: boolean
  platformQuotas?: PlatformQuotaItem[] | null
}>()
const { t } = useI18n()
const router = inject(routerKey, null)

const navigateTo = (path: string) => {
  void router?.push(path)
}

const PLATFORM_LABELS: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok',
  kimi: 'Kimi',
  zhipu: 'Zhipu GLM',
  deepseek: 'DeepSeek',
  minimax: 'MiniMax',
  typesafe: 'TypeSafe / Jev',
}

const platformLabel = (p: string) => PLATFORM_LABELS[p] ?? p

// 处理"各平台之和 < 总值"的差值：后端按平台聚合时过滤了无法归属平台的行
// （group 与 account 都缺 platform）。这里把差值作为"其他"卡片显式展示，
// 避免 Row 1 总值与 Row 3 平台拆分加总对不上、用户困惑。
const OTHER_THRESHOLD = 0.0001
const platformCards = computed<FusedPlatformCard[]>(() => {
  // 建立 by_platform Map
  const byPlat = new Map<string, PlatformDashboardStats>()
  for (const item of props.stats?.by_platform ?? []) byPlat.set(item.platform, item)

  // 建立 quota Map。三档全空的记录不产生卡片，挂到卡片上也不渲染配额区。
  const byQuota = new Map<string, PlatformQuotaItem>()
  for (const q of props.platformQuotas ?? []) byQuota.set(q.platform, q)

  // 卡片集合 = 有用量的平台 ∪ 至少配置了一档限额的平台。
  // 三档全空的限额记录等价于不限额，不单独产生卡片。
  // 后端 by_platform / quota 接口均不会返回 platform='__other__'，
  // 无需显式排除；__other__ 由下方差值补差逻辑单独追加。
  const platforms = new Set<string>(byPlat.keys())
  for (const [platform, q] of byQuota) {
    if (hasAnyLimit(q)) platforms.add(platform)
  }

  const PLATFORM_ORDER = ['anthropic', 'openai', 'gemini', 'antigravity', 'grok', 'typesafe']
  const cards: FusedPlatformCard[] = []

  for (const p of platforms) {
    const stat = byPlat.get(p)
    cards.push({
      platform: p,
      total_actual_cost: stat?.total_actual_cost ?? 0,
      today_actual_cost: stat?.today_actual_cost ?? 0,
      total_requests: stat?.total_requests ?? 0,
      total_tokens: stat?.total_tokens ?? 0,
      quota: byQuota.get(p),
    })
  }

  // 排序：按 PLATFORM_ORDER，未知平台按名称排序
  cards.sort((a, b) => {
    const ai = PLATFORM_ORDER.indexOf(a.platform)
    const bi = PLATFORM_ORDER.indexOf(b.platform)
    if (ai === -1 && bi === -1) return a.platform.localeCompare(b.platform)
    if (ai === -1) return 1
    if (bi === -1) return -1
    return ai - bi
  })

  // __other__ 补差逻辑：只对 by_platform 有 usage 数据的总和计算
  const total = props.stats?.total_actual_cost ?? 0
  const today = props.stats?.today_actual_cost ?? 0
  const sumTotal = cards.reduce((s, c) => s + c.total_actual_cost, 0)
  const sumToday = cards.reduce((s, c) => s + c.today_actual_cost, 0)
  const diffTotal = Math.max(0, total - sumTotal)
  const diffToday = Math.max(0, today - sumToday)

  if (diffTotal > OTHER_THRESHOLD || diffToday > OTHER_THRESHOLD) {
    cards.push({
      platform: '__other__',
      total_actual_cost: diffTotal,
      today_actual_cost: diffToday,
      total_requests: 0,
      total_tokens: 0,
      isOther: true,
    })
  }

  return cards
})

// 标题右侧的平台计数 = 实际渲染的平台卡片数，不含"其他"差额卡。
const platformCount = computed(() => platformCards.value.filter((c) => !c.isOther).length)

// Quota helpers

type QuotaWindow = 'daily' | 'weekly' | 'monthly'
type QuotaField = `${QuotaWindow}_limit_usd` | `${QuotaWindow}_usage_usd` | `${QuotaWindow}_window_resets_at`

function quotaVal(q: PlatformQuotaItem | undefined, key: QuotaField): PlatformQuotaItem[QuotaField] {
  return q?.[key]
}

function hasAnyLimit(q: PlatformQuotaItem | undefined): boolean {
  if (!q) return false
  return q.daily_limit_usd != null || q.weekly_limit_usd != null || q.monthly_limit_usd != null
}

function calcPercent(usage: number, limit: number): number {
  if (!limit || limit <= 0) return 0
  return Math.min(100, Math.max(0, Math.round((usage / limit) * 100)))
}

function quotaBarClass(p: number): string {
  if (p >= 95) return 'bg-red-500'
  if (p >= 75) return 'bg-amber-500'
  return 'bg-green-500'
}

// 与 formatBalance 一致使用 Intl.NumberFormat 做半偶舍入，避免 toFixed 在不同 JS 引擎
// 下偶发截断而非四舍五入（与后端展示精度不一致）。
const usdFormatter = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})
function formatUsd(n: number): string {
  if (!Number.isFinite(n)) return '0.00'
  return usdFormatter.format(n)
}

function formatResetTime(iso: string | null | undefined): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString(undefined, {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}

const formatBalance = (b: number) =>
  new Intl.NumberFormat('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  }).format(b)

const formatNumber = (n: number) => n.toLocaleString()
const formatCost = (c: number) => c.toFixed(4)
const formatTokens = (t: number) => {
  if (t >= 1_000_000) return `${(t / 1_000_000).toFixed(1)}M`
  if (t >= 1000) return `${(t / 1000).toFixed(1)}K`
  return t.toString()
}
const formatDuration = (ms: number) => ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${ms.toFixed(0)}ms`
</script>
