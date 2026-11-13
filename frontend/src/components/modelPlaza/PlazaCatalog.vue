<template>
  <div class="model-catalog">
    <div class="catalog-page">
      <header class="catalog-hero">
        <div>
          <h1>{{ t('publicSite.market.heading') }}</h1>
          <p>{{ t('publicSite.models.subtitle') }}</p>
        </div>
        <RouterLink v-if="!descriptionHtml" to="/home#api-example" class="market-guide"><Icon name="document" size="sm" />{{ t('publicSite.nav.apiDocs') }}</RouterLink>
        <Button v-else variant="ghost" size="sm" @click="descriptionOpen = !descriptionOpen" :aria-expanded="descriptionOpen" aria-controls="market-description">
          <Icon name="document" size="sm" />{{ t('publicSite.models.priceDescription') }}
        </Button>
      </header>
      <div v-if="descriptionHtml" v-show="descriptionOpen" id="market-description" class="catalog-description">
        <div class="description-markdown" v-html="descriptionHtml"></div>
      </div>

      <div class="market-layout">
        <aside class="market-sidebar">
          <h2>{{ t('publicSite.models.modelCount', { count: uniqueModelCount }) }}</h2>
          <nav v-if="!loading && !error" :aria-label="t('publicSite.models.filterModels')">
            <details open>
              <summary>{{ t('publicSite.market.type') }}<Icon name="chevronDown" size="xs" /></summary>
              <button v-for="mode in modes" :key="mode" type="button" :aria-pressed="selectedMode === mode" @click="selectedMode = mode">{{ modeLabel(mode) }}</button>
            </details>
            <details open>
              <summary>{{ t('publicSite.models.provider') }}<Icon name="chevronDown" size="xs" /></summary>
              <button type="button" :aria-pressed="selectedPlatform === 'all'" @click="selectedPlatform = 'all'">{{ t('publicSite.models.allProviders') }}</button>
              <button v-for="platform in platforms" :key="platform" type="button" :aria-pressed="selectedPlatform === platform" :disabled="!platformEnabled(platform)" @click="selectedPlatform = platform">
                <PlatformIcon :platform="platform as GroupPlatform" size="sm" />{{ platformLabel(platform) }}
              </button>
            </details>
            <details open>
              <summary>{{ t('publicSite.models.group') }}<Icon name="chevronDown" size="xs" /></summary>
              <select v-model="selectedGroupId" :aria-label="t('publicSite.models.group')">
                <option value="all">{{ t('publicSite.models.allGroups') }}</option>
                <option v-for="group in groupOptions" :key="group.id" :value="group.id" :disabled="!groupEnabled(group)">{{ group.name }}</option>
              </select>
            </details>
            <details open>
              <summary>{{ t('publicSite.models.rate') }}<Icon name="chevronDown" size="xs" /></summary>
              <select v-model="selectedRate" :aria-label="t('publicSite.models.rate')">
                <option value="all">{{ t('publicSite.models.allRates') }}</option>
                <option v-for="rate in rates" :key="rate" :value="rate" :disabled="!rateEnabled(rate)">{{ rate }}x</option>
              </select>
            </details>
            <button v-if="activeFilterCount" type="button" @click="resetFilters">{{ t('publicSite.models.clearFilters') }}</button>
          </nav>
        </aside>

        <section class="market-results" :aria-label="t('publicSite.market.heading')">
          <div class="catalog-toolbar">
            <Button variant="outline" size="sm" type="button" class="catalog-button filter-button" :disabled="loading || error" :aria-expanded="filterOpen" @click="filterOpen = true">
              <Icon name="filter" size="sm" />{{ t('publicSite.models.filter') }}
              <span v-if="activeFilterCount" class="filter-count">{{ activeFilterCount }}</span>
            </Button>
            <select v-if="viewMode === 'cards'" v-model="sortOrder" class="market-sort" :aria-label="t('publicSite.market.sort')">
              <option value="default">{{ t('publicSite.market.defaultSort') }}</option>
              <option value="latest">{{ t('publicSite.market.latest') }}</option>
              <option value="name">{{ t('publicSite.market.byName') }}</option>
            </select>
            <div class="catalog-search">
              <Icon name="search" size="sm" />
              <label for="public-model-search" class="sr-only">{{ t('publicSite.models.searchLabel') }}</label>
              <input id="public-model-search" ref="searchInput" v-model="searchQuery" type="search" :placeholder="t('publicSite.market.search')" autocomplete="off" />
            </div>
            <div class="summary-actions">
              <Button variant="outline" size="sm" type="button" class="catalog-icon-button" :title="t('publicSite.models.exportCsv')" :aria-label="t('publicSite.models.exportCsvLabel')" :disabled="!entries.length" @click="exportCsv"><Icon name="download" size="sm" /></Button>
              <Button variant="outline" size="sm" type="button" class="catalog-button rate-toggle" :aria-pressed="showRate" @click="showRate = !showRate">{{ t('publicSite.models.rate') }}</Button>
              <div class="catalog-view-switch" role="group" :aria-label="t('publicSite.models.viewMode')">
                <button type="button" :class="{ active: viewMode === 'cards' }" :aria-pressed="viewMode === 'cards'" :aria-label="t('publicSite.models.cardView')" @click="viewMode = 'cards'"><Icon name="grid" size="sm" /></button>
                <button type="button" :class="{ active: viewMode === 'table' }" :aria-pressed="viewMode === 'table'" :aria-label="t('publicSite.models.tableView')" @click="viewMode = 'table'"><Icon name="menu" size="sm" /></button>
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
              <article v-for="{ entry, groupCount } in cards" :key="entry.key" class="catalog-card">
                <div class="market-card-heading">
                  <button type="button" class="card-main" @click="selectedEntry = entry">
                    <PlatformIcon :platform="entry.model.platform as GroupPlatform" size="md" />
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
                    <div><dt>{{ t('publicSite.market.input') }}</dt><dd>{{ formatCatalogPrice(entry.prices.input, 1_000_000) }}<small v-if="entry.prices.input != null">/M</small></dd></div>
                    <div><dt>{{ t('publicSite.market.output') }}</dt><dd>{{ formatCatalogPrice(entry.prices.output, 1_000_000) }}<small v-if="entry.prices.output != null">/M</small></dd></div>
                    <div><dt>{{ t('publicSite.market.cache') }}</dt><dd>{{ formatCatalogPrice(entry.prices.cache, 1_000_000) }}<small v-if="entry.prices.cache != null">/M</small></dd></div>
                  </template>
                  <div v-else><dt>{{ billingLabel(entry.model) }}</dt><dd>{{ formatCatalogPrice(entry.prices.request) }}</dd></div>
                  <div><dt>{{ t(showRate ? 'publicSite.models.rate' : 'publicSite.market.billing') }}</dt><dd>{{ showRate ? entryRate(entry) + 'x' : billingLabel(entry.model) }}</dd></div>
                </dl>
                <footer class="card-footer">
                  <span class="card-group" :title="entry.group.name">{{ entry.group.name }}<template v-if="groupCount > 1"> · {{ t('publicSite.models.groupCount', { count: groupCount }) }}</template></span>
                  <button type="button" :aria-label="t('publicSite.models.priceDetailsLabel', { model: entry.model.name })" @click="selectedEntry = entry">{{ t('publicSite.market.details') }} <Icon name="arrowRight" size="xs" /></button>
                </footer>
              </article>
            </div>
            <div v-else class="catalog-tables"><PlazaGroupSection v-for="group in filteredGroups" :key="group.id" :group="group" /></div>
            <p v-if="entries.length && viewMode === 'cards'" class="catalog-price-note">{{ t('publicSite.market.priceNote') }}</p>
            <p v-if="!isAuthenticated" class="catalog-price-note">{{ t('modelPlaza.anonymousHint') }}</p>
          </template>
        </section>
      </div>
    </div>
    <dialog
      ref="filterDialog"
      class="catalog-filter-dialog"
      aria-labelledby="catalog-filter-title"
      @cancel.prevent="filterOpen = false"
      @close="filterOpen = false"
      @click="closeBackdrop($event, 'filter')"
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
              <span class="manufacturer-letter">A</span> {{ t('publicSite.models.allProviders') }}
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

    <dialog
      ref="detailDialog"
      class="catalog-detail-dialog"
      aria-labelledby="catalog-detail-title"
      @cancel.prevent="selectedEntry = null"
      @close="selectedEntry = null"
      @click="closeBackdrop($event, 'detail')"
    >
      <header class="catalog-dialog-header">
        <h2 id="catalog-detail-title">{{ selectedEntry?.model.name || t('publicSite.models.modelDetails') }}</h2>
        <Button variant="outline" size="sm" type="button" class="catalog-icon-button" :aria-label="t('publicSite.models.closeDetails')" @click="selectedEntry = null"><Icon name="x" size="md" /></Button>
      </header>
      <div class="catalog-dialog-body">
        <PlazaGroupSection v-for="entry in detailEntries" :key="entry.key" :group="{ ...entry.group, models: [entry.model] }" />
      </div>
      <footer class="catalog-dialog-footer">
        <Button variant="outline" size="sm" v-if="selectedEntry" type="button" class="catalog-button" @click="copyToClipboard(selectedEntry.model.name)"><Icon name="copy" size="sm" /> {{ t('publicSite.models.copyName') }}</Button>
      </footer>
    </dialog>
  </div>
</template>

<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAuthStore } from '@/stores/auth'
import { platformLabel } from '@/utils/platformColors'
import { BILLING_MODE_IMAGE, BILLING_MODE_TOKEN, BILLING_MODE_VIDEO } from '@/constants/channel'
import type { ModelPlazaGroup, ModelPlazaResponse, PlazaModel } from '@/api/modelPlaza'
import type { GroupPlatform } from '@/types'
import { catalogPrices, formatCatalogPrice, groupRate, modelRate } from './catalogPricing'
import PlazaGroupSection from './PlazaGroupSection.vue'

const props = defineProps<{ response: ModelPlazaResponse | null; loading: boolean; error?: boolean }>()
interface CatalogEntry { key: string; group: ModelPlazaGroup; model: PlazaModel; prices: ReturnType<typeof catalogPrices> }
const { t } = useI18n()
const route = useRoute()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()
const isAuthenticated = computed(() => authStore.isAuthenticated)
const selectedPlatform = ref('all')
const selectedMode = ref('all')
const sortOrder = ref('default')
const descriptionOpen = ref(false)
const selectedGroupId = ref<number | 'all'>('all')
const selectedRate = ref<number | 'all'>('all')
const searchQuery = ref(typeof route.query.search === 'string' ? route.query.search : '')
watch(() => route.query.search, search => {
  searchQuery.value = typeof search === 'string' ? search : ''
})
const searchInput = ref<HTMLInputElement | null>(null)
const filterOpen = ref(false)
const selectedEntry = ref<CatalogEntry | null>(null)
const filterDialog = ref<HTMLDialogElement | null>(null)
const detailDialog = ref<HTMLDialogElement | null>(null)
const showRate = ref(false)
const viewMode = ref<'cards' | 'table'>('cards')
const groups = computed(() => props.response?.groups ?? [])
const effectiveRate = groupRate
const modes = computed(() => ['all', ...new Set(groups.value.flatMap(group => group.models.map(model => model.pricing?.billing_mode || BILLING_MODE_TOKEN)))])
const latestOrder = computed(() => new Map((props.response?.latest_models ?? []).map((model, index) => [modelKey(model), index])))
const platforms = computed(() => [...new Set(groups.value.map(group => group.platform).filter(Boolean))].sort())
const groupOptions = computed(() => groups.value.map(group => ({ id: group.id, name: group.name, platform: group.platform, rate: effectiveRate(group) })))
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
watch(selectedEntry, entry => {
  if (entry && !detailDialog.value?.open) detailDialog.value?.showModal()
  else if (!entry) detailDialog.value?.close()
}, { flush: 'post' })
watch([filterOpen, selectedEntry], ([filter, entry]) => {
  document.documentElement.classList.toggle('catalog-modal-open', filter || !!entry)
})

watch(groups, () => {
  if (selectedPlatform.value !== 'all' && !platforms.value.includes(selectedPlatform.value)) selectedPlatform.value = 'all'
  if (selectedGroupId.value !== 'all' && !groups.value.some(group => group.id === selectedGroupId.value)) selectedGroupId.value = 'all'
  if (selectedRate.value !== 'all' && !rates.value.includes(selectedRate.value)) selectedRate.value = 'all'
  if (!modes.value.includes(selectedMode.value)) selectedMode.value = 'all'
  selectedEntry.value = null
})

const filteredGroups = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  return groups.value
    .filter(group => (selectedPlatform.value === 'all' || group.platform === selectedPlatform.value) && (selectedGroupId.value === 'all' || group.id === selectedGroupId.value) && (selectedRate.value === 'all' || effectiveRate(group) === selectedRate.value))
    .map(group => ({ ...group, models: group.models.filter(model => {
      const matchesMode = selectedMode.value === 'all' || selectedMode.value === (model.pricing?.billing_mode || BILLING_MODE_TOKEN)
      const matchesQuery = !query || [model.name, group.name, model.platform, platformLabel(model.platform)].some(value => value.toLowerCase().includes(query))
      return matchesMode && matchesQuery
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
const detailEntries = computed(() => selectedEntry.value ? entriesByModel.value.get(modelKey(selectedEntry.value.model)) ?? [] : [])

function platformEnabled(platform: string) {
  return groupOptions.value.some(group => group.platform === platform && (selectedGroupId.value === 'all' || group.id === selectedGroupId.value) && (selectedRate.value === 'all' || group.rate === selectedRate.value))
}
function groupEnabled(group: { platform: string; rate: number }) {
  return (selectedPlatform.value === 'all' || group.platform === selectedPlatform.value) && (selectedRate.value === 'all' || group.rate === selectedRate.value)
}
function rateEnabled(rate: number) {
  return groupOptions.value.some(group => group.rate === rate && (selectedPlatform.value === 'all' || group.platform === selectedPlatform.value) && (selectedGroupId.value === 'all' || group.id === selectedGroupId.value))
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
function entryRate({ model, group }: CatalogEntry) {
  return modelRate(model, group)
}
function onShortcut(event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k' && !filterOpen.value && !selectedEntry.value) {
    event.preventDefault()
    searchInput.value?.focus()
  }
}
function closeBackdrop(event: MouseEvent, target: 'filter' | 'detail') {
  const dialog = event.currentTarget as HTMLDialogElement
  if (event.target !== dialog) return
  const bounds = dialog.getBoundingClientRect()
  if (event.clientX >= bounds.left && event.clientX <= bounds.right && event.clientY >= bounds.top && event.clientY <= bounds.bottom) return
  if (target === 'filter') filterOpen.value = false
  else selectedEntry.value = null
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
.model-catalog { --surface:#fff; --ink:#151922; --muted:#68758b; --line:#dfe5ee; --soft:#f3f5fb; --accent:#4b4cf3; color:var(--ink); background:linear-gradient(180deg,#edf2ff 0, var(--surface) 70px); }
.dark .model-catalog { --surface:#151922; --ink:#edf0f9; --muted:#9da8be; --line:#303949; --soft:#202637; --accent:#aeb1ff; background:linear-gradient(180deg,#1d2543 0,var(--surface) 90px); }
.catalog-page { max-width:1536px; margin:auto; padding:34px 48px 72px; }
.catalog-hero { display:flex; justify-content:space-between; align-items:flex-start; gap:16px; margin-bottom:40px; }
.catalog-hero h1 { font-size:30px; font-weight:750; letter-spacing:-.035em; line-height:1.2; }
.catalog-hero p { color:var(--muted); font-size:14px; margin-top:14px; }
.market-guide { display:flex; align-items:center; gap:6px; font-size:13px; white-space:nowrap; padding:6px 0; }
.market-guide:hover { color:var(--accent); }
.catalog-description { border:1px solid var(--line); border-radius:10px; padding:14px 16px; margin:-22px 0 24px; font-size:14px; overflow-wrap:anywhere; }
.description-markdown :deep(a) { text-decoration:underline; }
.market-layout { display:grid; grid-template-columns:200px minmax(0,1fr); gap:24px; }
.market-sidebar h2 { font-size:18px; font-weight:650; margin:1px 0 22px; }
.market-sidebar details { margin-bottom:20px; }
.market-sidebar summary { display:flex; justify-content:space-between; cursor:pointer; align-items:center; color:var(--muted); font-size:13px; margin:0 0 9px; list-style:none; }
.market-sidebar summary::-webkit-details-marker { display:none; }
.market-sidebar button { width:100%; text-align:left; display:flex; gap:8px; align-items:center; padding:7px 8px; border-radius:6px; font-size:13px; overflow-wrap:anywhere; }
.market-sidebar button:hover, .market-sidebar button[aria-pressed="true"] { color:var(--accent); background:var(--soft); }
.market-sidebar button:disabled { opacity:.4; }
.market-sidebar select, .market-sort { min-width:0; min-height:34px; border:1px solid var(--line); border-radius:6px; padding:4px 10px; color:var(--ink); background:var(--surface); font-size:13px; }
.market-sidebar select { width:100%; }
.catalog-toolbar { display:flex; flex-wrap:wrap; align-items:center; gap:8px; margin-bottom:16px; }
.catalog-search { display:flex; align-items:center; gap:7px; width:min(320px,100%); min-height:34px; padding:0 10px; border:1px solid var(--line); border-radius:6px; color:var(--muted); background:var(--surface); }
.catalog-search:focus-within { outline:2px solid var(--accent); outline-offset:1px; }
.catalog-search input { width:100%; min-width:0; border:0; padding:0; background:transparent; color:var(--ink); box-shadow:none; outline:none; font-size:13px; }
.catalog-search input::placeholder { color:var(--muted); }
.summary-actions { display:flex; align-items:center; gap:6px; margin-left:auto; }
.catalog-view-switch { display:flex; gap:2px; border-radius:7px; padding:2px; background:var(--soft); }
.catalog-view-switch button { padding:6px 9px; border-radius:5px; color:var(--muted); }
.catalog-view-switch .active { background:var(--surface); color:var(--ink); box-shadow:0 1px 5px #172e6b1c; }
.filter-button { display:none; }
.filter-count { border-radius:4px; padding:0 4px; color:var(--accent); background:var(--soft); }
.catalog-grid { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:16px; }
.catalog-card { min-height:328px; display:flex; flex-direction:column; min-width:0; border:1px solid var(--line); border-radius:12px; padding:20px; background:var(--surface); box-shadow:0 2px 8px #1c274305; transition:border-color .18s, box-shadow .18s; }
.catalog-card:hover { border-color:var(--accent); box-shadow:0 5px 18px #3e52a116; }
.market-card-heading { display:flex; align-items:center; gap:6px; }
.card-main { display:flex; align-items:center; gap:10px; min-width:0; text-align:left; flex:1; }
.card-main h3 { font-size:18px; font-weight:650; overflow-wrap:anywhere; }
.market-copy { color:var(--muted); border-radius:6px; padding:4px; }
.market-copy:hover { background:var(--soft); color:var(--accent); }
.card-badges { display:flex; flex-wrap:wrap; gap:5px; margin:12px 0; }
.card-badges span { font-size:12px; line-height:20px; padding:0 7px; background:var(--soft); border-radius:4px; }
.card-description { color:var(--muted); font-size:13px; min-height:38px; line-height:19px; margin-bottom:15px; display:-webkit-box; -webkit-box-orient:vertical; -webkit-line-clamp:2; overflow:hidden; overflow-wrap:anywhere; }
.market-metrics { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:13px 10px; border-top:1px solid var(--line); padding:14px 0; margin-top:auto; }
.market-metrics dt { color:var(--muted); font-size:12px; }
.market-metrics dd { margin-top:4px; font-size:16px; font-weight:600; font-variant-numeric:tabular-nums; overflow-wrap:anywhere; }
.market-metrics small { margin-left:2px; font-weight:400; color:var(--muted); }
.card-footer { display:flex; align-items:center; justify-content:space-between; gap:9px; border-top:1px solid var(--line); padding-top:13px; font-size:12px; color:var(--muted); }
.card-group { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.card-footer button { display:inline-flex; flex-shrink:0; gap:3px; align-items:center; color:var(--accent); }
.catalog-price-note { margin-top:14px; font-size:12px; color:var(--muted); }
.catalog-tables { display:grid; gap:15px; }
.catalog-state { display:grid; place-content:center; justify-items:center; gap:12px; min-height:250px; border:1px dashed var(--line); border-radius:12px; color:var(--muted); font-size:14px; }
.catalog-spinner { width:30px; height:30px; border:2px solid var(--line); border-top-color:var(--accent); border-radius:50%; animation:market-spin .8s linear infinite; }
@keyframes market-spin { to { transform:rotate(360deg); } }
.catalog-filter-dialog, .catalog-detail-dialog { position:fixed; inset:0; margin:auto; width:min(calc(100% - 26px),600px); max-height:calc(100dvh - 36px); padding:0; border:1px solid var(--line); border-radius:12px; background:var(--surface); color:var(--ink); }
.catalog-detail-dialog { width:min(calc(100% - 26px),1160px); }
.catalog-filter-dialog::backdrop, .catalog-detail-dialog::backdrop { background:#14192c70; }
.catalog-dialog-header, .catalog-dialog-footer { display:flex; align-items:center; justify-content:space-between; gap:12px; padding:14px 16px; border-bottom:1px solid var(--line); }
.catalog-dialog-footer { border-top:1px solid var(--line); border-bottom:0; }
.catalog-dialog-header h2 { overflow-wrap:anywhere; font-weight:650; }
.catalog-dialog-body { display:grid; gap:12px; padding:16px; }
.catalog-filter-content { display:grid; gap:18px; }
.catalog-filter-content h3 { font-size:13px; margin-bottom:8px; color:var(--muted); }
.catalog-filter-pills, .manufacturer-options, .filter-footer { display:flex; flex-wrap:wrap; gap:8px; }
.catalog-filter-content button { display:flex; gap:6px; align-items:center; border:1px solid var(--line); border-radius:7px; padding:5px 9px; font-size:13px; }
.catalog-filter-content button.selected { background:var(--soft); border-color:var(--accent); color:var(--accent); }
.catalog-filter-content button:disabled { opacity:.4; }
:global(html.catalog-modal-open) { overflow:hidden; }
@media (max-width:1190px) { .catalog-grid { grid-template-columns:repeat(2,minmax(0,1fr)); } .catalog-page { padding-inline:30px; } }
@media (max-width:800px) { .market-layout { grid-template-columns:minmax(0,1fr); } .market-sidebar { display:none; } .filter-button { display:inline-flex; } .catalog-page { padding:27px 18px 52px; } .catalog-hero { margin-bottom:24px; } }
@media (max-width:530px) { .catalog-grid { grid-template-columns:minmax(0,1fr); } .catalog-search { flex:1; order:-1; width:100%; flex-basis:100%; } .catalog-hero h1 { font-size:25px; } .market-sort { max-width:120px; } }
</style>
