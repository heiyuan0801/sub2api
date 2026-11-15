<template>
  <SiteLayout surface="canvas">
    <template #header>
      <SiteHeader>
        <nav class="mx-auto flex min-h-14 max-w-5xl items-center justify-between gap-3 px-4 sm:px-6">
          <SiteBrand :name="siteName" :logo="siteLogo" />
          <div class="flex shrink-0 items-center gap-1 sm:gap-2">
            <LocaleSwitcher />
            <a
              v-if="docUrl"
              :href="docUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="rounded-lg p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              :title="t('home.viewDocs')"
            >
              <Icon name="book" size="md" />
            </a>
            <button
              @click="toggleTheme"
              class="rounded-lg p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            >
              <Icon v-if="isDark" name="sun" size="md" />
              <Icon v-else name="moon" size="md" />
            </button>
          </div>
        </nav>
      </SiteHeader>
    </template>

    <!-- Main Content -->
    <main class="mx-auto w-full max-w-5xl flex-1 space-y-5 px-4 py-8 sm:px-6 sm:py-10">
      <PageHeader :title="t('keyUsage.title')" :description="t('keyUsage.subtitle')" />

      <SectionCard>
        <Toolbar>
          <div class="relative min-w-0 flex-1 basis-52">
            <div class="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground">
              <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
              </svg>
            </div>
            <input
              v-model="apiKey"
              :type="keyVisible ? 'text' : 'password'"
              :placeholder="t('keyUsage.placeholder')"
              :aria-label="t('keys.apiKey')"
              class="input-ring h-10 w-full rounded-md border border-input bg-background pl-12 pr-12 text-sm text-foreground placeholder:text-muted-foreground"
              @keydown.enter="queryKey"
            />
            <button
              type="button"
              :aria-label="t('keyUsage.showKey')" :aria-pressed="keyVisible"
              @click="keyVisible = !keyVisible"
              class="absolute right-4 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
            >
              <svg v-if="!keyVisible" class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/>
                <line x1="1" y1="1" x2="23" y2="23"/>
              </svg>
              <svg v-else class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/>
              </svg>
            </button>
          </div>
          <Button
            type="button"
            @click="queryKey"
            :disabled="isQuerying"
            class="h-10 whitespace-nowrap bg-brand text-brand-foreground hover:bg-brand/90 sm:px-5"
          >
            <svg v-if="isQuerying" class="w-4 h-4 animate-spin" viewBox="0 0 24 24" fill="none">
              <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" opacity="0.25"/>
              <path d="M12 2a10 10 0 0 1 10 10" stroke="currentColor" stroke-width="3" stroke-linecap="round"/>
            </svg>
            <svg v-else class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
            </svg>
            {{ isQuerying ? t('keyUsage.querying') : t('keyUsage.query') }}
          </Button>
        </Toolbar>
        <p class="mt-2 text-xs text-muted-foreground">
          {{ t('keyUsage.privacyNote') }}
        </p>

        <!-- Date Range Picker -->
        <div v-if="showDatePicker" class="mt-4">
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-xs text-muted-foreground">{{ t('keyUsage.dateRange') }}</span>
            <button
              v-for="range in dateRanges"
              :key="range.key"
              @click="setDateRange(range.key)"
              class="rounded-md px-3 py-1.5 text-xs border transition-all"
              :class="currentRange === range.key
                ? 'bg-brand text-brand-foreground border-brand'
                : 'border-border bg-card text-foreground hover:border-brand/50 hover:bg-muted'"
            >{{ range.label }}</button>
            <div v-if="currentRange === 'custom'" class="flex min-w-0 flex-wrap items-center gap-2 sm:ml-1">
              <input
                v-model="customStartDate"
                type="date"
                class="input-ring text-xs px-2 py-1.5 rounded-lg border border-border bg-background text-foreground"
              />
              <span class="text-xs text-muted-foreground">-</span>
              <input
                v-model="customEndDate"
                type="date"
                class="input-ring text-xs px-2 py-1.5 rounded-lg border border-border bg-background text-foreground"
              />
              <button
                @click="queryKey"
                class="rounded-md px-3 py-1.5 text-xs bg-brand text-brand-foreground hover:bg-brand/90"
              >{{ t('keyUsage.apply') }}</button>
            </div>
          </div>
        </div>
      </SectionCard>

      <!-- Results Container -->
      <div v-if="showResults">
        <!-- Loading Skeleton -->
        <div v-if="showLoading" class="space-y-4" aria-busy="true">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div class="rounded-lg border border-border bg-card p-5">
              <div class="skeleton h-5 w-24 mb-6"></div>
              <div class="flex justify-center"><div class="skeleton w-44 h-44 rounded-full"></div></div>
            </div>
            <div class="rounded-lg border border-border bg-card p-5">
              <div class="skeleton h-5 w-24 mb-6"></div>
              <div class="flex justify-center"><div class="skeleton w-44 h-44 rounded-full"></div></div>
            </div>
          </div>
          <div class="rounded-lg border border-border bg-card p-5">
            <div class="skeleton h-5 w-32 mb-6"></div>
            <div class="space-y-4">
              <div class="skeleton h-4 w-full"></div>
              <div class="skeleton h-4 w-3/4"></div>
              <div class="skeleton h-4 w-5/6"></div>
              <div class="skeleton h-4 w-2/3"></div>
            </div>
          </div>
        </div>

        <!-- Result Content -->
        <div v-else-if="resultData" class="space-y-6">
          <!-- Status Badge -->
          <div v-if="statusInfo" class="fade-up flex flex-wrap items-center gap-2">
            <StatusBadge :tone="statusInfo.isActive ? 'success' : 'danger'" :label="statusInfo.label" />
            <span class="text-xs text-muted-foreground">{{ statusInfo.statusText }}</span>
          </div>

          <!-- Ring Cards Grid -->
          <div v-if="ringItems.length > 0" :class="ringGridClass">
            <div
              v-for="(ring, i) in ringItems"
              :key="i"
              class="fade-up min-w-0 rounded-lg border border-border bg-card p-5"
              :class="`fade-up-delay-${Math.min(i + 1, 4)}`"
            >
              <div class="flex items-center justify-between mb-6">
                <h3 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
                  {{ ring.title }}
                </h3>
                <!-- Clock icon -->
                <svg v-if="ring.iconType === 'clock'" class="w-5 h-5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
                </svg>
                <!-- Calendar icon -->
                <svg v-else-if="ring.iconType === 'calendar'" class="w-5 h-5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/>
                </svg>
                <!-- Dollar icon -->
                <svg v-else class="w-5 h-5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>
                </svg>
              </div>
              <div class="flex justify-center">
                <div class="relative">
                  <svg class="w-44 h-44" viewBox="0 0 160 160">
                    <circle cx="80" cy="80" r="68" fill="none" stroke="hsl(var(--muted))" stroke-width="10"/>
                    <circle
                      class="progress-ring"
                      cx="80" cy="80" r="68" fill="none"
                      :stroke="`url(#ring-grad-${i})`"
                      stroke-width="10" stroke-linecap="round"
                      :stroke-dasharray="CIRCUMFERENCE.toFixed(2)"
                      :stroke-dashoffset="getRingOffset(ring)"
                    />
                    <defs>
                      <linearGradient :id="`ring-grad-${i}`" x1="0%" y1="0%" x2="100%" y2="100%">
                        <stop offset="0%" :stop-color="RING_GRADIENTS[i % 4].from"/>
                        <stop offset="100%" :stop-color="RING_GRADIENTS[i % 4].to"/>
                      </linearGradient>
                    </defs>
                  </svg>
                  <div class="absolute inset-0 flex flex-col items-center justify-center">
                    <template v-if="ring.isBalance">
                      <span class="text-2xl font-bold tabular-nums" :style="{ color: RING_GRADIENTS[i % 4].from }">
                        {{ ring.amount }}
                      </span>
                    </template>
                    <template v-else>
                      <span class="text-3xl font-bold tabular-nums text-foreground">
                        {{ displayPcts[i] ?? 0 }}%
                      </span>
                      <span class="text-xs text-muted-foreground mt-0.5">{{ t('keyUsage.used') }}</span>
                      <span
                        class="text-sm font-semibold mt-1 tabular-nums"
                        :style="{ color: RING_GRADIENTS[i % 4].from }"
                      >{{ ring.amount }}</span>
                      <p v-if="ring.resetAt && formatResetTime(ring.resetAt)" class="text-xs text-muted-foreground mt-0.5 tabular-nums">
                        ⟳ {{ formatResetTime(ring.resetAt) }}
                      </p>
                    </template>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Detail Card -->
          <SectionCard
            v-if="detailRows.length > 0"
            :title="t('keyUsage.detailInfo')"
            class="fade-up fade-up-delay-3 overflow-hidden"
            flush
          >
            <div class="divide-y divide-border">
              <div
                v-for="(row, i) in detailRows"
                :key="i"
                class="flex flex-wrap items-center justify-between gap-2 px-4 py-3"
              >
                <div class="flex items-center gap-3">
                  <div class="w-8 h-8 rounded-lg flex items-center justify-center" :class="row.iconBg">
                    <svg
                      class="w-4 h-4"
                      :class="row.iconColor"
                      viewBox="0 0 24 24" fill="none" stroke="currentColor"
                      stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
                      v-html="row.iconSvg"
                    ></svg>
                  </div>
                  <span class="text-sm text-foreground">{{ row.label }}</span>
                </div>
                <span class="text-sm font-semibold tabular-nums" :class="row.valueClass || 'text-foreground'">
                  {{ row.value }}
                </span>
              </div>
            </div>
          </SectionCard>

          <!-- Usage Stats Card -->
          <SectionCard
            v-if="usageStatCells.length > 0"
            :title="t('keyUsage.tokenStats')"
            class="fade-up fade-up-delay-3 overflow-hidden"
            flush
          >
            <div class="grid grid-cols-2 md:grid-cols-4 gap-px bg-muted">
              <div
                v-for="(cell, i) in usageStatCells"
                :key="i"
                class="min-w-0 bg-card px-4 py-3"
              >
                <div class="text-xs text-muted-foreground mb-1">{{ cell.label }}</div>
                <div class="break-words text-sm font-semibold tabular-nums text-foreground">{{ cell.value }}</div>
              </div>
            </div>
          </SectionCard>

          <!-- Daily Usage Table -->
          <SectionCard
            v-if="showDailyUsage"
            :title="t('keyUsage.dailyDetail')"
            class="fade-up fade-up-delay-4 overflow-hidden"
            flush
          >
            <template #actions>
              <div class="inline-flex rounded-md border border-border bg-muted p-0.5" role="group" :aria-label="t('keyUsage.dailyDetail')">
                <button
                  v-for="option in dailyUsageOptions"
                  :key="option.value"
                  type="button"
                  :aria-pressed="dailyUsageDays === option.value"
                  @click="setDailyUsageDays(option.value)"
                  class="min-w-12 rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
                  :class="dailyUsageDays === option.value
                    ? 'bg-brand text-brand-foreground'
                    : 'text-muted-foreground hover:bg-card hover:text-foreground'"
                >
                  {{ option.label }}
                </button>
              </div>
            </template>
            <Table v-if="dailyUsageRows.length > 0" class="min-w-[680px]">
              <TableHeader>
                <TableRow class="border-b border-border bg-canvas">
                  <TableHead class="px-4 py-2 text-left text-xs font-medium text-muted-foreground">{{ t('keyUsage.date') }}</TableHead>
                  <TableHead class="px-4 py-2 text-right text-xs font-medium text-muted-foreground">{{ t('keyUsage.requests') }}</TableHead>
                  <TableHead class="px-4 py-2 text-right text-xs font-medium text-muted-foreground">{{ t('keyUsage.inputTokens') }}</TableHead>
                  <TableHead class="px-4 py-2 text-right text-xs font-medium text-muted-foreground">{{ t('keyUsage.outputTokens') }}</TableHead>
                  <TableHead class="px-4 py-2 text-right text-xs font-medium text-muted-foreground">{{ t('keyUsage.cacheReadTokens') }}</TableHead>
                  <TableHead class="px-4 py-2 text-right text-xs font-medium text-muted-foreground">{{ t('keyUsage.cacheWriteTokens') }}</TableHead>
                  <TableHead class="px-4 py-2 text-right text-xs font-medium text-muted-foreground">{{ t('keyUsage.cost') }}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow
                  v-for="row in dailyUsageRows"
                  :key="row.date"
                  class="border-b border-border last:border-b-0 hover:bg-muted/40"
                >
                  <TableCell class="px-4 py-2 text-sm font-medium whitespace-nowrap text-foreground">{{ row.date }}</TableCell>
                  <TableCell class="px-4 py-2 text-sm tabular-nums text-right text-foreground">{{ fmtNum(row.requests) }}</TableCell>
                  <TableCell class="px-4 py-2 text-sm tabular-nums text-right text-foreground">{{ fmtNum(row.input_tokens) }}</TableCell>
                  <TableCell class="px-4 py-2 text-sm tabular-nums text-right text-foreground">{{ fmtNum(row.output_tokens) }}</TableCell>
                  <TableCell class="px-4 py-2 text-sm tabular-nums text-right text-foreground">{{ fmtNum(row.cache_read_tokens) }}</TableCell>
                  <TableCell class="px-4 py-2 text-sm tabular-nums text-right text-foreground">{{ fmtNum(row.cache_write_tokens) }}</TableCell>
                  <TableCell class="px-4 py-2 text-sm tabular-nums text-right font-medium text-foreground">{{ usd(row.actual_cost != null ? row.actual_cost : row.cost) }}</TableCell>
                </TableRow>
              </TableBody>
            </Table>
            <EmptyState v-else :title="t('keyUsage.noDailyUsage')" />
          </SectionCard>

          <!-- Model Stats Table -->
          <SectionCard
            v-if="modelStats.length > 0"
            :title="t('keyUsage.modelStats')"
            class="fade-up fade-up-delay-4 overflow-hidden"
            flush
          >
            <Table class="min-w-[680px]">
              <TableHeader>
                <TableRow class="border-b border-border bg-canvas">
                  <TableHead class="px-4 py-2 text-left text-xs font-medium text-muted-foreground">{{ t('keyUsage.model') }}</TableHead>
                  <TableHead class="px-4 py-2 text-right text-xs font-medium text-muted-foreground">{{ t('keyUsage.requests') }}</TableHead>
                  <TableHead class="px-4 py-2 text-right text-xs font-medium text-muted-foreground">{{ t('keyUsage.inputTokens') }}</TableHead>
                  <TableHead class="px-4 py-2 text-right text-xs font-medium text-muted-foreground">{{ t('keyUsage.outputTokens') }}</TableHead>
                  <TableHead class="px-4 py-2 text-right text-xs font-medium text-muted-foreground">{{ t('keyUsage.cacheCreationTokens') }}</TableHead>
                  <TableHead class="px-4 py-2 text-right text-xs font-medium text-muted-foreground">{{ t('keyUsage.cacheReadTokens') }}</TableHead>
                  <TableHead class="px-4 py-2 text-right text-xs font-medium text-muted-foreground">{{ t('keyUsage.totalTokens') }}</TableHead>
                  <TableHead class="px-4 py-2 text-right text-xs font-medium text-muted-foreground">{{ t('keyUsage.cost') }}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow
                  v-for="(m, i) in modelStats"
                  :key="i"
                  class="border-b border-border last:border-b-0 hover:bg-muted/40"
                >
                  <TableCell class="px-4 py-2 text-sm font-medium whitespace-nowrap text-foreground">{{ m.model || '-' }}</TableCell>
                  <TableCell class="px-4 py-2 text-sm tabular-nums text-right text-foreground">{{ fmtNum(m.requests) }}</TableCell>
                  <TableCell class="px-4 py-2 text-sm tabular-nums text-right text-foreground">{{ fmtNum(m.input_tokens) }}</TableCell>
                  <TableCell class="px-4 py-2 text-sm tabular-nums text-right text-foreground">{{ fmtNum(m.output_tokens) }}</TableCell>
                  <TableCell class="px-4 py-2 text-sm tabular-nums text-right text-foreground">{{ fmtNum(m.cache_creation_tokens) }}</TableCell>
                  <TableCell class="px-4 py-2 text-sm tabular-nums text-right text-foreground">{{ fmtNum(m.cache_read_tokens) }}</TableCell>
                  <TableCell class="px-4 py-2 text-sm tabular-nums text-right text-foreground">{{ fmtNum(m.total_tokens) }}</TableCell>
                  <TableCell class="px-4 py-2 text-sm tabular-nums text-right font-medium text-foreground">{{ usd(m.actual_cost != null ? m.actual_cost : m.cost) }}</TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </SectionCard>
        </div>
      </div>
    </main>

    <template #footer>
      <SiteFooter class="px-4 py-4 sm:px-6">
        <div class="mx-auto flex max-w-5xl flex-col items-center justify-between gap-4 text-center sm:flex-row sm:text-left">
          <p class="text-sm text-muted-foreground">
            &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
          </p>
          <div class="flex items-center gap-4">
            <a
              v-if="docUrl"
              :href="docUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="text-sm text-muted-foreground transition-colors hover:text-foreground"
            >{{ t('home.docs') }}</a>
            <a
              :href="githubUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="text-sm text-muted-foreground transition-colors hover:text-foreground"
            >GitHub</a>
          </div>
        </div>
      </SiteFooter>
    </template>
  </SiteLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { EmptyState, PageHeader, SectionCard, StatusBadge, Toolbar } from '@/components/page'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import SiteBrand from '@/components/site-shell/SiteBrand.vue'
import SiteFooter from '@/components/site-shell/SiteFooter.vue'
import SiteHeader from '@/components/site-shell/SiteHeader.vue'
import SiteLayout from '@/components/site-shell/SiteLayout.vue'
import { buildGatewayUrl } from '@/api/client'
import { formatDateLocalInput } from '@/utils/format'
import { sanitizeUrl } from '@/utils/url'

const { t, locale } = useI18n()
const appStore = useAppStore()
const subscriptionFeatureEnabled = computed(() => resolveFeatureFlag(appStore.cachedPublicSettings, FeatureFlags.subscription))

// ==================== Site Settings (same as HomeView) ====================

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const githubUrl = 'https://github.com/Wei-Shaw/sub2api'

// ==================== Theme (same as HomeView) ====================

const isDark = ref(document.documentElement.classList.contains('dark'))

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

const currentYear = computed(() => new Date().getFullYear())

// ==================== Key Query State ====================

const apiKey = ref('')
const keyVisible = ref(false)
const isQuerying = ref(false)
const showResults = ref(false)
const showLoading = ref(false)
const showDatePicker = ref(false)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const resultData = ref<any>(null)
const now = ref(new Date())
let resetTimer: ReturnType<typeof setInterval> | null = null

// ==================== Date Range State ====================

type DateRangeKey = 'today' | '7d' | '30d' | 'custom'
const currentRange = ref<DateRangeKey>('today')
const customStartDate = ref('')
const customEndDate = ref('')
const dailyUsageDays = ref<7 | 30 | 90>(30)

const dateRanges = computed(() => [
  { key: 'today' as const, label: t('keyUsage.dateRangeToday') },
  { key: '7d' as const, label: t('keyUsage.dateRange7d') },
  { key: '30d' as const, label: t('keyUsage.dateRange30d') },
  { key: 'custom' as const, label: t('keyUsage.dateRangeCustom') },
])

const dailyUsageOptions = computed(() => [
  { value: 7 as const, label: t('keyUsage.dateRange7d') },
  { value: 30 as const, label: t('keyUsage.dateRange30d') },
  { value: 90 as const, label: t('keyUsage.dateRange90d') },
])

function setDateRange(key: DateRangeKey) {
  currentRange.value = key
  if (key !== 'custom') {
    queryKey()
  }
}

function getDateParams(): string {
  const now = new Date()
  const params = new URLSearchParams()

  if (currentRange.value === 'custom') {
    if (customStartDate.value && customEndDate.value) {
      params.set('start_date', customStartDate.value)
      params.set('end_date', customEndDate.value)
    }
  } else {
    const end = formatDateLocalInput(now)
    let start: string
    switch (currentRange.value) {
      case 'today': start = end; break
      case '7d': start = formatDateLocalInput(new Date(now.getTime() - 7 * 86400000)); break
      case '30d': start = formatDateLocalInput(new Date(now.getTime() - 30 * 86400000)); break
      default: start = formatDateLocalInput(new Date(now.getTime() - 30 * 86400000))
    }
    params.set('start_date', start)
    params.set('end_date', end)
  }
  params.set('days', String(dailyUsageDays.value))
  params.set('timezone', getBrowserTimezone())
  return params.toString()
}

function setDailyUsageDays(days: 7 | 30 | 90) {
  if (dailyUsageDays.value === days) return
  dailyUsageDays.value = days
  if (resultData.value && apiKey.value.trim()) {
    queryKey()
  }
}

// ==================== Ring Animation ====================

const CIRCUMFERENCE = 2 * Math.PI * 68
const RING_GRADIENTS = [
  { from: 'hsl(var(--brand))', to: 'hsl(var(--brand) / .55)' },
  { from: 'hsl(var(--info))', to: 'hsl(var(--info) / .55)' },
  { from: 'hsl(var(--success))', to: 'hsl(var(--success) / .55)' },
  { from: 'hsl(var(--warning))', to: 'hsl(var(--warning) / .55)' },
]

const ringAnimated = ref(false)
const displayPcts = ref<number[]>([])

interface RingItem {
  title: string
  pct: number
  amount: string
  isBalance?: boolean
  iconType: 'clock' | 'calendar' | 'dollar'
  resetAt?: string | null
}

function getRingOffset(ring: RingItem): number {
  if (!ringAnimated.value) return CIRCUMFERENCE
  if (ring.isBalance) return 0
  return CIRCUMFERENCE - (Math.min(ring.pct, 100) / 100) * CIRCUMFERENCE
}

function triggerRingAnimation(items: RingItem[]) {
  ringAnimated.value = false
  displayPcts.value = items.map(() => 0)

  nextTick(() => {
    requestAnimationFrame(() => {
      setTimeout(() => {
        ringAnimated.value = true

        // Animate percentage numbers
        const duration = 1000
        const startTime = performance.now()
        const targets = items.map(item => item.isBalance ? 0 : item.pct)

        function tick() {
          const elapsed = performance.now() - startTime
          const p = Math.min(elapsed / duration, 1)
          const ease = 1 - Math.pow(1 - p, 3)
          displayPcts.value = targets.map(target => Math.round(ease * target))
          if (p < 1) requestAnimationFrame(tick)
        }
        requestAnimationFrame(tick)
      }, 50)
    })
  })
}

// ==================== Computed Data ====================

const statusInfo = computed(() => {
  const data = resultData.value
  if (!data) return null

  if (data.mode === 'quota_limited') {
    const isValid = data.isValid !== false
    const statusMap: Record<string, string> = {
      active: 'Active',
      quota_exhausted: 'Quota Exhausted',
      expired: 'Expired',
    }
    return {
      label: t('keyUsage.quotaMode'),
      statusText: statusMap[data.status] || data.status || 'Unknown',
      isActive: isValid && data.status === 'active',
    }
  }

  return {
    label: data.planName || t('keyUsage.walletBalance'),
    statusText: 'Active',
    isActive: true,
  }
})

const ringItems = computed<RingItem[]>(() => {
  const data = resultData.value
  if (!data) return []

  const items: RingItem[] = []

  if (data.mode === 'quota_limited') {
    if (data.quota) {
      const pct = data.quota.limit > 0 ? Math.min(Math.round((data.quota.used / data.quota.limit) * 100), 100) : 0
      items.push({ title: t('keyUsage.totalQuota'), pct, amount: `${usd(data.quota.used)} / ${usd(data.quota.limit)}`, iconType: 'dollar' })
    }
    if (data.rate_limits) {
      const windowLabels: Record<string, string> = { '5h': t('keyUsage.limit5h'), '1d': t('keyUsage.limitDaily'), '7d': t('keyUsage.limit7d') }
      const windowIcons: Record<string, 'clock' | 'calendar'> = { '5h': 'clock', '1d': 'calendar', '7d': 'calendar' }
      for (const rl of data.rate_limits) {
        const pct = rl.limit > 0 ? Math.min(Math.round((rl.used / rl.limit) * 100), 100) : 0
        items.push({
          title: windowLabels[rl.window] || rl.window,
          pct,
          amount: `${usd(rl.used)} / ${usd(rl.limit)}`,
          iconType: windowIcons[rl.window] || 'clock',
          resetAt: rl.reset_at,
        })
      }
    }
  } else {
    if (data.subscription) {
      const sub = data.subscription
      const limits = [
        { label: t('keyUsage.limitDaily'), usage: sub.daily_usage_usd, limit: sub.daily_limit_usd },
        { label: t('keyUsage.limitWeekly'), usage: sub.weekly_usage_usd, limit: sub.weekly_limit_usd },
        { label: t('keyUsage.limitMonthly'), usage: sub.monthly_usage_usd, limit: sub.monthly_limit_usd },
      ]
      for (const l of limits) {
        if (l.limit != null && l.limit > 0) {
          const pct = Math.min(Math.round((l.usage / l.limit) * 100), 100)
          items.push({ title: l.label, pct, amount: `${usd(l.usage)} / ${usd(l.limit)}`, iconType: 'calendar' })
        }
      }
    }
    if (!data.subscription && data.balance != null) {
      items.push({ title: t('keyUsage.walletBalance'), pct: 0, amount: usd(data.balance), isBalance: true, iconType: 'dollar' })
    }
  }

  return items
})

const ringGridClass = computed(() => {
  const len = ringItems.value.length
  if (len === 1) return 'grid grid-cols-1 max-w-md mx-auto gap-6'
  if (len === 2) return 'grid grid-cols-1 md:grid-cols-2 gap-6'
  return 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6'
})

interface DetailRow {
  iconBg: string
  iconColor: string
  iconSvg: string
  label: string
  value: string
  valueClass: string
}

function getUsageColor(pct: number): string {
  if (pct > 90) return 'text-rose-500'
  if (pct > 70) return 'text-amber-500'
  return 'text-emerald-500'
}

const detailRows = computed<DetailRow[]>(() => {
  const data = resultData.value
  if (!data) return []

  const rows: DetailRow[] = []
  const ICON_SHIELD = '<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>'
  const ICON_CALENDAR = '<rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/>'
  const ICON_DOLLAR = '<line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>'
  const ICON_CHECK = '<polyline points="20 6 9 17 4 12"/>'

  if (data.mode === 'quota_limited') {
    if (data.quota) {
      const remainColor = data.quota.remaining <= 0 ? 'text-rose-500'
        : data.quota.remaining < data.quota.limit * 0.1 ? 'text-amber-500'
        : 'text-emerald-500'
      rows.push({
        iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-500', iconSvg: ICON_SHIELD,
        label: t('keyUsage.remainingQuota'), value: usd(data.quota.remaining), valueClass: remainColor,
      })
    }
    if (data.expires_at) {
      const daysLeft = data.days_until_expiry
      let expiryStr = formatDate(data.expires_at)
      if (daysLeft != null) {
        expiryStr += daysLeft > 0 ? ` ${t('keyUsage.daysLeft', { days: daysLeft })}` : daysLeft === 0 ? ` ${t('keyUsage.todayExpires')}` : ''
      }
      rows.push({
        iconBg: 'bg-amber-500/10', iconColor: 'text-amber-500', iconSvg: ICON_CALENDAR,
        label: t('keyUsage.expiresAt'), value: expiryStr, valueClass: '',
      })
    }
    if (data.rate_limits) {
      const windowMap: Record<string, string> = { '5h': '5H', '1d': locale.value === 'zh' ? '日' : 'D', '7d': '7D' }
      for (const rl of data.rate_limits) {
        const pct = rl.limit > 0 ? (rl.used / rl.limit) * 100 : 0
        let valueStr = `${usd(rl.used)} / ${usd(rl.limit)}`
        const resetStr = formatResetTime(rl.reset_at)
        if (resetStr) {
          valueStr += ` (⟳ ${resetStr})`
        }
        rows.push({
          iconBg: 'bg-brand/10', iconColor: 'text-brand', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${windowMap[rl.window] || rl.window})`,
          value: valueStr,
          valueClass: getUsageColor(pct),
        })
      }
    }
  } else {
    rows.push({
      iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-500', iconSvg: ICON_CHECK,
      // 订阅功能关闭后这一行只会是「钱包余额」，标签改用不带「订阅」字样的「计费方式」。
      label: subscriptionFeatureEnabled.value ? t('keyUsage.subscriptionType') : t('keyUsage.billingType'),
      value: data.planName || t('keyUsage.walletBalance'), valueClass: '',
    })

    if (data.subscription) {
      const sub = data.subscription
      if (sub.daily_limit_usd > 0) {
        const pct = (sub.daily_usage_usd / sub.daily_limit_usd) * 100
        rows.push({
          iconBg: 'bg-brand/10', iconColor: 'text-brand', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${locale.value === 'zh' ? '日' : 'D'})`, value: `${usd(sub.daily_usage_usd)} / ${usd(sub.daily_limit_usd)}`, valueClass: getUsageColor(pct),
        })
      }
      if (sub.weekly_limit_usd > 0) {
        const pct = (sub.weekly_usage_usd / sub.weekly_limit_usd) * 100
        rows.push({
          iconBg: 'bg-indigo-500/10', iconColor: 'text-indigo-500', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${locale.value === 'zh' ? '周' : 'W'})`, value: `${usd(sub.weekly_usage_usd)} / ${usd(sub.weekly_limit_usd)}`, valueClass: getUsageColor(pct),
        })
      }
      if (sub.monthly_limit_usd > 0) {
        const pct = (sub.monthly_usage_usd / sub.monthly_limit_usd) * 100
        rows.push({
          iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-500', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${locale.value === 'zh' ? '月' : 'M'})`, value: `${usd(sub.monthly_usage_usd)} / ${usd(sub.monthly_limit_usd)}`, valueClass: getUsageColor(pct),
        })
      }
      if (sub.expires_at) {
        rows.push({
          iconBg: 'bg-amber-500/10', iconColor: 'text-amber-500', iconSvg: ICON_CALENDAR,
          label: t('keyUsage.subscriptionExpires'), value: formatDate(sub.expires_at), valueClass: '',
        })
      }
    }

    const remainColor = data.remaining != null
      ? (data.remaining <= 0 ? 'text-rose-500' : data.remaining < 10 ? 'text-amber-500' : 'text-emerald-500')
      : ''
    rows.push({
      iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-500', iconSvg: ICON_SHIELD,
      label: t('keyUsage.remainingQuota'), value: data.remaining != null ? usd(data.remaining) : '-', valueClass: remainColor,
    })
  }

  return rows
})

interface StatCell {
  label: string
  value: string
}

const usageStatCells = computed<StatCell[]>(() => {
  const usage = resultData.value?.usage
  if (!usage) return []

  const today = usage.today || {}
  const total = usage.total || {}

  return [
    { label: t('keyUsage.todayRequests'), value: fmtNum(today.requests) },
    { label: t('keyUsage.todayInputTokens'), value: fmtNum(today.input_tokens) },
    { label: t('keyUsage.todayOutputTokens'), value: fmtNum(today.output_tokens) },
    { label: t('keyUsage.todayTokens'), value: fmtNum(today.total_tokens) },
    { label: t('keyUsage.todayCacheCreation'), value: fmtNum(today.cache_creation_tokens) },
    { label: t('keyUsage.todayCacheRead'), value: fmtNum(today.cache_read_tokens) },
    { label: t('keyUsage.todayCost'), value: usd(today.actual_cost) },
    { label: t('keyUsage.rpmTpm'), value: `${usage.rpm || 0} / ${usage.tpm || 0}` },
    { label: t('keyUsage.totalRequests'), value: fmtNum(total.requests) },
    { label: t('keyUsage.totalInputTokens'), value: fmtNum(total.input_tokens) },
    { label: t('keyUsage.totalOutputTokens'), value: fmtNum(total.output_tokens) },
    { label: t('keyUsage.totalTokensLabel'), value: fmtNum(total.total_tokens) },
    { label: t('keyUsage.totalCacheCreation'), value: fmtNum(total.cache_creation_tokens) },
    { label: t('keyUsage.totalCacheRead'), value: fmtNum(total.cache_read_tokens) },
    { label: t('keyUsage.totalCost'), value: usd(total.actual_cost) },
    { label: t('keyUsage.avgDuration'), value: usage.average_duration_ms ? `${Math.round(usage.average_duration_ms)} ms` : '-' },
  ]
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const modelStats = computed<any[]>(() => resultData.value?.model_stats || [])

interface DailyUsageRow {
  date: string
  requests: number
  input_tokens: number
  output_tokens: number
  cache_read_tokens: number
  cache_write_tokens: number
  cost: number
  actual_cost?: number
}

const dailyUsageRows = computed<DailyUsageRow[]>(() => {
  const rows = resultData.value?.daily_usage
  return Array.isArray(rows) ? rows : []
})

const showDailyUsage = computed(() => Boolean(resultData.value && Array.isArray(resultData.value.daily_usage)))

// ==================== Utility Functions ====================

function usd(value: number | null | undefined): string {
  if (value == null || value < 0) return '-'
  return '$' + Number(value).toFixed(2)
}

function fmtNum(val: number | null | undefined): string {
  if (val == null) return '-'
  return val.toLocaleString()
}

function formatDate(iso: string | null | undefined): string {
  if (!iso) return '-'
  const d = new Date(iso)
  const loc = locale.value === 'zh' ? 'zh-CN' : 'en-US'
  return d.toLocaleDateString(loc, { year: 'numeric', month: 'long', day: 'numeric' })
}

function getBrowserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
  } catch {
    return 'UTC'
  }
}

// ==================== API Query ====================

async function fetchUsage(key: string) {
  const dateParams = getDateParams()
  const url = buildGatewayUrl('/v1/usage') + (dateParams ? '?' + dateParams : '')
  const res = await fetch(url, {
    headers: { 'Authorization': 'Bearer ' + key },
  })
  if (!res.ok) {
    const body = await res.json().catch(() => null)
    const msg = body?.error?.message || body?.message || `${t('keyUsage.queryFailed')} (${res.status})`
    throw new Error(msg)
  }
  return await res.json()
}

async function queryKey() {
  if (isQuerying.value) return
  const key = apiKey.value.trim()
  if (!key) {
    appStore.showInfo(t('keyUsage.enterApiKey'))
    return
  }

  isQuerying.value = true
  showResults.value = true
  showLoading.value = true
  resultData.value = null

  try {
    const data = await fetchUsage(key)
    resultData.value = data
    showLoading.value = false
    showDatePicker.value = true

    // Trigger ring animations after DOM update
    nextTick(() => {
      triggerRingAnimation(ringItems.value)
    })

    appStore.showSuccess(t('keyUsage.querySuccess'))
  } catch (err) {
    showResults.value = false
    showLoading.value = false
    appStore.showError((err as Error).message || t('keyUsage.queryFailedRetry'))
  } finally {
    isQuerying.value = false
  }
}

// ==================== Lifecycle ====================

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark' || (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

function formatResetTime(resetAt: string | null | undefined): string {
  if (!resetAt) return ''
  const diff = new Date(resetAt).getTime() - now.value.getTime()
  if (diff <= 0) return t('keyUsage.resetNow')
  const days = Math.floor(diff / 86400000)
  const hours = Math.floor((diff % 86400000) / 3600000)
  const mins = Math.floor((diff % 3600000) / 60000)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${mins}m`
  return `${mins}m`
}

onMounted(() => {
  initTheme()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
  resetTimer = setInterval(() => { now.value = new Date() }, 60000)
})

onUnmounted(() => {
  if (resetTimer) clearInterval(resetTimer)
})
</script>

<style scoped>
/* Input focus ring */
.input-ring {
  transition: box-shadow 0.2s ease, border-color 0.2s ease;
}
.input-ring:focus {
  box-shadow: 0 0 0 3px hsl(var(--brand) / .2);
  border-color: hsl(var(--brand));
  outline: none;
}

/* Ring animation */
.progress-ring {
  transition: stroke-dashoffset 1.2s cubic-bezier(0.4, 0, 0.2, 1);
  transform: rotate(-90deg);
  transform-origin: 50% 50%;
}

/* Skeleton loading */
@keyframes shimmer-kv {
  0%   { background-position: -200% 0; }
  100% { background-position: 200% 0; }
}
.skeleton {
  background: linear-gradient(90deg, hsl(var(--muted)) 25%, hsl(var(--canvas)) 50%, hsl(var(--muted)) 75%);
  background-size: 200% 100%;
  animation: shimmer-kv 1.8s ease-in-out infinite;
  border-radius: 8px;
}

/* Fade up animation */
@keyframes fade-up-kv {
  from { opacity: 0; transform: translateY(16px); }
  to { opacity: 1; transform: translateY(0); }
}
.fade-up {
  animation: fade-up-kv 0.5s cubic-bezier(0.4, 0, 0.2, 1) forwards;
}
.fade-up-delay-1 { animation-delay: 0.1s; opacity: 0; }
.fade-up-delay-2 { animation-delay: 0.2s; opacity: 0; }
.fade-up-delay-3 { animation-delay: 0.3s; opacity: 0; }
.fade-up-delay-4 { animation-delay: 0.4s; opacity: 0; }

@media (prefers-reduced-motion: reduce) {
  .skeleton, .fade-up { animation: none; opacity: 1; }
  .progress-ring { transition: none; }
}

/* Tabular nums */
.tabular-nums {
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.02em;
}
</style>
