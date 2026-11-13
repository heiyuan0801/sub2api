<template>
  <PlazaCatalog v-if="!embedded" :response="response" :loading="loading" :error="error" />
  <div v-else class="space-y-5">
    <!-- 全局价格说明(管理员配置,Markdown) -->
    <div
      v-if="descriptionHtml"
      class="plaza-description rounded-2xl border border-gray-100 bg-white px-5 py-4 text-sm shadow-card dark:border-dark-700/50 dark:bg-dark-800/50"
      v-html="descriptionHtml"
    ></div>

    <!-- 未登录提示 -->
    <p
      v-if="!isAuthenticated"
      class="flex items-center gap-1.5 text-xs text-gray-400 dark:text-dark-500"
    >
      <Icon name="infoCircle" size="xs" class="h-3.5 w-3.5" />
      {{ t('modelPlaza.anonymousHint') }}
    </p>

    <!-- 加载/错误/空 -->
    <div v-if="loading" class="flex min-h-[240px] items-center justify-center">
      <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-600/25 border-t-primary-600 dark:border-primary-400/25 dark:border-t-primary-400"></div>
    </div>
    <div
      v-else-if="error"
      class="rounded-2xl border border-red-200 bg-red-50 px-5 py-8 text-center text-sm text-red-600 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300"
    >
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

      <div v-if="providerSections.length" class="space-y-7">
        <section v-for="{ platform, groups } in providerSections" :key="platform" class="space-y-3">
          <h2 class="flex items-center gap-2 text-base font-semibold text-gray-900 dark:text-white">
            <PlatformIcon :platform="platform as GroupPlatform" size="sm" />
            {{ platformLabel(platform) }}
            <span class="text-xs font-normal text-gray-500 dark:text-dark-400">{{ t('publicSite.models.groupCount', { count: groups.length }) }}</span>
          </h2>
          <PlazaGroupSection v-for="group in groups" :key="group.id" :group="group" prices-only />
        </section>
      </div>
      <div
        v-else
        class="rounded-2xl border border-dashed border-gray-300 px-5 py-12 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
      >
        {{ searchActive ? t('modelPlaza.noSearchResult') : t('modelPlaza.empty') }}
      </div>
    </template>
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
import type { ModelPlazaGroup, ModelPlazaResponse } from '@/api/modelPlaza'
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

const searchActive = computed(() => searchQuery.value.trim() !== '')

const descriptionHtml = computed(() => {
  const md = props.response?.description?.trim()
  if (!md) return ''
  return DOMPurify.sanitize(marked.parse(md) as string)
})

const groups = computed(() => props.response?.groups ?? [])
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
.plaza-description {
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
