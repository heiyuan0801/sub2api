<template>
  <AppLayout>
    <div class="availability-page space-y-5 pb-10">
      <header class="availability-header">
        <div class="monitor-header-controls">
          <h1 class="sr-only">{{ t('channelMonitorV2.channels.title') }}</h1>
          <div class="availability-ranges" role="group" :aria-label="t('channelMonitorV2.timeRange')">
            <button v-for="range in ranges" :key="range" type="button" :class="{ selected: filter.range === range }" :aria-pressed="filter.range === range" @click="setRange(range)">{{ t(`channelMonitorV2.ranges.${range}`) }}</button>
          </div>
          <p class="monitor-source">{{ t(monitorDisabled ? 'channelMonitorV2.availability.disabled' : 'channelMonitorV2.channels.source') }}</p>
        </div>
        <div class="monitor-header-controls">
          <div v-if="snapshot" class="availability-summary" :aria-label="t('channelMonitorV2.summaryAria')">
            <span>{{ t('channelMonitorV2.cards.availabilityValue', { value: successValue(snapshot.metrics) }) }}</span>
            <span aria-hidden="true">·</span>
            <span>{{ t('channelMonitorV2.metrics.cacheRateValue', { value: cacheValue(snapshot.metrics) }) }}</span>
          </div>
          <button class="availability-settings" type="button" :aria-label="t('common.refresh')" :title="t('common.refresh')" :disabled="loading || refreshing" @click="reload(false)">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading || refreshing }" />
          </button>
          <WebsiteStatusLink />
        </div>
      </header>
      <p class="sr-only" aria-live="polite">
        <template v-if="refreshing">{{ t('channelMonitorV2.updating') }}</template>
        <template v-else-if="snapshot?.coverage.data_through">{{ t('channelMonitorV2.updatedTo', { time: formatTime(snapshot.coverage.data_through) }) }}</template>
      </p>
      <div v-if="isAdmin || (snapshot && !snapshot.coverage.coverage_complete && !bootstrapActive)" class="flex flex-wrap items-center justify-end gap-2">
        <span v-if="snapshot && !snapshot.coverage.coverage_complete && !bootstrapActive" class="mr-auto text-xs text-gray-500 dark:text-gray-400">{{ t('channelMonitorV2.partialCoverage') }}</span>
        <template v-if="isAdmin">
          <router-link to="/admin/ops?open_alert_rules=1" class="availability-settings"><Icon name="bell" size="sm" />{{ t('channelMonitorV2.preferences.alerts') }}</router-link>
          <router-link to="/admin/channels/monitor" class="availability-settings"><Icon name="cog" size="sm" />{{ t('channelMonitorV2.availability.settings') }}</router-link>
        </template>
      </div>

      <div v-if="loadError" class="availability-notice" role="alert">
        <Icon name="exclamationCircle" size="sm" class="shrink-0" />
        <div class="min-w-0 flex-1">
          <p>{{ monitorDisabled ? t('channelMonitorV2.availability.disabled') : loadError }}</p>
          <p v-if="snapshot && !monitorDisabled" class="mt-1 text-xs">{{ t('channelMonitorV2.availability.stale') }}</p>
        </div>
        <button type="button" class="shrink-0 underline underline-offset-4" :disabled="refreshing" @click="reload(false)">{{ t('channelMonitorV2.availability.retry') }}</button>
      </div>

      <div v-if="dataStale && !monitorDisabled" class="availability-notice" role="status" data-testid="freshness-warning">
        <Icon name="clock" size="sm" class="shrink-0" />
        <div class="min-w-0 flex-1"><p>{{ t('channelMonitorV2.freshness.title') }}</p><p class="mt-1 text-xs">{{ t('channelMonitorV2.freshness.description', { minutes: Math.max(1, Math.floor(freshness.ageSeconds / 60)) }) }}</p></div>
        <button type="button" class="shrink-0 underline underline-offset-4" :disabled="refreshing" @click="reload(false)">{{ t('common.refresh') }}</button>
      </div>

      <div v-if="bootstrapActive" class="rounded-xl border border-gray-200 p-4 dark:border-dark-700" role="status" aria-live="polite">
        <p class="text-sm font-semibold">{{ t('channelMonitorV2.bootstrap.title') }}</p>
        <p class="mt-1 text-xs leading-relaxed text-gray-500 dark:text-gray-400">{{ t('channelMonitorV2.bootstrap.description') }}</p>
        <div class="mt-3 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700" role="progressbar" :aria-valuenow="bootstrapPercent" aria-valuemin="0" aria-valuemax="100" :aria-label="t('channelMonitorV2.bootstrap.working')">
          <div class="h-full rounded-full bg-gray-900 transition-[width] duration-500 dark:bg-gray-200" :style="{ width: `${bootstrapPercent}%` }" />
        </div>
      </div>

      <section class="availability-workspace" :aria-label="t('channelMonitorV2.channels.title')">
        <details class="channel-filters">
          <summary>{{ t('channelMonitorV2.cards.viewSettings') }}<Icon name="chevronDown" size="xs" /></summary>
          <div class="availability-platforms" role="group" :aria-label="t('channelMonitorV2.filters.platform')">
            <button type="button" :class="{ selected: filter.platforms.length === 0 }" :aria-pressed="filter.platforms.length === 0" :disabled="!dimensionsLoaded" data-testid="platform-all" @click="selectPlatforms([])">{{ t('channelMonitorV2.filters.allPlatforms') }}</button>
            <button v-for="platform in platformShortcuts" :key="platform.value" type="button" :class="{ selected: filter.platforms.length === 1 && filter.platforms[0] === platform.value }" :aria-pressed="filter.platforms.length === 1 && filter.platforms[0] === platform.value" :data-testid="`platform-${platform.value}`" @click="selectPlatforms([platform.value])">{{ platform.label }}</button>
          </div>
          <div class="availability-toolbar monitor-toolbar">
            <form class="availability-search" role="search" @submit.prevent="applySearch">
              <label for="monitor-group-search" class="sr-only">{{ t('channelMonitorV2.channels.search') }}</label>
              <input id="monitor-group-search" v-model="searchDraft" type="search" class="input min-w-0 flex-1" :placeholder="t('channelMonitorV2.channels.search')" @input="onSearchInput" />
              <button type="submit" class="availability-action"><Icon name="search" size="sm" />{{ t('channelMonitorV2.availability.search') }}</button>
            </form>
            <div class="flex min-w-0 flex-wrap items-center gap-2">
              <Select v-model="sortOrder" :options="sortOptions" :aria-label="t('channelMonitorV2.preferences.sortLabel')" class="availability-status" />
              <button type="button" class="availability-settings" :class="{ '!border-gray-900 !text-gray-950 dark:!border-gray-100 dark:!text-white': favoritesOnly }" :aria-pressed="favoritesOnly" @click="favoritesOnly = !favoritesOnly">{{ t('channelMonitorV2.preferences.favoritesOnly') }}</button>
              <FilterMultiSelect v-model="selectedGroupIds" compact :label="t('channelMonitorV2.filters.group')" :all-label="t('channelMonitorV2.availability.allGroups')" :options="groupOptions" />
              <Select v-model="statusFilter" :options="statusOptions" :aria-label="t('channelMonitorV2.availability.filterStatus')" class="availability-status" />
            </div>
          </div>
          <div class="availability-legend">
            <div class="flex flex-wrap items-center gap-x-4 gap-y-2">
              <span v-for="item in legendItems" :key="item.state" class="inline-flex items-center gap-1.5"><i class="h-2 w-2 rounded-sm" :class="`legend-${item.state}`" aria-hidden="true" />{{ item.label }}</span>
            </div>
            <span v-if="summary.noData || summary.insufficient">{{ t('channelMonitorV2.availability.pendingSummary', { empty: summary.noData, insufficient: summary.insufficient }) }}</span>
            <button v-if="hasDimensionFilter || appliedSearch || statusFilter !== 'all' || favoritesOnly" type="button" class="underline underline-offset-4" @click="resetAllFilters">{{ t('channelMonitorV2.clearFilters') }}</button>
          </div>
        </details>
        <div class="availability-results" :aria-busy="refreshing">
          <MonitorTrendChart v-if="trendView === 'line'" :trend="snapshot?.trend || []" :coverage="snapshot?.coverage || null" :loading="loading && !snapshot" />
          <MonitorChannelCards v-else-if="matrix" :rows="visibleMatrixRows" :coverage="matrix.coverage" :health-mode="healthMode" :show-throughput="showThroughput" :loading="loading" :favorite-keys="favoriteKeys" :stale="dataStale" :group-rates="groupRates" :refresh-seconds="refreshSeconds" @toggle-favorite="toggleFavorite" @inspect="inspectIncident" :empty-description="hasDimensionFilter || appliedSearch || statusFilter !== 'all' || favoritesOnly ? t('channelMonitorV2.channels.emptyFiltered') : undefined" @select-group="selectGroup" />
          <div v-else-if="loading" class="min-h-[280px] space-y-4" aria-hidden="true">
            <div v-for="i in 3" :key="i" class="h-28 animate-pulse rounded-xl bg-gray-100 dark:bg-dark-800" />
          </div>
          <div v-else class="py-16 text-center text-sm text-gray-500 dark:text-gray-400">{{ monitorDisabled ? t('channelMonitorV2.availability.disabledHint') : loadError || t('channelMonitorV2.empty.title') }}</div>
        </div>
      </section>

        <details class="group availability-advanced">
          <summary class="flex cursor-pointer list-none items-center justify-between gap-3 rounded-b-3xl px-5 py-3 text-xs font-medium text-gray-600 outline-offset-[-4px] hover:bg-gray-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-gray-400 dark:text-gray-300 dark:hover:bg-dark-800 sm:px-6">
            <span class="flex items-center gap-2">
              <Icon name="filter" size="sm" />{{ t('channelMonitorV2.cards.viewSettings') }}
              <span v-if="hasDimensionFilter" class="badge badge-gray">{{ t('channelMonitorV2.filters.selectedCount', { count: filter.platforms.length + filter.groupIds.length }) }}</span>
            </span>
            <Icon name="chevronDown" size="sm" class="transition-transform group-open:rotate-180" />
          </summary>
          <div class="monitor-toolbar flex flex-wrap items-center gap-2 border-t border-gray-100 px-5 py-4 dark:border-dark-700 sm:px-6">
            <FilterMultiSelect v-model="selectedPlatformIds" compact :label="t('channelMonitorV2.filters.platform')" :all-label="t('channelMonitorV2.filters.allPlatforms')" :options="platformOptions" />
            <FilterMultiSelect v-model="selectedGroupIds" compact :label="t('channelMonitorV2.filters.group')" :all-label="t('channelMonitorV2.filters.allGroups')" :options="groupOptions" />
            <button type="button" class="btn btn-ghost btn-sm !px-2 !py-1 text-xs" :disabled="!hasDimensionFilter" @click="clearDimensions">{{ t('channelMonitorV2.clearFilters') }}</button>
            <div class="tabs inline-flex" role="group" :aria-label="t('channelMonitorV2.trendView.label')">
              <button type="button" class="tab !px-3 !py-1.5 text-xs" :class="trendView === 'pulse' ? 'tab-active' : ''" :aria-pressed="trendView === 'pulse'" @click="trendView = 'pulse'">{{ t('channelMonitorV2.cards.cardView') }}</button>
              <button type="button" class="tab !px-3 !py-1.5 text-xs" :class="trendView === 'line' ? 'tab-active' : ''" :aria-pressed="trendView === 'line'" @click="trendView = 'line'">{{ t('channelMonitorV2.trendView.line') }}</button>
            </div>
            <div v-if="trendView === 'pulse'" class="tabs inline-flex flex-wrap" role="group" :aria-label="t('channelMonitorV2.healthMode.label')">
              <button v-for="option in healthModeOptions" :key="option.value" type="button" class="tab !px-2.5 !py-1.5 text-xs" :class="healthMode === option.value ? 'tab-active' : ''" :aria-pressed="healthMode === option.value" @click="healthMode = option.value">{{ option.label }}</button>
            </div>
          </div>
          <div class="px-5 pb-5 sm:px-6">
            <!-- Overview KPI: success · TTFT · tokens/s(optional) · cache · (+ RPM when throughput visible) -->
            <section
              v-if="snapshot"
              class="grid grid-cols-2 gap-3 sm:grid-cols-3"
              :class="showThroughput ? 'xl:grid-cols-5' : 'xl:grid-cols-4'"
              :aria-label="t('channelMonitorV2.summaryAria')"
            >
              <MetricCell
                :label="t('channelMonitorV2.metrics.successRate')"
                :value="successValue(snapshot.metrics)"
                :detail="t('channelMonitorV2.metrics.errorRateValue', { value: formatPercent(snapshot.metrics.error_rate) })"
                :state="dataStale ? 'unknown' : snapshot.health.success_rate || snapshot.health.error_rate"
              />
              <MetricCell
                :label="t('channelMonitorV2.metrics.ttftP50')"
                :value="formatMs(snapshot.metrics.ttft.p50_ms)"
                :detail="latencyKpiSecondary(snapshot.metrics.ttft)"
                :title="latencyDetail(snapshot.metrics.ttft)"
                :state="dataStale ? 'unknown' : ttftCellState(snapshot.health.ttft, snapshot.metrics.ttft)"
              />
              <MetricCell
                v-if="showThroughput"
                :label="t('channelMonitorV2.metrics.tps')"
                :value="formatTps(snapshot.metrics.tpm)"
                :detail="t('channelMonitorV2.metrics.tpsDetail')"
                :title="exactTps(snapshot.metrics.tpm)"
              />
              <MetricCell
                :label="t('channelMonitorV2.metrics.cacheRate')"
                :value="cacheValue(snapshot.metrics)"
                :detail="t('channelMonitorV2.metrics.cacheDetail')"
                :state="dataStale ? 'unknown' : snapshot.health.cache || snapshot.health.overall"
              />
              <MetricCell
                v-if="showThroughput"
                :label="t('channelMonitorV2.metrics.rpm')"
                :value="formatRate(snapshot.metrics.rpm)"
                :detail="t('channelMonitorV2.metrics.rpmDetail')"
                :title="exactRate(snapshot.metrics.rpm)"
              />
            </section>
            <section
              v-else-if="loading"
              class="grid grid-cols-2 gap-3 sm:grid-cols-3"
              :class="showThroughput ? 'xl:grid-cols-5' : 'xl:grid-cols-4'"
              aria-hidden="true"
            >
              <div
                v-for="i in (showThroughput ? 5 : 4)"
                :key="i"
                class="h-24 animate-pulse rounded-2xl bg-gray-50 dark:bg-dark-900/30"
              />
            </section>

          </div>
        </details>

      <details class="card min-h-0 overflow-hidden !rounded-3xl p-0 shadow-sm ring-1 ring-gray-900/5 dark:ring-dark-700">
        <summary class="flex cursor-pointer list-none items-center justify-between gap-3 px-5 py-4 text-sm font-semibold text-gray-900 outline-offset-[-4px] focus-visible:outline focus-visible:outline-2 focus-visible:outline-gray-400 dark:text-gray-100 sm:px-6">
          {{ t('channelMonitorV2.cards.details') }}
          <Icon name="chevronDown" size="sm" />
        </summary>
        <div class="border-b border-gray-100 px-5 pt-4 dark:border-dark-700 sm:px-6">
          <nav class="tabs w-full max-w-md sm:w-auto" role="tablist" :aria-label="t('channelMonitorV2.tabs.aria')">
            <button
              v-for="item in tabs"
              :key="item.value"
              type="button"
              role="tab"
              class="tab flex-1 sm:flex-none"
              :aria-selected="activeTab === item.value"
              :class="activeTab === item.value ? 'tab-active' : ''"
              @click="activeTab = item.value"
            >
              {{ item.label }}
            </button>
          </nav>
        </div>
        <div class="min-h-0 max-h-[min(52vh,520px)] overflow-auto p-4 sm:p-5">
          <div v-if="activeTab === 'errors'" class="space-y-3">
            <div
              v-for="row in errorRows"
              :key="row.category"
              class="rounded-2xl bg-gray-50 p-4 text-sm dark:bg-dark-900/30"
              :class="row.ignored ? 'opacity-60' : ''"
            >
              <button
                type="button"
                class="grid w-full grid-cols-[minmax(100px,200px)_1fr_auto_auto] items-center gap-3 text-left"
                @click="toggleError(row.category)"
              >
                <span class="flex min-w-0 items-center gap-1.5 truncate text-gray-700 dark:text-gray-200">
                  <span class="truncate">{{ errorLabel(row.category) }}</span>
                  <span v-if="row.ignored" class="badge badge-gray shrink-0 !px-1.5 !py-0 text-[10px]">{{ t('channelMonitorV2.ignored') }}</span>
                </span>
                <span class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                  <i
                    class="block h-full rounded-full"
                    :class="row.ignored ? 'bg-gray-400 dark:bg-gray-500' : 'bg-gradient-to-r from-red-400 to-red-500'"
                    :style="{ width: `${Math.max(2, row.rate * 100)}%` }"
                  ></i>
                </span>
                <small
                  class="w-14 text-right text-xs tabular-nums"
                  :class="row.ignored ? 'text-gray-400' : 'text-gray-500'"
                >{{ formatPercent(row.rate) }}</small>
                <Icon name="chevronDown" size="sm" :class="['text-gray-400 transition-transform', expandedErrors.has(row.category) ? 'rotate-180' : '']" />
              </button>
              <div v-if="expandedErrors.has(row.category)" class="mt-3 space-y-2 border-t border-gray-100 pt-3 dark:border-dark-700">
                <template v-if="isAdmin && (row.details || []).length">
                  <div
                    v-for="(detail, index) in row.details || []"
                    :key="`${row.category}:${index}:${detail.message}`"
                    class="rounded-lg bg-gray-50 px-3 py-2 text-xs text-gray-600 dark:bg-dark-900/50 dark:text-dark-300"
                  >
                    <div class="mb-1 flex flex-wrap items-center gap-2">
                      <span class="badge badge-gray !px-1.5 !py-0 text-[10px]">{{ detail.platform || '-' }}</span>
                      <span class="truncate font-medium">{{ detail.model || '-' }}</span>
                      <span v-if="detail.status_code" class="text-gray-400">{{ t('channelMonitorV2.errorDetail.http', { code: detail.status_code }) }}</span>
                      <span v-if="detail.upstream_status_code" class="text-gray-400">{{ t('channelMonitorV2.errorDetail.upstream', { code: detail.upstream_status_code }) }}</span>
                      <span class="ml-auto text-gray-400">×{{ detail.count }}</span>
                    </div>
                    <p class="break-words leading-relaxed">{{ detail.message || detail.error_type || t('channelMonitorV2.errorDetail.noMessage') }}</p>
                  </div>
                </template>
                <p v-else class="text-xs text-gray-400">{{ t('channelMonitorV2.errorDetail.empty') }}</p>
              </div>
            </div>
          </div>

          <div v-else class="table-container border-0">
            <table class="table monitor-table min-w-[640px]">
              <thead>
                <tr>
                  <th class="w-16">{{ t('channelMonitorV2.table.rank') }}</th>
                  <th>{{ t('channelMonitorV2.table.user') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.successRate') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.ttftP50') }}</th>
                  <th v-if="showThroughput">{{ t('channelMonitorV2.metrics.tps') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.cacheRate') }}</th>
                  <th v-if="showThroughput">{{ t('channelMonitorV2.metrics.rpm') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="row in userRows"
                  :key="row.user_id || row.display_label"
                  :class="row.is_self
                    ? 'bg-primary-50 ring-1 ring-inset ring-primary-200/80 dark:bg-primary-900/25 dark:ring-primary-700/50'
                    : ''"
                >
                  <td><MonitorRankBadge :rank="row.rank" /></td>
                  <td>
                    <strong
                      class="font-semibold"
                      :class="row.is_self ? 'text-primary-700 dark:text-primary-300' : 'text-gray-900 dark:text-white'"
                    >
                      {{ row.display_label }}
                      <span
                        v-if="row.is_self"
                        class="badge badge-primary ml-2 !px-1.5 !py-0 text-[10px]"
                      >{{ t('channelMonitorV2.currentUser') }}</span>
                    </strong>
                  </td>
                  <td>
                    <span class="block">{{ successValue(row.metrics, row) }}</span>
                    <small class="text-xs text-gray-400">{{ t('channelMonitorV2.metrics.errorRateValue', { value: formatPercent(row.metrics.error_rate) }) }}</small>
                  </td>
                  <td>
                    <span class="block">{{ formatMs(row.metrics.ttft.p50_ms) }}</span>
                    <small class="text-xs text-gray-400">{{ latencyDetail(row.metrics.ttft) }}</small>
                  </td>
                  <td v-if="showThroughput" :title="exactTps(row.metrics.tpm)">{{ formatTps(row.metrics.tpm) }}</td>
                  <td>{{ cacheValue(row.metrics, row) }}</td>
                  <td v-if="showThroughput">{{ formatRate(row.metrics.rpm) }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div v-if="tabLoading" class="empty-state py-10 text-sm text-gray-400">{{ t('common.loading') }}</div>
          <div v-else-if="activeRowsEmpty" class="empty-state py-10">
            <p class="empty-state-title text-base">
              {{
                bootstrapActive
                  ? t('channelMonitorV2.bootstrap.title')
                  : t('channelMonitorV2.empty.title')
              }}
            </p>
            <p class="empty-state-description">
              {{
                bootstrapActive
                  ? t('channelMonitorV2.bootstrap.description')
                  : t('channelMonitorV2.empty.description')
              }}
            </p>
          </div>
        </div>
      </details>
      <MonitorIncidentDialog :show="Boolean(incidentRow)" :row="incidentRow" :coverage="incidentCoverage" :window="incidentWindow" :metrics="incidentMetrics" :health="incidentHealth" :errors="incidentErrors" :loading="incidentLoading" :error="incidentError" :is-admin="isAdmin" :stale="dataStale" @close="closeIncident" @retry="loadIncident" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import WebsiteStatusLink from '@/components/user/monitor/WebsiteStatusLink.vue'
import Icon from '@/components/icons/Icon.vue'
import Select from '@/components/common/Select.vue'
import { CONCRETE_PLATFORM_OPTIONS } from '@/constants/platforms'
import FilterMultiSelect from '@/features/channel-monitor-v2/FilterMultiSelect.vue'
import MetricCell from '@/features/channel-monitor-v2/MetricCell.vue'
import MonitorRankBadge from '@/features/channel-monitor-v2/MonitorRankBadge.vue'
import MonitorTrendChart from '@/features/channel-monitor-v2/MonitorTrendChart.vue'
import MonitorChannelCards from '@/features/channel-monitor-v2/MonitorChannelCards.vue'
import MonitorIncidentDialog from '@/features/channel-monitor-v2/MonitorIncidentDialog.vue'
import { monitorFreshness, monitorRowKey, preferenceKey, readPreferences, savePreferences, sortMonitorRows, type MonitorSort } from '@/features/channel-monitor-v2/monitorPreferences'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { isChannelMonitorThroughputHidden, isChannelMonitorUserRankingHidden } from '@/utils/featureFlags'
import * as api from '@/api/channelMonitorV2'
import { getAvailable, getUserGroupRates } from '@/api/groups'
import { useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'
import type {
  HealthState,
  MonitorCoverage,
  MonitorDimensions,
  MonitorErrorRow,
  MonitorFilter,
  MonitorHealth,
  MonitorMatrixRow,
  MonitorMetric,
  MonitorMatrixResponse,
  MonitorRange,
  MonitorSnapshot,
  MonitorUserRow,
} from '@/api/channelMonitorV2'
import {
  formatLatencyKpiSecondary,
  formatLatencyPrivacy,
  formatMonitorMs,
  formatMonitorPercent,
  formatMonitorThroughput,
  formatMonitorTokensPerSecond,
  tokensPerSecondFromTpm,
  monitorErrorCategoryLabel,
  ttftDisplayState,
} from '@/features/channel-monitor-v2/monitorFormat'

type Tab = 'errors' | 'users'
const MATRIX_GROUP_BY = 'platform_group' as const
type HealthMode = 'overall' | 'success' | 'ttft' | 'cache'
type TrendView = 'pulse' | 'line'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()
const { t, te, locale } = useI18n()
const { providerLabel } = useChannelMonitorFormat()
const isAdmin = computed(() => authStore.isAdmin)
/** Admins always see RPM/TPM; users honor the hide-throughput system setting. */
const showThroughput = computed(() => isAdmin.value || !isChannelMonitorThroughputHidden())
/** Admins always see ranking; users honor the hide-user-ranking system setting. */
const showUserRanking = computed(() => isAdmin.value || !isChannelMonitorUserRankingHidden())

const ranges: MonitorRange[] = ['90m', '24h', '7d', '30d']
const tabs = computed(() => {
  const items: Array<{ value: Tab; label: string }> = [{ value: 'errors', label: t('channelMonitorV2.tabs.errors') }]
  if (showUserRanking.value) items.push({ value: 'users', label: t('channelMonitorV2.tabs.users') })
  return items
})
const healthModeOptions = computed(() => [
  { value: 'overall' as HealthMode, label: t('channelMonitorV2.healthMode.overall') },
  { value: 'success' as HealthMode, label: t('channelMonitorV2.healthMode.success') },
  { value: 'ttft' as HealthMode, label: t('channelMonitorV2.healthMode.ttft') },
  { value: 'cache' as HealthMode, label: t('channelMonitorV2.healthMode.cache') },
])

const saved = readPreferences(preferenceKey(authStore.user?.id))
function initialValue(key: keyof typeof saved): unknown {
  return Object.prototype.hasOwnProperty.call(route.query, key) ? route.query[key] : saved[key]
}
const filter = ref<MonitorFilter>({
  range: parseRange(initialValue('range')),
  platforms: csv(initialValue('platform')),
  groupIds: csv(initialValue('group')).map(Number).filter(id => Number.isSafeInteger(id) && id > 0),
  models: [],
})
const activeTab = ref<Tab>(parseTab(initialValue('tab'), showUserRanking.value))
const healthMode = ref<HealthMode>(parseHealthMode(initialValue('health_mode')))
const trendView = ref<TrendView>(parseTrendView(initialValue('trend_view')))
const dimensions = ref<MonitorDimensions>({ platforms: [], groups: [], models: [] })
const dimensionsLoaded = ref(false)
const snapshot = ref<MonitorSnapshot | null>(null)
const matrix = ref<MonitorMatrixResponse | null>(null)
const errorRows = ref<MonitorErrorRow[]>([])
const userRows = ref<MonitorUserRow[]>([])
const loading = ref(false)
const tabLoading = ref(false)
const refreshing = ref(false)
const loadError = ref('')
const monitorDisabled = ref(false)
type AvailabilityStatus = 'all' | 'healthy' | 'warning' | 'critical' | 'noData' | 'insufficient'
const legacyGrouping = initialValue('group_by')
const searchDraft = ref(!['platform_group_model', 'platform_model'].includes(String(legacyGrouping)) && typeof initialValue('q') === 'string' ? String(initialValue('q')) : '')
const appliedSearch = ref(searchDraft.value.trim())
const statusFilter = ref<AvailabilityStatus>(parseStatus(initialValue('status')))
const sortOrder = ref<MonitorSort>(['health', 'success', 'latency', 'name'].includes(String(initialValue('sort'))) ? initialValue('sort') as MonitorSort : 'health')
const favoritesOnly = ref(initialValue('favorites_only') === true || initialValue('favorites_only') === '1')
// Promote older model favorites to their corresponding channel group.
const favoriteKeys = ref([...new Set((saved.favorites || []).flatMap(key => {
  const [platform, , groupId] = JSON.parse(key)
  return groupId ? [monitorRowKey({ platform, group_id: groupId })] : []
}))])
const groupRates = ref<Record<number, number>>({})
const nextRefresh = ref(0)
const refreshSeconds = computed(() => Math.max(0, Math.ceil((nextRefresh.value - now.value) / 1000)))
const now = ref(Date.now())
const freshness = computed(() => monitorFreshness(snapshot.value?.coverage, snapshot.value?.config.refresh_interval_seconds || 300, now.value))
const dataStale = computed(() => !monitorDisabled.value && freshness.value.stale)
const sortOptions = computed(() => (['health', 'success', 'latency', 'name'] as const).map(value => ({ value, label: t(`channelMonitorV2.preferences.sort.${value}`) })))
const expandedErrors = ref(new Set<string>())
const incidentRow = ref<MonitorMatrixRow | null>(null)
const incidentWindow = ref<{start: string; end: string} | null>(null)
const incidentCoverage = ref<MonitorCoverage | null>(null)
const incidentMetrics = ref<MonitorMetric | null>(null)
const incidentHealth = ref<MonitorHealth | null>(null)
const incidentErrors = ref<MonitorErrorRow[]>([])
const incidentLoading = ref(false)
const incidentError = ref('')
let incidentController: AbortController | null = null
let incidentSequence = 0
let freshnessTimer: number | null = null
let controller: AbortController | null = null
let sequence = 0
let autoRefreshTimer: number | null = null

const hasDimensionFilter = computed(
  () => filter.value.platforms.length + filter.value.groupIds.length > 0
)
// Full platform catalog (never pruned). Groups/models cascade by selected platforms
// so choosing a platform narrows the other pickers without collapsing platforms.
const platformOptions = computed(() =>
  (dimensions.value.platforms || []).map((item) => ({
    value: item.value,
    label: item.label && item.label !== item.value
      ? item.label
      : item.value === 'anthropic'
        ? 'Claude'
        : CONCRETE_PLATFORM_OPTIONS.find(option => option.value === item.value)?.label || item.value,
  }))
)
const platformShortcuts = computed(() => {
  const priority = ['openai', 'anthropic', 'grok', 'deepseek']
  return [...platformOptions.value].sort((a, b) => {
    const rank = (value: string) => priority.includes(value) ? priority.indexOf(value) : priority.length
    return rank(a.value) - rank(b.value) || a.label.localeCompare(b.label)
  })
})
function selectPlatforms(platforms: string[]) {
  const groups = new Set(dimensions.value.groups.filter(item => groupInScope(item, platforms)).map(item => item.id))
  const groupIds = dimensionsLoaded.value ? filter.value.groupIds.filter(id => groups.has(id)) : filter.value.groupIds
  filter.value = { ...filter.value, platforms, groupIds }
}
const selectedPlatformIds = computed({ get: () => filter.value.platforms, set: selectPlatforms })
function groupInScope(item: MonitorDimensions['groups'][number], platforms: string[]) {
  if (!platforms.length) return true
  return item.platforms === undefined
    ? !item.platform || platforms.includes(item.platform)
    : item.platforms.some(platform => platforms.includes(platform))
}
const groupOptions = computed(() =>
  (dimensions.value.groups || [])
    .filter(item => groupInScope(item, filter.value.platforms))
    .map((item) => ({
      value: String(item.id),
      label: item.platform ? `${item.platform} / ${item.name || `#${item.id}`}` : item.name || `#${item.id}`,
    }))
)
const selectedGroupIds = computed({
  get: () => filter.value.groupIds.map(String),
  set: (value: string[]) => { filter.value.groupIds = value.map(Number).filter(id => Number.isSafeInteger(id) && id > 0) },
})
watch([dimensionsLoaded, groupOptions], () => {
  if (!dimensionsLoaded.value) return
  const groups = new Set(groupOptions.value.map(item => item.value))
  const groupIds = filter.value.groupIds.filter(id => groups.has(String(id)))
  if (groupIds.length !== filter.value.groupIds.length) filter.value.groupIds = groupIds
}, { flush: 'post' })
const activeRowsEmpty = computed(() => (activeTab.value === 'errors' ? errorRows.value : userRows.value).length === 0)
/** First-upgrade backfill toward 90m/24h/7d/30d; banner hides when backend omits bootstrap. */
const bootstrapActive = computed(() => Boolean(snapshot.value?.coverage?.bootstrap?.active))
const bootstrapPercent = computed(() => {
  const raw = snapshot.value?.coverage?.bootstrap?.progress_percent
  if (typeof raw !== 'number' || Number.isNaN(raw)) return 0
  return Math.min(100, Math.max(0, Math.round(raw)))
})
const matrixRows = computed(() => (matrix.value?.items || []).filter(row => row.group_id && row.group_id > 0 && !row.model))

function hasMetricData(metrics: MonitorMetric, row?: { metrics: MonitorMetric; buckets?: unknown[] }) {
  return metrics.has_data ?? Boolean(row?.buckets?.length || metrics.request_count > 0 || metrics.ttft.p50_ms != null || metrics.duration.p50_ms != null || (metrics.success_rate ?? 0) > 0 || metrics.error_rate > 0)
}
function successValue(metrics: MonitorMetric, row?: { metrics: MonitorMetric; buckets?: unknown[] }) {
  if (!hasMetricData(metrics, row)) return '—'
  return formatPercent(metrics.success_rate ?? 1 - metrics.error_rate)
}
function cacheValue(metrics: MonitorMetric, row?: { metrics: MonitorMetric; buckets?: unknown[] }) {
  if (!hasMetricData(metrics, row) || metrics.cache_rate_available === false) return '—'
  return formatPercent(metrics.cache_rate)
}
function rowAvailability(row: MonitorMatrixRow): Exclude<AvailabilityStatus, 'all'> {
  if (!hasMetricData(row.metrics, row)) return 'noData'
  return row.health.overall === 'unknown' ? 'insufficient' : row.health.overall
}
function parseStatus(value: unknown): AvailabilityStatus {
  return ['healthy', 'warning', 'critical', 'noData', 'insufficient'].includes(String(value)) ? value as AvailabilityStatus : 'all'
}
const searchedMatrixRows = computed(() => {
  const query = appliedSearch.value.toLocaleLowerCase()
  return matrixRows.value.filter(row =>
    !query || [row.platform, providerLabel(row.platform), row.group_name].some(value => value?.toLocaleLowerCase().includes(query))
  )
})
const visibleMatrixRows = computed(() => sortMonitorRows(searchedMatrixRows.value.filter(row =>
  (statusFilter.value === 'all' || rowAvailability(row) === statusFilter.value) &&
  (!favoritesOnly.value || favoriteKeys.value.includes(monitorRowKey(row)))
), sortOrder.value, favoriteKeys.value))
const summary = computed(() => {
  const counts = { total: 0, healthy: 0, warning: 0, critical: 0, noData: 0, insufficient: 0 }
  for (const row of searchedMatrixRows.value) { counts.total++; counts[rowAvailability(row)]++ }
  return counts
})
const legendItems = computed(() => [
  { state: 'healthy', label: t('channelMonitorV2.availability.healthy') },
  { state: 'warning', label: t('channelMonitorV2.availability.warning') },
  { state: 'critical', label: t('channelMonitorV2.availability.critical') },
  { state: 'unknown', label: t('channelMonitorV2.availability.unknownLegend') },
])
const statusOptions = computed(() => [
  { value: 'all', label: t('channelMonitorV2.availability.allStatuses') },
  ...(['healthy', 'warning', 'critical', 'noData', 'insufficient'] as const).map(value => ({ value, label: t(`channelMonitorV2.availability.${value}`) })),
])
function applySearch() { appliedSearch.value = searchDraft.value.trim() }
function onSearchInput() { if (!searchDraft.value) appliedSearch.value = '' }
function resetAllFilters() { searchDraft.value = ''; appliedSearch.value = ''; statusFilter.value = 'all'; favoritesOnly.value = false; clearDimensions() }
function selectGroup(row: MonitorMatrixRow) {
  if (row.group_id) selectedGroupIds.value = [String(row.group_id)]
}
function closeIncident() {
  incidentController?.abort()
  incidentSequence++
  incidentRow.value = null
  incidentLoading.value = false
}
function inspectIncident(selection: {row: MonitorMatrixRow; bucketStart?: string; bucketEnd?: string}) {
  const coverage = matrix.value?.coverage
  if (!coverage) return
  const start = selection.bucketStart || coverage.requested_start
  const end = selection.bucketEnd || coverage.requested_end || coverage.data_through
  if (!Number.isFinite(Date.parse(start)) || !Number.isFinite(Date.parse(end)) || Date.parse(start) >= Date.parse(end)) return
  incidentRow.value = selection.row
  incidentCoverage.value = coverage
  incidentWindow.value = { start, end }
  void loadIncident()
}
async function loadIncident() {
  if (!incidentRow.value || !incidentWindow.value) return
  incidentController?.abort()
  const request = new AbortController()
  incidentController = request
  const id = ++incidentSequence
  const row = incidentRow.value
  const scoped: MonitorFilter = {
    range: filter.value.range, platforms: [row.platform],
    groupIds: row.group_id ? [row.group_id] : [...filter.value.groupIds],
    models: [], ...incidentWindow.value,
  }
  incidentLoading.value = true
  incidentError.value = ''
  incidentErrors.value = []
  incidentMetrics.value = null
  incidentHealth.value = null
  try {
    const [metrics, errors] = await Promise.all([
      api.getMatrix(scoped, MATRIX_GROUP_BY, isAdmin.value, request.signal),
      api.getErrors(scoped, isAdmin.value, request.signal),
    ])
    if (id !== incidentSequence) return
    const result = metrics.items.find(item => monitorRowKey(item) === monitorRowKey(row))
    incidentCoverage.value = metrics.coverage
    incidentMetrics.value = result?.metrics || null
    incidentHealth.value = result?.health || null
    incidentErrors.value = errors.items || []
  } catch (error) {
    const e = error as {name?: string; code?: string}
    if (id === incidentSequence && e.name !== 'AbortError' && e.name !== 'CanceledError' && e.code !== 'ERR_CANCELED') incidentError.value = extractApiErrorMessage(error, t('channelMonitorV2.detailLoadFailed'))
  } finally {
    if (id === incidentSequence) incidentLoading.value = false
  }
}
function handleLoadFailure(error: unknown, id: number) {
  const e = error as { name?: string; code?: string; reason?: string }
  if (id !== sequence || e.name === 'AbortError' || e.name === 'CanceledError' || e.code === 'ERR_CANCELED') return
  monitorDisabled.value = e.reason === 'CHANNEL_MONITOR_DISABLED'
  loadError.value = extractApiErrorMessage(error, t('channelMonitorV2.loadFailed'))
  if (monitorDisabled.value) { closeIncident(); snapshot.value = null; matrix.value = null; errorRows.value = []; userRows.value = [] }
  scheduleAutoRefresh()
}
function onVisibilityChange() {
  if (document.visibilityState === 'visible' && !loading.value && !refreshing.value) void reload(true)
}

function csv(value: unknown) {
  return Array.isArray(value) ? value.map(String).filter(Boolean) : typeof value === 'string' ? value.split(',').filter(Boolean) : []
}
function parseRange(value: unknown): MonitorRange {
  return ['90m', '24h', '7d', '30d'].includes(String(value)) ? (value as MonitorRange) : '90m'
}
function parseTab(value: unknown, allowUsers: boolean): Tab {
  return value === 'users' && allowUsers ? 'users' : 'errors'
}
function parseHealthMode(value: unknown): HealthMode {
  const allowed: HealthMode[] = ['overall', 'success', 'ttft', 'cache']
  return allowed.includes(value as HealthMode) ? (value as HealthMode) : 'overall'
}
function parseTrendView(value: unknown): TrendView {
  return value === 'line' ? 'line' : 'pulse'
}
function persistPreferences() {
  savePreferences(preferenceKey(authStore.user?.id), {
    range: filter.value.range, platform: filter.value.platforms, group: filter.value.groupIds,
    q: appliedSearch.value, status: statusFilter.value, group_by: MATRIX_GROUP_BY, health_mode: healthMode.value,
    trend_view: trendView.value, tab: activeTab.value, sort: sortOrder.value,
    favorites_only: favoritesOnly.value, favorites: favoriteKeys.value,
  })
}
function toggleFavorite(row: MonitorMatrixRow) {
  const key = monitorRowKey(row)
  favoriteKeys.value = favoriteKeys.value.includes(key) ? favoriteKeys.value.filter(item => item !== key) : [...favoriteKeys.value, key].slice(-500)
}
function syncQuery() {
  persistPreferences()
  void router.replace({
    query: {
      q: appliedSearch.value || undefined,
      status: statusFilter.value !== 'all' ? statusFilter.value : undefined,
      sort: sortOrder.value !== 'health' ? sortOrder.value : undefined,
      favorites_only: favoritesOnly.value ? '1' : undefined,
      range: filter.value.range,
      platform: filter.value.platforms.join(',') || undefined,
      group: filter.value.groupIds.join(',') || undefined,
      group_by: MATRIX_GROUP_BY,
      health_mode: healthMode.value,
      trend_view: trendView.value === 'line' ? 'line' : undefined,
      tab: activeTab.value,
    },
  })
}
/** Dimensions catalog: range only — never re-filtered by platform/group/model selection. */
async function loadDimensions(signal?: AbortSignal, id = sequence) {
  const rangeOnly: MonitorFilter = {
    range: filter.value.range,
    platforms: [],
    groupIds: [],
    models: [],
  }
  const next = await api.getDimensions(rangeOnly, isAdmin.value, signal)
  if (id !== sequence) return
  dimensions.value = next
  dimensionsLoaded.value = true
}

async function loadMetrics(signal?: AbortSignal, id = sequence) {
  const [nextSnapshot, nextMatrix] = await Promise.all([
    api.getSnapshot(filter.value, isAdmin.value, signal),
    api.getMatrix(filter.value, MATRIX_GROUP_BY, isAdmin.value, signal),
  ])
  if (id !== sequence) return
  now.value = Date.now()
  snapshot.value = nextSnapshot
  matrix.value = nextMatrix
  loadError.value = ''
  monitorDisabled.value = false
  scheduleAutoRefresh()
  await loadTab(signal, id)
}

async function reload(silent = true) {
  controller?.abort()
  const request = new AbortController()
  controller = request
  const id = ++sequence
  refreshing.value = true
  if (!silent) loading.value = true
  try {
    // Catalog + metrics in parallel; catalog ignores dimension filters so options never shrink.
    await Promise.all([
      loadDimensions(request.signal, id),
      loadMetrics(request.signal, id),
    ])
  } catch (error) {
    handleLoadFailure(error, id)
  } finally {
    if (id === sequence) {
      loading.value = false
      tabLoading.value = false
      refreshing.value = false
    }
  }
}

/** When only range changes, still refresh dimensions; dimension filters only re-load metrics. */
async function reloadMetricsOnly(silent = true) {
  controller?.abort()
  const request = new AbortController()
  controller = request
  const id = ++sequence
  refreshing.value = true
  if (!silent) loading.value = true
  try {
    await loadMetrics(request.signal, id)
  } catch (error) {
    handleLoadFailure(error, id)
  } finally {
    if (id === sequence) {
      loading.value = false
      tabLoading.value = false
      refreshing.value = false
    }
  }
}
async function loadTab(signal?: AbortSignal, id = sequence) {
  tabLoading.value = true
  try {
    if (activeTab.value === 'errors') {
      const result = await api.getErrors(filter.value, isAdmin.value, signal)
      if (id === sequence && activeTab.value === 'errors') errorRows.value = result.items || []
    } else if (showUserRanking.value) {
      const result = await api.getUsers(filter.value, isAdmin.value, signal)
      if (id === sequence && activeTab.value === 'users') userRows.value = result.items || []
    } else {
      userRows.value = []
    }
  } catch (error) {
    const e = error as { name?: string; code?: string }
    if (e?.name === 'AbortError' || e?.name === 'CanceledError' || e?.code === 'ERR_CANCELED') return
    if (id === sequence) appStore.showError(extractApiErrorMessage(error, t('channelMonitorV2.detailLoadFailed')))
  } finally {
    if (id === sequence) tabLoading.value = false
  }
}
function setRange(value: MonitorRange) {
  filter.value.range = value
}
function clearDimensions() {
  // Replace arrays so deep watch always fires and metrics reload full window.
  filter.value = {
    ...filter.value,
    platforms: [],
    groupIds: [],
    models: [],
  }
}
function scheduleAutoRefresh() {
  if (autoRefreshTimer) {
    window.clearInterval(autoRefreshTimer)
    autoRefreshTimer = null
  }
  // Poll faster while first-upgrade bootstrap is filling 90m→30d so the progress bar moves.
  const seconds = bootstrapActive.value
    ? 10
    : snapshot.value?.config?.refresh_interval_seconds || 300
  nextRefresh.value = Date.now() + seconds * 1000
  autoRefreshTimer = window.setInterval(() => {
    if (document.visibilityState !== 'hidden' && !loading.value && !refreshing.value) {
      void reload(true)
    }
  }, Math.max(bootstrapActive.value ? 10 : 60, seconds) * 1000)
}
function formatRate(value: number) {
  return formatMonitorThroughput(value)
}
function exactRate(value: number) {
  return Intl.NumberFormat(locale.value || undefined, { maximumFractionDigits: 2 }).format(value || 0)
}
function formatTps(tpm: number | null | undefined) {
  return formatMonitorTokensPerSecond(tpm)
}
function exactTps(tpm: number | null | undefined) {
  return Intl.NumberFormat(locale.value || undefined, { maximumFractionDigits: 3 }).format(
    tokensPerSecondFromTpm(tpm),
  )
}
function formatPercent(value: number) {
  return formatMonitorPercent(value)
}
function formatMs(value: number | null) {
  return formatMonitorMs(value)
}
function ttftCellState(state: HealthState | undefined, metric: { p50_ms: number | null; sample_count?: number }) {
  return ttftDisplayState(state, metric)
}
function latencyDetail(metric: {
  p50_ms: number | null
  p90_ms?: number | null
  p95_ms: number | null
  avg_ms?: number | null
}) {
  return formatLatencyPrivacy(metric.p50_ms, metric.p90_ms, metric.avg_ms, metric.p95_ms)
}
/** KPI secondary: AVG · P90 under the P50 primary value. */
function latencyKpiSecondary(metric: {
  p90_ms?: number | null
  p95_ms: number | null
  avg_ms?: number | null
}) {
  return formatLatencyKpiSecondary(metric.avg_ms, metric.p90_ms, metric.p95_ms)
}
function formatTime(value: string) {
  if (!Number.isFinite(Date.parse(value))) return '—'
  return new Intl.DateTimeFormat(locale.value || undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}
function errorLabel(value: string) {
  const key = `channelMonitorV2.errorCategories.${value}`
  return te(key) ? t(key) : monitorErrorCategoryLabel(value)
}
function toggleError(category: string) {
  const next = new Set(expandedErrors.value)
  if (next.has(category)) next.delete(category)
  else next.add(category)
  expandedErrors.value = next
}

let lastRange: MonitorRange = filter.value.range
watch(
  filter,
  () => {
    closeIncident()
    syncQuery()
    const rangeChanged = filter.value.range !== lastRange
    lastRange = filter.value.range
    if (rangeChanged) void reload(true)
    else void reloadMetricsOnly(true)
  },
  { deep: true }
)
watch(healthMode, syncQuery)
watch(trendView, syncQuery)
watch(activeTab, () => {
  syncQuery()
  void loadTab()
})
watch(showUserRanking, (allowed) => {
  if (!allowed && activeTab.value === 'users') {
    activeTab.value = 'errors'
  }
})
watch([appliedSearch, statusFilter, sortOrder, favoritesOnly], syncQuery)
watch(favoriteKeys, persistPreferences)
onMounted(() => {
  freshnessTimer = window.setInterval(() => { now.value = Date.now() }, 1000)
  document.addEventListener('visibilitychange', onVisibilityChange)
  void reload(false)
  void Promise.allSettled([getAvailable(), getUserGroupRates()]).then(([groups, overrides]) => {
    const baseRates = groups.status === 'fulfilled' ? Object.fromEntries(groups.value.map(group => [group.id, group.rate_multiplier])) : {}
    groupRates.value = { ...baseRates, ...(overrides.status === 'fulfilled' ? overrides.value : {}) }
  })
})
onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', onVisibilityChange)
  controller?.abort()
  incidentController?.abort()
  if (freshnessTimer) window.clearInterval(freshnessTimer)
  if (autoRefreshTimer) window.clearInterval(autoRefreshTimer)
})
</script>

<style scoped>
.availability-page {
  --channel-text: #171a22;
  --channel-muted: #636874;
  --channel-border: #e0e1e5;
  --channel-surface: #fff;
  --channel-subtle: #f3f4f6;
  --channel-gap: #cbd0d8;
  max-width: 104rem;
  margin-inline: auto;
  color: var(--channel-text);
}
.dark .availability-page {
  --channel-text: #f4f5f9;
  --channel-muted: #aab2c0;
  --channel-border: #343843;
  --channel-surface: #191b23;
  --channel-subtle: #252833;
  --channel-gap: #535968;
}
.availability-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: .65rem 1rem;
  padding: .65rem .85rem;
  border: 1px solid var(--channel-border);
  border-radius: 10px;
  background: var(--channel-surface);
  box-shadow: 0 1px 2px rgb(15 23 42 / 4%);
}
.monitor-header-controls, .availability-summary {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: .45rem .8rem;
  min-width: 0;
  color: var(--channel-muted);
  font-size: .8rem;
}
.availability-summary { gap: .25rem; font-variant-numeric: tabular-nums; }
.availability-ranges { display: flex; align-items: center; gap: .1rem; border-radius: 9px; background: var(--channel-subtle); padding: .2rem; }
.availability-ranges button { border: 1px solid transparent; border-radius: 7px; padding: .3rem .68rem; color: var(--channel-muted); font-size: .8rem; white-space: nowrap; }
.availability-ranges button.selected { border-color: var(--channel-border); background: var(--channel-surface); color: var(--channel-text); font-weight: 650; box-shadow: 0 1px 2px rgb(15 23 42 / 8%); }
.availability-action, .availability-settings { display: inline-flex; flex: none; min-height: 2rem; align-items: center; justify-content: center; gap: .35rem; border: 1px solid var(--channel-border); border-radius: 6px; background: var(--channel-surface); padding: .35rem .6rem; color: var(--channel-text); font-size: .75rem; font-weight: 600; }
.availability-action:hover:not(:disabled), .availability-settings:hover:not(:disabled) { background: var(--channel-subtle); }
.availability-settings:disabled { cursor: wait; opacity: .6; }
.availability-action:focus-visible, .availability-ranges button:focus-visible, .availability-settings:focus-visible { outline: 2px solid var(--channel-muted); outline-offset: 2px; }
.availability-workspace, .availability-results { min-width: 0; }
.channel-filters { margin-bottom: 1rem; color: var(--channel-muted); }
.channel-filters summary { display: flex; width: fit-content; align-items: center; gap: .4rem; cursor: pointer; border-radius: 5px; font-size: .75rem; }
.channel-filters[open] { border-bottom: 1px solid var(--channel-border); padding-bottom: .55rem; }
.availability-platforms, .availability-toolbar { display: flex; flex-wrap: wrap; align-items: center; gap: .55rem; margin-top: .65rem; }
.availability-platforms button { border: 1px solid var(--channel-border); border-radius: 7px; padding: .35rem .65rem; color: var(--channel-muted); font-size: .75rem; }
.availability-platforms button:hover, .availability-platforms button.selected { background: var(--channel-subtle); color: var(--channel-text); }
.availability-platforms button.selected { font-weight: 650; }
.availability-toolbar { min-width: 0; justify-content: space-between; }
.availability-search { display: flex; width: min(100%, 26rem); min-width: 0; align-items: center; gap: .5rem; }
.availability-search input { min-height: 2rem; border-radius: 6px; font-size: .8125rem; }
.availability-status { max-width: 100%; }
.availability-legend { display: flex; flex-wrap: wrap; gap: .5rem 1rem; margin-top: .7rem; font-size: .6875rem; }
.legend-healthy { background: #00976e; }
.legend-warning { background: #f9c32f; }
.legend-critical { background: #ef4444; }
.legend-unknown { background: var(--channel-gap); }
.availability-notice { display: flex; align-items: flex-start; gap: .75rem; border: 1px solid #fcd34d; border-radius: .75rem; background: #fffbeb; padding: 1rem; color: #92400e; font-size: .8125rem; }
.dark .availability-notice { border-color: #92400e; background: #451a03; color: #fde68a; }
.availability-advanced { border: 1px solid var(--channel-border); border-radius: .875rem; background: var(--channel-surface); }
@media (prefers-reduced-motion: reduce) { .availability-page :deep(*) { animation: none !important; transition: none !important; } }
details > summary::-webkit-details-marker {
  display: none;
}
</style>
