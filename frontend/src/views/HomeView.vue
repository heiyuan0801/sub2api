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
          <RouterLink v-else :to="localizedPublicPath('/docs', locale)" :title="t('docs.title')" :aria-label="t('docs.title')" class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800">
            <Icon name="book" size="md" />
          </RouterLink>
          <router-link
            v-if="showModelPlazaEntry"
            :to="localizedPublicPath('/model-plaza', locale)"
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
    <PublicSiteHeader variant="landing" />

    <main>
      <section class="gateway-hero" aria-labelledby="hero-heading">
        <div class="gateway-hero-copy">
          <RouterLink
            v-if="showModelPlazaEntry && featuredModel"
            :to="modelLink(featuredModel)"
            class="gateway-announcement featured-model-announcement"
            :aria-label="t('publicHome.announcement.label', { model: featuredModel })"
          >
            {{ t('publicHome.announcement.newModel') }} · {{ featuredModel }}
            <span><ArrowRight :size="14" aria-hidden="true" /></span>
          </RouterLink>
          <a v-else href="#quick-start" class="gateway-announcement">
            {{ landing.guide }}
            <span><ArrowRight :size="14" aria-hidden="true" /></span>
          </a>

          <h1 id="hero-heading">
            <span>{{ siteName }} — {{ landing.title[0] }}</span>
            <span>{{ landing.title[1] }}</span>
            <span>{{ landing.title[2] }}</span>
          </h1>
          <p class="gateway-hero-lead">{{ heroSubtitle }}</p>
          <RouterLink to="/keys" class="gateway-key-button">
            {{ landing.getKey }}
            <ArrowRight :size="22" aria-hidden="true" />
          </RouterLink>

          <ul class="gateway-hero-benefits">
            <li v-for="benefit in benefits" :key="benefit">
              <Check :size="15" aria-hidden="true" />
              {{ t(benefit) }}
            </li>
          </ul>
        </div>

        <div class="gateway-shell gateway-hero-bottom">
          <div v-if="showModelPlazaEntry && marketData" class="gateway-stats">
            <dl>
              <div v-for="stat in catalogStats" :key="stat.label">
                <dt>{{ stat.label }}</dt>
                <dd>{{ stat.value.toLocaleString(locale) }}</dd>
              </div>
            </dl>
            <div class="gateway-stats-caption">
              <p>{{ landing.stats }}</p>
              <RouterLink :to="localizedPublicPath('/model-plaza', locale)">
                {{ t('publicSite.market.explore') }} <ArrowRight :size="14" aria-hidden="true" />
              </RouterLink>
            </div>
          </div>

          <GatewayDashboardDemo :start-path="startPath" :site-name="siteName" :site-logo="siteLogo" />
        </div>
      </section>

      <GatewayHomeFeatures
        :site-name="siteName"
        :providers="providers"
        :groups="marketGroups"
        :show-model-plaza="showModelPlazaEntry"
        :show-monitor="showMonitor"
      />
      <GatewayHomeDetails
        :site-name="siteName"
        :providers="providers"
        :model-count="catalogModels.length"
        :group-count="marketGroups.length"
        :show-models="showModelPlazaEntry"
        :show-monitor="showMonitor"
        :gateway-root="gatewayRoot"
        :start-path="startPath"
        :is-authenticated="isAuthenticated"
        :protocols="protocols"
        :request-examples="requestExamples"
      />
    </main>

    <PublicSiteFooter variant="landing" :providers="providers" />
  </div>
</template>

<script setup lang="ts">
import { ArrowRight, Check } from '@lucide/vue'

import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { localizedPublicPath } from '@/i18n/localeConfig'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import PublicSiteHeader from '@/components/public/PublicSiteHeader.vue'
import PublicSiteFooter from '@/components/public/PublicSiteFooter.vue'
import GatewayDashboardDemo from '@/components/public/home/GatewayDashboardDemo.vue'
import GatewayHomeFeatures from '@/components/public/home/GatewayHomeFeatures.vue'
import GatewayHomeDetails from '@/components/public/home/GatewayHomeDetails.vue'
import { landingCopy } from '@/components/public/home/landingCopy'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { getModelPlaza, type ModelPlazaResponse, type PlazaModel } from '@/api/modelPlaza'
import { BILLING_MODE_TOKEN } from '@/constants/channel'

const { t, locale } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const landing = computed(() => landingCopy(locale.value))
const benefits = [
  'publicHome.services.unifiedApi.title',
  'publicHome.services.keys.title',
  'publicHome.services.usage.title',
]

// Custom and compact home options retain their existing priority and behavior.
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || t('publicHome.siteSubtitle'))
const heroSubtitle = computed(() => {
  const subtitle = appStore.cachedPublicSettings?.site_subtitle?.trim()
  return subtitle && subtitle !== 'Subscription to API Conversion Platform' ? subtitle : landing.value.description
})
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
const catalogModels = computed(() => {
  const models = new Map<string, PlazaModel>()
  for (const group of marketGroups.value) {
    for (const model of group.models) models.set(`${model.platform}:${model.name}`, model)
  }
  return [...models.values()]
})
const providers = computed(() => {
  const counts = new Map<string, number>()
  for (const model of catalogModels.value) counts.set(model.platform, (counts.get(model.platform) ?? 0) + 1)
  return [...counts].map(([name, count]) => ({ name, count })).sort((a, b) => b.count - a.count)
})
const showMonitor = computed(() => isFeatureFlagEnabled(FeatureFlags.channelMonitor))
const catalogStats = computed(() => [
  { value: marketGroups.value.length, label: t('publicSite.market.groups') },
  { value: new Set(marketGroups.value.flatMap(group => group.models.map(model => model.pricing?.billing_mode || BILLING_MODE_TOKEN))).size, label: landing.value.billingModes },
  { value: catalogModels.value.length, label: t('publicSite.market.availableModels') },
  { value: providers.value.length, label: t('publicSite.market.providers') },
])
const modelLink = (search: string) => localizedPublicPath(`/model-plaza?${new URLSearchParams({ search })}`, locale.value)
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

const protocols = computed(() => [
  { id: 'chat', label: t('publicHome.examples.protocols.chat') },
  { id: 'responses', label: t('publicHome.examples.protocols.responses') },
  { id: 'claude', label: t('publicHome.examples.protocols.claude') },
  { id: 'gemini', label: t('publicHome.examples.protocols.gemini') }
] as const)
type Protocol = typeof protocols.value[number]['id']
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
@font-face {
  font-family: 'Gateway Inter';
  src: url('@/assets/fonts/inter-variable.woff2') format('woff2');
  font-style: normal;
  font-weight: 100 900;
  font-display: swap;
}
.public-home {
  --gateway-ink: #0a0a0a;
  --gateway-muted: #71717a;
  --gateway-accent: #5756e8;
  --gateway-line: #e4e4e7;
  --gateway-surface: #fff;
  --gateway-soft: #fafafa;
  --gateway-width: 1280px;
  --gateway-gutter: clamp(16px, 3vw, 40px);
  min-height: 100vh;
  background: var(--gateway-surface);
  color: var(--gateway-ink);
  font-family: 'Gateway Inter', Inter, system-ui, -apple-system, BlinkMacSystemFont, 'PingFang SC', 'Microsoft YaHei', sans-serif;
}
.dark .public-home {
  --gateway-ink: #fafafa;
  --gateway-muted: #a1a1aa;
  --gateway-accent: #8685ff;
  --gateway-line: #29292f;
  --gateway-surface: #09090b;
  --gateway-soft: #141417;
}
.public-home main { min-width: 0; }
.gateway-shell { width: min(calc(100% - 2 * var(--gateway-gutter)), var(--gateway-width)); margin-inline: auto; }
.gateway-hero {
  background: radial-gradient(ellipse 66% 405px at 0% 20px, color-mix(in srgb, var(--gateway-soft) 93%, var(--gateway-line)), transparent 95%);
  padding: 64px 0 0;
}
.gateway-hero-copy { padding-inline: var(--gateway-gutter); text-align: center; }
.gateway-announcement {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 13px;
  max-width: 100%;
  padding: 3px 4px 3px 15px;
  border: 1px solid var(--gateway-line);
  border-radius: 999px;
  background: color-mix(in srgb, var(--gateway-soft) 55%, var(--gateway-line));
  color: var(--gateway-ink);
  box-shadow: 0 2px 5px #09090912, inset 0 1px 1px #fff3;
  font-size: 13px;
  font-weight: 500;
  line-height: 1.4;
  overflow-wrap: anywhere;
}
.gateway-announcement span {
  display: grid;
  place-items: center;
  flex: none;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--gateway-surface);
}
.gateway-announcement:hover { border-color: var(--gateway-muted); }
.gateway-hero h1 {
  max-width: 896px;
  margin: 44px auto 0;
  color: var(--gateway-ink);
  font-size: clamp(35px, 4.6vw, 60px);
  font-weight: 700;
  letter-spacing: -.025em;
  line-height: 1;
  overflow-wrap: anywhere;
  text-wrap: balance;
}
.gateway-hero h1 span { display: block; }
.gateway-hero-lead {
  max-width: 620px;
  margin: 27px auto 0;
  color: var(--gateway-muted);
  font-size: 18px;
  line-height: 28px;
  overflow-wrap: anywhere;
  text-wrap: pretty;
}
.gateway-key-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 14px;
  min-height: 62px;
  margin-top: 39px;
  padding: 12px 48px;
  border-radius: 999px;
  background: #5554dc;
  color: #fff;
  box-shadow: 0 3px 8px #413ed72e, 0 10px 46px #5554dc58;
  font-size: 23px;
  font-weight: 650;
  line-height: 1.25;
  transition: background .18s ease, transform .18s ease;
}
.gateway-key-button:hover { transform: translateY(-1px); background: #4847ce; }
.gateway-hero-benefits {
  display: flex;
  justify-content: center;
  flex-wrap: wrap;
  gap: 8px 24px;
  margin-top: 24px;
  color: var(--gateway-muted);
  font-size: 14px;
}
.gateway-hero-benefits li { display: inline-flex; align-items: center; gap: 7px; }
.gateway-hero-benefits svg { color: #3b9254; }
.dark .gateway-hero-benefits svg { color: #7fcb94; }
.gateway-hero-bottom { margin-top: 78px; }
.gateway-stats { margin-bottom: 42px; }
.gateway-stats dl {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  overflow: hidden;
  border: 1px solid var(--gateway-line);
  border-radius: 16px;
  background: var(--gateway-surface);
}
.gateway-stats dl div {
  display: flex;
  flex-direction: column-reverse;
  justify-content: center;
  gap: 4px;
  min-width: 0;
  padding: 17px 20px;
}
.gateway-stats dl div + div { border-inline-start: 1px solid var(--gateway-line); }
.gateway-stats dt {
  color: var(--gateway-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 10px;
  letter-spacing: .16em;
  text-transform: uppercase;
}
.gateway-stats dd {
  font-size: 30px;
  font-weight: 700;
  letter-spacing: -.035em;
  line-height: 1.1;
  font-variant-numeric: tabular-nums;
}
.gateway-stats-caption {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px 16px;
  padding: 10px 3px 0;
  color: var(--gateway-muted);
  font-size: 13px;
}
.gateway-stats-caption a { display: inline-flex; align-items: center; gap: 6px; color: var(--gateway-ink); font-weight: 600; }
.gateway-stats-caption a:hover { text-decoration: underline; text-underline-offset: 4px; }
.gateway-hero :is(a, button):focus-visible { outline: 2px solid var(--gateway-accent); outline-offset: 4px; }
@media (max-width: 767px) {
  .gateway-hero { padding-top: 46px; }
  .gateway-hero h1 { margin-top: 32px; font-size: clamp(26px, 7vw, 42px); line-height: 1.13; }
  .gateway-hero-lead { margin-top: 20px; font-size: 16px; line-height: 25px; }
  .gateway-key-button { min-height: 52px; margin-top: 29px; padding-inline: 27px; font-size: 18px; }
  .gateway-hero-benefits { gap: 5px 12px; font-size: 12px; }
  .gateway-hero-bottom { margin-top: 44px; }
  .gateway-stats { margin-bottom: 26px; }
  .gateway-stats dl { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .gateway-stats dl div { padding: 13px 14px; }
  .gateway-stats dl div:nth-child(3) { border-inline-start: 0; }
  .gateway-stats dl div:nth-child(n+3) { border-top: 1px solid var(--gateway-line); }
  .gateway-stats dd { font-size: 24px; }
  .gateway-stats-caption { align-items: flex-start; font-size: 12px; }
}
@media (prefers-reduced-motion: reduce) {
  .gateway-key-button { transition: none; }
}
</style>
