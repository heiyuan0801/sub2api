<template>
  <section class="demo-slot" :class="{ 'demo-slot-expanded': fullscreen }" :aria-label="copy.title">
    <dialog ref="fullDialog" class="demo-fullscreen" :aria-label="copy.title" @close="restoreInline">
      <div :id="fullscreenTarget" />
    </dialog>
    <Teleport defer :to="'#' + fullscreenTarget" :disabled="!fullscreen">
      <div class="demo-frame">
        <header class="demo-browser">
          <span class="window-dots" aria-hidden="true"><i /><i /><i /></span>
          <div class="demo-address" aria-hidden="true">
            <LockKeyhole :size="11" />
            <span>{{ address }}<span>/dashboard</span></span>
          </div>
          <div class="browser-actions">
            <span class="demo-readonly">{{ copy.readonly }}</span>
            <RouterLink :to="startPath" class="demo-primary">{{ t('publicHome.actions.getStarted') }}</RouterLink>
            <button
              ref="fullscreenButton"
              type="button"
              :aria-label="fullscreen ? copy.exitFullscreen : copy.fullscreen"
              :title="fullscreen ? copy.exitFullscreen : copy.fullscreen"
              @click="toggleFullscreen"
            >
              <Minimize2 v-if="fullscreen" :size="14" /><Maximize2 v-else :size="14" />
            </button>
            <button
              type="button"
              :aria-label="locked ? copy.unlock : copy.lock"
              :title="locked ? copy.unlock : copy.lock"
              :aria-pressed="locked"
              @click="toggleLock"
            >
              <LockKeyhole v-if="locked" :size="14" /><LockKeyholeOpen v-else :size="14" />
            </button>
          </div>
        </header>

        <div class="demo-content" :inert="showCover">
          <aside
            v-if="narrow ? mobileSidebar : !desktopSidebarHidden"
            class="demo-sidebar"
          >
            <div class="demo-brand"><img :src="siteLogo || '/logo.svg'" alt="" /><strong>{{ siteName }}</strong></div>
            <div class="demo-workspace"><span>{{ workspaceInitials }}</span><strong>{{ copy.workspace }}</strong></div>
            <label class="sidebar-search">
              <Search :size="13" aria-hidden="true" />
              <input v-model="filter" type="search" :placeholder="copy.searchNav" :aria-label="copy.searchNav" />
            </label>
            <nav class="demo-nav" :aria-label="copy.title">
              <template v-for="group in sidebarGroups" :key="group.label">
                <div v-if="group.screens.length" class="demo-nav-group">
                  <span>{{ group.label }}</span>
                  <button
                    v-for="screen in group.screens"
                    :key="screen.id"
                    type="button"
                    :aria-current="activeScreen === screen.id ? 'page' : undefined"
                    @click="openScreen(screen.id)"
                  >
                    <component :is="screen.icon" :size="15" aria-hidden="true" />
                    {{ screen.label }}
                  </button>
                </div>
              </template>
              <p v-if="filter && !sidebarGroups.some(group => group.screens.length)" class="demo-nav-empty">{{ t('common.noData') }}</p>
            </nav>
            <div class="sidebar-account"><span class="sidebar-avatar"><UserRound :size="15" /></span><strong>{{ copy.user }}</strong></div>
          </aside>

          <div class="demo-main">
            <div class="demo-toolbar">
              <button
                ref="sidebarButton"
                class="demo-icon"
                type="button"
                :aria-label="sidebarVisible ? t('nav.collapse') : t('nav.expand')"
                :aria-expanded="sidebarVisible"
                @click="toggleSidebar"
              ><PanelLeft :size="16" /></button>
              <span class="toolbar-divider" aria-hidden="true" />
              <span>{{ copy.workspace }}</span>
              <ChevronRight :size="13" aria-hidden="true" />
              <strong>{{ activeItem.label }}</strong>
              <span class="toolbar-spacer" />
              <RouterLink :to="startPath" class="toolbar-console">{{ t('publicSite.nav.console') }} <ArrowUpRight :size="13" /></RouterLink>
            </div>

            <div ref="scrollArea" class="demo-pane" tabindex="0" :aria-label="activeItem.label">
              <div class="demo-heading">
                <div>
                  <h2>{{ activeItem.label }}</h2>
                  <p>{{ activeItem.description }}</p>
                </div>
                <button v-if="activeScreen === 'overview'" type="button" class="demo-primary" @click="openScreen('keys')">
                  <Plus :size="14" />{{ t('publicHome.quickstart.createKey.title') }}
                </button>
              </div>

              <template v-if="activeScreen === 'overview' || activeScreen === 'usage'">
                <div class="demo-range">
                  <CalendarDays :size="13" aria-hidden="true" />
                  <select v-model.number="days" :aria-label="copy.range">
                    <option :value="7">{{ copy.lastWeek }}</option>
                    <option :value="30">{{ copy.lastMonth }}</option>
                  </select>
                </div>
                <div class="demo-metrics">
                  <div v-for="metric in metrics" :key="metric.label" class="metric-tile">
                    <div><span>{{ metric.label }}</span><component :is="metric.icon" :size="13" aria-hidden="true" /></div>
                    <strong>{{ metric.value }}</strong>
                    <small>{{ metric.note }}</small>
                  </div>
                </div>
              </template>

              <template v-if="activeScreen === 'overview'">
                <div class="demo-token-row">
                  <div v-for="metric in tokenMetrics" :key="metric.label"><span>{{ metric.label }}</span><strong>{{ metric.value }}</strong></div>
                </div>

                <div class="demo-overview">
                  <section class="demo-card demo-usage-chart">
                    <div class="card-heading">
                      <div><h3>{{ copy.usageOverview }}</h3><p>{{ copy.chartNote }}</p></div>
                      <div class="chart-switch">
                        <button
                          v-for="option in metricOptions"
                          :key="option.id"
                          type="button"
                          :aria-pressed="chartMetric === option.id"
                          @click="chartMetric = option.id"
                        >{{ option.label }}</button>
                      </div>
                    </div>
                    <div class="chart" role="img" :aria-label="chartSummary">
                      <div class="chart-axis" aria-hidden="true">
                        <span v-for="tick in chartTicks" :key="tick">{{ formatChartValue(tick) }}</span>
                      </div>
                      <div class="chart-bars" :style="{ '--bar-count': chartPoints.length }" aria-hidden="true">
                        <div v-for="(point, index) in chartPoints" :key="point.date" class="chart-column">
                          <i
                            :title="formatDate(point.date) + ' · ' + formatChartValue(point[chartMetric])"
                            :style="{ height: (point[chartMetric] / chartTop) * 100 + '%' }"
                          />
                          <span v-if="days === 7 || index % 7 === 0">{{ formatDate(point.date) }}</span>
                        </div>
                      </div>
                    </div>
                  </section>

                  <section class="demo-card shortcuts-card">
                    <div class="card-heading"><div><h3>{{ copy.quickActions }}</h3><p>{{ copy.shortcutsNote }}</p></div></div>
                    <button v-for="item in screens.slice(1)" :key="item.id" class="demo-shortcut" type="button" @click="openScreen(item.id)">
                      <component :is="item.icon" :size="15" />
                      {{ item.label }}
                      <ChevronRight :size="13" />
                    </button>
                  </section>
                </div>
              </template>

              <section v-if="activeScreen === 'usage'" class="demo-card demo-table">
                <div class="card-heading"><div><h3>{{ t('nav.usage') }}</h3><p>{{ copy.chartNote }}</p></div></div>
                <div class="demo-table-scroll" tabindex="0" :aria-label="t('nav.usage')">
                  <table>
                    <thead><tr><th scope="col">{{ copy.date }}</th><th scope="col">{{ copy.requests }}</th><th scope="col">Token</th><th scope="col">{{ copy.cost }}</th></tr></thead>
                    <tbody><tr v-for="point in [...chartPoints].reverse()" :key="point.date">
                      <td>{{ formatDate(point.date) }}</td><td>{{ formatNumber(point.requests) }}</td><td>{{ formatNumber(point.tokens, true) }}</td><td>{{ formatMoney(point.cost) }}</td>
                    </tr></tbody>
                  </table>
                </div>
              </section>

              <section v-if="activeScreen === 'models' || activeScreen === 'status'" class="demo-card demo-table">
                <div class="demo-table-scroll" tabindex="0" :aria-label="activeItem.label">
                  <table>
                    <thead><tr>
                      <th scope="col">{{ t('publicSite.modelDetail.modelId') }}</th>
                      <th scope="col">{{ t('publicSite.modelDetail.provider') }}</th>
                      <template v-if="activeScreen === 'status'">
                        <th scope="col">{{ copy.success }}</th><th scope="col">{{ copy.latency }}</th>
                      </template>
                      <template v-else>
                        <th scope="col">{{ t('publicSite.market.input') }}</th><th scope="col">{{ t('publicSite.market.output') }}</th>
                      </template>
                    </tr></thead>
                    <tbody><tr v-for="model in sampleModels" :key="model.name">
                      <td>{{ model.name }}</td>
                      <td><span class="demo-provider"><PlatformIcon :platform="model.platform" size="sm" />{{ platformLabel(model.platform) }}</span></td>
                      <template v-if="activeScreen === 'status'">
                        <td><span class="demo-status">{{ model.availability }}</span></td><td>{{ model.latency }}</td>
                      </template>
                      <template v-else>
                        <td>{{ formatMoney(model.input) }} / 1M</td><td>{{ formatMoney(model.output) }} / 1M</td>
                      </template>
                    </tr></tbody>
                  </table>
                </div>
              </section>

              <section v-if="activeScreen === 'keys'" class="demo-card demo-table">
                <div class="demo-table-scroll" tabindex="0" :aria-label="activeItem.label">
                  <table>
                    <thead><tr><th scope="col">{{ t('common.name') }}</th><th scope="col">{{ t('nav.apiKeys') }}</th><th scope="col">{{ t('common.status') }}</th></tr></thead>
                    <tbody><tr v-for="key in sampleKeys" :key="key.name">
                      <td>{{ key.name }}</td><td><code>•••• {{ key.mask }}</code></td><td><span class="demo-status">{{ copy.active }}</span></td>
                    </tr></tbody>
                  </table>
                </div>
              </section>
            </div>
          </div>
        </div>

        <div v-if="showCover" class="demo-cover">
          <div class="demo-cover-card">
            <Monitor :size="23" aria-hidden="true" />
            <strong>{{ copy.title }}</strong>
            <p>{{ copy.coverNote }}</p>
            <button ref="startButton" class="demo-primary" type="button" @click="startDemo"><Play :size="14" />{{ copy.start }}</button>
          </div>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, useId } from 'vue'
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useMediaQuery } from '@vueuse/core'
import {
  Activity, ArrowUpRight, BarChart3, CalendarDays, ChevronRight, Coins, Database, KeyRound, LayoutDashboard,
  LockKeyhole, LockKeyholeOpen, Maximize2, Minimize2, Monitor, PanelLeft, Play, Plus, Search, UserRound, Wallet,
} from '@lucide/vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { isLocaleCode } from '@/i18n/localeConfig'
import { platformLabel } from '@/utils/platformColors'
import { gatewayDemoCopy } from './gatewayDemoCopy'

withDefaults(defineProps<{ startPath: RouteLocationRaw; siteName?: string; siteLogo?: string }>(), {
  siteName: 'Sub2API',
  siteLogo: '',
})

const { t, locale } = useI18n()
const copy = computed(() => gatewayDemoCopy[isLocaleCode(locale.value) ? locale.value : 'en'])
const address = window.location.host
const workspaceInitials = 'DE'
const narrow = useMediaQuery('(max-width: 760px)')
const desktopSidebarHidden = ref(false)
const mobileSidebar = ref(false)
const activated = ref(false)
const locked = ref(false)
const showCover = computed(() => locked.value || (narrow.value && !activated.value))
const sidebarVisible = computed(() => narrow.value ? mobileSidebar.value : !desktopSidebarHidden.value)

type Screen = 'overview' | 'usage' | 'models' | 'keys' | 'status'
const activeScreen = ref<Screen>('overview')
const screens = computed(() => [
  { id: 'overview', label: t('nav.dashboard'), description: copy.value.overviewNote, icon: LayoutDashboard },
  { id: 'usage', label: t('nav.usage'), description: t('publicHome.services.usage.description'), icon: Activity },
  { id: 'models', label: t('publicSite.market.heading'), description: t('publicSite.market.featuredDescription'), icon: Database },
  { id: 'keys', label: t('nav.apiKeys'), description: t('publicHome.services.keys.description'), icon: KeyRound },
  { id: 'status', label: t('nav.channelStatus'), description: copy.value.statusNote, icon: BarChart3 },
] as const)
const activeItem = computed(() => screens.value.find(screen => screen.id === activeScreen.value) ?? screens.value[0])
const filter = ref('')
const sidebarGroups = computed(() => {
  const includesQuery = (screen: { label: string }) => screen.label.toLocaleLowerCase().includes(filter.value.trim().toLocaleLowerCase())
  return [
    { label: copy.value.workspaceLabel, screens: screens.value.slice(0, 2).filter(includesQuery) },
    { label: copy.value.management, screens: screens.value.slice(2).filter(includesQuery) },
  ]
})

const fullscreenTarget = useId()
const fullDialog = ref<HTMLDialogElement | null>(null)
const fullscreenButton = ref<HTMLButtonElement | null>(null)
const sidebarButton = ref<HTMLButtonElement | null>(null)
const startButton = ref<HTMLButtonElement | null>(null)
const scrollArea = ref<HTMLDivElement | null>(null)
const fullscreen = ref(false)

async function toggleFullscreen() {
  if (fullscreen.value) {
    fullDialog.value?.close()
    return
  }
  fullscreen.value = true
  await nextTick()
  fullDialog.value?.showModal()
  fullscreenButton.value?.focus()
}

async function restoreInline() {
  fullscreen.value = false
  await nextTick()
  fullscreenButton.value?.focus()
}

async function startDemo() {
  activated.value = true
  locked.value = false
  await nextTick()
  sidebarButton.value?.focus()
}

async function toggleLock() {
  if (showCover.value) {
    await startDemo()
    return
  }
  locked.value = true
  await nextTick()
  startButton.value?.focus()
}

function toggleSidebar() {
  if (narrow.value) mobileSidebar.value = !mobileSidebar.value
  else desktopSidebarHidden.value = !desktopSidebarHidden.value
}

async function openScreen(screen: Screen) {
  activeScreen.value = screen
  mobileSidebar.value = false
  await nextTick()
  const pane = scrollArea.value
  if (!pane) return
  pane.scrollTop = 0
  if (narrow.value) pane.focus({ preventScroll: true })
}

// One deterministic dataset supports the range filters, totals and usage rows.
// The demo stays independent of the signed-in account and never calls a usage API.
const sampleRequests = [
  45430, 52612, 51142, 59351, 47315, 38763, 40245, 47333, 50943, 61198,
  52651, 50132, 42782, 39082, 49132, 54432, 58908, 58453, 53301, 40273,
  36273, 49313, 60935, 56913, 51870, 50420, 35115, 52410, 43750, 38142,
]

const sampleUsage = sampleRequests.map((requests, index) => ({
  date: Date.UTC(2026, 8, index + 1),
  requests,
  tokens: requests * 3279,
  cost: Math.round(requests * 0.84) / 100,
}))

const sampleModels = [
  { name: 'gpt-5', platform: 'openai', input: 1.25, output: 10, availability: '99.98%', latency: '382 ms' },
  { name: 'claude-sonnet-4-5', platform: 'anthropic', input: 3, output: 15, availability: '99.91%', latency: '521 ms' },
  { name: 'deepseek-chat', platform: 'deepseek', input: 0.28, output: 0.42, availability: '99.82%', latency: '671 ms' },
] as const
const sampleKeys = [
  { name: 'web-app', mask: '3a2d' },
  { name: 'background-worker', mask: '6f90' },
  { name: 'research', mask: '9e13' },
]
const days = ref(7)
const chartMetric = ref<'cost' | 'requests'>('cost')
const metricOptions = computed(() => [
  { label: copy.value.cost, id: 'cost' },
  { label: copy.value.requests, id: 'requests' },
] as const)
const chartPoints = computed(() => sampleUsage.slice(-days.value))
const totals = computed(() => chartPoints.value.reduce(
  (sum, day) => ({ requests: sum.requests + day.requests, tokens: sum.tokens + day.tokens, cost: sum.cost + day.cost }),
  { requests: 0, tokens: 0, cost: 0 },
))

const chartStep = computed(() => chartMetric.value === 'cost' ? 150 : 20_000)
const chartTop = computed(() => Math.ceil(Math.max(...chartPoints.value.map(day => day[chartMetric.value])) / chartStep.value) * chartStep.value)
const chartTicks = computed(() => Array.from({ length: chartTop.value / chartStep.value + 1 }, (_, index) => chartTop.value - index * chartStep.value))
const chartSummary = computed(() => copy.value.usageOverview + ' · ' + metricOptions.value.find(option => option.id === chartMetric.value)?.label + ' · ' +
  chartPoints.value.map(point => formatDate(point.date) + ' ' + formatChartValue(point[chartMetric.value])).join(', '))

const metrics = computed(() => [
  { label: copy.value.balance, value: formatMoney(8420.5), note: copy.value.available, icon: Wallet },
  { label: copy.value.totalRequests, value: formatNumber(totals.value.requests), note: copy.value.rangeNote, icon: BarChart3 },
  { label: copy.value.totalCost, value: formatMoney(totals.value.cost), note: copy.value.rangeNote, icon: Coins },
  { label: copy.value.totalTokens, value: formatNumber(totals.value.tokens, true), note: copy.value.rangeNote, icon: Database },
])
const tokenMetrics = computed(() => [
  { label: copy.value.inputTokens, value: formatNumber(Math.round(totals.value.tokens * .78), true) },
  { label: copy.value.outputTokens, value: formatNumber(Math.round(totals.value.tokens * .22), true) },
  { label: copy.value.cachedTokens, value: formatNumber(Math.round(totals.value.tokens * .184), true) },
  { label: copy.value.topModel, value: sampleModels[0].name },
])

function formatNumber(value: number, compact = false) {
  return new Intl.NumberFormat(locale.value, {
    notation: compact ? 'compact' : 'standard',
    maximumFractionDigits: compact ? 1 : 0,
  }).format(value)
}

function formatDate(date: number) {
  return new Intl.DateTimeFormat(locale.value, { month: 'short', day: 'numeric', timeZone: 'UTC' }).format(date)
}

function formatMoney(value: number, fractions = 2) {
  return '$' + new Intl.NumberFormat(locale.value, { minimumFractionDigits: fractions, maximumFractionDigits: fractions }).format(value)
}

function formatChartValue(value: number) {
  return chartMetric.value === 'cost' ? formatMoney(value, 0) : formatNumber(value, true)
}
</script>

<style scoped>
.demo-slot {
  --demo-height: 802px;
  min-width: 0;
  scroll-margin-top: 90px;
}
.demo-slot-expanded { min-height: var(--demo-height); }
.demo-frame {
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100%;
  height: var(--demo-height);
  overflow: hidden;
  border: 1px solid var(--gateway-line, #e4e4e7);
  border-radius: 16px;
  color: var(--gateway-ink, #0a0a0a);
  background: var(--gateway-surface, #fff);
  box-shadow: 0 18px 50px -28px rgb(0 0 0 / .26);
  text-align: start;
}
.demo-fullscreen {
  position: fixed;
  inset: 0;
  width: calc(100vw - 32px);
  height: max-content;
  max-width: none;
  max-height: none;
  margin: auto;
  padding: 0;
  overflow: hidden;
  border: 0;
  border-radius: 16px;
  background: var(--gateway-surface, #fff);
}
.demo-fullscreen::backdrop { background: rgb(6 8 15 / .68); }
.demo-fullscreen .demo-frame { height: calc(100dvh - 32px); }
.demo-browser {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  min-height: 44px;
  gap: 12px;
  padding: 6px 12px;
  border-bottom: 1px solid var(--gateway-line, #e4e4e7);
  background: var(--gateway-soft, #fafafa);
}
.window-dots { display: flex; align-items: center; gap: 5px; flex-shrink: 0; }
.window-dots i { width: 8px; height: 8px; border-radius: 50%; background: color-mix(in srgb, var(--gateway-muted, #71717a) 44%, transparent); }
.demo-address {
  display: flex;
  gap: 7px;
  align-items: center;
  justify-content: center;
  min-width: 0;
  width: min(39%, 458px);
  min-height: 26px;
  padding: 2px 9px;
  margin-inline-start: clamp(0px, 9%, 122px);
  margin-inline-end: auto;
  border: 1px solid var(--gateway-line, #e4e4e7);
  border-radius: 7px;
  color: var(--gateway-muted, #71717a);
  background: var(--gateway-surface, #fff);
  font: 10px ui-monospace, SFMono-Regular, Consolas, monospace;
}
.demo-address > span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.demo-address span span { opacity: .85; }
.browser-actions { display: flex; align-items: center; gap: 7px; flex-shrink: 0; }
.demo-readonly {
  white-space: nowrap;
  color: var(--gateway-muted, #71717a);
  font: 600 9px ui-monospace, SFMono-Regular, Consolas, monospace;
  letter-spacing: .12em;
}
.demo-primary, .browser-actions button, .demo-icon, .toolbar-console, .chart-switch button, .demo-shortcut, .demo-nav button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}
.demo-primary {
  border: 1px solid transparent;
  border-radius: 7px;
  background: var(--gateway-ink, #0a0a0a);
  color: var(--gateway-surface, #fff);
  min-height: 30px;
  padding: 5px 10px;
  white-space: nowrap;
  font-size: 11px;
  font-weight: 600;
  text-decoration: none;
}
.demo-primary:hover { opacity: .82; }
.browser-actions button, .demo-icon {
  min-width: 28px;
  height: 28px;
  border: 1px solid var(--gateway-line, #e4e4e7);
  border-radius: 6px;
  background: var(--gateway-surface, #fff);
}
.browser-actions button:hover, .demo-icon:hover, .demo-shortcut:hover, .demo-nav button:hover {
  background: color-mix(in srgb, var(--gateway-ink, #0a0a0a) 6%, var(--gateway-surface, #fff));
}
.demo-frame :is(button, a, select, input, [tabindex]):focus-visible {
  outline: 2px solid var(--gateway-accent, #5756e8);
  outline-offset: 2px;
}
.demo-frame :is(button, a, select, input, [tabindex]) { scroll-margin-top: 80px; }
.demo-content { display: flex; position: relative; min-width: 0; min-height: 0; flex: 1 1 auto; }
.demo-sidebar {
  display: flex;
  flex-direction: column;
  flex: 0 0 min(23%, 256px);
  gap: 12px;
  min-width: 0;
  padding: 14px 13px;
  overflow-y: auto;
  border-right: 1px solid var(--gateway-line, #e4e4e7);
  background: var(--gateway-soft, #fafafa);
}
.demo-brand, .demo-workspace, .sidebar-account { display: flex; align-items: center; gap: 8px; min-height: 30px; font-size: 13px; }
.demo-brand img { width: 21px; height: 21px; object-fit: contain; }
.demo-workspace {
  padding: 6px;
  border: 1px solid var(--gateway-line, #e4e4e7);
  border-radius: 7px;
  background: var(--gateway-surface, #fff);
}
.demo-workspace span, .sidebar-avatar {
  display: grid;
  flex: 0 0 25px;
  width: 25px;
  height: 25px;
  place-items: center;
  border-radius: 5px;
  background: color-mix(in srgb, var(--gateway-accent, #5756e8) 12%, var(--gateway-surface, #fff));
  font-size: 10px;
  color: var(--gateway-ink, #0a0a0a);
}
.sidebar-search {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 7px;
  border: 1px solid var(--gateway-line, #e4e4e7);
  border-radius: 6px;
  color: var(--gateway-muted, #71717a);
  background: var(--gateway-surface, #fff);
}
.sidebar-search input { width: 100%; min-width: 0; color: var(--gateway-ink, #0a0a0a); background: transparent; border: 0; font-size: 12px; }
.sidebar-search input::placeholder { color: var(--gateway-muted, #71717a); }
.demo-nav-group { display: grid; gap: 2px; margin-bottom: 16px; }
.demo-nav-group > span { margin: 0 7px 4px; font-size: 11px; color: var(--gateway-muted, #71717a); }
.demo-nav-empty { margin: 5px 7px; font-size: 12px; color: var(--gateway-muted, #71717a); }
.demo-nav button {
  justify-content: flex-start;
  gap: 8px;
  min-height: 33px;
  padding: 6px 8px;
  border-radius: 6px;
  text-align: start;
  font-size: 12px;
}
.demo-nav button[aria-current="page"] { font-weight: 650; background: color-mix(in srgb, var(--gateway-ink, #0a0a0a) 8%, var(--gateway-soft, #fafafa)); }
.sidebar-account { margin-top: auto; padding-top: 9px; border-top: 1px solid var(--gateway-line, #e4e4e7); font-size: 11px; }
.demo-main { display: flex; flex: 1 1 auto; flex-direction: column; min-width: 0; }
.demo-toolbar {
  display: flex;
  flex: 0 0 62px;
  gap: 10px;
  align-items: center;
  min-width: 0;
  padding: 0 17px;
  border-bottom: 1px solid var(--gateway-line, #e4e4e7);
  color: var(--gateway-muted, #71717a);
  font-size: 12px;
}
.demo-toolbar strong { color: var(--gateway-ink, #0a0a0a); white-space: nowrap; }
.demo-toolbar .demo-icon { border-color: transparent; }
.toolbar-divider { height: 16px; width: 1px; background: var(--gateway-line, #e4e4e7); }
.toolbar-spacer { flex: 1 1 auto; }
.toolbar-console { flex-shrink: 0; font-size: 11px; }
.demo-pane {
  flex: 1 1 auto;
  padding: 20px;
  min-width: 0;
  overflow: auto;
  overscroll-behavior: contain;
  scrollbar-width: thin;
  scrollbar-color: var(--gateway-line, #e4e4e7) transparent;
}
.demo-heading, .card-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 10px; }
.demo-heading h2 { margin: 0; font-size: clamp(20px, 3vw, 30px); font-weight: 700; letter-spacing: -.035em; line-height: 1.06; }
.demo-heading p, .card-heading p { color: var(--gateway-muted, #71717a); margin: 6px 0 0; line-height: 1.4; font-size: 12px; }
.demo-range {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 16px;
  padding: 3px 8px;
  min-height: 29px;
  border: 1px solid var(--gateway-line, #e4e4e7);
  border-radius: 6px;
  font-size: 11px;
}
.demo-range select { border: 0; padding: 0 3px 0 0; background: var(--gateway-surface, #fff); color: var(--gateway-ink, #0a0a0a); cursor: pointer; }
.demo-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0,1fr));
  gap: 9px;
  margin-top: 12px;
}
.metric-tile, .demo-card { min-width: 0; border: 1px solid var(--gateway-line, #e4e4e7); border-radius: 9px; }
.metric-tile { padding: 12px; }
.metric-tile > div { display: flex; justify-content: space-between; align-items: center; gap: 4px; }
.metric-tile > div > span, .demo-token-row span {
  font-size: 9px;
  font-weight: 600;
  letter-spacing: .08em;
  text-transform: uppercase;
  color: var(--gateway-muted, #71717a);
}
.metric-tile > div > svg { flex-shrink: 0; color: var(--gateway-muted, #71717a); }
.metric-tile strong { display: block; margin-top: 8px; font-size: clamp(16px, 2.5vw, 24px); line-height: 1; font-weight: 700; font-variant-numeric: tabular-nums; letter-spacing: -.035em; }
.metric-tile small { display: block; margin-top: 6px; color: var(--gateway-muted, #71717a); font-size: 10px; }
.demo-token-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 1px;
  margin: 12px 0;
  border: 1px solid var(--gateway-line, #e4e4e7);
  border-radius: 9px;
  overflow: hidden;
  background: var(--gateway-line, #e4e4e7);
}
.demo-token-row > div { display: grid; padding: 7px 12px; gap: 3px; background: var(--gateway-surface, #fff); }
.demo-token-row strong { font-size: 14px; font-weight: 620; white-space: nowrap; }
.demo-overview { display: grid; grid-template-columns: minmax(0, 2.25fr) minmax(0, .9fr); gap: 12px; }
.demo-card { padding: 14px; background: var(--gateway-surface, #fff); }
.card-heading h3 { margin: 0; font-size: 13px; font-weight: 650; }
.card-heading p { margin: 3px 0 0; font-size: 11px; }
.chart-switch { display: flex; gap: 2px; padding: 2px; border: 1px solid var(--gateway-line, #e4e4e7); border-radius: 6px; background: var(--gateway-soft, #fafafa); }
.chart-switch button { padding: 3px 6px; border: 1px solid transparent; border-radius: 4px; color: var(--gateway-muted, #71717a); white-space: nowrap; font-size: 10px; }
.chart-switch button[aria-pressed="true"] { color: var(--gateway-ink, #0a0a0a); background: var(--gateway-surface, #fff); border-color: var(--gateway-line, #e4e4e7); }
.chart {
  display: grid;
  grid-template-columns: minmax(27px, max-content) minmax(0,1fr);
  gap: 8px;
  height: 202px;
  margin: 19px 0 17px;
}
.chart-axis { display: flex; flex-direction: column; justify-content: space-between; color: var(--gateway-muted, #71717a); font-size: 10px; }
.chart-bars {
  display: grid;
  grid-template-columns: repeat(var(--bar-count, 7), minmax(0, 1fr));
  align-items: end;
  background: repeating-linear-gradient(to top, var(--gateway-line, #e4e4e7) 0 1px, transparent 1px 25%);
}
.chart-column { display: flex; justify-content: center; align-items: flex-end; position: relative; height: 100%; min-width: 0; }
.chart-column i {
  display: block;
  width: 76%;
  max-width: 56px;
  border-radius: 3px 3px 0 0;
  background: color-mix(in srgb, var(--gateway-accent, #5756e8) 88%, #3c74ef);
}
.chart-column i:hover { opacity: .75; }
.chart-column span { position: absolute; bottom: -19px; color: var(--gateway-muted, #71717a); font-size: 9px; white-space: nowrap; }
.shortcuts-card .card-heading { margin-bottom: 9px; }
.demo-shortcut { width: 100%; padding: 7px; justify-content: flex-start; border-radius: 6px; font-size: 11px; }
.demo-shortcut > svg:last-child { margin-inline-start: auto; }
.demo-table { margin-top: 14px; padding: 0; overflow: hidden; }
.demo-table .card-heading { padding: 14px; }
.demo-table-scroll { overflow: auto; }
.demo-table table { width: 100%; border-collapse: collapse; font-size: 12px; white-space: nowrap; }
.demo-table :is(th, td) { padding: 11px 14px; text-align: start; }
.demo-table th { color: var(--gateway-muted, #71717a); background: var(--gateway-soft, #fafafa); font-weight: 580; }
.demo-table td { border-top: 1px solid var(--gateway-line, #e4e4e7); }
.demo-provider { display: inline-flex; align-items: center; gap: 7px; }
.demo-status { color: #15803d; }
:global(.dark) .demo-status { color: #4ade80; }

.demo-cover {
  position: absolute;
  inset: 44px 0 0;
  display: grid;
  place-items: center;
  padding: 12px;
  background: color-mix(in srgb, var(--gateway-surface, #fff) 48%, transparent);
  backdrop-filter: blur(2px);
}
.demo-cover-card {
  display: grid;
  justify-items: center;
  text-align: center;
  gap: 6px;
  max-width: 260px;
  padding: 18px;
  border: 1px solid var(--gateway-line, #e4e4e7);
  border-radius: 12px;
  background: var(--gateway-surface, #fff);
  box-shadow: 0 12px 38px -14px rgb(0 0 0 / .28);
}
.demo-cover-card strong { font-size: 15px; }
.demo-cover-card p { margin: 0 0 3px; color: var(--gateway-muted, #71717a); font-size: 11px; }

@media (max-width: 1000px) {
  .demo-address { margin-inline-start: min(3%, 22px); }
  .demo-overview { grid-template-columns: minmax(0, 1fr); }
  .shortcuts-card { display: none; }
}
@media (max-width: 760px) {
  .demo-slot { --demo-height: 540px; }
  .demo-frame { border-radius: 11px; }
  .demo-fullscreen { width: calc(100vw - 16px); border-radius: 11px; }
  .demo-fullscreen .demo-frame { height: calc(100dvh - 16px); }
  .demo-browser { gap: 7px; padding-inline: 8px; }
  .demo-address { display: none; }
  .browser-actions { margin-inline-start: auto; gap: 4px; }
  .browser-actions .demo-primary { display: none; }
  .demo-readonly { font-size: 8px; letter-spacing: .04em; }
  .demo-sidebar {
    position: absolute;
    z-index: 1;
    top: 0;
    bottom: 0;
    left: 0;
    width: min(215px, 78%);
    box-shadow: 9px 0 20px rgb(0 0 0 / .12);
  }
  .demo-toolbar { flex-basis: 42px; padding-inline: 8px; gap: 5px; }
  .demo-toolbar > span:first-of-type + span, .demo-toolbar > svg { display: none; }
  .demo-pane { padding: 12px; }
  .demo-heading { flex-wrap: wrap; }
  .demo-heading .demo-primary { display: none; }
  .demo-metrics, .demo-token-row { grid-template-columns: repeat(2, minmax(0,1fr)); }
  .metric-tile { padding: 9px; }
  .demo-card { padding: 9px; }
  .demo-table { padding: 0; }
  .chart { height: 140px; }
  .chart-column span { font-size: 8px; }
  .chart-column:nth-child(2n) span { display: none; }
  .chart-switch button { padding-inline: 3px; }
}
</style>
