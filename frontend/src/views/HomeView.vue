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
      <section class="hero" aria-labelledby="hero-heading">
        <div class="hero-grid" aria-hidden="true"></div>
        <div class="hero-inner">
          <router-link
            v-if="showModelPlazaEntry && featuredModel"
            :to="{ path: '/model-plaza', query: { search: featuredModel } }"
            class="model-announcement featured-model-announcement"
            :aria-label="t('publicHome.announcement.label', { model: featuredModel })"
            :title="t('publicHome.announcement.label', { model: featuredModel })"
          >
            <span class="announcement-label"><Icon name="sparkles" size="md" aria-hidden="true" />{{ t('publicHome.announcement.newModel') }}</span>
            <span class="announcement-model">{{ featuredModel }}</span>
            <Icon name="arrowRight" size="lg" class="announcement-arrow" aria-hidden="true" />
          </router-link>
          <div v-else class="model-announcement">
            <span class="announcement-label">{{ t('publicHome.announcement.unifiedApi') }}</span>
            <span>{{ t('publicHome.announcement.simplifyDevelopment') }}</span>
            <Icon name="sparkles" size="sm" aria-hidden="true" />
          </div>

          <h1 id="hero-heading" class="hero-title">
            <span>{{ t('publicHome.hero.title') }}</span>
            <span>{{ t('publicHome.hero.subtitle') }}</span>
          </h1>
          <p class="hero-description">{{ heroSubtitle }}</p>
          <div class="hero-actions">
            <router-link :to="startPath" class="public-button button-primary">
              {{ isAuthenticated ? t('publicHome.actions.dashboard') : t('publicHome.actions.getStarted') }}
              <Icon name="arrowRight" size="sm" aria-hidden="true" />
            </router-link>
            <a
              :href="docUrl || '#quick-start'"
              :target="docUrl ? '_blank' : undefined"
              :rel="docUrl ? 'noopener noreferrer' : undefined"
              class="public-button button-secondary"
            >{{ t('publicHome.actions.documentation') }}</a>
          </div>
        </div>
      </section>

      <section id="features" class="home-section services-section" aria-labelledby="services-heading">
        <div class="home-container">
          <div class="section-heading">
            <span class="section-eyebrow"><span class="status-dot"></span>{{ t('publicHome.services.eyebrow') }}</span>
            <h2 id="services-heading">{{ t('publicHome.services.title') }}</h2>
            <p>{{ t('publicHome.services.description') }}</p>
          </div>
          <div class="service-grid">
            <article v-for="feature in features" :key="feature.title" class="service-card">
              <div class="service-icon"><Icon :name="feature.icon" size="lg" aria-hidden="true" /></div>
              <h3>{{ feature.title }}</h3>
              <p>{{ feature.description }}</p>
            </article>
          </div>
        </div>
      </section>

      <section id="quick-start" class="home-section quickstart-section" aria-labelledby="quickstart-heading">
        <div class="home-container">
          <div class="section-heading">
            <span class="section-eyebrow"><Icon name="terminal" size="sm" aria-hidden="true" />{{ t('publicHome.quickstart.eyebrow') }}</span>
            <h2 id="quickstart-heading">{{ t('publicHome.quickstart.title') }}</h2>
            <p>{{ t('publicHome.quickstart.description') }}</p>
          </div>
          <div class="quickstart-layout">
            <div class="steps-list">
              <router-link :to="startPath" class="step-card">
                <span class="step-number">1</span>
                <div>
                  <h3>{{ accountStep.title }}</h3>
                  <p>{{ accountStep.description }}</p>
                </div>
                <Icon name="arrowRight" size="sm" class="step-arrow" aria-hidden="true" />
              </router-link>
              <router-link to="/keys" class="step-card">
                <span class="step-number">2</span>
                <div><h3>{{ t('publicHome.quickstart.createKey.title') }}</h3><p>{{ t('publicHome.quickstart.createKey.description') }}</p></div>
                <Icon name="arrowRight" size="sm" class="step-arrow" aria-hidden="true" />
              </router-link>
              <a
                :href="docUrl || '#api-example'"
                :target="docUrl ? '_blank' : undefined"
                :rel="docUrl ? 'noopener noreferrer' : undefined"
                class="step-card"
              >
                <span class="step-number">3</span>
                <div><h3>{{ t('publicHome.quickstart.connectApp.title') }}</h3><p>{{ t('publicHome.quickstart.connectApp.description') }}</p></div>
                <Icon name="arrowRight" size="sm" class="step-arrow" aria-hidden="true" />
              </a>
              <div class="endpoint-note">
                <Icon name="link" size="sm" aria-hidden="true" />
                <span>{{ t('publicHome.quickstart.apiEndpoint') }}</span>
                <code>{{ gatewayRoot }}</code>
                <button type="button" :title="t('publicHome.quickstart.copyEndpoint')" :aria-label="t('publicHome.quickstart.copyEndpoint')" @click="copyToClipboard(gatewayRoot)">
                  <Icon name="copy" size="sm" aria-hidden="true" />
                </button>
              </div>
            </div>

            <div id="api-example" class="code-panel">
              <div class="code-window-header">
                <div class="window-dots" aria-hidden="true"><span></span><span></span><span></span></div>
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
              <div id="protocol-code" role="tabpanel" :aria-labelledby="`protocol-tab-${activeProtocol}`">
                <div class="code-toolbar">
                  <span class="code-block-label">{{ t('publicHome.examples.request') }}</span>
                  <button type="button" class="code-copy" :title="copied ? t('publicHome.examples.requestCopied') : t('publicHome.examples.copyRequest')" :aria-label="copied ? t('publicHome.examples.requestCopied') : t('publicHome.examples.copyRequest')" @click="copyToClipboard(requestExamples[activeProtocol])">
                    <Icon :name="copied ? 'check' : 'copy'" size="sm" aria-hidden="true" />
                    <span>{{ copied ? t('publicHome.examples.copied') : t('publicHome.examples.copy') }}</span>
                  </button>
                </div>
                <pre class="example-code request-code" :aria-label="t('publicHome.examples.requestLabel')" tabindex="0"><code>{{ requestExamples[activeProtocol] }}</code></pre>
                <div class="code-toolbar response-toolbar"><span class="code-block-label">{{ t('publicHome.examples.response') }}</span></div>
                <pre class="example-code response-code" :aria-label="t('publicHome.examples.responseLabel')" tabindex="0"><code>{{ responseExamples[activeProtocol] }}</code></pre>
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

      <section class="home-section applications-section" aria-labelledby="applications-heading">
        <div class="home-container">
          <div class="section-heading">
            <span class="section-eyebrow"><Icon name="sparkles" size="sm" aria-hidden="true" />{{ t('publicHome.applications.eyebrow') }}</span>
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

      <section class="capabilities-section" :aria-label="t('publicHome.capabilities.label')">
        <div class="home-container capability-grid">
          <div v-for="capability in capabilities" :key="capability.id"><span>{{ capability.title }}</span><p>{{ capability.description }}</p></div>
        </div>
      </section>

    </main>

    <PublicSiteFooter />
  </div>
</template>

<script setup lang="ts">
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
import { getModelPlaza } from '@/api/modelPlaza'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const { copied, copyToClipboard } = useClipboard()

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
    const latestModel = data.latest_models?.[0]?.name
    featuredModel.value = typeof latestModel === 'string' ? latestModel.trim() : ''
  } catch {
    if (!controller.signal.aborted && featuredModelMounted && generation === featuredModelGeneration) {
      featuredModel.value = ''
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

const capabilities = computed(() => [
  { id: 'unifiedApi', title: t('publicHome.capabilities.unifiedApi.title'), description: t('publicHome.capabilities.unifiedApi.description') },
  { id: 'models', title: t('publicHome.capabilities.models.title'), description: t('publicHome.capabilities.models.description') },
  { id: 'usage', title: t('publicHome.capabilities.usage.title'), description: t('publicHome.capabilities.usage.description') },
  { id: 'control', title: t('publicHome.capabilities.control.title'), description: t('publicHome.capabilities.control.description') }
])

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
  --home-surface: #fafafa;
  --home-text: #171717;
  --home-muted: #737373;
  --home-border: #ededed;
  min-height: 100vh;
  background: var(--home-bg);
  color: var(--home-text);
  font-family: Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', sans-serif;
  -webkit-font-smoothing: antialiased;
}
.dark .public-home {
  --home-bg: #111214;
  --home-surface: #17181b;
  --home-text: #f5f5f5;
  --home-muted: #a3a3a3;
  --home-border: #2a2b2e;
}
.public-home main { min-width: 0; }
.public-home a, .public-home button { -webkit-tap-highlight-color: transparent; }
.public-home a:focus-visible, .public-home button:focus-visible, .example-code:focus-visible {
  outline: 2px solid var(--home-text);
  outline-offset: 4px;
}
.home-container { width: min(1152px, calc(100% - 64px)); margin: 0 auto; }
.hero { position: relative; isolation: isolate; min-height: 506px; text-align: center; background: var(--home-bg); }
.hero-grid {
  position: absolute;
  z-index: -1;
  inset: 0;
  pointer-events: none;
  background-image: linear-gradient(to right, rgba(120, 120, 120, .065) 1px, transparent 1px), linear-gradient(to bottom, rgba(120, 120, 120, .065) 1px, transparent 1px);
  background-size: 100px 100px;
  background-position: center top;
  -webkit-mask-image: radial-gradient(ellipse at 50% 30%, #000 12%, transparent 77%);
  mask-image: radial-gradient(ellipse at 50% 30%, #000 12%, transparent 77%);
}
.hero::after { content: ''; position: absolute; z-index: -1; inset: 0; pointer-events: none; background: radial-gradient(ellipse at 50% 28%, rgba(120, 120, 120, .045), transparent 65%); }
.hero-inner { width: min(920px, calc(100% - 40px)); margin: 0 auto; padding: 62px 0 72px; }
.model-announcement { display: inline-flex; align-items: center; justify-content: center; gap: 10px; min-height: 33px; max-width: 100%; padding: 4px 12px 4px 5px; border: 1px solid var(--home-border); border-radius: 999px; background: var(--home-bg); color: var(--home-muted); font-size: 12px; }
.announcement-label { flex-shrink: 0; border-radius: 999px; background: var(--home-text); color: var(--home-bg); padding: 3px 9px; font-size: 10px; font-weight: 500; }
.featured-model-announcement { gap: 17px; min-height: 66px; padding: 10px 30px 10px 16px; font-size: 28px; line-height: 1; text-decoration: none; transition: border-color .18s, background-color .18s; }
.featured-model-announcement:hover { border-color: color-mix(in srgb, var(--home-muted) 35%, var(--home-bg)); background: var(--home-surface); }
.featured-model-announcement .announcement-label { display: inline-flex; align-items: center; justify-content: center; gap: 8px; min-height: 40px; padding: 7px 15px; font-size: 22px; line-height: 1; white-space: nowrap; }
.featured-model-announcement .announcement-label svg { flex-shrink: 0; width: 24px; height: 24px; }
.announcement-model { min-width: 0; overflow: hidden; color: #595959; font-weight: 500; text-overflow: ellipsis; white-space: nowrap; }
.announcement-arrow { flex-shrink: 0; width: 30px; height: 30px; margin-left: 10px; color: #595959; }
.dark .announcement-model, .dark .announcement-arrow { color: #c5c5c5; }
.hero-title { margin: 24px 0 0; font-weight: 750; letter-spacing: -.045em; line-height: 1.3; }
.hero-title span { display: block; font-size: 56px; }
.hero-title span + span { margin-top: 5px; font-size: 44px; letter-spacing: -.035em; }
.hero-description { max-width: 720px; margin: 22px auto 0; color: var(--home-muted); font-size: 17px; line-height: 1.9; white-space: pre-line; overflow-wrap: anywhere; }
.hero-actions { display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 16px; margin-top: 30px; }
.public-button { display: inline-flex; align-items: center; justify-content: center; gap: 15px; min-height: 54px; border: 1px solid transparent; border-radius: 999px; padding: 0 29px; font-size: 14px; font-weight: 500; text-decoration: none; transition: background-color .18s, border-color .18s, box-shadow .18s; }
.button-primary { min-width: 158px; border-color: var(--home-text); background: var(--home-text); color: var(--home-bg); }
.button-primary:hover { opacity: .86; }
.button-secondary { min-width: 128px; border-color: var(--home-border); background: var(--home-bg); color: var(--home-text); box-shadow: 0 1px 2px rgba(0, 0, 0, .02); }
.button-secondary:hover { background: var(--home-surface); border-color: var(--home-muted); }
.home-section { padding: 80px 0; scroll-margin-top: 80px; }
.section-heading { text-align: center; margin-bottom: 44px; }
.section-eyebrow { display: inline-flex; align-items: center; gap: 7px; border: 1px solid var(--home-border); border-radius: 999px; padding: 5px 11px; color: var(--home-muted); background: var(--home-bg); font-size: 11px; line-height: 1.5; }
.status-dot { width: 6px; height: 6px; border-radius: 50%; background: #22c55e; box-shadow: 0 0 0 2px rgba(34, 197, 94, .09); }
.section-heading h2 { margin: 17px 0 0; color: var(--home-text); font-size: 36px; font-weight: 700; line-height: 1.4; letter-spacing: -.03em; }
.section-heading p { margin: 13px 0 0; color: var(--home-muted); font-size: 15px; line-height: 1.8; }
.service-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 22px; }
.service-card { min-width: 0; border: 1px solid var(--home-border); border-radius: 20px; padding: 30px 28px; background: var(--home-bg); transition: border-color .18s, box-shadow .18s; }
.service-card:hover { border-color: color-mix(in srgb, var(--home-muted) 40%, var(--home-bg)); box-shadow: 0 6px 24px rgba(0, 0, 0, .035); }
.service-icon { display: flex; align-items: center; justify-content: center; width: 46px; height: 46px; margin-bottom: 23px; border: 1px solid var(--home-border); border-radius: 50%; background: var(--home-bg); box-shadow: 0 3px 7px rgba(0, 0, 0, .035); }
.service-card h3, .application-card h3 { margin: 0; font-size: 17px; font-weight: 600; line-height: 1.5; }
.service-card p, .application-card p { margin: 10px 0 0; color: var(--home-muted); font-size: 13px; line-height: 1.9; }
.quickstart-section { border-top: 1px solid var(--home-border); background: var(--home-bg); }
.quickstart-layout { display: grid; grid-template-columns: .85fr 1.15fr; gap: 32px; align-items: start; }
.steps-list { display: flex; min-width: 0; flex-direction: column; gap: 12px; border: 1px solid var(--home-border); border-radius: 20px; padding: 16px; background: var(--home-bg); }
.step-card { display: flex; align-items: center; gap: 16px; min-height: 102px; border: 1px solid var(--home-border); border-radius: 12px; padding: 21px 18px; background: var(--home-bg); transition: border-color .18s; }
.step-card:hover { border-color: var(--home-muted); }
.step-number { display: flex; flex-shrink: 0; align-items: center; justify-content: center; width: 34px; height: 34px; border-radius: 50%; background: var(--home-text); color: var(--home-bg); font-size: 13px; font-weight: 600; }
.step-card > div { flex: 1; min-width: 0; }
.step-card h3 { margin: 0; font-size: 15px; font-weight: 600; line-height: 1.6; }
.step-card p { margin: 5px 0 0; color: var(--home-muted); font-size: 12px; line-height: 1.8; }
.step-arrow { flex-shrink: 0; color: var(--home-muted); }
.endpoint-note { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; padding: 3px 2px; color: var(--home-muted); font-size: 11px; }
.endpoint-note > svg { flex-shrink: 0; }
.endpoint-note code { min-width: 0; overflow-wrap: anywhere; color: var(--home-text); font-size: 11px; }
.endpoint-note button { display: flex; align-items: center; justify-content: center; width: 26px; height: 26px; border-radius: 6px; }
.endpoint-note button:hover { background: var(--home-border); }
.code-panel { min-width: 0; overflow: hidden; border: 1px solid var(--home-border); border-radius: 20px; background: var(--home-bg); box-shadow: 0 5px 20px rgba(0, 0, 0, .025); scroll-margin-top: 90px; }
.code-window-header { display: flex; align-items: center; justify-content: space-between; gap: 18px; min-height: 56px; padding: 12px 20px; border-bottom: 1px solid var(--home-border); }
.window-dots { display: flex; flex-shrink: 0; gap: 5px; }
.window-dots span { width: 6px; height: 6px; border-radius: 50%; background: #d4d4d4; }
.protocol-tabs { display: flex; align-items: center; gap: 5px; min-width: 0; overflow-x: auto; padding: 2px; }
.protocol-tabs button { flex-shrink: 0; min-height: 27px; border-radius: 999px; padding: 4px 11px; color: var(--home-muted); font-size: 10px; white-space: nowrap; }
.protocol-tabs button.active { background: var(--home-text); color: var(--home-bg); font-weight: 500; }
.code-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 10px; min-height: 37px; padding: 15px 22px 0; }
.code-block-label { color: var(--home-muted); font-size: 9px; font-weight: 500; letter-spacing: 1.5px; }
.response-toolbar { min-height: 29px; border-top: 1px solid var(--home-border); padding-top: 15px; }
.code-copy { display: flex; align-items: center; gap: 5px; border-radius: 5px; padding: 5px; color: var(--home-muted); font-size: 10px; }
.code-copy:hover { color: var(--home-text); background: var(--home-surface); }
.example-code { margin: 0; overflow: auto; padding: 12px 22px 20px; color: var(--home-text); font-size: 11px; line-height: 1.9; tab-size: 2; }
.request-code { min-height: 227px; max-height: 300px; }
.response-code { max-height: 170px; color: #15803d; }
.dark .response-code { color: #86efac; }
.example-code code { font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', monospace; }
.code-caption { display: flex; align-items: center; gap: 6px; border-top: 1px solid var(--home-border); padding: 12px 20px; color: var(--home-muted); font-size: 10px; line-height: 1.6; }
.code-caption code { min-width: 0; overflow-wrap: anywhere; font-size: 10px; }
.request-method { flex-shrink: 0; font-size: 9px; font-weight: 600; }
.response-status { display: flex; flex-shrink: 0; align-items: center; gap: 4px; margin-left: auto; color: #15803d; font-size: 9px; }
.response-status small { color: var(--home-muted); font-size: 9px; }
.dark .response-status { color: #86efac; }
.example-note { margin: 0; padding: 0 20px 12px; color: var(--home-muted); font-size: 10px; line-height: 1.6; }
.application-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 18px; }
.application-card { min-width: 0; border: 1px solid var(--home-border); border-radius: 17px; padding: 27px 22px; background: var(--home-bg); }
.application-card:first-child { grid-column: span 2; }
.application-icon { margin-bottom: 22px; color: var(--home-text); }
.application-card h3 { font-size: 15px; }
.application-card p { font-size: 12px; }
.capabilities-section { border-top: 1px solid var(--home-border); border-bottom: 1px solid var(--home-border); padding: 47px 0; background: var(--home-surface); }
.capability-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); text-align: center; }
.capability-grid > div + div { border-left: 1px solid var(--home-border); }
.capability-grid span { font-size: 24px; font-weight: 650; letter-spacing: -.03em; }
.capability-grid p { margin: 8px 0 0; color: var(--home-muted); font-size: 12px; }
@media (max-width: 1100px) {
  .home-container { width: min(calc(100% - 64px), 1000px); }
  .service-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .application-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .application-card:first-child { grid-column: auto; }
  .quickstart-layout { gap: 24px; }
  .step-card { padding: 20px 14px; gap: 12px; }
  .code-window-header { gap: 10px; padding-inline: 16px; }
  .protocol-tabs button { padding-inline: 9px; }
}
@media (max-width: 760px) {
  .home-container { width: calc(100% - 40px); }
  .hero { min-height: auto; }
  .hero-inner { padding: 55px 0 65px; }
  .featured-model-announcement { gap: 13px; min-height: 54px; padding: 8px 23px 8px 12px; font-size: 22px; }
  .featured-model-announcement .announcement-label { gap: 6px; min-height: 34px; padding: 6px 12px; font-size: 18px; }
  .featured-model-announcement .announcement-label svg { width: 22px; height: 22px; }
  .announcement-arrow { width: 26px; height: 26px; margin-left: 4px; }
  .hero-title span { font-size: 44px; }
  .hero-title span + span { font-size: 32px; }
  .hero-description { max-width: 530px; font-size: 15px; line-height: 1.9; }
  .home-section { padding: 60px 0; }
  .section-heading { margin-bottom: 32px; }
  .section-heading h2 { font-size: 29px; }
  .section-heading p { font-size: 13px; }
  .service-grid { gap: 16px; }
  .service-card { padding: 25px 22px; }
  .quickstart-layout { grid-template-columns: minmax(0, 1fr); gap: 24px; }
  .steps-list { gap: 12px; }
  .step-card { min-height: 94px; padding: 20px; }
  .application-grid { grid-template-columns: minmax(0, 1fr); gap: 15px; }
  .capability-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); row-gap: 29px; }
  .capability-grid > div:nth-child(3) { border-left: 0; }
  .capability-grid span { font-size: 22px; }
}
@media (max-width: 480px) {
  .hero-title span { font-size: 36px; }
  .hero-title span + span { margin-top: 10px; font-size: clamp(24px, 6.3vw, 30px); }
  .hero-description { font-size: 14px; }
  .model-announcement { gap: 8px; font-size: 11px; }
  .featured-model-announcement { gap: 11px; min-height: 46px; padding: 6px 16px 6px 9px; font-size: clamp(15px, 4.2vw, 18px); }
  .featured-model-announcement .announcement-label { gap: 5px; min-height: 32px; padding: 6px 10px; font-size: clamp(12px, 3.4vw, 14px); }
  .featured-model-announcement .announcement-label svg { width: 19px; height: 19px; }
  .announcement-arrow { width: 22px; height: 22px; margin-left: 2px; }
  .hero-actions { gap: 12px; }
  .public-button { min-height: 48px; padding-inline: 23px; }
  .button-primary { min-width: 144px; }
  .button-secondary { min-width: 118px; }
  .service-grid { grid-template-columns: minmax(0, 1fr); }
  .service-card { padding: 27px 25px; }
  .service-icon { margin-bottom: 18px; }
  .application-card { padding: 24px 18px; }
  .application-card p { font-size: 11px; }
  .capability-grid p { font-size: 11px; }
  .code-window-header { gap: 9px; padding-inline: 14px; }
  .protocol-tabs { gap: 2px; }
  .protocol-tabs button { padding-inline: 8px; }
  .code-toolbar { padding-inline: 14px; }
  .example-code { padding: 18px 14px; font-size: 10px; }
  .code-caption { padding-inline: 14px; }
}
@media (prefers-reduced-motion: reduce) {
  .public-home *, .public-home *::before, .public-home *::after { transition: none !important; scroll-behavior: auto !important; }
}
</style>
