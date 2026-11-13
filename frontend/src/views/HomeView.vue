<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="hasHomeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Compact Home Page -->
  <div
    v-else-if="compactHomeEnabled"
    data-testid="compact-home"
    class="flex min-h-screen flex-col bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white"
  >
    <header class="border-b border-gray-200 px-4 py-4 sm:px-6 dark:border-dark-800">
      <nav class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <img
            :src="siteLogo || '/logo.svg'"
            alt="Logo"
            class="h-9 w-9 shrink-0 rounded-lg object-contain"
          />
          <span class="min-w-0 truncate text-base font-semibold">{{ siteName }}</span>
        </div>
        <div class="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <RouterLink v-else to="/docs" :title="t('docs.title')" :aria-label="t('docs.title')" class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800">
            <Icon name="book" size="md" />
          </RouterLink>
          <router-link
            v-if="showModelPlazaEntry"
            to="/model-plaza"
            class="flex h-10 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-sm font-medium text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('nav.modelPlaza')"
          >
            <Icon name="grid" size="md" />
            <span class="hidden sm:inline">{{ t('nav.modelPlaza') }}</span>
          </router-link>
          <button
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="flex min-w-0 flex-1 items-center justify-center px-4 py-16 sm:px-6">
      <div class="min-w-0 max-w-2xl text-center">
        <img
          :src="siteLogo || '/logo.svg'"
          alt="Logo"
          class="mx-auto mb-6 h-20 w-20 rounded-2xl object-contain"
        />
        <h1 class="[overflow-wrap:anywhere] text-3xl font-bold md:text-4xl">{{ siteName }}</h1>
        <p class="mt-4 whitespace-pre-wrap [overflow-wrap:anywhere] text-base text-gray-600 dark:text-dark-300">{{ siteSubtitle }}</p>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="mt-8 inline-flex min-h-10 items-center justify-center rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-700"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.login') }}
        </router-link>
      </div>
    </main>

    <footer class="min-w-0 border-t border-gray-200 px-4 py-5 text-center text-sm text-gray-500 [overflow-wrap:anywhere] sm:px-6 dark:border-dark-800 dark:text-dark-400">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>

  <!-- Default public home; custom and compact home settings take precedence. -->
  <div v-else class="public-home">
    <PublicSiteHeader />

    <main>
      <section class="market-home-hero" aria-labelledby="hero-heading">
        <div class="market-home-shell">
          <div class="market-home-copy">
            <RouterLink v-if="showModelPlazaEntry && featuredModel" :to="modelLink(featuredModel)" class="featured-model-announcement market-new">
              <Icon name="sparkles" size="sm" /><span>{{ t('publicHome.announcement.newModel') }}</span><strong>{{ featuredModel }}</strong><Icon name="arrowRight" size="xs" />
            </RouterLink>
            <h1 id="hero-heading">{{ t('publicSite.market.homeTitle') }}</h1>
            <p class="market-lead">{{ heroSubtitle }}</p>
            <div class="market-hero-actions">
              <RouterLink v-if="showModelPlazaEntry" to="/model-plaza" class="market-primary">{{ t('publicSite.market.explore') }}<ArrowUpRight :size="16" /></RouterLink>
              <RouterLink :to="startPath" :class="showModelPlazaEntry ? 'market-secondary' : 'market-primary'">{{ isAuthenticated ? t('publicHome.actions.dashboard') : t('publicHome.actions.getStarted') }}</RouterLink>
              <a v-if="!showModelPlazaEntry" :href="docUrl || '/docs/quickstart'" class="market-secondary">{{ t('publicHome.actions.documentation') }}</a>
            </div>
            <dl v-if="marketGroups.length" class="market-totals">
              <div><dd>{{ catalogModels.length }}</dd><dt>{{ t('publicSite.market.availableModels') }}</dt></div>
              <div><dd>{{ providerCount }}</dd><dt>{{ t('publicSite.market.providers') }}</dt></div>
              <div><dd>{{ marketGroups.length }}</dd><dt>{{ t('publicSite.market.groups') }}</dt></div>
            </dl>
          </div>
          <img :src="marketHero" class="market-home-art" width="1320" height="880" alt="" fetchpriority="high" />
        </div>
      </section>

      <section v-if="featuredCards.length" class="market-home-section featured-section" aria-labelledby="featured-heading">
        <div class="home-container">
          <div class="market-section-title">
            <h2 id="featured-heading">{{ t('publicSite.market.featured') }}</h2>
            <p>{{ t('publicSite.market.featuredDescription') }} <RouterLink to="/model-plaza">{{ t('publicSite.market.explore') }} <ArrowUpRight :size="15" /></RouterLink></p>
          </div>
          <div class="featured-grid">
            <RouterLink v-for="{ model, group, groups, prices } in featuredCards" :key="model.platform + ':' + model.name" :to="modelLink(model.name)" class="featured-tile">
              <article>
                <h3><PlatformIcon :platform="model.platform as GroupPlatform" size="md" />{{ model.name }}</h3>
                <div class="featured-tags"><span>{{ platformLabel(model.platform) }}</span><span v-if="latestNames.includes(model.name)">{{ t('publicHome.announcement.newModel') }}</span></div>
                <p :title="group.description || undefined">{{ group.name }} · {{ t('publicSite.models.groupCount', { count: groups }) }}</p>
                <footer>
                  <dl v-if="prices.mode === BILLING_MODE_TOKEN && (prices.input !== null || prices.output !== null)" class="featured-prices">
                    <div v-if="prices.input !== null"><dt>{{ t('publicSite.market.input') }}</dt><dd>{{ formatCatalogPrice(prices.input, 1e6) }}<small>/M</small></dd></div>
                    <div v-if="prices.output !== null"><dt>{{ t('publicSite.market.output') }}</dt><dd>{{ formatCatalogPrice(prices.output, 1e6) }}<small>/M</small></dd></div>
                  </dl>
                  <dl v-else-if="prices.request !== null && prices.mode !== BILLING_MODE_TOKEN" class="featured-prices">
                    <div><dt>{{ t(`publicSite.models.${prices.mode === BILLING_MODE_IMAGE ? 'imageBilling' : prices.mode === BILLING_MODE_VIDEO ? 'videoBilling' : 'requestBilling'}`) }}</dt><dd>{{ formatCatalogPrice(prices.request) }}</dd></div>
                  </dl>
                  <span>{{ t('publicSite.market.details') }} <ArrowUpRight :size="15" /></span>
                </footer>
              </article>
            </RouterLink>
          </div>
        </div>
      </section>

      <section v-if="providers.length" class="market-home-section provider-section" aria-labelledby="provider-heading">
        <div class="home-container">
          <div class="market-section-title">
            <h2 id="provider-heading">{{ t('publicSite.market.overview') }}</h2>
            <p>{{ t('publicSite.market.overviewDescription') }}</p>
          </div>
          <ul class="provider-grid">
            <li v-for="provider in providers.slice(0, 8)" :key="provider.name">
              <RouterLink :to="modelLink(platformLabel(provider.name))" class="provider-tile">
                <span class="provider-symbol"><PlatformIcon :platform="provider.name as GroupPlatform" size="lg" /></span>
                <span class="provider-label"><strong>{{ platformLabel(provider.name) }}</strong><small>{{ t('publicSite.models.modelCount', { count: provider.count }) }}</small></span>
                <span class="provider-bar" aria-hidden="true"><span :style="{ width: `${100 * provider.count / catalogModels.length}%` }"></span></span>
                <ArrowUpRight :size="16" />
              </RouterLink>
            </li>
          </ul>
          <RouterLink to="/model-plaza" class="market-section-link">{{ t('publicSite.market.explore') }} <ArrowUpRight :size="16" /></RouterLink>
        </div>
      </section>

      <section id="features" class="market-home-section services-section" aria-labelledby="services-heading">
        <div class="home-container">
          <div class="market-section-title">
            <h2 id="services-heading">{{ t('publicHome.services.title') }}</h2>
            <p>{{ t('publicHome.services.description') }}</p>
          </div>
          <div class="service-grid">
            <Card v-for="feature in features" :key="feature.title" class="service-card">
              <div class="service-icon"><Icon :name="feature.icon" size="lg" aria-hidden="true" /></div>
              <h3>{{ feature.title }}</h3>
              <p>{{ feature.description }}</p>
            </Card>
          </div>
        </div>
      </section>

      <section id="quick-start" class="market-home-section quickstart-section" aria-labelledby="quickstart-heading">
        <div class="home-container">
          <div class="market-section-title">
            <h2 id="quickstart-heading">{{ t('publicHome.quickstart.title') }}</h2>
            <p>{{ t('publicHome.quickstart.description') }}</p>
          </div>
          <div class="quickstart-layout">
            <div class="steps-list">
              <router-link :to="startPath" class="step-card">
                <span class="step-icon"><Icon name="user" size="md" aria-hidden="true" /></span>
                <div>
                  <h3>{{ accountStep.title }}</h3>
                  <p>{{ accountStep.description }}</p>
                </div>
                <Icon name="arrowRight" size="sm" class="step-arrow" aria-hidden="true" />
              </router-link>
              <router-link to="/keys" class="step-card">
                <span class="step-icon"><Icon name="key" size="md" aria-hidden="true" /></span>
                <div><h3>{{ t('publicHome.quickstart.createKey.title') }}</h3><p>{{ t('publicHome.quickstart.createKey.description') }}</p></div>
                <Icon name="arrowRight" size="sm" class="step-arrow" aria-hidden="true" />
              </router-link>
              <a
                :href="docUrl || '/docs/api'"
                :target="docUrl ? '_blank' : undefined"
                :rel="docUrl ? 'noopener noreferrer' : undefined"
                class="step-card"
              >
                <span class="step-icon"><Icon name="terminal" size="md" aria-hidden="true" /></span>
                <div><h3>{{ t('publicHome.quickstart.connectApp.title') }}</h3><p>{{ t('publicHome.quickstart.connectApp.description') }}</p></div>
                <Icon name="arrowRight" size="sm" class="step-arrow" aria-hidden="true" />
              </a>
            </div>

            <div id="api-example" class="code-panel">
              <div class="code-window-header">
                <div class="endpoint-note">
                  <Icon name="link" size="sm" aria-hidden="true" />
                  <span>{{ t('publicHome.quickstart.apiEndpoint') }}</span>
                  <code>{{ gatewayRoot }}</code>
                  <button type="button" :title="t('publicHome.quickstart.copyEndpoint')" :aria-label="t('publicHome.quickstart.copyEndpoint')" @click="copyEndpoint(gatewayRoot)">
                    <Icon :name="endpointCopied ? 'check' : 'copy'" size="sm" aria-hidden="true" />
                  </button>
                </div>
                <div class="protocol-tabs" role="tablist" :aria-label="t('publicHome.examples.protocolsLabel')">
                  <button
                    v-for="protocol in protocols"
                    :id="`protocol-tab-${protocol.id}`"
                    :key="protocol.id"
                    type="button"
                    role="tab"
                    :aria-selected="activeProtocol === protocol.id"
                    aria-controls="protocol-code"
                    :tabindex="activeProtocol === protocol.id ? 0 : -1"
                    :class="{ active: activeProtocol === protocol.id }"
                    @click="activeProtocol = protocol.id"
                    @keydown="handleProtocolKeydown($event, protocol.id)"
                  >{{ protocol.label }}</button>
                </div>
              </div>
              <div id="protocol-code" class="code-examples" role="tabpanel" :aria-labelledby="`protocol-tab-${activeProtocol}`">
                <div class="code-pane">
                  <div class="code-toolbar">
                    <span class="code-block-label">{{ t('publicHome.examples.request') }}</span>
                    <button type="button" class="code-copy" :title="copied ? t('publicHome.examples.requestCopied') : t('publicHome.examples.copyRequest')" :aria-label="copied ? t('publicHome.examples.requestCopied') : t('publicHome.examples.copyRequest')" @click="copyToClipboard(requestExamples[activeProtocol])">
                      <Icon :name="copied ? 'check' : 'copy'" size="sm" aria-hidden="true" />
                      <span>{{ copied ? t('publicHome.examples.copied') : t('publicHome.examples.copy') }}</span>
                    </button>
                  </div>
                  <pre class="example-code" :aria-label="t('publicHome.examples.requestLabel')" tabindex="0"><code>{{ requestExamples[activeProtocol] }}</code></pre>
                </div>
                <div class="code-pane">
                  <div class="code-toolbar"><span class="code-block-label">{{ t('publicHome.examples.response') }}</span></div>
                  <pre class="example-code response-code" :aria-label="t('publicHome.examples.responseLabel')" tabindex="0"><code>{{ responseExamples[activeProtocol] }}</code></pre>
                </div>
              </div>
              <div class="code-caption">
                <span class="request-method">POST</span>
                <code>{{ endpointPaths[activeProtocol] }}</code>
                <span class="response-status">200 OK <small>{{ t('publicHome.examples.example') }}</small></span>
              </div>
              <p class="example-note">{{ t('publicHome.examples.note') }}</p>
            </div>
          </div>
        </div>
      </section>

      <section class="market-home-section applications-section" aria-labelledby="applications-heading">
        <div class="home-container">
          <div class="market-section-title">
            <h2 id="applications-heading">{{ t('publicHome.applications.title') }}</h2>
            <p>{{ t('publicHome.applications.description') }}</p>
          </div>
          <div class="application-grid">
            <article v-for="application in applications" :key="application.title" class="application-card">
              <div class="application-icon"><Icon :name="application.icon" size="lg" aria-hidden="true" /></div>
              <h3>{{ application.title }}</h3>
              <p>{{ application.description }}</p>
            </article>
          </div>
        </div>
      </section>

      <section class="market-home-section start-section" aria-labelledby="start-heading">
        <div class="home-container">
          <div class="start-panel">
            <div>
              <h2 id="start-heading">{{ t('publicHome.getStarted.title', { siteName }) }}</h2>
              <p>{{ t('publicHome.getStarted.description') }}</p>
            </div>
            <div class="start-actions">
              <RouterLink :to="startPath" class="market-primary">{{ isAuthenticated ? t('publicHome.actions.dashboard') : t('publicHome.actions.getStarted') }} <ArrowUpRight :size="16" /></RouterLink>
              <RouterLink to="/docs/quickstart" class="market-secondary">{{ t('publicSite.nav.quickStart') }}</RouterLink>
            </div>
          </div>
        </div>
      </section>

    </main>

    <PublicSiteFooter variant="home" />
  </div>
</template>

<script setup lang="ts">
import { Card } from '@/components/ui/card'
import { ArrowUpRight } from '@lucide/vue'

import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import PublicSiteHeader from '@/components/public/PublicSiteHeader.vue'
import PublicSiteFooter from '@/components/public/PublicSiteFooter.vue'
import { useClipboard } from '@/composables/useClipboard'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { getModelPlaza, type ModelPlazaResponse, type ModelPlazaGroup, type PlazaModel } from '@/api/modelPlaza'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { catalogPrices, formatCatalogPrice } from '@/components/modelPlaza/catalogPricing'
import { BILLING_MODE_IMAGE, BILLING_MODE_TOKEN, BILLING_MODE_VIDEO } from '@/constants/channel'
import { platformLabel } from '@/utils/platformColors'
import type { GroupPlatform } from '@/types'
import marketHero from '@/assets/market-hero.webp'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const { copied, copyToClipboard } = useClipboard()
const { copied: endpointCopied, copyToClipboard: copyEndpoint } = useClipboard()

// Custom and compact home options retain their existing priority and behavior.
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || t('publicHome.siteSubtitle'))
const heroSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || t('publicHome.hero.description'))
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const modelPlazaEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza))
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const isAuthenticated = computed(() => authStore.isAuthenticated)
const modelPlazaRequiresAuth = computed(() => appStore.cachedPublicSettings?.model_plaza_require_auth === true)
const showModelPlazaEntry = computed(() => modelPlazaEnabled.value && (isAuthenticated.value || !modelPlazaRequiresAuth.value))
const dashboardPath = computed(() => authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
const registrationEnabled = computed(() => appStore.cachedPublicSettings?.registration_enabled === true)
const startPath = computed(() => isAuthenticated.value ? dashboardPath.value : registrationEnabled.value ? '/register' : '/login')
const currentYear = computed(() => new Date().getFullYear())
const featuredModel = ref('')
const marketData = ref<ModelPlazaResponse | null>(null)
const marketGroups = computed(() => marketData.value?.groups ?? [])
const latestNames = computed(() => marketData.value?.latest_models?.map(model => model.name) ?? [])
const catalogModels = computed(() => {
  const models = new Map<string, { model: PlazaModel; group: ModelPlazaGroup; groups: number }>()
  for (const group of marketGroups.value) {
    for (const model of group.models) {
      const key = `${model.platform}:${model.name}`
      const entry = models.get(key)
      if (entry) entry.groups++
      else models.set(key, { model, group, groups: 1 })
    }
  }
  return [...models.values()]
})
const providers = computed(() => {
  const counts = new Map<string, number>()
  for (const { model } of catalogModels.value) counts.set(model.platform, (counts.get(model.platform) ?? 0) + 1)
  return [...counts].map(([name, count]) => ({ name, count })).sort((a, b) => b.count - a.count)
})
const providerCount = computed(() => providers.value.length)
const featuredCards = computed(() => {
  const preferred = latestNames.value.map(name => catalogModels.value.find(entry => entry.model.name === name)).filter(entry => entry !== undefined)
  const firstByProvider = providers.value.map(provider => catalogModels.value.find(entry => entry.model.platform === provider.name)).filter(entry => entry !== undefined)
  return [...new Set([...preferred, ...firstByProvider, ...catalogModels.value])].slice(0, 4)
    .map(entry => ({ ...entry, prices: catalogPrices(entry.model, entry.group) }))
})
const modelLink = (search: string) => ({ path: '/model-plaza', query: { search } })
const canLoadFeaturedModel = computed(() =>
  !hasHomeContent.value && !compactHomeEnabled.value && showModelPlazaEntry.value &&
  (!appStore.backendModeEnabled || authStore.isAdmin)
)
const featuredModelRefreshInterval = 60_000
let featuredModelMounted = false
let featuredModelRequest: AbortController | null = null
let featuredModelTimer: ReturnType<typeof setTimeout> | null = null
let featuredModelGeneration = 0
let refreshFeaturedModelAfterRequest = false

function clearFeaturedModelTimer() {
  if (featuredModelTimer !== null) {
    clearTimeout(featuredModelTimer)
    featuredModelTimer = null
  }
}

function resetFeaturedModel() {
  clearFeaturedModelTimer()
  featuredModel.value = ''
  marketData.value = null
  featuredModelGeneration++
  refreshFeaturedModelAfterRequest = false
  featuredModelRequest?.abort()
}

async function refreshFeaturedModel() {
  if (!featuredModelMounted || !canLoadFeaturedModel.value || document.visibilityState !== 'visible') return
  clearFeaturedModelTimer()
  // Wait for cancellation/completion before starting another request.
  if (featuredModelRequest) {
    refreshFeaturedModelAfterRequest = true
    return
  }

  const controller = new AbortController()
  const generation = featuredModelGeneration
  featuredModelRequest = controller
  try {
    const data = await getModelPlaza({ signal: controller.signal })
    if (controller.signal.aborted || !featuredModelMounted || generation !== featuredModelGeneration || !canLoadFeaturedModel.value) return
    marketData.value = data
    const latestModel = data.latest_models?.[0]?.name
    featuredModel.value = typeof latestModel === 'string' ? latestModel.trim() : ''
  } catch {
    if (!controller.signal.aborted && featuredModelMounted && generation === featuredModelGeneration) {
      featuredModel.value = ''
      marketData.value = null
    }
  } finally {
    if (featuredModelRequest === controller) featuredModelRequest = null
    if (featuredModelMounted && canLoadFeaturedModel.value && document.visibilityState === 'visible') {
      if (refreshFeaturedModelAfterRequest) {
        refreshFeaturedModelAfterRequest = false
        void refreshFeaturedModel()
      } else {
        featuredModelTimer = setTimeout(() => void refreshFeaturedModel(), featuredModelRefreshInterval)
      }
    }
  }
}

function handleFeaturedModelVisibility() {
  clearFeaturedModelTimer()
  if (document.visibilityState !== 'visible') {
    featuredModelGeneration++
    refreshFeaturedModelAfterRequest = false
    featuredModelRequest?.abort()
    return
  }
  void refreshFeaturedModel()
}

watch(
  [canLoadFeaturedModel, modelPlazaRequiresAuth, isAuthenticated, () => authStore.user?.id, () => authStore.token],
  () => {
    // Anonymous and signed-in catalogs may contain different private models.
    resetFeaturedModel()
    void refreshFeaturedModel()
  },
  { flush: 'sync' }
)

const accountStep = computed(() => {
  const step = isAuthenticated.value ? 'dashboard' : registrationEnabled.value ? 'register' : 'login'
  return {
    title: t(`publicHome.quickstart.${step}.title`),
    description: t(`publicHome.quickstart.${step}.description`)
  }
})

const features = computed(() => [
  { icon: 'globe', title: t('publicHome.services.unifiedApi.title'), description: t('publicHome.services.unifiedApi.description') },
  { icon: 'grid', title: t('publicHome.services.modelSelection.title'), description: t('publicHome.services.modelSelection.description') },
  { icon: 'chart', title: t('publicHome.services.usage.title'), description: t('publicHome.services.usage.description') },
  { icon: 'creditCard', title: t('publicHome.services.costs.title'), description: t('publicHome.services.costs.description') },
  { icon: 'shield', title: t('publicHome.services.keys.title'), description: t('publicHome.services.keys.description') },
  { icon: 'sparkles', title: t('publicHome.services.multimodal.title'), description: t('publicHome.services.multimodal.description') }
] as const)

const applications = computed(() => [
  { icon: 'chatBubble', title: t('publicHome.applications.chat.title'), description: t('publicHome.applications.chat.description') },
  { icon: 'terminal', title: t('publicHome.applications.coding.title'), description: t('publicHome.applications.coding.description') },
  { icon: 'document', title: t('publicHome.applications.writing.title'), description: t('publicHome.applications.writing.description') },
  { icon: 'sparkles', title: t('publicHome.applications.images.title'), description: t('publicHome.applications.images.description') },
  { icon: 'chartBar', title: t('publicHome.applications.analysis.title'), description: t('publicHome.applications.analysis.description') }
] as const)

const protocols = computed(() => [
  { id: 'chat', label: t('publicHome.examples.protocols.chat') },
  { id: 'responses', label: t('publicHome.examples.protocols.responses') },
  { id: 'claude', label: t('publicHome.examples.protocols.claude') },
  { id: 'gemini', label: t('publicHome.examples.protocols.gemini') }
] as const)
type Protocol = typeof protocols.value[number]['id']
const activeProtocol = ref<Protocol>('chat')
const endpointPaths: Record<Protocol, string> = {
  chat: '/v1/chat/completions',
  responses: '/v1/responses',
  claude: '/v1/messages',
  gemini: '/v1beta/models/MODEL_ID:generateContent'
}
const gatewayRoot = computed(() => {
  const configured = sanitizeUrl(appStore.cachedPublicSettings?.api_base_url || appStore.apiBaseUrl || '')
  return (configured || window.location.origin).replace(/\/+$/, '').replace(/\/v1$/, '')
})

const requestExamples = computed<Record<Protocol, string>>(() => ({
  chat: `curl ${gatewayRoot.value}/v1/chat/completions \\\n  -H "Authorization: Bearer YOUR_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "model": "MODEL_ID",\n    "messages": [\n      { "role": "user", "content": ${JSON.stringify(t('publicHome.examples.prompt'))} }\n    ]\n  }'`,
  responses: `curl ${gatewayRoot.value}/v1/responses \\\n  -H "Authorization: Bearer YOUR_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "model": "MODEL_ID",\n    "input": ${JSON.stringify(t('publicHome.examples.prompt'))}\n  }'`,
  claude: `curl ${gatewayRoot.value}/v1/messages \\\n  -H "x-api-key: YOUR_API_KEY" \\\n  -H "anthropic-version: 2023-06-01" \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "model": "MODEL_ID",\n    "max_tokens": 1024,\n    "messages": [\n      { "role": "user", "content": ${JSON.stringify(t('publicHome.examples.prompt'))} }\n    ]\n  }'`,
  gemini: `curl ${gatewayRoot.value}/v1beta/models/MODEL_ID:generateContent \\\n  -H "x-goog-api-key: YOUR_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "contents": [\n      { "parts": [{ "text": ${JSON.stringify(t('publicHome.examples.prompt'))} }] }\n    ]\n  }'`
}))

const responseExamples = computed<Record<Protocol, string>>(() => ({
  chat: JSON.stringify({ id: 'chatcmpl_example', object: 'chat.completion', model: 'MODEL_ID', choices: [{ index: 0, message: { role: 'assistant', content: t('publicHome.examples.reply') }, finish_reason: 'stop' }] }, null, 2),
  responses: JSON.stringify({ id: 'resp_example', object: 'response', model: 'MODEL_ID', output: [{ type: 'message', role: 'assistant', content: [{ type: 'output_text', text: t('publicHome.examples.reply') }] }] }, null, 2),
  claude: JSON.stringify({ id: 'msg_example', type: 'message', role: 'assistant', model: 'MODEL_ID', content: [{ type: 'text', text: t('publicHome.examples.reply') }], stop_reason: 'end_turn' }, null, 2),
  gemini: JSON.stringify({ candidates: [{ content: { role: 'model', parts: [{ text: t('publicHome.examples.reply') }] }, finishReason: 'STOP' }] }, null, 2)
}))

function handleProtocolKeydown(event: KeyboardEvent, protocol: Protocol) {
  const index = protocols.value.findIndex((entry) => entry.id === protocol)
  let nextIndex: number
  if (event.key === 'ArrowRight') nextIndex = (index + 1) % protocols.value.length
  else if (event.key === 'ArrowLeft') nextIndex = (index - 1 + protocols.value.length) % protocols.value.length
  else if (event.key === 'Home') nextIndex = 0
  else if (event.key === 'End') nextIndex = protocols.value.length - 1
  else return
  event.preventDefault()
  activeProtocol.value = protocols.value[nextIndex].id
  document.getElementById(`protocol-tab-${activeProtocol.value}`)?.focus()
}

// The compact variant keeps its existing theme control.
const isDark = ref(document.documentElement.classList.contains('dark'))
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}
function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark' || (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}
onMounted(() => {
  initTheme()
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) appStore.fetchPublicSettings()
  featuredModelMounted = true
  document.addEventListener('visibilitychange', handleFeaturedModelVisibility)
  void refreshFeaturedModel()
})
onBeforeUnmount(() => {
  featuredModelMounted = false
  resetFeaturedModel()
  document.removeEventListener('visibilitychange', handleFeaturedModelVisibility)
})
</script>

<style scoped>
.public-home {
  --home-bg: #fff;
  --home-text: #171717;
  --home-muted: #737373;
  --home-border: #e0e6f2;
  --lower-surface: #f5f7fd;
  --lower-muted: #546179;
  --lower-glow: #e1e7ff;
  --lower-accent: #4236cd;
  --lower-icon-bg: #eaf0ff;
  --lower-icon-border: #dce5ff;
  --lower-violet-bg: #f1ebff;
  --lower-violet-border: #e8ddff;
  --lower-violet-text: #6740b0;
  --lower-teal-bg: #e7f8f6;
  --lower-teal-border: #d5efed;
  --lower-teal-text: #176c60;
  min-height: 100vh;
  background: var(--home-bg);
  color: var(--home-text);
  font-family: Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', sans-serif;
  -webkit-font-smoothing: antialiased;
}
.dark .public-home {
  --home-bg: #111214;
  --home-text: #f5f5f5;
  --home-muted: #a3a3a3;
  --home-border: #303a52;
  --lower-surface: #1a2133;
  --lower-muted: #abb8d0;
  --lower-glow: #273860;
  --lower-accent: #c0c8ff;
  --lower-icon-bg: #262f51;
  --lower-icon-border: #37466d;
  --lower-violet-bg: #32274d;
  --lower-violet-border: #4b386a;
  --lower-violet-text: #dfccff;
  --lower-teal-bg: #203c40;
  --lower-teal-border: #2f5357;
  --lower-teal-text: #9be4d8;
}
.public-home main { min-width: 0; }
.public-home a, .public-home button { -webkit-tap-highlight-color: transparent; }
.public-home a:focus-visible, .public-home button:focus-visible, .example-code:focus-visible {
  outline: 2px solid var(--home-text);
  outline-offset: 4px;
}
/* Shared styles for the sections below the hero. */
.market-home-section { scroll-margin-top: 80px; }
.market-home-section .market-section-title { margin-bottom: 32px; }
.services-section, .applications-section { background: var(--home-bg); }
.service-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 16px; }
.service-card {
  min-width: 0;
  border: 1px solid var(--home-border);
  border-radius: 16px;
  padding: 24px;
  background: linear-gradient(145deg, var(--lower-surface), var(--home-bg) 56%);
  color: var(--home-text);
  box-shadow: 0 4px 12px rgb(22 42 80 / 3%);
}
.service-icon, .step-icon, .application-icon {
  display: grid;
  flex: none;
  place-items: center;
  width: 42px;
  height: 42px;
  border: 1px solid var(--lower-icon-border);
  border-radius: 11px;
  background: var(--lower-icon-bg);
  color: var(--lower-accent);
}
.service-card:nth-child(3n + 2) .service-icon, .step-card:nth-child(2) .step-icon, .application-card:nth-child(2n) .application-icon {
  border-color: var(--lower-violet-border);
  background: var(--lower-violet-bg);
  color: var(--lower-violet-text);
}
.service-card:nth-child(3n) .service-icon, .step-card:nth-child(3) .step-icon, .application-card:nth-child(3) .application-icon {
  border-color: var(--lower-teal-border);
  background: var(--lower-teal-bg);
  color: var(--lower-teal-text);
}
.service-card h3, .application-card h3 { margin-top: 16px; font-size: 17px; font-weight: 650; line-height: 1.5; }
.service-card p, .application-card p { margin-top: 7px; color: var(--lower-muted); font-size: 14px; line-height: 1.7; }
.quickstart-section {
  border-block: 1px solid var(--home-border);
  background: radial-gradient(ellipse at 100% 0%, var(--lower-glow), transparent 65%), var(--lower-surface);
}
.quickstart-layout { display: grid; gap: 18px; }
.steps-list { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 14px; }
.step-card {
  display: flex;
  align-items: center;
  gap: 13px;
  min-width: 0;
  min-height: 102px;
  padding: 16px;
  border: 1px solid var(--home-border);
  border-radius: 13px;
  background: var(--home-bg);
  transition: border-color .18s, box-shadow .18s;
}
.step-card:hover, .provider-tile:hover {
  border-color: var(--lower-accent);
  box-shadow: 0 4px 16px rgb(76 79 187 / 8%);
}
.step-card > div { flex: 1; min-width: 0; }
.step-card h3 { font-size: 15px; font-weight: 650; line-height: 1.5; }
.step-card p { margin-top: 5px; color: var(--lower-muted); font-size: 13px; line-height: 1.6; }
.step-arrow { flex: none; color: var(--lower-accent); }
.code-panel { min-width: 0; border: 1px solid var(--home-border); border-radius: 16px; background: var(--home-bg); box-shadow: 0 10px 30px rgb(22 42 80 / 5%); scroll-margin-top: 80px; }
.code-window-header { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 9px 16px; padding: 12px 18px; border-bottom: 1px solid var(--home-border); }
.endpoint-note { display: flex; align-items: center; flex-wrap: wrap; gap: 6px 9px; min-width: 0; color: var(--lower-muted); font-size: 12px; }
.endpoint-note > svg { flex: none; color: var(--lower-accent); }
.endpoint-note code { min-width: 0; overflow-wrap: anywhere; color: var(--home-text); }
.endpoint-note button, .code-copy { display: inline-flex; align-items: center; justify-content: center; gap: 5px; min-height: 30px; padding: 4px 7px; border-radius: 6px; color: var(--lower-accent); }
.endpoint-note button:hover, .code-copy:hover { background: var(--lower-icon-bg); }
.protocol-tabs { display: flex; align-items: center; gap: 3px; min-width: 0; max-width: 100%; overflow-x: auto; padding: 4px; border-radius: 9px; background: var(--lower-surface); }
.protocol-tabs button { flex: none; min-height: 30px; padding: 4px 12px; border-radius: 6px; color: var(--lower-muted); font-size: 12px; white-space: nowrap; }
.protocol-tabs button.active { background: var(--lower-icon-bg); color: var(--lower-accent); font-weight: 650; }
.protocol-tabs button:focus-visible, .example-code:focus-visible { outline-offset: -2px; }
.code-examples { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); }
.code-pane { min-width: 0; }
.code-pane + .code-pane { border-left: 1px solid var(--home-border); }
.code-toolbar { display: flex; justify-content: space-between; align-items: center; gap: 10px; min-height: 44px; padding: 5px 18px; background: var(--lower-surface); color: var(--lower-muted); font-size: 12px; }
.code-block-label { font-weight: 650; }
.example-code { min-height: 205px; max-height: 230px; margin: 0; overflow: auto; padding: 16px 18px; color: var(--home-text); font-size: 12px; line-height: 1.65; tab-size: 2; }
.example-code code { font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', monospace; }
.response-code, .response-status { color: var(--lower-teal-text); }
.code-caption { display: flex; flex-wrap: wrap; align-items: center; gap: 7px; padding: 10px 18px 5px; border-top: 1px solid var(--home-border); color: var(--lower-muted); font-size: 12px; }
.code-caption code { min-width: 0; overflow-wrap: anywhere; }
.request-method { padding: 2px 6px; border-radius: 4px; background: var(--lower-icon-bg); color: var(--lower-accent); font-size: 11px; font-weight: 700; }
.response-status { display: flex; flex-shrink: 0; align-items: center; gap: 5px; margin-left: auto; font-size: 11px; }
.response-status small { color: var(--lower-muted); font-size: 11px; }
.example-note { padding: 1px 18px 11px; color: var(--lower-muted); font-size: 11px; line-height: 1.5; }
.application-grid { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 14px; }
.application-card { min-width: 0; padding: 18px; border: 1px solid var(--home-border); border-radius: 13px; background: var(--lower-surface); }
.application-card h3 { font-size: 15px; }
.application-card p { font-size: 13px; }
.market-home-section.start-section { padding-top: 12px; }
.start-panel {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 24px;
  padding: clamp(24px, 4vw, 52px);
  border: 1px solid var(--lower-icon-border);
  border-radius: 20px;
  background: radial-gradient(ellipse at 85% 15%, var(--lower-glow), transparent 70%), var(--lower-surface);
}
.start-panel h2 { overflow-wrap: anywhere; font-size: clamp(25px, 3vw, 38px); font-weight: 730; line-height: 1.2; letter-spacing: -.035em; }
.start-panel p { margin-top: 10px; color: var(--lower-muted); font-size: 14px; line-height: 1.7; }
.start-actions { display: flex; flex-wrap: wrap; gap: 10px; }
@media (max-width: 1000px) {
  .application-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
}
@media (max-width: 800px) {
  .service-grid, .application-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .steps-list { grid-template-columns: minmax(0, 1fr); }
  .step-card { min-height: 80px; }
  .start-panel { flex-direction: column; align-items: flex-start; }
}
@media (max-width: 540px) {
  .service-grid, .code-examples { grid-template-columns: minmax(0, 1fr); }
  .service-card, .application-card { padding: 16px; }
  .code-pane + .code-pane { border-left: 0; border-top: 1px solid var(--home-border); }
  .code-window-header, .code-toolbar, .example-code, .code-caption, .example-note { padding-inline: 12px; }
  .protocol-tabs button { padding-inline: 8px; }
}
@media (prefers-reduced-motion: reduce) {
  .public-home *, .public-home *::before, .public-home *::after { transition: none !important; scroll-behavior: auto !important; }
}
.market-home-hero { position:relative; min-height:610px; display:flex; padding:0; border:0; text-align:left; background:radial-gradient(ellipse at 81% 82%,#e3e7ff 0,transparent 59%),linear-gradient(180deg,#f3f8fc,#edf1fc); overflow:hidden; }
.market-home-shell { display:grid; grid-template-columns:minmax(0,1fr) minmax(0,1.16fr); align-items:center; gap:12px; width:min(100%,1536px); padding:55px 58px 60px; margin:auto; }
.market-home-copy { position:relative; z-index:1; color:#141923; }
.market-home-copy h1 { margin:0; max-width:650px; font-size:clamp(38px,3.65vw,54px); font-weight:760; line-height:1.18; letter-spacing:-.04em; }
.market-lead { max-width:600px; margin-top:20px; color:#566379; font-size:15px; line-height:1.9; overflow-wrap:anywhere; }
.market-home-art { width:112%; max-width:none; height:auto; margin-left:-5%; object-fit:contain; mask-image:radial-gradient(ellipse 72% 74% at center,#000 45%,transparent 84%); }
.market-new { display:inline-flex; flex-wrap:wrap; gap:7px; align-items:center; max-width:100%; padding:6px 10px; border:1px solid #cdd8fe; border-radius:100px; margin-bottom:19px; font-size:12px; color:#423bd1; background:#eff2ff; overflow-wrap:anywhere; }
.market-new:hover { background:#e1e6ff; }
.market-hero-actions { display:flex; flex-wrap:wrap; gap:12px; margin-top:28px; }
.market-primary,.market-secondary { display:inline-flex; align-items:center; justify-content:center; gap:7px; min-height:46px; padding:0 20px; border-radius:8px; font-size:15px; font-weight:650; }
.market-primary { color:white; background:linear-gradient(95deg,#2e50fb,#805ffa); }
.market-primary:hover { filter:brightness(1.08); }
.market-secondary { color:#4139e0; border:1px solid #6659fc; background:#ffffff4d; }
.market-secondary:hover { background:#fff; }
.market-totals { display:flex; flex-wrap:wrap; gap:20px 36px; margin-top:44px; }
.market-totals dd { font-size:25px; font-weight:750; line-height:1.3; }
.market-totals dt { margin-top:4px; font-size:13px; color:#566379; }
.market-home-section { padding:54px 0 58px; }
.market-home-section .home-container { width:100%; max-width:1536px; padding:0 58px; margin-inline:auto; }
.featured-section { background:linear-gradient(180deg,var(--home-bg),#f4f7fc); }
.provider-section { background:#f4f7fc; }
.market-section-title { text-align:center; margin-bottom:34px; }
.market-section-title h2 { font-size:clamp(28px,3.3vw,40px); font-weight:730; letter-spacing:-.035em; }
.market-section-title p { margin-top:10px; color:var(--home-muted); font-size:14px; line-height:1.7; }
.market-section-title a { display:inline-flex; align-items:center; gap:4px; margin-left:7px; color:var(--home-text); font-weight:650; }
.featured-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:20px; }
.featured-tile { min-width:0; padding:18px; border-radius:12px; background:radial-gradient(ellipse at 85% 14%,#d8e4ff,transparent 70%),linear-gradient(130deg,#92dbff,#4361fc 68%,#dbdfff); }
.featured-tile:nth-child(2) { background:radial-gradient(ellipse at 75% 15%,#f4d8ff,transparent 70%),linear-gradient(145deg,#d9c5fe,#7844fa 70%,#f9bcfc); }
.featured-tile:nth-child(3) { background:linear-gradient(140deg,#a480dc,#ffd0e8 46%,#b19ded); }
.featured-tile:nth-child(4) { background:linear-gradient(140deg,#a9dcff,#82e4ef 46%,#b6f4cd); }
.featured-tile article { height:100%; padding:20px; border-radius:10px; background:var(--home-bg); color:var(--home-text); transition:transform .18s; }
.featured-tile:hover article { transform:translateY(-2px); }
.featured-tile h3 { display:flex; align-items:center; gap:9px; font-size:20px; font-weight:680; overflow-wrap:anywhere; }
.featured-tags { display:flex; flex-wrap:wrap; gap:6px; margin:9px 0; }
.featured-tags span { border-radius:4px; padding:3px 7px; color:#4439d4; background:#edf0ff; font-size:12px; }
.featured-tile p { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; color:var(--home-muted); font-size:14px; }
.featured-tile footer { display:flex; align-items:center; justify-content:space-between; flex-wrap:wrap; gap:10px; border-top:1px solid var(--home-border); margin-top:14px; padding-top:13px; color:var(--home-muted); font-size:13px; }
.featured-tile footer > span:last-child { display:flex; align-items:center; gap:4px; margin-left:auto; color:var(--lower-accent); }
.featured-prices { display: flex; flex-wrap: wrap; gap: 8px 22px; }
.featured-prices dt { color: var(--lower-muted); font-size: 12px; }
.featured-prices dd { margin-top: 2px; color: var(--home-text); font-size: 16px; font-weight: 680; }
.featured-prices small { margin-left: 2px; color: var(--lower-muted); font-size: 11px; font-weight: 400; }
.provider-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px 18px; }
.provider-tile {
  display: grid;
  grid-template-columns: 38px minmax(85px, 1fr) minmax(34px, 1.1fr) auto;
  gap: 12px;
  align-items: center;
  min-width: 0;
  height: 100%;
  padding: 14px;
  border: 1px solid var(--home-border);
  border-radius: 12px;
  background: var(--home-bg);
  transition: border-color .18s, box-shadow .18s;
}
.provider-symbol { display: grid; place-items: center; width: 38px; height: 38px; border-radius: 9px; background: var(--lower-surface); }
.provider-label { min-width: 0; }
.provider-tile strong, .provider-tile small { display: block; overflow-wrap: anywhere; }
.provider-tile small { color: var(--lower-muted); font-size: 12px; margin-top: 3px; }
.provider-tile > svg:last-child { color: var(--lower-accent); }
.provider-bar { height: 5px; border-radius: 999px; background: var(--lower-icon-bg); }
.provider-bar span { display: block; height: 100%; border-radius: inherit; background: linear-gradient(90deg,#4a6ff6,#9271fa); }
.market-section-link { display: flex; align-items: center; gap: 5px; width: fit-content; margin: 20px auto 0; color: var(--lower-accent); font-size: 14px; font-weight: 650; }
.market-section-link:hover { text-decoration: underline; text-underline-offset: 4px; }
.dark .market-home-hero { background:radial-gradient(ellipse at 80% 63%,#283760,transparent 55%),linear-gradient(180deg,#171e31,#151923); }
.dark .market-home-copy { color:#f1f4ff; }
.dark .market-lead,.dark .market-totals dt { color:#abb8d0; }
.dark .market-home-art { opacity:.82; }
.dark .market-secondary { background:#222a49; color:#c6cdff; border-color:#6c72c8; }
.dark .featured-section,.dark .provider-section { background:#151923; }
@media(max-width:1050px) { .market-home-shell,.market-home-section .home-container { padding-inline:32px; } .market-home-hero { min-height:515px; } .market-totals { gap:18px 22px; } }
@media(max-width:760px) { .market-home-shell { grid-template-columns:minmax(0,1fr); gap:0; padding:37px 22px 10px; } .market-home-art { width:min(100%,520px); margin:-8px auto 0; } .market-home-copy h1 { font-size:clamp(32px,8vw,48px); } .market-totals { margin-top:25px; } .market-home-section .home-container { padding-inline:22px; } .market-home-section { padding:38px 0; } .provider-grid { grid-template-columns:minmax(0,1fr); } .market-home-section.start-section { padding-top:8px; } }
@media(max-width:540px) { .featured-grid { grid-template-columns:minmax(0,1fr); } .featured-tile { padding:12px; } .featured-tile article { padding:16px; } .market-hero-actions { margin-top:20px; } .market-home-art { margin-top:0; } }
</style>
