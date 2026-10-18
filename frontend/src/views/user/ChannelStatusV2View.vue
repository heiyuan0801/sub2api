<template>
  <AppLayout>
    <div class="availability-page space-y-5 pb-10">
      <header class="availability-header">
        <div class="min-w-0">
          <h1 class="text-2xl font-bold tracking-tight text-gray-950 dark:text-white">{{ t('channelMonitorV2.availability.title') }}</h1>
          <p class="mt-3 text-xs leading-6 text-gray-500 dark:text-gray-400">{{ t('channelMonitorV2.availability.description') }}</p>
        </div>
        <div class="flex flex-col items-end gap-3">
          <div class="flex flex-wrap items-center justify-end gap-3">
            <div class="availability-ranges" role="group" :aria-label="t('channelMonitorV2.timeRange')">
              <button v-for="option in ranges" :key="option.value" type="button" :class="{ selected: filter.range === option.value }" :aria-pressed="filter.range === option.value" @click="setRange(option.value)">{{ option.label }}</button>
            </div>
            <button class="availability-action" type="button" :disabled="loading || refreshing" @click="reload(false)">
              <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading || refreshing }" />
              {{ t('common.refresh') }}
            </button>
          </div>
          <span class="text-xs tabular-nums text-gray-500 dark:text-gray-400" aria-live="polite">
            <template v-if="refreshing">{{ t('channelMonitorV2.updating') }}</template>
            <template v-else-if="snapshot?.coverage.data_through">{{ t('channelMonitorV2.updatedTo', { time: formatTime(snapshot.coverage.data_through) }) }}</template>
          </span>
        </div>
      </header>

      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex flex-wrap items-center gap-3 text-xs text-gray-500 dark:text-gray-400">
          <span class="availability-source">{{ t('channelMonitorV2.availability.realCalls') }}</span>
          <span class="inline-flex items-center gap-1.5">
            <span class="h-1.5 w-1.5 rounded-full" :class="snapshot && !monitorDisabled && !dataStale ? 'bg-emerald-500' : 'bg-gray-400'" aria-hidden="true" />
            {{ t(monitorDisabled ? 'channelMonitorV2.availability.disabled' : 'channelMonitorV2.availability.passive') }}
          </span>
          <span v-if="snapshot && !snapshot.coverage.coverage_complete && !bootstrapActive">{{ t('channelMonitorV2.partialCoverage') }}</span>
        </div>
        <div v-if="isAdmin" class="flex flex-wrap gap-2">
        <router-link to="/admin/ops?open_alert_rules=1" class="availability-settings"><Icon name="bell" size="sm" />{{ t('channelMonitorV2.preferences.alerts') }}</router-link>
        <router-link to="/admin/channels/monitor" class="availability-settings">
          <Icon name="cog" size="sm" />{{ t('channelMonitorV2.availability.settings') }}
        </router-link>
        </div>
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

      <section class="availability-summary" :aria-label="t('channelMonitorV2.summaryAria')">
        <div v-for="item in summaryStats" :key="item.key" class="availability-stat" :data-testid="`summary-${item.key}`">
          <span class="availability-stat-icon" :class="item.tone"><Icon :name="item.icon" size="md" /></span>
          <div class="min-w-0">
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ item.label }}</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums leading-none">{{ (loading && !matrix) || (dataStale && item.key !== 'total') ? '—' : item.count }}</p>
          </div>
        </div>
      </section>

      <section class="availability-workspace" :aria-label="t('channelMonitorV2.availability.title')">
        <div class="availability-platforms" role="group" :aria-label="t('channelMonitorV2.filters.platform')">
          <button type="button" :class="{ selected: filter.platforms.length === 0 }" :aria-pressed="filter.platforms.length === 0" :disabled="!dimensionsLoaded" data-testid="platform-all" @click="selectPlatforms([])">{{ t('channelMonitorV2.filters.allPlatforms') }}</button>
          <button v-for="platform in platformShortcuts" :key="platform.value" type="button" :class="{ selected: filter.platforms.length === 1 && filter.platforms[0] === platform.value }" :aria-pressed="filter.platforms.length === 1 && filter.platforms[0] === platform.value" :data-testid="`platform-${platform.value}`" @click="selectPlatforms([platform.value])">{{ platform.label }}</button>
        </div>
        <div class="availability-toolbar monitor-toolbar">
          <form class="availability-search" role="search" @submit.prevent="applySearch">
            <label for="monitor-model-search" class="sr-only">{{ t('channelMonitorV2.availability.searchPlaceholder') }}</label>
            <input id="monitor-model-search" v-model="searchDraft" type="search" class="input min-w-0 flex-1" :placeholder="t('channelMonitorV2.availability.searchPlaceholder')" @input="onSearchInput" />
            <button type="submit" class="availability-action"><Icon name="search" size="sm" />{{ t('channelMonitorV2.availability.search') }}</button>
          </form>
          <div class="flex min-w-0 flex-wrap items-center gap-2">
            <Select v-model="sortOrder" :options="sortOptions" :aria-label="t('channelMonitorV2.preferences.sortLabel')" class="availability-status" />
            <button type="button" class="availability-settings" :class="{ '!border-gray-900 !text-gray-950 dark:!border-gray-100 dark:!text-white': favoritesOnly }" :aria-pressed="favoritesOnly" @click="favoritesOnly = !favoritesOnly">{{ t('channelMonitorV2.preferences.favoritesOnly') }}</button>
            <Select v-model="modelTypeFilter" :options="modelTypeOptions" :aria-label="t('channelMonitorV2.availability.modelType')" class="availability-status" />
            <FilterMultiSelect v-model="selectedGroupIds" compact :label="t('channelMonitorV2.filters.group')" :all-label="t('channelMonitorV2.availability.allGroups')" :options="groupOptions" />
            <FilterMultiSelect v-model="filter.models" compact :label="t('channelMonitorV2.filters.model')" :all-label="t('channelMonitorV2.filters.allModels')" :options="modelOptions" />
            <Select v-model="statusFilter" :options="statusOptions" :aria-label="t('channelMonitorV2.availability.filterStatus')" class="availability-status" />
          </div>
        </div>
        <div class="availability-legend">
          <div class="flex flex-wrap items-center gap-x-4 gap-y-2">
            <span v-for="item in legendItems" :key="item.state" class="inline-flex items-center gap-1.5"><i class="h-2 w-2 rounded-sm" :class="`legend-${item.state}`" aria-hidden="true" />{{ item.label }}</span>
          </div>
          <span v-if="summary.noData || summary.insufficient">{{ t('channelMonitorV2.availability.pendingSummary', { empty: summary.noData, insufficient: summary.insufficient }) }}</span>
          <button v-if="hasDimensionFilter || appliedSearch || statusFilter !== 'all' || modelTypeFilter !== 'all' || favoritesOnly" type="button" class="underline underline-offset-4" @click="resetAllFilters">{{ t('channelMonitorV2.clearFilters') }}</button>
        </div>
        <div class="availability-results" :aria-busy="refreshing">
          <MonitorTrendChart v-if="trendView === 'line'" :trend="snapshot?.trend || []" :coverage="snapshot?.coverage || null" :loading="loading && !snapshot" />
          <MonitorChannelCards v-else-if="matrix" :rows="visibleMatrixRows" :coverage="matrix.coverage" :health-mode="healthMode" :show-throughput="showThroughput" :loading="loading" :favorite-keys="favoriteKeys" :stale="dataStale" @toggle-favorite="toggleFavorite" @inspect="inspectIncident" :empty-description="hasDimensionFilter || appliedSearch || statusFilter !== 'all' || modelTypeFilter !== 'all' || favoritesOnly ? t('channelMonitorV2.availability.emptyDescription') : undefined" @select-group="selectGroup" />
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
              <span v-if="hasDimensionFilter" class="badge badge-gray">{{ t('channelMonitorV2.filters.selectedCount', { count: filter.platforms.length + filter.groupIds.length + filter.models.length }) }}</span>
            </span>
            <Icon name="chevronDown" size="sm" class="transition-transform group-open:rotate-180" />
          </summary>
          <div class="monitor-toolbar flex flex-wrap items-center gap-2 border-t border-gray-100 px-5 py-4 dark:border-dark-700 sm:px-6">
            <FilterMultiSelect v-model="selectedPlatformIds" compact :label="t('channelMonitorV2.filters.platform')" :all-label="t('channelMonitorV2.filters.allPlatforms')" :options="platformOptions" />
            <FilterMultiSelect v-model="selectedGroupIds" compact :label="t('channelMonitorV2.filters.group')" :all-label="t('channelMonitorV2.filters.allGroups')" :options="groupOptions" />
            <FilterMultiSelect v-model="filter.models" compact :label="t('channelMonitorV2.filters.model')" :all-label="t('channelMonitorV2.filters.allModels')" :options="modelOptions" />
            <button type="button" class="btn btn-ghost btn-sm !px-2 !py-1 text-xs" :disabled="!hasDimensionFilter" @click="clearDimensions">{{ t('channelMonitorV2.clearFilters') }}</button>
            <Select v-model="matrixGroupBy" :options="matrixGroupOptions" :placeholder="t('channelMonitorV2.groupBy.label')" class="monitor-toolbar-select w-full sm:w-44" />
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
          <div v-if="activeTab === 'models'" class="table-container border-0">
            <table class="table monitor-table min-w-[720px]">
              <thead>
                <tr>
                  <th>{{ t('channelMonitorV2.table.platformModel') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.successRate') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.ttftP50') }}</th>
                  <th v-if="showThroughput">{{ t('channelMonitorV2.metrics.tps') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.cacheRate') }}</th>
                  <th v-if="showThroughput">{{ t('channelMonitorV2.metrics.rpm') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="row in modelRows"
                  :key="`${row.platform}:${row.model}`"
                  class="cursor-pointer"
                  @click="drillModel(row)"
                >
                  <td>
                    <div class="flex items-center gap-2">
                      <span :class="statusDot(row.health)" aria-hidden="true"></span>
                      <div>
                        <span class="block text-xs text-gray-500 dark:text-dark-400">{{ row.platform }}</span>
                        <strong class="font-semibold text-gray-900 dark:text-white">
                          {{ row.model === '__other__' ? t('channelMonitorV2.otherModels') : row.model }}
                        </strong>
                      </div>
                    </div>
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

          <div v-else-if="activeTab === 'errors'" class="space-y-3">
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
import type {
  HealthState,
  MonitorCoverage,
  MonitorDimensions,
  MonitorErrorRow,
  MonitorFilter,
  MonitorHealth,
  MonitorMatrixGroupBy,
  MonitorMatrixRow,
  MonitorMetric,
  MonitorMatrixResponse,
  MonitorModelRow,
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
  healthScoreClass,
  monitorErrorCategoryLabel,
  ttftDisplayState,
} from '@/features/channel-monitor-v2/monitorFormat'

type Tab = 'models' | 'errors' | 'users'
type HealthMode = 'overall' | 'success' | 'ttft' | 'cache'
type TrendView = 'pulse' | 'line'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()
const { t, te, locale } = useI18n()
const isAdmin = computed(() => authStore.isAdmin)
/** Admins always see RPM/TPM; users honor the hide-throughput system setting. */
const showThroughput = computed(() => isAdmin.value || !isChannelMonitorThroughputHidden())
/** Admins always see ranking; users honor the hide-user-ranking system setting. */
const showUserRanking = computed(() => isAdmin.value || !isChannelMonitorUserRankingHidden())

const ranges = computed(() => [
  { value: '24h' as MonitorRange, label: t('channelMonitorV2.availability.range24h') },
  { value: '7d' as MonitorRange, label: t('channelMonitorV2.availability.range7d') },
  { value: '90m' as MonitorRange, label: t('channelMonitorV2.ranges.90m') },
  { value: '30d' as MonitorRange, label: t('channelMonitorV2.ranges.30d') },
])
const tabs = computed(() => {
  const items: Array<{ value: Tab; label: string }> = [
    { value: 'models', label: t('channelMonitorV2.tabs.models') },
    { value: 'errors', label: t('channelMonitorV2.tabs.errors') },
  ]
  if (showUserRanking.value) {
    items.push({ value: 'users', label: t('channelMonitorV2.tabs.users') })
  }
  return items
})
const matrixGroupOptions = computed(() => [
  { value: 'platform' as MonitorMatrixGroupBy, label: t('channelMonitorV2.groupBy.platform') },
  { value: 'platform_group' as MonitorMatrixGroupBy, label: t('channelMonitorV2.groupBy.platformGroup') },
  { value: 'platform_model' as MonitorMatrixGroupBy, label: t('channelMonitorV2.groupBy.platformModel') },
  { value: 'platform_group_model' as MonitorMatrixGroupBy, label: t('channelMonitorV2.groupBy.platformGroupModel') },
])
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
  models: csv(initialValue('model')),
})
const activeTab = ref<Tab>(parseTab(initialValue('tab'), showUserRanking.value))
const matrixGroupBy = ref<MonitorMatrixGroupBy>(parseMatrixGroupBy(initialValue('group_by')))
const healthMode = ref<HealthMode>(parseHealthMode(initialValue('health_mode')))
const trendView = ref<TrendView>(parseTrendView(initialValue('trend_view')))
const dimensions = ref<MonitorDimensions>({ platforms: [], groups: [], models: [] })
const dimensionsLoaded = ref(false)
const snapshot = ref<MonitorSnapshot | null>(null)
const matrix = ref<MonitorMatrixResponse | null>(null)
const modelRows = ref<MonitorModelRow[]>([])
const errorRows = ref<MonitorErrorRow[]>([])
const userRows = ref<MonitorUserRow[]>([])
const loading = ref(false)
const tabLoading = ref(false)
const refreshing = ref(false)
const loadError = ref('')
const monitorDisabled = ref(false)
type AvailabilityStatus = 'all' | 'healthy' | 'warning' | 'critical' | 'noData' | 'insufficient'
const searchDraft = ref(typeof initialValue('q') === 'string' ? String(initialValue('q')) : '')
const appliedSearch = ref(searchDraft.value.trim())
const statusFilter = ref<AvailabilityStatus>(parseStatus(initialValue('status')))
type ModelTypeFilter = 'all' | 'text' | 'image'
const modelTypeFilter = ref<ModelTypeFilter>(initialValue('model_type') === 'image' || initialValue('model_type') === 'text' ? initialValue('model_type') as ModelTypeFilter : 'all')
const sortOrder = ref<MonitorSort>(['health', 'success', 'latency', 'name'].includes(String(initialValue('sort'))) ? initialValue('sort') as MonitorSort : 'health')
const favoritesOnly = ref(initialValue('favorites_only') === true || initialValue('favorites_only') === '1')
const favoriteKeys = ref<string[]>(saved.favorites || [])
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
  () => filter.value.platforms.length + filter.value.groupIds.length + filter.value.models.length > 0
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
  const models = new Set(dimensions.value.models.filter(item => modelInScope(item, platforms, groupIds)).map(item => item.value))
  // Apply the new scope together so no request uses an incompatible old group/model.
  filter.value = {
    ...filter.value,
    platforms,
    groupIds,
    models: dimensionsLoaded.value ? filter.value.models.filter(model => models.has(model)) : filter.value.models,
  }
}
const selectedPlatformIds = computed({ get: () => filter.value.platforms, set: selectPlatforms })
function groupInScope(item: MonitorDimensions['groups'][number], platforms: string[]) {
  if (!platforms.length) return true
  return item.platforms === undefined
    ? !item.platform || platforms.includes(item.platform)
    : item.platforms.some(platform => platforms.includes(platform))
}
function modelInScope(item: MonitorDimensions['models'][number], platforms: string[], groupIds: number[]) {
  return (!platforms.length || !item.platform || platforms.includes(item.platform)) &&
    (!groupIds.length || item.group_ids === undefined || item.group_ids.some(id => groupIds.includes(id)))
}
const groupOptions = computed(() =>
  (dimensions.value.groups || [])
    .filter(item => groupInScope(item, filter.value.platforms))
    .map((item) => ({
      value: String(item.id),
      label: item.platform ? `${item.platform} / ${item.name || `#${item.id}`}` : item.name || `#${item.id}`,
    }))
)
const modelOptions = computed(() => {
  // Model filters use real names across platforms, so repeated names must share
  // one option while platform selection continues to narrow the request scope.
  const options = new Map<string, { value: string; label: string; platforms: Set<string> }>()
  for (const item of dimensions.value.models || []) {
    if (!modelInScope(item, filter.value.platforms, filter.value.groupIds)) continue
    const option = options.get(item.value) || { value: item.value, label: item.label, platforms: new Set<string>() }
    if (item.platform) option.platforms.add(item.platform)
    options.set(item.value, option)
  }
  return [...options.values()].map(item => ({
    value: item.value,
    label: item.platforms.size ? `${[...item.platforms].join(' / ')} / ${item.label}` : item.label,
  }))
})
const selectedGroupIds = computed({
  get: () => filter.value.groupIds.map(String),
  set: (value: string[]) => {
    const groupIds = value.map(Number).filter((id) => Number.isInteger(id) && id > 0)
    const models = new Set(dimensions.value.models.filter(item => modelInScope(item, filter.value.platforms, groupIds)).map(item => item.value))
    filter.value = {
      ...filter.value,
      groupIds,
      models: dimensionsLoaded.value ? filter.value.models.filter(model => models.has(model)) : filter.value.models,
    }
  },
})
// Wait for the full catalog before pruning saved filters. Once loaded, an empty
// platform catalog also invalidates old group/model choices.
watch(
  [dimensionsLoaded, groupOptions, modelOptions],
  () => {
    if (!dimensionsLoaded.value) return
    const groups = new Set(groupOptions.value.map((item) => item.value))
    const groupIds = filter.value.groupIds.filter((id) => groups.has(String(id)))
    if (groupIds.length !== filter.value.groupIds.length) {
      filter.value.groupIds = groupIds
    }
    const allowedModels = new Set(modelOptions.value.map((item) => item.value))
    const models = filter.value.models.filter((model) => allowedModels.has(model))
    if (models.length !== filter.value.models.length) {
      filter.value.models = models
    }
  },
  { flush: 'post' },
)
const activeRowsEmpty = computed(() =>
  activeTab.value === 'models'
    ? modelRows.value.length === 0
    : activeTab.value === 'errors'
      ? errorRows.value.length === 0
      : userRows.value.length === 0
)
/** First-upgrade backfill toward 90m/24h/7d/30d; banner hides when backend omits bootstrap. */
const bootstrapActive = computed(() => Boolean(snapshot.value?.coverage?.bootstrap?.active))
const bootstrapPercent = computed(() => {
  const raw = snapshot.value?.coverage?.bootstrap?.progress_percent
  if (typeof raw !== 'number' || Number.isNaN(raw)) return 0
  return Math.min(100, Math.max(0, Math.round(raw)))
})
const matrixRows = computed(() => {
  const items = matrix.value?.items || []
  // platform_group views should only show real groups, never bare platform placeholders.
  if (matrixGroupBy.value === 'platform_group' || matrixGroupBy.value === 'platform_group_model') {
    return items.filter((row) => row.group_id != null && Number(row.group_id) > 0)
  }
  return items
})


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
    (modelTypeFilter.value === 'all' || Boolean(row.is_image_model) === (modelTypeFilter.value === 'image')) &&
    (!query || [row.model, row.platform, row.group_name].some(value => value?.toLocaleLowerCase().includes(query)))
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
const summaryStats = computed(() => [
  { key: 'total', label: t('channelMonitorV2.availability.total'), count: summary.value.total, icon: 'cube' as const, tone: 'stat-total' },
  { key: 'healthy', label: t('channelMonitorV2.availability.healthy'), count: summary.value.healthy, icon: 'checkCircle' as const, tone: 'stat-healthy' },
  { key: 'warning', label: t('channelMonitorV2.availability.warning'), count: summary.value.warning, icon: 'trendingUp' as const, tone: 'stat-warning' },
  { key: 'critical', label: t('channelMonitorV2.availability.critical'), count: summary.value.critical, icon: 'exclamationCircle' as const, tone: 'stat-critical' },
])
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
const modelTypeOptions = computed(() => [
  { value: 'all', label: t('channelMonitorV2.availability.allModelTypes') },
  { value: 'text', label: t('channelMonitorV2.availability.textModels') },
  { value: 'image', label: t('channelMonitorV2.availability.imageModels') },
])
function applySearch() { appliedSearch.value = searchDraft.value.trim() }
function onSearchInput() { if (!searchDraft.value) appliedSearch.value = '' }
function resetAllFilters() { searchDraft.value = ''; appliedSearch.value = ''; statusFilter.value = 'all'; modelTypeFilter.value = 'all'; favoritesOnly.value = false; clearDimensions() }
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
    models: row.model ? [row.model] : [...filter.value.models], ...incidentWindow.value,
  }
  incidentLoading.value = true
  incidentError.value = ''
  incidentErrors.value = []
  incidentMetrics.value = null
  incidentHealth.value = null
  try {
    const [metrics, errors] = await Promise.all([
      api.getMatrix(scoped, matrixGroupBy.value, isAdmin.value, request.signal),
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
  if (monitorDisabled.value) { closeIncident(); snapshot.value = null; matrix.value = null; modelRows.value = []; errorRows.value = []; userRows.value = [] }
  scheduleAutoRefresh()
}
function onVisibilityChange() {
  if (document.visibilityState === 'visible' && !loading.value && !refreshing.value) void reload(true)
}

function csv(value: unknown) {
  return Array.isArray(value) ? value.map(String).filter(Boolean) : typeof value === 'string' ? value.split(',').filter(Boolean) : []
}
function parseRange(value: unknown): MonitorRange {
  return ['90m', '24h', '7d', '30d'].includes(String(value)) ? (value as MonitorRange) : '24h'
}
function parseMatrixGroupBy(value: unknown): MonitorMatrixGroupBy {
  const allowed: MonitorMatrixGroupBy[] = [
    'platform',
    'platform_group',
    'platform_model',
    'platform_group_model',
  ]
  return allowed.includes(value as MonitorMatrixGroupBy)
    ? (value as MonitorMatrixGroupBy)
    : 'platform_group_model'
}
function parseTab(value: unknown, allowUsers: boolean): Tab {
  const allowed: Tab[] = allowUsers ? ['models', 'errors', 'users'] : ['models', 'errors']
  return allowed.includes(value as Tab) ? (value as Tab) : 'models'
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
    model: filter.value.models, q: appliedSearch.value, status: statusFilter.value,
    model_type: modelTypeFilter.value, group_by: matrixGroupBy.value, health_mode: healthMode.value,
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
      model_type: modelTypeFilter.value !== 'all' ? modelTypeFilter.value : undefined,
      sort: sortOrder.value !== 'health' ? sortOrder.value : undefined,
      favorites_only: favoritesOnly.value ? '1' : undefined,
      range: filter.value.range,
      platform: filter.value.platforms.join(',') || undefined,
      group: filter.value.groupIds.join(',') || undefined,
      model: filter.value.models.join(',') || undefined,
      group_by: matrixGroupBy.value,
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
    api.getMatrix(filter.value, matrixGroupBy.value, isAdmin.value, signal),
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
    if (activeTab.value === 'models') {
      const result = await api.getModels(filter.value, isAdmin.value, signal)
      if (id === sequence && activeTab.value === 'models') modelRows.value = result.items || []
    } else if (activeTab.value === 'errors') {
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
  autoRefreshTimer = window.setInterval(() => {
    if (document.visibilityState !== 'hidden' && !loading.value && !refreshing.value) {
      void reload(true)
    }
  }, Math.max(bootstrapActive.value ? 10 : 60, seconds) * 1000)
}
function drillModel(row: MonitorModelRow) {
  selectPlatforms([row.platform])
  filter.value.models = [row.model]
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
  return new Intl.DateTimeFormat(locale.value || undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}
function statusDot(health?: MonitorHealth | HealthState) {
  if (dataStale.value) return 'status-dot health-unknown'
  if (!health || typeof health === 'string') {
    return `status-dot health-${health || 'unknown'}`
  }
  // Prefer multi-band score when available; otherwise fall back to the coarse
  // overall state for mixed-version/older payloads.
  const klass =
    health.score != null
      ? healthScoreClass(health, 'overall', 0)
      : `health-${health.overall || 'unknown'}`
  return `status-dot ${klass}`
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
watch(matrixGroupBy, () => {
  syncQuery()
  void reloadMetricsOnly(true)
})
watch(healthMode, syncQuery)
watch(trendView, syncQuery)
watch(activeTab, () => {
  syncQuery()
  void loadTab()
})
watch(showUserRanking, (allowed) => {
  if (!allowed && activeTab.value === 'users') {
    activeTab.value = 'models'
  }
})
watch([appliedSearch, statusFilter, modelTypeFilter, sortOrder, favoritesOnly], syncQuery)
watch(favoriteKeys, persistPreferences)
onMounted(() => {
  freshnessTimer = window.setInterval(() => { now.value = Date.now() }, 15_000)
  document.addEventListener('visibilitychange', onVisibilityChange)
  void reload(false)
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
.availability-page { color: #171717; }
.availability-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 1.5rem; padding: .25rem 0 .5rem; }
.availability-ranges { display: flex; flex-wrap: wrap; align-items: center; gap: .125rem; border: 1px solid #e5e5e5; border-radius: 9999px; background: #fff; padding: .25rem; }
.availability-ranges button { border-radius: 9999px; padding: .375rem .75rem; color: #737373; font-size: .75rem; white-space: nowrap; }
.availability-ranges button.selected { background: #171717; color: #fff; font-weight: 600; }
.availability-action { display: inline-flex; flex: none; min-height: 2.25rem; align-items: center; justify-content: center; gap: .4rem; border: 1px solid #171717; border-radius: .5rem; background: #171717; padding: .5rem .75rem; color: #fff; font-size: .75rem; font-weight: 600; }
.availability-action:hover:not(:disabled) { background: #404040; }
.availability-action:disabled { cursor: wait; opacity: .55; }
.availability-action:focus-visible, .availability-ranges button:focus-visible, .availability-settings:focus-visible { outline: 2px solid #737373; outline-offset: 3px; }
.availability-source { border-radius: 9999px; background: #171717; padding: .4rem .85rem; color: #fff; font-weight: 600; }
.availability-settings { display: inline-flex; flex: none; align-items: center; gap: .4rem; border: 1px solid #e5e5e5; border-radius: .5rem; background: #fff; padding: .4rem .65rem; color: #262626; font-size: .75rem; }
.availability-summary { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: .75rem; }
.availability-stat { display: flex; min-width: 0; align-items: center; gap: .85rem; border: 1px solid #e5e5e5; border-radius: .875rem; background: #fff; padding: 1.25rem; }
.availability-stat-icon { display: inline-flex; width: 2.5rem; height: 2.5rem; flex: none; align-items: center; justify-content: center; border-radius: 9999px; color: #fff; }
.stat-total { background: #171717; }
.stat-healthy, .legend-healthy { background: #00b982; }
.stat-warning, .legend-warning { background: #ff9500; }
.stat-critical, .legend-critical { background: #ff2d42; }
.legend-unknown { background: #d4d4d4; }
.availability-workspace { min-width: 0; border: 1px solid #e5e5e5; border-radius: .875rem; background: #fff; }
.availability-platforms { display: flex; flex-wrap: wrap; align-items: center; gap: .5rem; border-bottom: 1px solid #ededed; padding: 1rem 1.25rem; }
.availability-platforms button { border: 1px solid #e5e5e5; border-radius: 9999px; padding: .45rem .9rem; color: #525252; font-size: .75rem; white-space: nowrap; transition: background-color .15s, color .15s; }
.availability-platforms button:hover { background: #f5f5f5; }
.availability-platforms button.selected { border-color: #171717; background: #171717; color: #fff; font-weight: 600; }
.availability-platforms button:focus-visible { outline: 2px solid #737373; outline-offset: 3px; }
.availability-toolbar { display: flex; min-width: 0; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: .75rem; padding: 1.25rem; }
.availability-search { display: flex; width: min(100%, 26rem); min-width: 0; align-items: center; gap: .5rem; }
.availability-search input { height: 2.25rem; border-radius: .5rem; font-size: .8125rem; }
.availability-status { min-width: 10rem; max-width: 100%; }
.availability-legend { display: flex; flex-wrap: wrap; align-items: center; gap: .5rem 1.5rem; border-top: 1px solid #ededed; padding: .75rem 1.25rem; color: #737373; font-size: .6875rem; }
.availability-results { min-width: 0; border-top: 1px solid #ededed; padding: 1.25rem; }
.availability-notice { display: flex; align-items: flex-start; gap: .75rem; border: 1px solid #fcd34d; border-radius: .75rem; background: #fffbeb; padding: 1rem; color: #92400e; font-size: .8125rem; }
.availability-advanced { border: 1px solid #e5e5e5; border-radius: .875rem; background: #fff; }
.dark .availability-page { color: #f5f5f5; }
.dark .availability-ranges, .dark .availability-settings, .dark .availability-stat, .dark .availability-workspace, .dark .availability-advanced { border-color: #404040; background: #262626; }
.dark .availability-legend, .dark .availability-results, .dark .availability-platforms { border-color: #404040; }
.dark .availability-platforms button { border-color: #525252; color: #d4d4d4; }
.dark .availability-platforms button:hover { background: #404040; }
.dark .availability-platforms button.selected { border-color: #f5f5f5; background: #f5f5f5; color: #171717; }
.dark .availability-ranges button { color: #a3a3a3; }
.dark .availability-ranges button.selected, .dark .availability-action, .dark .availability-source, .dark .stat-total { background: #f5f5f5; color: #171717; }
.dark .availability-settings { color: #e5e5e5; }
.dark .availability-legend { color: #a3a3a3; }
.dark .availability-notice { border-color: #92400e; background: #451a03; color: #fde68a; }
@media (max-width: 768px) {
  .availability-header { flex-direction: column; gap: 1rem; }
  .availability-header > div:last-child { width: 100%; align-items: flex-start; }
  .availability-header > div:last-child > div { justify-content: flex-start; }
  .availability-summary { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .availability-stat { gap: .625rem; padding: 1rem; }
  .availability-toolbar { align-items: stretch; padding: 1rem; }
  .availability-platforms { padding: 1rem; }
  .availability-search { width: 100%; }
  .availability-results { padding: .875rem; }
  .availability-legend { padding: .75rem 1rem; }
}
@media (prefers-reduced-motion: reduce) { .availability-page :deep(*) { animation: none !important; transition: none !important; } }
.status-dot {
  display: inline-block;
  height: 0.5rem;
  width: 0.5rem;
  flex: none;
  border-radius: 9999px;
}
/* Multi-stop green → yellow → red score bands */
.health-score10 { background: #16a34a; }
.health-score9  { background: #22c55e; }
.health-score8  { background: #4ade80; }
.health-score7  { background: #a3e635; }
.health-score6  { background: #facc15; }
.health-score5  { background: #fbbf24; }
.health-score4  { background: #f59e0b; }
.health-score3  { background: #f97316; }
.health-score2  { background: #fb7185; }
.health-score1  { background: #f87171; }
.health-score0  { background: rgb(239, 67, 67); }
.health-healthy  { background: #22c55e; }
.health-warning  { background: #f59e0b; }
.health-critical { background: #ef4444; }
.health-unknown  { background: #9ca3af; }
.matrix-select {
  min-width: 10rem;
}
details > summary::-webkit-details-marker {
  display: none;
}
</style>
