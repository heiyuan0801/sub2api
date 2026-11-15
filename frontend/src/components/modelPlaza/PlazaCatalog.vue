<template>
  <div class="model-catalog">
    <div class="catalog-page">
      <header class="catalog-hero">
        <div class="catalog-intro">
          <div>
            <h1>{{ t('publicSite.market.heading') }}</h1>
            <p>{{ t('publicSite.market.featuredDescription') }}</p>
          </div>
          <RouterLink :to="localizedPublicPath('/docs/api', locale)" class="market-guide">
            <Icon name="document" size="sm" />{{ t('publicSite.nav.apiDocs') }}<Icon name="chevronRight" size="xs" />
          </RouterLink>
        </div>
        <div class="catalog-discovery">
          <div class="catalog-search">
            <Icon name="search" size="md" aria-hidden="true" />
            <label for="public-model-search" class="sr-only">{{ t('publicSite.models.searchLabel') }}</label>
            <input id="public-model-search" ref="searchInput" v-model="searchQuery" type="search" :placeholder="t('publicSite.market.search')" autocomplete="off" />
            <kbd aria-hidden="true">{{ searchShortcut }}</kbd>
          </div>
          <Button variant="outline" size="sm" type="button" class="catalog-button filter-button" :disabled="loading || error" :aria-expanded="filterOpen" @click="filterOpen = true">
            <Icon name="filter" size="sm" />{{ t('publicSite.models.filter') }}
            <span v-if="activeFilterCount" class="filter-count">{{ activeFilterCount }}</span>
          </Button>
          <button v-if="activeFilterCount" type="button" class="clear-filters" @click="resetFilters">{{ t('publicSite.models.clearFilters') }}</button>
        </div>
        <nav v-if="!loading && !error && platforms.length" class="provider-navigation" :aria-label="t('publicSite.models.provider')">
          <button type="button" :aria-pressed="selectedPlatform === 'all'" @click="selectedPlatform = 'all'">{{ t('publicSite.models.allProviders') }}</button>
          <button v-for="platform in platforms" :key="platform" type="button" :aria-pressed="selectedPlatform === platform" :disabled="!platformEnabled(platform)" @click="selectedPlatform = platform">
            <PlatformIcon :platform="platform as GroupPlatform" size="sm" />{{ platformLabel(platform) }}
          </button>
        </nav>
      </header>

      <dl v-if="!loading && !error && groups.length" class="catalog-totals" aria-live="polite">
        <div v-for="stat in catalogStats" :key="stat.label" class="catalog-stat">
          <dt>{{ t(stat.label) }}</dt>
          <dd>{{ n(stat.value) }}</dd>
        </div>
      </dl>
      <div v-if="descriptionHtml" class="catalog-description">
        <Button variant="ghost" size="sm" class="market-guide" @click="descriptionOpen = !descriptionOpen" :aria-expanded="descriptionOpen" aria-controls="market-description">
          <Icon name="document" size="sm" />{{ t('publicSite.models.priceDescription') }}
          <Icon name="chevronDown" size="xs" />
        </Button>
        <div v-show="descriptionOpen" id="market-description" class="description-markdown" v-html="descriptionHtml"></div>
      </div>

      <section class="market-results" :aria-labelledby="loading || error ? undefined : 'catalog-results-title'">
        <div v-if="!loading && !error" class="catalog-toolbar">
          <h2 id="catalog-results-title" ref="resultsTitle" tabindex="-1" class="model-count">
            {{ viewMode === 'cards' ? t('publicSite.models.modelCount', { count: n(uniqueModelCount) }) : t('publicSite.market.offerCount', { count: n(entries.length) }) }}
          </h2>
          <div class="summary-actions">
            <SelectRoot v-model="sortOrder">
              <SelectTrigger class="market-sort" :aria-label="t('publicSite.market.sort')">
                <SelectValue />
                <Icon name="chevronDown" size="sm" class="sort-chevron" aria-hidden="true" />
              </SelectTrigger>
              <SelectPortal>
                <SelectContent class="market-sort-content" position="popper" align="start" :side-offset="8" :collision-padding="12" :body-lock="false">
                  <SelectViewport class="market-sort-options">
                    <SelectItem v-for="option in sortOptions" :key="option.value" :value="option.value" class="market-sort-option">
                      <SelectItemText>{{ option.label }}</SelectItemText>
                      <SelectItemIndicator class="sort-check"><Icon name="check" size="sm" aria-hidden="true" /></SelectItemIndicator>
                    </SelectItem>
                  </SelectViewport>
                </SelectContent>
              </SelectPortal>
            </SelectRoot>
            <Button variant="outline" size="sm" type="button" class="catalog-icon-button" :title="t('publicSite.models.exportCsv')" :aria-label="t('publicSite.models.exportCsvLabel')" :disabled="!entries.length" @click="exportCsv"><Icon name="download" size="sm" /></Button>
            <Button variant="outline" size="sm" type="button" class="catalog-button rate-toggle" :aria-pressed="showRate" @click="showRate = !showRate">{{ t('publicSite.models.rate') }}</Button>
            <div class="catalog-view-switch" :data-view="viewMode" role="group" :aria-label="t('publicSite.models.viewMode')">
              <button type="button" :class="{ active: viewMode === 'table' }" :aria-pressed="viewMode === 'table'" :aria-label="t('publicSite.models.tableView')" @click="viewMode = 'table'"><Icon name="menu" size="sm" /></button>
              <button type="button" :class="{ active: viewMode === 'cards' }" :aria-pressed="viewMode === 'cards'" :aria-label="t('publicSite.models.cardView')" @click="viewMode = 'cards'"><Icon name="grid" size="sm" /></button>
            </div>
          </div>
        </div>

        <div v-if="loading" class="catalog-state" role="status" aria-live="polite"><span class="catalog-spinner"></span><span class="sr-only">{{ t('common.loading') }}</span></div>
        <div v-else-if="error" class="catalog-state catalog-error" role="alert"><Icon name="exclamationCircle" size="lg" /><p>{{ t('modelPlaza.loadFailed') }}</p></div>
        <template v-else>
          <div v-if="!entries.length" class="catalog-state catalog-empty">
            <Icon name="search" size="lg" />
            <p>{{ searchQuery.trim() || activeFilterCount ? t('modelPlaza.noSearchResult') : t('modelPlaza.empty') }}</p>
            <Button v-if="searchQuery.trim() || activeFilterCount" variant="outline" size="sm" @click="resetFilters">{{ t('publicSite.models.clearFilters') }}</Button>
          </div>
          <div v-else-if="viewMode === 'cards'" class="catalog-grid">
            <article v-for="{ entry, groupCount } in visibleCards" :key="entry.key" class="catalog-card">
              <div class="market-card-heading">
                <button type="button" class="card-main" @click="selectedEntry = entry">
                  <span class="card-brand"><PlatformIcon :platform="entry.model.platform as GroupPlatform" size="md" /></span>
                  <h3>{{ entry.model.name }}</h3>
                </button>
                <button type="button" class="market-copy" :aria-label="t('publicSite.models.copyNameLabel', { model: entry.model.name })" :title="t('publicSite.models.copyName')" @click="copyToClipboard(entry.model.name)"><Icon name="copy" size="sm" /></button>
              </div>
              <div class="card-badges">
                <span>{{ platformLabel(entry.model.platform) }}</span><span>{{ modeLabel(entry.prices.mode) }}</span>
                <span v-if="entry.model.time_pricing?.periods.length">{{ t('publicSite.models.timePricing') }}</span>
                <span v-if="entry.group.is_exclusive">{{ t('publicSite.models.exclusive') }}</span>
              </div>
              <p class="card-description">{{ entry.group.description || entry.group.name }}</p>
              <dl class="market-metrics">
                <template v-if="entry.prices.mode === BILLING_MODE_TOKEN">
                  <div v-for="column in tokenColumns" :key="column.key">
                    <dt>{{ t(column.label) }}</dt>
                    <dd>{{ formatCatalogPrice(entry.prices[column.key], 1_000_000) }}<small v-if="entry.prices[column.key] != null">/M</small></dd>
                  </div>
                </template>
                <div v-else>
                  <dt>{{ billingLabel(entry.model) }}</dt>
                  <dd>
                    {{ formatCatalogPrice(entry.prices.request) }}
                    <small v-if="entry.prices.request != null">{{ unitSuffix(entry.prices.mode) }}</small>
                  </dd>
                </div>
                <div class="metric-meta"><dt>{{ t(showRate ? 'publicSite.models.rate' : 'publicSite.market.billing') }}</dt><dd>{{ showRate ? entryRate(entry) + 'x' : billingLabel(entry.model) }}</dd></div>
              </dl>
              <footer class="card-footer">
                <span class="card-group" :title="entry.group.name">{{ entry.group.name }}<template v-if="groupCount > 1"> · {{ t('publicSite.models.groupCount', { count: groupCount }) }}</template></span>
                <button type="button" :aria-label="t('publicSite.models.priceDetailsLabel', { model: entry.model.name })" @click="selectedEntry = entry">{{ t('publicSite.market.details') }} <Icon name="arrowRight" size="xs" /></button>
              </footer>
              <GroupDisplayTags :tags="entry.group.display_tags" :custom-tags="entry.group.custom_display_tags" class="mt-2" />
            </article>
          </div>
          <div v-else class="catalog-table-scroll" tabindex="0" role="region" aria-labelledby="catalog-results-title">
            <table class="catalog-table">
              <thead>
                <tr>
                  <th scope="col">{{ t('publicSite.modelDetail.modelId') }}</th>
                  <th scope="col">{{ t('publicSite.models.provider') }}</th>
                  <th scope="col">{{ t('publicSite.models.group') }}</th>
                  <th v-for="column in tokenColumns" :key="column.key" scope="col" class="catalog-number">
                    {{ t(column.label) }}
                    <small>{{ t('publicSite.modelDetail.perMillionTokens') }}</small>
                  </th>
                  <th scope="col">{{ t(showRate ? 'publicSite.models.rate' : 'publicSite.market.billing') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="entry in visibleEntries" :key="entry.key">
                  <th scope="row">
                    <div class="offer-model">
                      <button type="button" class="offer-detail" @click="selectedEntry = entry">{{ entry.model.name }}</button>
                      <button type="button" class="market-copy" :aria-label="t('publicSite.models.copyNameLabel', { model: entry.model.name })" :title="t('publicSite.models.copyName')" @click="copyToClipboard(entry.model.name)">
                        <Icon name="copy" size="sm" />
                      </button>
                    </div>
                  </th>
                  <td><span class="offer-provider"><PlatformIcon :platform="entry.model.platform as GroupPlatform" size="sm" />{{ platformLabel(entry.model.platform) }}</span></td>
                  <td>
                    <div>{{ entry.group.name }}</div>
                    <GroupDisplayTags :tags="entry.group.display_tags" :custom-tags="entry.group.custom_display_tags" class="mt-1" />
                  </td>
                  <template v-if="entry.prices.mode === BILLING_MODE_TOKEN">
                    <td v-for="column in tokenColumns" :key="column.key" class="catalog-number offer-price">{{ formatCatalogPrice(entry.prices[column.key], 1_000_000) }}</td>
                  </template>
                  <td v-else colspan="3" class="offer-price offer-unit">
                    {{ formatCatalogPrice(entry.prices.request) }}
                    <small v-if="entry.prices.request != null">{{ unitSuffix(entry.prices.mode) }}</small>
                  </td>
                  <td>
                    <div class="offer-terms">
                      <span class="offer-mode">{{ showRate ? entryRate(entry) + 'x' : billingLabel(entry.model) }}</span>
                      <span v-if="entry.model.time_pricing?.periods.length || entry.group.peak_rate_enabled" class="offer-dynamic">{{ t('publicSite.modelDetail.dynamicBilling') }}</span>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <p v-if="entries.length && viewMode === 'table'" class="catalog-scroll-hint">
            <Icon name="arrowRight" size="xs" aria-hidden="true" />{{ t('publicSite.market.scrollPrices') }}
          </p>
          <div v-if="pageCount > 1" class="catalog-pagination">
            <button type="button" class="catalog-page-button" :aria-label="t('pagination.previous')" :title="t('pagination.previous')" :disabled="page === 1" @click="changePage(page - 1)"><Icon name="chevronLeft" size="sm" /></button>
            <span aria-live="polite">{{ n(page) }} / {{ n(pageCount) }}</span>
            <button type="button" class="catalog-page-button" :aria-label="t('pagination.next')" :title="t('pagination.next')" :disabled="page === pageCount" @click="changePage(page + 1)"><Icon name="chevronRight" size="sm" /></button>
          </div>
          <p v-if="entries.length" class="catalog-price-note">{{ t(viewMode === 'cards' ? 'publicSite.market.priceNote' : 'publicSite.market.comparisonPriceNote') }}</p>
          <p v-if="!isAuthenticated" class="catalog-price-note">{{ t('modelPlaza.anonymousHint') }}</p>
        </template>
      </section>
    </div>
    <dialog
      ref="filterDialog"
      class="catalog-filter-dialog"
      aria-labelledby="catalog-filter-title"
      @cancel.prevent="filterOpen = false"
      @close="filterOpen = false"
      @click="closeBackdrop"
    >
      <header class="catalog-dialog-header">
        <h2 id="catalog-filter-title">{{ t('publicSite.models.filterModels') }}</h2>
        <Button variant="outline" size="sm" type="button" class="catalog-icon-button" :aria-label="t('publicSite.models.closeFilters')" @click="filterOpen = false"><Icon name="x" size="md" /></Button>
      </header>
      <div class="catalog-dialog-body">
        <div class="catalog-filter-content">
        <section>
          <h3>{{ t('publicSite.market.type') }}</h3>
          <div class="catalog-filter-pills">
            <button v-for="mode in modes" :key="mode" type="button" :class="{ selected: selectedMode === mode }" :aria-pressed="selectedMode === mode" @click="selectedMode = mode">{{ modeLabel(mode) }}</button>
          </div>
        </section>
        <section>
          <h3>{{ t('publicSite.models.provider') }}</h3>
          <div class="manufacturer-options">
            <button type="button" :class="{ selected: selectedPlatform === 'all' }" :aria-pressed="selectedPlatform === 'all'" @click="selectedPlatform = 'all'">
              {{ t('publicSite.models.allProviders') }}
            </button>
            <button v-for="platform in platforms" :key="platform" type="button" :class="{ selected: selectedPlatform === platform }" :aria-pressed="selectedPlatform === platform" :disabled="!platformEnabled(platform)" @click="selectedPlatform = platform">
              <PlatformIcon :platform="platform as GroupPlatform" size="md" /> {{ platformLabel(platform) }}
            </button>
          </div>
        </section>
        <section>
          <h3>{{ t('publicSite.models.group') }}</h3>
          <div class="catalog-filter-pills">
            <button type="button" :class="{ selected: selectedGroupId === 'all' }" :aria-pressed="selectedGroupId === 'all'" @click="selectedGroupId = 'all'">{{ t('publicSite.models.allGroups') }}</button>
            <button v-for="group in groupOptions" :key="group.id" type="button" :class="{ selected: selectedGroupId === group.id }" :aria-pressed="selectedGroupId === group.id" :disabled="!groupEnabled(group)" @click="selectedGroupId = group.id">
              {{ group.name }} <span>{{ group.rate }}x</span>
            </button>
          </div>
        </section>
        <section>
          <h3>{{ t('publicSite.models.rate') }}</h3>
          <div class="catalog-filter-pills">
            <button type="button" :class="{ selected: selectedRate === 'all' }" :aria-pressed="selectedRate === 'all'" @click="selectedRate = 'all'">{{ t('publicSite.models.allRates') }}</button>
            <button v-for="rate in rates" :key="rate" type="button" :class="{ selected: selectedRate === rate }" :aria-pressed="selectedRate === rate" :disabled="!rateEnabled(rate)" @click="selectedRate = rate">{{ rate }}x</button>
          </div>
        </section>
        </div>
      </div>
      <footer class="catalog-dialog-footer">
        <div class="filter-footer">
          <Button variant="outline" size="sm" type="button" class="catalog-button" @click="resetFilters">{{ t('publicSite.models.resetFilters') }}</Button>
          <Button variant="outline" size="sm" type="button" class="catalog-button catalog-button-dark" @click="filterOpen = false">{{ t('publicSite.models.showModels', { count: uniqueModelCount }) }}</Button>
        </div>
      </footer>
    </dialog>

    <PlazaModelDetail v-if="selectedEntry" :model="selectedEntry.model" :groups="detailGroups" :initial-group-id="selectedEntry.group.id" @close="selectedEntry = null" />
  </div>
</template>

<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { SelectContent, SelectItem, SelectItemIndicator, SelectItemText, SelectPortal, SelectRoot, SelectTrigger, SelectValue, SelectViewport } from 'reka-ui'
import { useI18n } from 'vue-i18n'
import GroupDisplayTags from '@/components/common/GroupDisplayTags.vue'
import { useRoute } from 'vue-router'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAuthStore } from '@/stores/auth'
import { platformLabel } from '@/utils/platformColors'
import { BILLING_MODE_IMAGE, BILLING_MODE_TOKEN, BILLING_MODE_VIDEO } from '@/constants/channel'
import { localizedPublicPath } from '@/i18n/localeConfig'
import type { ModelPlazaGroup, ModelPlazaResponse, PlazaModel } from '@/api/modelPlaza'
import type { GroupPlatform } from '@/types'
import { catalogPrices, formatCatalogPrice, groupRate, groupsForPlazaModel, modelRate } from './catalogPricing'
import PlazaModelDetail from './PlazaModelDetail.vue'

const props = defineProps<{ response: ModelPlazaResponse | null; loading: boolean; error?: boolean }>()
interface CatalogEntry { key: string; group: ModelPlazaGroup; model: PlazaModel; prices: ReturnType<typeof catalogPrices> }
const { t, n, locale } = useI18n()
const route = useRoute()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()
const isAuthenticated = computed(() => authStore.isAuthenticated)
const selectedPlatform = ref('all')
const selectedMode = ref('all')
const sortOrder = ref('default')
const sortOptions = computed(() => [
  { value: 'default', label: t('publicSite.market.defaultSort') },
  { value: 'latest', label: t('publicSite.market.latest') },
  { value: 'name', label: t('publicSite.market.byName') },
])
const descriptionOpen = ref(false)
const selectedGroupId = ref<number | 'all'>('all')
const selectedRate = ref<number | 'all'>('all')
const searchQuery = ref(typeof route.query.search === 'string' ? route.query.search : '')
watch(() => route.query.search, search => {
  searchQuery.value = typeof search === 'string' ? search : ''
})
const searchInput = ref<HTMLInputElement | null>(null)
const searchShortcut = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform) ? '⌘ K' : 'Ctrl K'
const filterOpen = ref(false)
const selectedEntry = ref<CatalogEntry | null>(null)
const filterDialog = ref<HTMLDialogElement | null>(null)
const showRate = ref(false)
const viewMode = ref<'cards' | 'table'>('table')
const pageSize = 24
const page = ref(1)
const resultsTitle = ref<HTMLHeadingElement | null>(null)
const tokenColumns = [
  { key: 'input', label: 'publicSite.market.input' },
  { key: 'output', label: 'publicSite.market.output' },
  { key: 'cache', label: 'publicSite.market.cache' },
] as const
const groups = computed(() => props.response?.groups ?? [])
const effectiveRate = groupRate
const modes = computed(() => ['all', ...new Set(groups.value.flatMap(group => group.models.map(model => model.pricing?.billing_mode || BILLING_MODE_TOKEN)))])
const latestOrder = computed(() => new Map((props.response?.latest_models ?? []).map((model, index) => [modelKey(model), index])))
const platforms = computed(() => [...new Set(groups.value.flatMap(group => group.models.map(model => model.platform)).filter(Boolean))].sort())
const groupOptions = computed(() => groups.value.map(group => ({
  id: group.id,
  name: group.name,
  platforms: new Set(group.models.map(model => model.platform)),
  rate: effectiveRate(group),
})))
const rates = computed(() => [...new Set(groups.value.map(effectiveRate))].sort((a, b) => a - b))
const activeFilterCount = computed(() => Number(selectedMode.value !== 'all') + Number(selectedPlatform.value !== 'all') + Number(selectedGroupId.value !== 'all') + Number(selectedRate.value !== 'all'))
const descriptionHtml = computed(() => {
  const description = props.response?.description?.trim()
  return description ? DOMPurify.sanitize(marked.parse(description) as string) : ''
})

// Native modal dialogs keep keyboard focus inside and restore focus when closed.
watch(filterOpen, open => {
  if (open) filterDialog.value?.showModal()
  else filterDialog.value?.close()
}, { flush: 'post' })
watch(filterOpen, open => {
  document.documentElement.classList.toggle('catalog-modal-open', open)
})

watch(groups, () => {
  if (selectedPlatform.value !== 'all' && !platforms.value.includes(selectedPlatform.value)) selectedPlatform.value = 'all'
  if (selectedGroupId.value !== 'all' && !groups.value.some(group => group.id === selectedGroupId.value)) selectedGroupId.value = 'all'
  if (selectedRate.value !== 'all' && !rates.value.includes(selectedRate.value)) selectedRate.value = 'all'
  if (!modes.value.includes(selectedMode.value)) selectedMode.value = 'all'
  if (selectedEntry.value && !detailGroups.value.length) selectedEntry.value = null
})

const filteredGroups = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  return groups.value
    .filter(group => (selectedGroupId.value === 'all' || group.id === selectedGroupId.value) && (selectedRate.value === 'all' || effectiveRate(group) === selectedRate.value))
    .map(group => ({ ...group, models: group.models.filter(model => {
      const matchesPlatform = selectedPlatform.value === 'all' || model.platform === selectedPlatform.value
      const matchesMode = selectedMode.value === 'all' || selectedMode.value === (model.pricing?.billing_mode || BILLING_MODE_TOKEN)
      const matchesQuery = !query || [model.name, group.name, model.platform, platformLabel(model.platform)].some(value => value.toLowerCase().includes(query))
      return matchesPlatform && matchesMode && matchesQuery
    }) }))
    .filter(group => group.models.length > 0)
    .sort((a, b) => effectiveRate(a) - effectiveRate(b) || a.name.localeCompare(b.name))
})
const entries = computed<CatalogEntry[]>(() => {
  const rows = filteredGroups.value.flatMap(group => group.models.map(model => ({
    key: `${group.id}:${model.platform}:${model.name}`, group, model, prices: catalogPrices(model, group),
  })))
  if (sortOrder.value === 'name') rows.sort((a, b) => a.model.name.localeCompare(b.model.name))
  if (sortOrder.value === 'latest') {
    const order = (model: PlazaModel) => latestOrder.value.get(modelKey(model)) ?? Number.MAX_SAFE_INTEGER
    rows.sort((a, b) => order(a.model) - order(b.model))
  }
  return rows
})
function modelKey(model: Pick<PlazaModel, 'platform' | 'name'>) {
  return model.platform + ':' + model.name
}
const entriesByModel = computed(() => {
  const grouped = new Map<string, CatalogEntry[]>()
  for (const entry of entries.value) {
    const key = modelKey(entry.model)
    const items = grouped.get(key)
    if (items) items.push(entry)
    else grouped.set(key, [entry])
  }
  return grouped
})
const cards = computed(() => [...entriesByModel.value.values()].map(items => ({ entry: items[0], groupCount: items.length })))
const uniqueModelCount = computed(() => entriesByModel.value.size)
const detailGroups = computed(() => groupsForPlazaModel(groups.value, selectedEntry.value?.model ?? null))
const catalogStats = computed(() => [
  { label: 'publicSite.market.availableModels', value: uniqueModelCount.value },
  { label: 'publicSite.market.providers', value: new Set(entries.value.map(entry => entry.model.platform)).size },
  { label: 'publicSite.market.groups', value: filteredGroups.value.length },
])
const pageCount = computed(() => Math.max(1, Math.ceil((viewMode.value === 'table' ? entries.value.length : cards.value.length) / pageSize)))
const pageStart = computed(() => (page.value - 1) * pageSize)
const visibleEntries = computed(() => entries.value.slice(pageStart.value, pageStart.value + pageSize))
const visibleCards = computed(() => cards.value.slice(pageStart.value, pageStart.value + pageSize))

watch([searchQuery, selectedMode, selectedPlatform, selectedGroupId, selectedRate, sortOrder, viewMode], () => {
  page.value = 1
})
watch(pageCount, total => {
  page.value = Math.min(page.value, total)
})

function changePage(nextPage: number) {
  page.value = nextPage
  resultsTitle.value?.focus()
}
function platformEnabled(platform: string) {
  return groupOptions.value.some(group => group.platforms.has(platform) && (selectedGroupId.value === 'all' || group.id === selectedGroupId.value) && (selectedRate.value === 'all' || group.rate === selectedRate.value))
}
function groupEnabled(group: { platforms: Set<string>; rate: number }) {
  return (selectedPlatform.value === 'all' || group.platforms.has(selectedPlatform.value)) && (selectedRate.value === 'all' || group.rate === selectedRate.value)
}
function rateEnabled(rate: number) {
  return groupOptions.value.some(group => group.rate === rate && (selectedPlatform.value === 'all' || group.platforms.has(selectedPlatform.value)) && (selectedGroupId.value === 'all' || group.id === selectedGroupId.value))
}
function resetFilters() {
  selectedPlatform.value = 'all'
  selectedMode.value = 'all'
  selectedGroupId.value = 'all'
  selectedRate.value = 'all'
  searchQuery.value = ''
}
function modeLabel(mode: string) {
  if (mode === 'all') return t('publicSite.market.allTypes')
  if (mode === BILLING_MODE_IMAGE) return t('publicSite.market.images')
  if (mode === BILLING_MODE_VIDEO) return t('publicSite.market.video')
  if (mode === BILLING_MODE_TOKEN) return t('publicSite.market.text')
  return t('publicSite.models.requestBilling')
}
function billingLabel(model: PlazaModel) {
  switch (model.pricing?.billing_mode) {
    case BILLING_MODE_IMAGE: return t('publicSite.models.imageBilling')
    case BILLING_MODE_VIDEO: return t('publicSite.models.videoBilling')
    case 'per_request': return t('publicSite.models.requestBilling')
    default: return t('publicSite.models.tokenBilling')
  }
}
function unitSuffix(mode: string) {
  return t(mode === BILLING_MODE_IMAGE ? 'modelPlaza.table.perUnitImage' : 'modelPlaza.table.perUnitRequest')
}
function entryRate({ model, group }: CatalogEntry) {
  return modelRate(model, group)
}
function onShortcut(event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k' && !filterOpen.value && !selectedEntry.value) {
    event.preventDefault()
    searchInput.value?.focus()
  }
}
function closeBackdrop(event: MouseEvent) {
  const dialog = event.currentTarget as HTMLDialogElement
  if (event.target !== dialog) return
  const bounds = dialog.getBoundingClientRect()
  if (event.clientX >= bounds.left && event.clientX <= bounds.right && event.clientY >= bounds.top && event.clientY <= bounds.bottom) return
  filterOpen.value = false
}

/** Export source prices with explicit units and the applicable rate, preserving all pricing details. */
function exportCsv() {
  const cell = (value: unknown) => {
    let text = value == null ? '' : String(value)
    // eslint-disable-next-line no-control-regex -- CSV formula detection also skips leading ASCII control characters.
    if (/^[\s\u0000-\u001f]*[=+@-]/.test(text)) text = `'${text}`
    return `"${text.replace(/"/g, '""')}"`
  }
  const columns = [t('publicSite.models.csv.provider'), t('publicSite.models.csv.model'), t('publicSite.models.csv.group'), t('publicSite.models.csv.billingMode'), t('publicSite.models.csv.effectiveRate'), t('publicSite.models.csv.inputPrice'), t('publicSite.models.csv.outputPrice'), t('publicSite.models.csv.cacheWritePrice'), t('publicSite.models.csv.cacheReadPrice'), t('publicSite.models.csv.requestPrice'), t('publicSite.models.csv.pricing'), t('publicSite.models.csv.longContextBasis'), t('publicSite.models.csv.timePricing'), t('publicSite.models.csv.peakPricing'), t('publicSite.models.csv.officialPricing')]
  const rows = entries.value.map(entry => {
    const { model, group } = entry
    return [model.platform, model.name, group.name, model.pricing?.billing_mode || BILLING_MODE_TOKEN, entryRate(entry), model.pricing?.input_price, model.pricing?.output_price, model.pricing?.cache_write_price, model.pricing?.cache_read_price, model.pricing?.per_request_price, JSON.stringify(model.pricing), model.long_context_basis ?? '', JSON.stringify(model.time_pricing ?? null), JSON.stringify({ enabled: group.peak_rate_enabled, start: group.peak_start, end: group.peak_end, multiplier: group.peak_rate_multiplier }), JSON.stringify(model.official_pricing)]
  })
  const blob = new Blob(['\uFEFF', [columns, ...rows].map(row => row.map(cell).join(',')).join('\r\n')], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'model-pricing.csv'
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}
onMounted(() => window.addEventListener('keydown', onShortcut))
onUnmounted(() => {
  window.removeEventListener('keydown', onShortcut)
  document.documentElement.classList.remove('catalog-modal-open')
})
</script>

<style scoped>
.model-catalog {
  --surface:#fff;
  --ink:#1d1d1f;
  --muted:#626267;
  --line:#d9d9de;
  --soft:#f5f5f7;
  --canvas:#f5f5f7;
  --accent:#0066cc;
  --on-accent:#fff;
  --accent-soft:#edf5ff;
  color:var(--ink);
  background:var(--canvas);
  font-family:-apple-system,BlinkMacSystemFont,"Inter","PingFang SC","Microsoft YaHei",sans-serif;
}
.dark .model-catalog {
  --surface:#1c1c1e;
  --ink:#f5f5f7;
  --muted:#ababaf;
  --line:#38383c;
  --soft:#252528;
  --canvas:#101012;
  --accent:#71b7ff;
  --on-accent:#101012;
  --accent-soft:#142e47;
}
.catalog-page { max-width:1320px; margin:auto; padding:clamp(36px,6vw,82px) 36px 72px; }
.catalog-hero { margin-bottom:24px; }
.catalog-intro { display:flex; justify-content:space-between; align-items:flex-end; gap:20px; }
.catalog-hero h1 { margin:0; font-size:clamp(36px,5vw,64px); font-weight:600; letter-spacing:-.045em; line-height:1.06; }
.catalog-intro p { max-width:52ch; margin:14px 0 0; color:var(--muted); font-size:clamp(15px,1.5vw,19px); line-height:1.5; }
.market-guide { flex:none; display:inline-flex; align-items:center; gap:7px; min-height:40px; padding:8px 16px; border:1px solid var(--accent); border-radius:24px; color:var(--accent); font-size:13px; font-weight:500; white-space:nowrap; }
.market-guide:hover { color:var(--on-accent); background:var(--accent); }
.catalog-discovery { display:flex; align-items:center; gap:10px; margin-top:34px; }
.catalog-search { display:flex; flex:1; align-items:center; gap:12px; min-width:0; min-height:48px; padding:0 16px; border:1px solid var(--line); border-radius:24px; color:var(--muted); background:var(--surface); }
.catalog-search:focus-within { outline:2px solid var(--accent); outline-offset:2px; }
.catalog-search input { flex:1; width:100%; min-width:0; min-height:46px; border:0; padding:0; background:transparent; color:var(--ink); box-shadow:none; outline:none; font-size:16px; }
.catalog-search input::placeholder { color:var(--muted); }
.catalog-search kbd { flex:none; border:1px solid var(--line); border-radius:6px; padding:3px 6px; font-size:11px; color:var(--muted); }
.catalog-totals { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:12px; margin:0 0 24px; }
.catalog-stat { display:flex; flex-direction:column-reverse; gap:5px; min-width:0; padding:18px 22px; border:1px solid var(--line); border-radius:18px; background:var(--surface); }
.catalog-stat dt { color:var(--muted); font-size:13px; line-height:1.4; }
.catalog-stat dd { margin:0; font-size:clamp(26px,3vw,34px); font-weight:600; line-height:1.1; letter-spacing:-.035em; font-variant-numeric:tabular-nums; }
.catalog-description { margin-bottom:22px; }
.catalog-description .market-guide { border-color:var(--line); background:var(--surface); }
.catalog-description .market-guide:hover { color:var(--accent); background:var(--accent-soft); }
.description-markdown { margin-top:12px; padding:16px 20px; border:1px solid var(--line); border-radius:16px; background:var(--surface); font-size:14px; line-height:1.65; overflow-wrap:anywhere; overflow:auto; }
.description-markdown :deep(a) { color:var(--accent); text-decoration:underline; }
.market-results { min-width:0; scroll-margin-top:88px; }
.model-count { margin:0; border-radius:4px; font-size:17px; font-weight:600; font-variant-numeric:tabular-nums; scroll-margin-top:88px; }
.provider-navigation { display:flex; align-items:center; flex-wrap:wrap; gap:8px; padding:4px; margin:12px -4px 0; }
.provider-navigation button { display:inline-flex; flex:none; align-items:center; justify-content:center; gap:7px; min-height:36px; padding:7px 13px; border:1px solid var(--line); border-radius:24px; background:var(--surface); font-size:13px; font-weight:500; transition:color 180ms,background-color 180ms,border-color 180ms; }
.provider-navigation button:hover:not(:disabled) { background:var(--soft); }
.provider-navigation button[aria-pressed="true"] { border-color:var(--accent); background:var(--accent-soft); color:var(--accent); }
.provider-navigation button:disabled { opacity:.35; cursor:not-allowed; }
.catalog-toolbar { display:flex; flex-wrap:wrap; align-items:center; justify-content:space-between; gap:10px; padding:0 0 12px; }
.market-sort { display:inline-flex; align-items:center; justify-content:space-between; gap:10px; min-width:0; max-width:175px; min-height:40px; border:1px solid var(--line); border-radius:22px; padding:7px 13px; color:var(--ink); background:var(--surface); font-size:13px; transition:background-color 180ms,border-color 180ms; }
.market-sort:hover, .market-sort[data-state="open"] { border-color:var(--accent); }
.sort-chevron { flex:none; color:var(--muted); transition:transform 220ms cubic-bezier(.22,1,.36,1); }
.market-sort[data-state="open"] .sort-chevron { transform:rotate(180deg); }
.summary-actions { display:flex; flex-wrap:wrap; align-items:center; gap:8px; }
.catalog-button, .catalog-icon-button { min-height:40px; height:auto; padding:8px 13px; border:1px solid var(--line); border-radius:22px; background:var(--surface); color:var(--ink); font-size:13px; box-shadow:none; }
.catalog-button:hover, .catalog-icon-button:hover { border-color:var(--accent); background:var(--surface); color:var(--accent); }
.catalog-icon-button { width:40px; padding:0; }
.catalog-button[aria-pressed="true"], .catalog-button-dark { border-color:var(--accent); background:var(--accent); color:var(--on-accent); }
.catalog-button-dark:hover { opacity:.85; background:var(--accent); color:var(--on-accent); }
.catalog-view-switch { position:relative; isolation:isolate; display:grid; grid-template-columns:repeat(2,1fr); width:88px; height:40px; padding:3px; border:1px solid var(--line); border-radius:24px; background:var(--surface); }
.catalog-view-switch::before { content:""; position:absolute; z-index:-1; inset:3px auto 3px 3px; width:calc((100% - 6px) / 2); border-radius:22px; background:var(--accent-soft); transition:transform 280ms cubic-bezier(.22,1,.36,1); }
.catalog-view-switch[data-view="cards"]::before { transform:translateX(100%); }
.catalog-view-switch button { display:grid; place-items:center; border-radius:22px; color:var(--muted); transition:color 220ms; }
.catalog-view-switch .active { color:var(--accent); }
.filter-button { display:inline-flex; min-height:48px; padding-inline:18px; }
.filter-count { display:grid; place-items:center; min-width:18px; height:18px; border-radius:50%; color:var(--on-accent); background:var(--accent); font-size:10px; }
.clear-filters { min-height:40px; padding-inline:5px; color:var(--accent); font-size:12px; }
.clear-filters:hover { text-decoration:underline; text-underline-offset:4px; }
.catalog-grid { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:18px; }
.catalog-card { display:flex; flex-direction:column; min-width:0; border:1px solid var(--line); border-radius:26px; padding:23px; background:var(--surface); box-shadow:0 3px 8px rgb(0 0 0 / 2%); transition:transform 260ms cubic-bezier(.22,1,.36,1), box-shadow 260ms, border-color 260ms; }
@media (hover:hover) { .catalog-card:hover { transform:translateY(-4px); border-color:color-mix(in srgb,var(--ink) 16%,var(--line)); box-shadow:0 12px 26px rgb(0 0 0 / 6%); } }
.market-card-heading { display:flex; align-items:center; gap:8px; }
.card-main { display:flex; align-items:center; gap:13px; min-width:0; min-height:50px; text-align:left; flex:1; border-radius:12px; }
.card-brand { flex:none; display:grid; place-items:center; width:48px; height:48px; border:1px solid var(--line); border-radius:16px; background:var(--soft); }
.card-main h3 { font-size:18px; line-height:1.4; font-weight:650; letter-spacing:-.035em; overflow-wrap:anywhere; }
.market-copy { flex:none; display:grid; place-items:center; width:34px; height:34px; color:var(--muted); border-radius:50%; }
.market-copy:hover { background:var(--soft); color:var(--ink); }
.card-badges { display:flex; flex-wrap:wrap; gap:6px; margin:16px 0 10px; }
.card-badges span { font-size:10px; line-height:22px; padding:0 8px; background:var(--soft); border-radius:20px; color:var(--muted); }
.card-description { color:var(--muted); font-size:12px; min-height:38px; line-height:19px; margin-bottom:20px; display:-webkit-box; -webkit-box-orient:vertical; -webkit-line-clamp:2; overflow:hidden; overflow-wrap:anywhere; }
.market-metrics { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:20px 18px; border-radius:18px; padding:18px; margin-top:auto; background:var(--soft); }
.market-metrics dt { color:var(--muted); font-size:10px; line-height:1.5; }
.market-metrics dd { margin-top:7px; font-size:24px; line-height:1.2; font-weight:550; letter-spacing:-.035em; font-variant-numeric:tabular-nums; overflow-wrap:anywhere; }
.market-metrics small { margin-left:3px; font-size:10px; font-weight:400; letter-spacing:0; color:var(--muted); }
.market-metrics .metric-meta dd { font-size:15px; line-height:1.9; letter-spacing:0; }
.card-footer { display:flex; align-items:center; justify-content:space-between; gap:9px; padding-top:18px; font-size:11px; color:var(--muted); }
.card-group { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.card-footer button { display:inline-flex; flex-shrink:0; gap:5px; align-items:center; min-height:36px; padding:8px 11px; border:1px solid var(--line); border-radius:22px; color:var(--accent); font-size:12px; font-weight:500; transition:background-color 180ms,color 180ms; }
.card-footer button:hover { background:var(--accent-soft); }
.catalog-price-note { margin-top:13px; font-size:12px; line-height:1.6; color:var(--muted); }
.catalog-table-scroll {
  max-width:100%;
  overflow-x:auto;
  border:1px solid var(--line);
  border-radius:18px;
  background:var(--surface);
  scrollbar-color:var(--line) var(--surface);
}
.catalog-table {
  width:100%;
  min-width:790px;
  text-align:left;
  font-size:13px;
  line-height:1.5;
}
.catalog-table :is(th,td) { padding:12px 14px; vertical-align:middle; }
.catalog-table thead { color:var(--muted); background:var(--soft); }
.catalog-table thead th { padding-block:13px; font-weight:600; }
.catalog-table thead small { display:block; font-size:11px; font-weight:400; }
.catalog-table tbody tr { border-top:1px solid var(--line); }
.catalog-table tbody tr:hover, .catalog-table tbody tr:focus-within { background:var(--soft); }
.catalog-table tbody th { width:25%; font-weight:600; }
.catalog-scroll-hint { display:none; }
.offer-model { display:flex; align-items:center; gap:4px; }
.offer-detail { min-height:36px; border-radius:6px; color:var(--accent); text-align:left; overflow-wrap:anywhere; }
.offer-detail:hover { text-decoration:underline; text-underline-offset:3px; }
.offer-model .market-copy { width:34px; height:34px; }
.offer-provider { display:inline-flex; align-items:center; gap:7px; }
.offer-provider :deep(svg) { flex:none; }
.catalog-number { text-align:right; white-space:nowrap; }
.offer-price { font-family:ui-monospace,SFMono-Regular,Consolas,monospace; font-weight:500; font-variant-numeric:tabular-nums; }
.offer-unit { text-align:center; }
.offer-unit small { margin-inline-start:3px; color:var(--muted); font-family:inherit; font-size:11px; }
.offer-terms { display:flex; flex-wrap:wrap; align-items:center; gap:5px; min-width:70px; }
.offer-mode, .offer-dynamic { border-radius:999px; padding:3px 8px; color:var(--muted); background:var(--soft); font-size:11px; font-weight:500; white-space:nowrap; }
.offer-dynamic { color:var(--accent); background:var(--accent-soft); }
.catalog-pagination { display:flex; align-items:center; justify-content:flex-end; gap:12px; margin-top:12px; color:var(--muted); font-size:13px; font-variant-numeric:tabular-nums; }
.catalog-page-button { display:grid; place-items:center; width:40px; height:40px; border:1px solid var(--line); border-radius:50%; color:var(--accent); background:var(--surface); }
.catalog-page-button:hover:not(:disabled) { border-color:var(--accent); background:var(--accent-soft); }
.catalog-page-button:disabled { opacity:.45; cursor:not-allowed; }
.catalog-state { display:grid; place-content:center; justify-items:center; gap:14px; min-height:300px; padding:28px; border:1px solid var(--line); border-radius:26px; background:var(--soft); color:var(--muted); font-size:14px; text-align:center; }
.catalog-spinner { width:28px; height:28px; border:2px solid var(--line); border-top-color:var(--accent); border-radius:50%; animation:market-spin .8s linear infinite; }
@keyframes market-spin { to { transform:rotate(360deg); } }
.catalog-filter-dialog { position:fixed; inset:0; margin:auto; width:min(calc(100% - 32px),620px); max-height:calc(100dvh - 48px); padding:0; border:1px solid var(--line); border-radius:26px; background:var(--surface); color:var(--ink); box-shadow:0 25px 80px rgb(0 0 0 / 18%); }
.catalog-filter-dialog[open] { display:flex; flex-direction:column; animation:market-filter-in 300ms cubic-bezier(.22,1,.36,1) both; }
@keyframes market-filter-in {
  from { opacity:0; transform:translateY(8px) scale(.98); }
  to { opacity:1; transform:translateY(0) scale(1); }
}
.catalog-filter-dialog::backdrop { background:rgb(18 18 18 / 35%); backdrop-filter:blur(4px); }
.catalog-dialog-header, .catalog-dialog-footer { flex:none; display:flex; align-items:center; justify-content:space-between; gap:12px; padding:20px 24px; }
.catalog-dialog-header { border-bottom:1px solid var(--line); }
.catalog-dialog-footer { border-top:1px solid var(--line); }
.catalog-dialog-header h2 { overflow-wrap:anywhere; font-size:19px; letter-spacing:-.03em; font-weight:650; }
.catalog-dialog-body { min-height:0; overflow:auto; overscroll-behavior:contain; padding:24px; }
.catalog-filter-content { display:grid; gap:25px; }
.catalog-filter-content h3 { font-size:12px; margin-bottom:11px; color:var(--muted); font-weight:500; }
.catalog-filter-pills, .manufacturer-options, .filter-footer { display:flex; flex-wrap:wrap; gap:8px; }
.filter-footer { width:100%; justify-content:space-between; }
.catalog-filter-content button { display:flex; gap:7px; align-items:center; min-height:40px; border:1px solid var(--line); border-radius:22px; padding:7px 12px; font-size:12px; transition:color 180ms,background-color 180ms; }
.catalog-filter-content button:hover:not(:disabled) { background:var(--soft); }
.catalog-filter-content button.selected { background:var(--accent-soft); border-color:var(--accent); color:var(--accent); }
.catalog-filter-content button:disabled { opacity:.35; cursor:not-allowed; }
.model-catalog :is(button,a,select,.model-count,.catalog-table-scroll):focus-visible { outline:2px solid var(--accent); outline-offset:2px; }
:global(html.catalog-modal-open) { overflow:hidden; }
@media (max-width:1024px) {
  .catalog-grid { grid-template-columns:repeat(2,minmax(0,1fr)); }
  .catalog-page { padding-inline:28px; }
}
@media (max-width:800px) {
  .catalog-page { padding:40px 18px 52px; }
  .catalog-card { padding:20px; border-radius:23px; }
  .market-metrics { padding:15px; gap:18px 12px; }
}
@media (max-width:850px) {
  .catalog-scroll-hint { display:flex; align-items:center; gap:6px; margin-top:8px; color:var(--muted); font-size:12px; }
}
@media (max-width:600px) {
  .catalog-page { padding-inline:16px; }
  .catalog-hero { margin-bottom:18px; }
  .catalog-intro { flex-wrap:wrap; gap:12px; }
  .catalog-intro p { margin-top:10px; }
  .catalog-discovery { flex-wrap:wrap; margin-top:22px; gap:8px; }
  .catalog-search { padding-inline:12px; min-height:46px; }
  .catalog-search input { min-height:44px; }
  .catalog-search kbd { display:none; }
  .market-guide { min-height:44px; }
  .filter-button { min-height:46px; padding-inline:12px; }
  .clear-filters { flex-basis:100%; min-height:32px; text-align:left; }
  .provider-navigation { flex-wrap:nowrap; overflow-x:auto; scrollbar-width:thin; scrollbar-color:var(--line) transparent; }
  .catalog-totals { gap:7px; margin-bottom:18px; }
  .catalog-stat { padding:13px 10px; border-radius:14px; }
  .catalog-stat dt { font-size:11px; overflow-wrap:anywhere; }
  .catalog-stat dd { font-size:25px; }
  .catalog-toolbar { align-items:flex-start; flex-direction:column; }
  .catalog-grid { grid-template-columns:minmax(0,1fr); }
  .market-sort { max-width:140px; }
  .rate-toggle { padding-inline:10px; }
  .catalog-button, .catalog-icon-button, .market-sort, .provider-navigation button, .catalog-page-button { min-height:44px; }
  .catalog-icon-button, .catalog-page-button { width:44px; }
  .summary-actions { gap:5px; }
  .catalog-view-switch { height:44px; }
  .catalog-table :is(th,td) { padding-inline:12px; }
  .catalog-card { padding:22px; }
  .market-metrics { padding:19px; gap:22px 18px; }
  .market-copy { width:40px; height:40px; }
  .card-footer button { min-height:40px; }
  .catalog-filter-dialog { width:calc(100% - 20px); max-height:calc(100dvh - 28px); border-radius:24px; }
  .catalog-dialog-header, .catalog-dialog-footer { padding:17px 18px; }
  .catalog-dialog-body { padding:20px 18px; }
  .catalog-filter-content button { min-height:44px; }
}
@media (prefers-reduced-motion:reduce) {
  .catalog-card, .catalog-view-switch::before, .catalog-view-switch button, .provider-navigation button, .card-footer button, .catalog-filter-content button { transition:none; }
  .catalog-card:hover { transform:none; }
  .catalog-spinner, .catalog-filter-dialog[open] { animation:none; }
  .market-sort, .sort-chevron { transition:none; }
  .catalog-filter-dialog::backdrop { backdrop-filter:none; }
}
</style>

<style>
/* Reka portals the listbox into the document body, outside the catalog scope. */
.market-sort-content { --sort-surface:#fff; --sort-ink:#1d1d1f; --sort-accent:#0066cc; --sort-line:#d9d9de; --sort-soft:#edf5ff; z-index:70; min-width:max(176px,var(--reka-select-trigger-width)); max-width:calc(100vw - 24px); max-height:var(--reka-select-content-available-height); overflow:hidden; border:1px solid var(--sort-line); border-radius:18px; background:var(--sort-surface); color:var(--sort-ink); box-shadow:0 12px 36px rgb(0 0 0 / 10%); transform-origin:var(--reka-select-content-transform-origin); }
.dark .market-sort-content { --sort-surface:#1c1c1e; --sort-ink:#f5f5f7; --sort-accent:#71b7ff; --sort-line:#38383c; --sort-soft:#142e47; }
.market-sort-content[data-state="open"] { animation:sort-open 220ms cubic-bezier(.22,1,.36,1); }
.market-sort-content[data-state="closed"] { animation:sort-close 140ms ease-in; }
.market-sort-content .market-sort-options { padding:6px; }
.market-sort-content .market-sort-option { position:relative; display:flex; align-items:center; justify-content:space-between; gap:22px; min-height:40px; padding:10px 12px; border-radius:12px; outline:none; font-size:12px; line-height:1.4; cursor:pointer; user-select:none; transition:background-color 150ms; }
.market-sort-content .market-sort-option[data-state="checked"] { font-weight:600; background:var(--sort-soft); }
.market-sort-content .market-sort-option[data-highlighted] { background:var(--sort-soft); box-shadow:inset 0 0 0 1px var(--sort-line); }
.market-sort-content .sort-check { display:flex; align-items:center; justify-content:center; flex:none; width:16px; color:var(--sort-accent); }
@keyframes sort-open { from { opacity:0; transform:translateY(-4px) scale(.97); } to { opacity:1; transform:translateY(0) scale(1); } }
@keyframes sort-close { from { opacity:1; transform:scale(1); } to { opacity:0; transform:scale(.98); } }
@media (max-width:600px) {
  .market-sort-content .market-sort-option { min-height:44px; }
}
@media (prefers-reduced-motion:reduce) {
  .market-sort-content[data-state] { animation:none; }
  .market-sort-content .market-sort-option { transition:none; }
}
</style>
