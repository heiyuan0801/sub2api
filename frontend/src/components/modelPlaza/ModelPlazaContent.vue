<template>
  <PlazaCatalog v-if="!embedded" :response="response" :loading="loading" :error="error" />
  <div v-else class="embedded-market">
    <div v-if="descriptionHtml" class="plaza-description" v-html="descriptionHtml" />

    <p v-if="!isAuthenticated" class="embedded-market-note">
      <Icon name="infoCircle" size="xs" />
      {{ t('modelPlaza.anonymousHint') }}
    </p>

    <div v-if="loading" class="market-state" role="status">
      <span class="market-spinner" aria-hidden="true" />
      {{ t('modelPlaza.loading') }}
    </div>
    <div v-else-if="error" class="market-state market-load-error" role="alert">
      <Icon name="exclamationCircle" size="md" />
      {{ t('modelPlaza.loadFailed') }}
    </div>
    <template v-else>
      <PlazaFilterBar
        :platforms="platforms"
        :groups="platformGroups"
        :platform="selectedPlatform"
        :group-id="selectedGroupId"
        :search="searchQuery"
        @update:platform="selectedPlatform = $event"
        @update:group-id="selectedGroupId = $event"
        @update:search="searchQuery = $event"
      />

      <div v-if="providerSections.length" class="market-provider-list">
        <section v-for="{ platform, groups } in providerSections" :key="platform" class="market-provider">
          <header class="provider-heading">
            <span class="provider-icon"><PlatformIcon :platform="platform as GroupPlatform" size="md" /></span>
            <h2>{{ platformLabel(platform) }}</h2>
            <span class="provider-count">{{ t('publicSite.models.groupCount', { count: groups.length }) }}</span>
          </header>
          <PlazaGroupSection
            v-for="group in groups"
            :key="group.id"
            :group="group"
            market
            prices-only
            model-details
            @select-model="selectedModel = $event"
          />
        </section>
      </div>
      <div v-else class="market-state" role="status">
        <Icon name="search" size="md" />
        {{ t(filtersActive ? 'modelPlaza.noSearchResult' : 'modelPlaza.empty') }}
        <button v-if="filtersActive" type="button" class="market-reset" @click="resetFilters">
          {{ t('publicSite.models.clearFilters') }}
        </button>
      </div>
    </template>
    <PlazaModelDetail v-if="selectedModel" :model="selectedModel" :groups="detailGroups" @close="selectedModel = null" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import PlazaFilterBar from './PlazaFilterBar.vue'
import PlazaGroupSection from './PlazaGroupSection.vue'
import PlazaCatalog from './PlazaCatalog.vue'
import PlazaModelDetail from './PlazaModelDetail.vue'
import { groupsForPlazaModel } from './catalogPricing'
import type { ModelPlazaGroup, ModelPlazaResponse, PlazaModel } from '@/api/modelPlaza'
import { useAuthStore } from '@/stores/auth'
import { platformLabel } from '@/utils/platformColors'
import type { GroupPlatform } from '@/types'

const props = defineProps<{
  response: ModelPlazaResponse | null
  loading: boolean
  error?: boolean
  /** 后台内嵌形态(AppLayout 内):隐藏页头。 */
  embedded?: boolean
}>()

const { t, locale } = useI18n()
const authStore = useAuthStore()
const isAuthenticated = computed(() => authStore.isAuthenticated)

const selectedPlatform = ref<string>('all')
const selectedGroupId = ref<number | 'all'>('all')
const searchQuery = ref('')
const selectedModel = ref<PlazaModel | null>(null)

const filtersActive = computed(() => Boolean(searchQuery.value.trim()) || selectedPlatform.value !== 'all' || selectedGroupId.value !== 'all')

function resetFilters() {
  searchQuery.value = ''
  selectedPlatform.value = 'all'
  selectedGroupId.value = 'all'
}

const descriptionHtml = computed(() => {
  const md = props.response?.description?.trim()
  if (!md) return ''
  return DOMPurify.sanitize(marked.parse(md) as string)
})

const groups = computed(() => props.response?.groups ?? [])
const detailGroups = computed(() => groupsForPlazaModel(groups.value, selectedModel.value))
watch(detailGroups, visible => {
  if (!visible.length) selectedModel.value = null
})
const platforms = computed(() =>
  [...new Set(groups.value.map(group => group.platform))].sort((a, b) =>
    platformLabel(a).localeCompare(platformLabel(b), locale.value)
  )
)
const platformGroups = computed(() =>
  groups.value.filter(group => selectedPlatform.value === 'all' || group.platform === selectedPlatform.value)
    .sort((a, b) => a.name.localeCompare(b.name, locale.value, { numeric: true }) || a.id - b.id)
)

watch(platforms, (list) => {
  if (selectedPlatform.value !== 'all' && !list.includes(selectedPlatform.value)) {
    selectedPlatform.value = 'all'
  }
})
watch(platformGroups, (list) => {
  if (selectedGroupId.value !== 'all' && !list.some(group => group.id === selectedGroupId.value)) {
    selectedGroupId.value = 'all'
  }
})

const providerSections = computed(() => {
  const byPlatform = new Map<string, ModelPlazaGroup[]>()
  const q = searchQuery.value.trim().toLowerCase()
  for (const group of platformGroups.value) {
    if (selectedGroupId.value !== 'all' && group.id !== selectedGroupId.value) continue
    const models = q ? group.models.filter(model => model.name.toLowerCase().includes(q)) : group.models
    if (!models.length) continue
    const section = byPlatform.get(group.platform)
    const matchingGroup = { ...group, models }
    if (section) section.push(matchingGroup)
    else byPlatform.set(group.platform, [matchingGroup])
  }
  return platforms.value.flatMap(platform => {
    const matchingGroups = byPlatform.get(platform)
    return matchingGroups ? [{ platform, groups: matchingGroups }] : []
  })
})
</script>

<style scoped>
.embedded-market {
  --market-surface: #fff;
  --market-ink: #171717;
  --market-muted: #676764;
  --market-line: #e9e9e7;
  --market-soft: #f6f6f4;
  display: grid;
  gap: 22px;
  min-width: 0;
  color: var(--market-ink);
}
.dark .embedded-market {
  --market-surface: #202020;
  --market-ink: #f5f5f5;
  --market-muted: #afafae;
  --market-line: #373737;
  --market-soft: #262626;
}
.embedded-market > * { min-width: 0; }
.embedded-market-note { display: flex; align-items: center; gap: 8px; color: var(--market-muted); font-size: .78rem; }
.market-provider-list { display: grid; gap: 30px; }
.market-provider { display: grid; min-width: 0; gap: 12px; }
.provider-heading { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; margin-bottom: 2px; }
.provider-heading h2 { font-size: 1.1rem; font-weight: 650; overflow-wrap: anywhere; }
.provider-icon { display: grid; width: 40px; height: 40px; place-items: center; border-radius: 13px; background: var(--market-soft); }
.provider-count { color: var(--market-muted); font-size: .76rem; }
.market-state { display: grid; gap: 13px; place-content: center; justify-items: center; min-height: 215px; padding: 24px; border-radius: 24px; background: var(--market-soft); color: var(--market-muted); font-size: .88rem; text-align: center; }
.market-load-error { color: #af2940; }
.dark .market-load-error { color: #ff99a7; }
.market-spinner { width: 24px; height: 24px; border: 2px solid var(--market-line); border-top-color: var(--market-ink); border-radius: 50%; animation: market-spin .75s linear infinite; }
.market-reset { padding: 10px 15px; border-radius: 999px; color: var(--market-surface); background: var(--market-ink); }
.market-reset:focus-visible { outline: 2px solid var(--market-ink); outline-offset: 3px; }
@keyframes market-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { .market-spinner { animation: none; } }

.plaza-description {
  padding: 18px 22px;
  border-radius: 24px;
  background: var(--market-soft);
  font-size: .86rem;
  line-height: 1.7;
  overflow-wrap: anywhere;
}

.plaza-description :deep(h1),
.plaza-description :deep(h2),
.plaza-description :deep(h3) {
  @apply mb-2 mt-3 font-semibold text-gray-900 first:mt-0 dark:text-white;
}

.plaza-description :deep(p) {
  @apply mb-2 text-gray-700 last:mb-0 dark:text-dark-200;
}

.plaza-description :deep(a) {
  @apply text-primary-600 underline underline-offset-4 hover:text-primary-700 dark:text-primary-300;
}

.plaza-description :deep(ul) {
  @apply mb-2 list-disc pl-5;
}

.plaza-description :deep(ol) {
  @apply mb-2 list-decimal pl-5;
}

.plaza-description :deep(li) {
  @apply mb-0.5 text-gray-700 dark:text-dark-200;
}

.plaza-description :deep(code) {
  @apply rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs dark:bg-dark-800;
}

.plaza-description :deep(blockquote) {
  @apply my-2 border-l-4 border-gray-300 pl-3 text-gray-600 dark:border-dark-600 dark:text-dark-300;
}
</style>
