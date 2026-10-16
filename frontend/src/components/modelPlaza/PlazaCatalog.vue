<template>
  <div class="model-catalog" :style="{
    '--catalog-input-label': JSON.stringify(t('publicSite.models.inputLabel')),
    '--catalog-output-label': JSON.stringify(t('publicSite.models.outputLabel'))
  }">
    <div class="catalog-toolbar">
      <button
        type="button"
        class="catalog-button filter-button"
        :disabled="loading || error"
        :aria-expanded="filterOpen"
        @click="filterOpen = true"
      >
        <Icon name="filter" size="sm" />
        <span>{{ t('publicSite.models.filter') }}</span>
        <span v-if="activeFilterCount" class="filter-count">{{ activeFilterCount }}</span>
      </button>
      <div class="catalog-search">
        <Icon name="search" size="sm" />
        <label for="public-model-search" class="sr-only">{{ t('publicSite.models.searchLabel') }}</label>
        <input
          id="public-model-search"
          ref="searchInput"
          v-model="searchQuery"
          type="search"
          :placeholder="t('publicSite.models.searchPlaceholder')"
          autocomplete="off"
        />
        <kbd>⌘ K</kbd>
      </div>
    </div>

    <div class="catalog-page">
      <header class="catalog-hero">
        <h1>{{ t('publicSite.models.title') }}</h1>
        <p>{{ t('publicSite.models.subtitle') }}</p>
      </header>

      <details v-if="descriptionHtml" class="catalog-description">
        <summary>{{ t('publicSite.models.priceDescription') }} <Icon name="chevronDown" size="sm" /></summary>
        <div class="description-markdown" v-html="descriptionHtml"></div>
      </details>

      <div v-if="loading" class="catalog-state" role="status" aria-live="polite">
        <span class="catalog-spinner"></span>
        <span class="sr-only">{{ t('common.loading') }}</span>
      </div>
      <div v-else-if="error" class="catalog-state catalog-error" role="alert">
        <Icon name="exclamationCircle" size="lg" />
        <p>{{ t('modelPlaza.loadFailed') }}</p>
      </div>
      <template v-else>
        <div class="catalog-summary">
          <div class="summary-provider">
            <span class="provider-initial" aria-hidden="true">
              <PlatformIcon v-if="selectedPlatform !== 'all'" :platform="selectedPlatform as GroupPlatform" size="lg" />
              <span v-else>A</span>
            </span>
            <div>
              <h2>{{ selectedPlatform === 'all' ? t('publicSite.models.allProviders') : platformLabel(selectedPlatform) }}</h2>
              <p>{{ t('publicSite.models.modelCount', { count: uniqueModelCount }) }}<span v-if="filteredGroups.length"> · {{ t('publicSite.models.groupCount', { count: filteredGroups.length }) }}</span></p>
            </div>
          </div>
          <div class="summary-actions">
            <button type="button" class="catalog-icon-button" :title="t('publicSite.models.exportCsv')" :aria-label="t('publicSite.models.exportCsvLabel')" :disabled="!entries.length" @click="exportCsv">
              <Icon name="download" size="sm" />
            </button>
            <button type="button" class="catalog-button rate-toggle" :class="{ active: showRate }" :aria-pressed="showRate" @click="showRate = !showRate">
              <span class="rate-switch" aria-hidden="true"><span></span></span> {{ t('publicSite.models.rate') }}
            </button>
            <div class="catalog-view-switch" role="group" :aria-label="t('publicSite.models.viewMode')">
              <button type="button" :class="{ active: viewMode === 'cards' }" :aria-pressed="viewMode === 'cards'" :aria-label="t('publicSite.models.cardView')" @click="viewMode = 'cards'">
                <Icon name="grid" size="sm" />
              </button>
              <button type="button" :class="{ active: viewMode === 'table' }" :aria-pressed="viewMode === 'table'" :aria-label="t('publicSite.models.tableView')" @click="viewMode = 'table'">
                <Icon name="menu" size="sm" />
              </button>
            </div>
          </div>
        </div>

        <p v-if="!isAuthenticated" class="catalog-anonymous">{{ t('modelPlaza.anonymousHint') }}</p>

        <div v-if="!entries.length" class="catalog-state catalog-empty">
          <Icon name="search" size="lg" />
          <p>{{ searchQuery.trim() || activeFilterCount ? t('modelPlaza.noSearchResult') : t('modelPlaza.empty') }}</p>
          <button v-if="searchQuery.trim() || activeFilterCount" type="button" class="catalog-button" @click="resetFilters">{{ t('publicSite.models.clearFilters') }}</button>
        </div>
        <div v-else-if="viewMode === 'cards'" class="catalog-grid">
          <article v-for="entry in entries" :key="entry.key" class="catalog-card">
            <div class="card-actions">
              <button type="button" :title="t('publicSite.models.copyName')" :aria-label="t('publicSite.models.copyNameLabel', { model: entry.model.name })" @click="copyToClipboard(entry.model.name)">
                <Icon name="copy" size="sm" />
              </button>
              <button type="button" :title="t('publicSite.models.priceDetails')" :aria-label="t('publicSite.models.priceDetailsLabel', { model: entry.model.name })" @click="selectedEntry = entry">
                <Icon name="externalLink" size="sm" />
              </button>
            </div>
            <button type="button" class="card-main" @click="selectedEntry = entry">
              <span class="card-provider" aria-hidden="true"><PlatformIcon :platform="entry.model.platform as GroupPlatform" size="lg" /></span>
              <h3>{{ entry.model.name }}</h3>
            </button>
            <!-- The existing price table owns every calculation, including user/media rates and tiers. -->
            <div class="card-price" :class="{ 'media-price': isMedia(entry.model) }" :aria-label="isMedia(entry.model) ? t('publicSite.models.mediaPriceLabel') : t('publicSite.models.tokenPriceLabel')">
              <PlazaModelPricingTable v-bind="pricingProps(entry)" />
            </div>
            <p class="card-description" :title="entry.group.description">
              {{ entry.group.description || platformLabel(entry.model.platform) }}
            </p>
            <footer class="card-footer">
              <div class="card-badges">
                <span>{{ billingLabel(entry.model) }}</span>
                <span v-if="entry.model.pricing?.intervals?.length">{{ t('publicSite.models.tieredPricing') }}</span>
                <span v-if="entry.model.time_pricing?.periods.length">{{ t('publicSite.models.timePricing') }}</span>
                <span v-if="entry.group.subscription_type === 'subscription'">{{ t('publicSite.models.subscription') }}</span>
                <span v-if="entry.group.is_exclusive">{{ t('publicSite.models.exclusive') }}</span>
              </div>
              <span class="card-group" :title="entry.group.name">{{ entry.group.name }}<template v-if="showRate"> · {{ entryRate(entry) }}x</template></span>
            </footer>
          </article>
        </div>
        <div v-else class="catalog-tables">
          <PlazaGroupSection v-for="group in filteredGroups" :key="group.id" :group="group" />
        </div>

        <p v-if="entries.length && viewMode === 'cards'" class="catalog-price-note">{{ t('publicSite.models.priceNote') }}</p>
      </template>
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
        <button type="button" class="catalog-icon-button" :aria-label="t('publicSite.models.closeFilters')" @click="filterOpen = false"><Icon name="x" size="md" /></button>
      </header>
      <div class="catalog-dialog-body">
        <div class="catalog-filter-content">
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
          <button type="button" class="catalog-button" @click="resetFilters">{{ t('publicSite.models.resetFilters') }}</button>
          <button type="button" class="catalog-button catalog-button-dark" @click="filterOpen = false">{{ t('publicSite.models.showModels', { count: uniqueModelCount }) }}</button>
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
        <button type="button" class="catalog-icon-button" :aria-label="t('publicSite.models.closeDetails')" @click="selectedEntry = null"><Icon name="x" size="md" /></button>
      </header>
      <div class="catalog-dialog-body">
        <PlazaGroupSection v-if="selectedEntry" :group="{ ...selectedEntry.group, models: [selectedEntry.model] }" />
      </div>
      <footer class="catalog-dialog-footer">
        <button v-if="selectedEntry" type="button" class="catalog-button" @click="copyToClipboard(selectedEntry.model.name)"><Icon name="copy" size="sm" /> {{ t('publicSite.models.copyName') }}</button>
      </footer>
    </dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { platformLabel } from '@/utils/platformColors'
import { formatPeakRateWindow, hasPeakRate, serverTimezoneLabel } from '@/utils/peak-rate'
import { BILLING_MODE_IMAGE, BILLING_MODE_TOKEN, BILLING_MODE_VIDEO } from '@/constants/channel'
import type { ModelPlazaGroup, ModelPlazaResponse, PlazaModel } from '@/api/modelPlaza'
import type { GroupPlatform } from '@/types'
import PlazaModelPricingTable from './PlazaModelPricingTable.vue'
import PlazaGroupSection from './PlazaGroupSection.vue'

const props = defineProps<{ response: ModelPlazaResponse | null; loading: boolean; error?: boolean }>()
interface CatalogEntry { key: string; group: ModelPlazaGroup; model: PlazaModel }
const { t } = useI18n()
const route = useRoute()
const authStore = useAuthStore()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()
const isAuthenticated = computed(() => authStore.isAuthenticated)
const selectedPlatform = ref('all')
const selectedGroupId = ref<number | 'all'>('all')
const selectedRate = ref<number | 'all'>('all')
const searchQuery = ref(typeof route.query.search === 'string' ? route.query.search : '')
watch(() => route.query.search, search => {
  if (typeof search === 'string') searchQuery.value = search
})
const searchInput = ref<HTMLInputElement | null>(null)
const filterOpen = ref(false)
const selectedEntry = ref<CatalogEntry | null>(null)
const filterDialog = ref<HTMLDialogElement | null>(null)
const detailDialog = ref<HTMLDialogElement | null>(null)
const showRate = ref(false)
const viewMode = ref<'cards' | 'table'>('cards')
const groups = computed(() => props.response?.groups ?? [])
const effectiveRate = (group: ModelPlazaGroup) => group.user_rate_multiplier ?? group.rate_multiplier
const platforms = computed(() => [...new Set(groups.value.map(group => group.platform).filter(Boolean))].sort())
const groupOptions = computed(() => groups.value.map(group => ({ id: group.id, name: group.name, platform: group.platform, rate: effectiveRate(group) })))
const rates = computed(() => [...new Set(groups.value.map(effectiveRate))].sort((a, b) => a - b))
const activeFilterCount = computed(() => Number(selectedPlatform.value !== 'all') + Number(selectedGroupId.value !== 'all') + Number(selectedRate.value !== 'all'))
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
  selectedEntry.value = null
})

const filteredGroups = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  return groups.value
    .filter(group => (selectedPlatform.value === 'all' || group.platform === selectedPlatform.value) && (selectedGroupId.value === 'all' || group.id === selectedGroupId.value) && (selectedRate.value === 'all' || effectiveRate(group) === selectedRate.value))
    .map(group => ({ ...group, models: query ? group.models.filter(model => model.name.toLowerCase().includes(query)) : group.models }))
    .filter(group => group.models.length > 0)
    .sort((a, b) => effectiveRate(a) - effectiveRate(b) || a.name.localeCompare(b.name))
})
const entries = computed<CatalogEntry[]>(() => filteredGroups.value.flatMap(group => group.models.map(model => ({ key: `${group.id}:${model.platform}:${model.name}`, group, model }))))
const uniqueModelCount = computed(() => new Set(entries.value.map(entry => entry.model.name)).size)

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
  selectedGroupId.value = 'all'
  selectedRate.value = 'all'
  searchQuery.value = ''
}
function isMedia(model: PlazaModel) {
  return (model.pricing?.billing_mode || BILLING_MODE_TOKEN) !== BILLING_MODE_TOKEN
}
function billingLabel(model: PlazaModel) {
  switch (model.pricing?.billing_mode) {
    case BILLING_MODE_IMAGE: return t('publicSite.models.imageBilling')
    case BILLING_MODE_VIDEO: return t('publicSite.models.videoBilling')
    case 'per_request': return t('publicSite.models.requestBilling')
    default: return t('publicSite.models.tokenBilling')
  }
}
function entryRate({ group, model }: CatalogEntry) {
  if (model.pricing?.billing_mode === BILLING_MODE_IMAGE && group.image_rate_independent) return group.image_rate_multiplier
  if (model.pricing?.billing_mode === BILLING_MODE_VIDEO && group.video_rate_independent) return Math.max(0, group.video_rate_multiplier ?? 1)
  return effectiveRate(group)
}
function pricingProps({ group, model }: CatalogEntry) {
  return {
    models: [model],
    platform: group.platform,
    rateMultiplier: group.rate_multiplier,
    userRateMultiplier: group.user_rate_multiplier ?? null,
    imageRateIndependent: group.image_rate_independent,
    imageRateMultiplier: group.image_rate_multiplier,
    videoRateIndependent: group.video_rate_independent,
    videoRateMultiplier: group.video_rate_multiplier,
    peakWindow: hasPeakRate(group) ? formatPeakRateWindow(group, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset)) : '',
    peakRateMultiplier: group.peak_rate_multiplier
  }
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
.model-catalog { --catalog-bg: #fff; --catalog-text: #18181b; --catalog-muted: #919195; --catalog-border: #eeeeef; --catalog-soft: #f7f7f8; color: var(--catalog-text); background: var(--catalog-bg); }
.catalog-toolbar { height: 50px; display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 0 16px; border-bottom: 1px solid #f3f3f3; }
.catalog-button, .catalog-icon-button { display: inline-flex; align-items: center; justify-content: center; gap: 6px; height: 34px; padding: 0 11px; border: 1px solid var(--catalog-border); border-radius: 8px; font-size: 12px; font-weight: 500; color: var(--catalog-text); background: var(--catalog-bg); transition: background .15s, border-color .15s; }
.catalog-button:hover, .catalog-icon-button:hover { background: var(--catalog-soft); border-color: #d4d4d8; }
.catalog-button:disabled, .catalog-icon-button:disabled, .catalog-filter-content button:disabled { cursor: not-allowed; opacity: .35; }
.catalog-button:focus-visible, .catalog-icon-button:focus-visible, .model-catalog button:focus-visible, .catalog-filter-content button:focus-visible { outline: 2px solid #737373; outline-offset: 3px; }
.filter-button { height: 30px; font-size: 12px; }
.filter-count { min-width: 16px; height: 16px; padding: 0 4px; border-radius: 4px; background: var(--catalog-text); color: var(--catalog-bg); font-size: 10px; }
.catalog-search { display: flex; align-items: center; gap: 8px; width: 315px; height: 34px; padding: 0 10px; border: 1px solid var(--catalog-border); border-radius: 8px; color: #b0b0b6; background: var(--catalog-soft); }
.catalog-search:focus-within { border-color: #a1a1aa; }
.catalog-search input { width: 100%; min-width: 0; border: 0; background: transparent; padding: 0; outline: none; box-shadow: none; font-size: 12px; color: var(--catalog-text); }
.catalog-search input::placeholder { color: #a1a1aa; }
.catalog-search kbd { flex-shrink: 0; font-family: inherit; font-size: 10px; line-height: 18px; color: #a1a1aa; }
.catalog-page { max-width: 1440px; margin: 0 auto; padding: 0 16px 70px; }
.catalog-hero { padding: 57px 16px 65px; text-align: center; }
.catalog-hero h1 { margin: 0; font-size: 36px; font-weight: 700; line-height: 1.3; letter-spacing: -1.2px; color: var(--catalog-text); }
.catalog-hero p { margin: 17px 0 0; font-size: 14px; line-height: 1.7; color: #89898e; }
.catalog-description { margin: -24px 0 22px; border: 1px solid var(--catalog-border); border-radius: 12px; font-size: 13px; }
.catalog-description summary { display: flex; align-items: center; justify-content: space-between; padding: 12px 16px; color: var(--catalog-muted); cursor: pointer; list-style: none; }
.catalog-description summary::-webkit-details-marker { display: none; }
.description-markdown { padding: 0 16px 16px; line-height: 1.7; overflow-wrap: anywhere; }
.description-markdown :deep(p) { margin: 8px 0; }
.description-markdown :deep(a) { text-decoration: underline; text-underline-offset: 3px; }
.description-markdown :deep(ul) { padding-left: 20px; list-style: disc; }
.description-markdown :deep(ol) { padding-left: 20px; list-style: decimal; }
.description-markdown :deep(h1), .description-markdown :deep(h2), .description-markdown :deep(h3) { margin: 12px 0 6px; font-weight: 600; }
.catalog-summary { display: flex; align-items: center; justify-content: space-between; gap: 16px; min-height: 68px; padding: 12px 16px; border: 1px solid var(--catalog-border); border-radius: 16px; }
.summary-provider { display: flex; align-items: center; min-width: 0; gap: 12px; }
.summary-provider > div { display: flex; align-items: center; flex-wrap: wrap; min-width: 0; gap: 3px 8px; }
.provider-initial { display: flex; align-items: center; justify-content: center; flex: 0 0 40px; width: 40px; height: 40px; border-radius: 12px; background: var(--catalog-soft); font: 700 19px/1 Arial, sans-serif; }
.provider-initial > span { display: grid; place-items: center; width: 25px; height: 25px; border: 1.5px solid currentColor; border-radius: 50%; font-size: 16px; }
.summary-provider h2 { margin: 0; font-size: 14px; font-weight: 600; line-height: 1.5; }
.summary-provider p { margin: 0; color: var(--catalog-muted); font-size: 11px; }
.summary-actions { display: flex; align-items: center; gap: 10px; flex-shrink: 0; }
.catalog-icon-button { width: 32px; padding: 0; border: 0; color: #8c8c92; }
.rate-toggle { height: 30px; color: #71717a; }
.rate-toggle.active { color: var(--catalog-text); background: var(--catalog-soft); border-color: #b4b4bb; }
.rate-switch { display: inline-flex; align-items: center; width: 28px; height: 17px; padding: 2px; border-radius: 999px; background: var(--catalog-soft); }
.rate-switch > span { width: 13px; height: 13px; border-radius: 50%; background: var(--catalog-bg); box-shadow: 0 1px 2px #00000012; transition: transform .15s; }
.rate-toggle.active .rate-switch { background: var(--catalog-text); }
.rate-toggle.active .rate-switch > span { transform: translateX(11px); }
.catalog-view-switch { display: flex; padding: 3px; height: 32px; border: 1px solid var(--catalog-border); border-radius: 8px; background: var(--catalog-soft); }
.catalog-view-switch button { display: grid; place-items: center; width: 30px; border-radius: 5px; color: #a1a1aa; }
.catalog-view-switch button.active { color: var(--catalog-text); background: var(--catalog-bg); box-shadow: 0 1px 3px #0000000b; }
.catalog-anonymous { margin: 10px 2px 0; color: var(--catalog-muted); font-size: 11px; line-height: 1.6; }
.catalog-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; margin-top: 24px; }
.catalog-card { position: relative; display: flex; flex-direction: column; min-width: 0; min-height: 264px; padding: 32px 18px 0; border: 1px solid var(--catalog-border); border-radius: 16px; background: var(--catalog-bg); transition: border-color .2s, box-shadow .2s; }
.catalog-card:hover { border-color: #d9d9dd; box-shadow: 0 3px 14px #00000004; }
.card-main { width: 100%; text-align: center; }
.card-provider { display: flex; justify-content: center; height: 42px; color: var(--catalog-text); }
.card-provider :deep(svg) { width: 32px; height: 32px; }
.card-main h3 { overflow: hidden; margin: 10px 0 0; font-size: 16px; font-weight: 650; line-height: 24px; letter-spacing: -.2px; text-overflow: ellipsis; white-space: nowrap; }
.card-actions { position: absolute; display: flex; gap: 4px; top: 12px; right: 12px; opacity: 0; transition: opacity .15s; }
.catalog-card:hover .card-actions, .catalog-card:focus-within .card-actions { opacity: 1; }
.card-actions button { display: grid; place-items: center; width: 26px; height: 26px; border-radius: 6px; color: #85858e; background: var(--catalog-bg); }
.card-actions button:hover { background: var(--catalog-soft); color: var(--catalog-text); }
.card-price { margin: 9px auto 0; max-width: 100%; font-size: 12px; }
.card-price :deep(.plaza-pricing-table) { overflow: visible; }
.card-price :deep(table) { display: block; width: auto; min-width: 0; font-size: 12px; }
.card-price :deep(colgroup), .card-price :deep(thead), .card-price :deep(tbody tr:not(:first-child)), .card-price :deep(tbody td) { display: none; }
.card-price :deep(tbody) { display: block; }
.card-price :deep(tbody tr:first-child) { display: flex; justify-content: center; align-items: baseline; gap: 10px; border: 0; background: transparent; }
.card-price :deep(tbody tr:first-child td.pz-cell:nth-child(2)), .card-price :deep(tbody tr:first-child td.pz-cell:nth-child(3)) { display: block; padding: 0; border: 0; background: none; color: var(--catalog-text); font-family: inherit; font-size: 12px; font-weight: 500; white-space: nowrap; }
.card-price :deep(tbody tr:first-child td.pz-cell:nth-child(2))::before { content: var(--catalog-input-label); color: var(--catalog-muted); font-weight: 400; }
.card-price :deep(tbody tr:first-child td.pz-cell:nth-child(3))::before { content: var(--catalog-output-label); color: var(--catalog-muted); font-weight: 400; }
.card-price :deep(tbody tr:first-child td.pz-cell:nth-child(3))::after { content: ' / M'; color: var(--catalog-muted); font-weight: 400; font-size: 10px; }
.card-price :deep(tbody tr:first-child td.pz-cell > div:not(:first-child)), .card-price :deep(tbody tr:first-child td.pz-cell > div > span:first-child) { display: none; }
.card-price.media-price :deep(tbody tr:first-child td.pz-cell)::before { content: ''; }
.card-price.media-price :deep(tbody tr:first-child td.pz-cell > div > span:first-child) { display: inline-flex; }
.card-price.media-price :deep(tbody tr:first-child td.pz-cell > div) { justify-content: center; }
.card-description { display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; min-height: 34px; max-width: 90%; margin: 14px auto 16px; color: var(--catalog-muted); font-size: 11px; line-height: 17px; text-align: center; overflow-wrap: anywhere; }
.card-footer { display: flex; align-items: center; justify-content: space-between; gap: 8px; min-height: 40px; margin-top: auto; border-top: 1px solid var(--catalog-border); color: var(--catalog-muted); font-size: 10px; }
.card-badges { display: flex; gap: 5px; min-width: 0; flex-wrap: wrap; padding: 8px 0; }
.card-badges > span { padding: 2px 5px; border-radius: 4px; background: var(--catalog-soft); white-space: nowrap; }
.card-group { max-width: 45%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.catalog-tables { display: flex; flex-direction: column; gap: 18px; margin-top: 24px; }
.catalog-tables :deep(section), .catalog-detail-dialog :deep(section) { border-color: var(--catalog-border); box-shadow: none; filter: grayscale(1); }
.catalog-tables :deep(.plaza-pricing-table), .catalog-detail-dialog :deep(.plaza-pricing-table) { --plaza-accent: #52525b !important; }
.catalog-price-note { margin: 24px 0 0; text-align: center; font-size: 11px; line-height: 1.7; color: var(--catalog-muted); }
.catalog-state { display: flex; align-items: center; justify-content: center; flex-direction: column; gap: 16px; min-height: 280px; padding: 24px; color: var(--catalog-muted); text-align: center; font-size: 13px; }
.catalog-error { color: #b91c1c; }
.catalog-empty { margin-top: 24px; border: 1px dashed var(--catalog-border); border-radius: 16px; }
.catalog-spinner { width: 25px; height: 25px; border: 2px solid var(--catalog-border); border-top-color: var(--catalog-text); border-radius: 50%; animation: catalog-spin .8s linear infinite; }
@keyframes catalog-spin { to { transform: rotate(360deg); } }
.catalog-filter-dialog, .catalog-detail-dialog { --catalog-bg: #fff; --catalog-text: #18181b; --catalog-muted: #919195; --catalog-border: #eeeeef; --catalog-soft: #f7f7f8; }
.catalog-filter-dialog, .catalog-detail-dialog { padding: 0; color: var(--catalog-text); background: var(--catalog-bg); border: 1px solid var(--catalog-border); }
.catalog-filter-dialog::backdrop, .catalog-detail-dialog::backdrop { background: #0006; }
.catalog-filter-dialog { position: fixed; inset: 0 auto 0 0; width: min(384px, 100vw); max-width: 384px; height: 100dvh; max-height: 100dvh; margin: 0; border: 0; border-radius: 0; }
.catalog-filter-dialog[open], .catalog-detail-dialog[open] { display: flex; flex-direction: column; }
.catalog-dialog-header { display: flex; flex-shrink: 0; align-items: center; justify-content: space-between; gap: 16px; min-height: 64px; padding: 14px 22px; border-bottom: 1px solid var(--catalog-border); }
.catalog-dialog-header h2 { min-width: 0; margin: 0; font-size: 16px; font-weight: 600; overflow-wrap: anywhere; }
.catalog-dialog-body { min-height: 0; overflow-y: auto; padding: 26px 22px; }
.catalog-filter-dialog .catalog-dialog-body { flex: 1; }
.catalog-dialog-footer { flex-shrink: 0; padding: 16px 22px; border-top: 1px solid var(--catalog-border); }
:global(html.catalog-modal-open) { overflow: hidden; }
.catalog-filter-content section + section { margin-top: 32px; }
.catalog-filter-content h3 { margin: 0 0 14px; color: var(--catalog-text); font-size: 12px; font-weight: 600; }
.manufacturer-options { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; }
.manufacturer-options button { display: flex; align-items: center; gap: 8px; min-height: 40px; padding: 8px 10px; border: 1px solid var(--catalog-border); border-radius: 8px; color: #71717a; background: var(--catalog-bg); text-align: left; font-size: 12px; overflow-wrap: anywhere; }
.manufacturer-letter { display: inline-grid; place-items: center; width: 16px; height: 16px; border: 1px solid currentColor; border-radius: 50%; font: 600 11px/1 Arial, sans-serif; }
.catalog-filter-pills { display: flex; flex-wrap: wrap; gap: 8px; }
.catalog-filter-pills button { min-height: 32px; padding: 6px 10px; border: 1px solid var(--catalog-border); border-radius: 7px; color: #71717a; font-size: 11px; background: var(--catalog-bg); }
.catalog-filter-pills button span { margin-left: 3px; opacity: .7; }
.catalog-filter-content button.selected { color: var(--catalog-bg); border-color: var(--catalog-text); background: var(--catalog-text); font-weight: 500; }
.catalog-filter-content button:not(:disabled):hover { border-color: #b7b7bf; }
.filter-footer { display: flex; justify-content: space-between; gap: 10px; }
.catalog-button-dark { color: #fff; border-color: #202023; background: #202023; }
.catalog-button-dark:hover { border-color: #3f3f46; background: #3f3f46; }
.catalog-detail-dialog { position: fixed; inset: 0; width: min(1280px, calc(100vw - 32px)); max-width: none; max-height: 90dvh; margin: auto; border-radius: 16px; }
.catalog-detail-dialog .catalog-dialog-body { padding: 16px; }
.dark .model-catalog, .dark .catalog-filter-dialog, .dark .catalog-detail-dialog { --catalog-bg: #101011; --catalog-text: #f4f4f5; --catalog-muted: #8d8d96; --catalog-border: #29292c; --catalog-soft: #1b1b1e; }
.dark .catalog-toolbar { border-color: #252528; }
@media (min-width: 1200px) { .catalog-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); } }
@media (min-width: 1536px) { .catalog-toolbar { padding-right: max(16px, calc((100vw - 1408px) / 2)); padding-left: max(16px, calc((100vw - 1408px) / 2)); } }
@media (max-width: 640px) {
  .catalog-search { width: min(265px, calc(100vw - 112px)); }
  .catalog-search kbd { display: none; }
  .catalog-page { padding-right: 14px; padding-left: 14px; }
  .catalog-hero { padding: 45px 0 47px; }
  .catalog-hero h1 { font-size: 30px; }
  .catalog-hero p { font-size: 12px; }
  .catalog-summary { gap: 10px; padding: 12px; border-radius: 12px; }
  .summary-provider { gap: 8px; }
  .summary-provider h2 { font-size: 13px; }
  .summary-provider p { font-size: 10px; }
  .summary-actions { gap: 4px; }
  .summary-actions .catalog-icon-button { width: 25px; }
  .rate-toggle { padding: 0 6px; font-size: 11px; }
  .rate-toggle :deep(svg) { display: none; }
  .provider-initial { width: 32px; height: 32px; flex-basis: 32px; border-radius: 9px; }
  .catalog-view-switch button { width: 25px; }
  .catalog-grid { grid-template-columns: minmax(0, 1fr); margin-top: 16px; }
  .catalog-card { min-height: 264px; }
  .card-actions { opacity: 1; }
  .catalog-detail-dialog .catalog-dialog-body { padding: 8px; }
}
@media (prefers-reduced-motion: reduce) { .catalog-spinner { animation: none; } .catalog-card, .card-actions, .rate-switch > span { transition: none; } }
</style>
