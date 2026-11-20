<template>
  <header class="public-header" :class="{ 'landing-header': isLanding }" @keydown.esc="closeMobileNavigation">
    <div class="public-header-inner">
      <RouterLink :to="localizedPublicPath('/home', locale)" class="public-brand" :aria-label="t('publicSite.nav.homeLabel', { siteName })">
        <img :src="siteLogo || '/logo.svg'" alt="" />
        <span>{{ siteName }}</span>
      </RouterLink>

      <form v-if="isLanding && showModels" class="public-search landing-search" role="search" @submit.prevent="searchModels">
        <Icon name="search" size="sm" aria-hidden="true" />
        <input ref="landingSearch" v-model="marketQuery" type="search" :aria-label="t('publicSite.models.searchLabel')" :placeholder="t('publicSite.models.searchLabel')" />
        <kbd aria-hidden="true">{{ shortcut }}</kbd>
      </form>

      <div class="public-navigation-and-tools" :class="{ 'is-open': mobileOpen }">
        <nav v-if="isLanding" id="public-navigation" class="public-navigation" :class="{ 'is-open': mobileOpen }" :aria-label="t('publicSite.nav.mainNavigation')">
          <RouterLink v-if="showModels" :to="localizedPublicPath('/model-plaza', locale)" @click="mobileOpen = false">{{ t('publicSite.market.heading') }}</RouterLink>
          <DropdownMenuRoot v-for="group in landingMenus" :key="group.label" :modal="false">
            <DropdownMenuTrigger class="landing-menu-trigger">{{ group.label }} <Icon name="chevronDown" size="xs" aria-hidden="true" /></DropdownMenuTrigger>
            <DropdownMenuContent class="landing-menu-items" :side-offset="10" :collision-padding="12" align="start">
              <DropdownMenuItem v-for="item in group.items" :key="item.label" as-child>
                <a v-if="typeof item.to === 'string' && item.to.startsWith('#')" :href="item.to" @click="mobileOpen = false">{{ item.label }}</a>
                <RouterLink v-else :to="item.to" @click="mobileOpen = false">{{ item.label }}</RouterLink>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenuRoot>
          <a href="#quick-start" @click="mobileOpen = false">API</a>
          <RouterLink :to="localizedPublicPath('/docs', locale)" @click="mobileOpen = false">{{ t('docs.title') }}</RouterLink>
          <RouterLink v-if="showModels" :to="localizedPublicPath('/model-plaza', locale)" @click="mobileOpen = false">{{ copy.pricing }}</RouterLink>
        </nav>
        <nav v-else id="public-navigation" class="public-navigation" :class="{ 'is-open': mobileOpen }" :aria-label="t('publicSite.nav.mainNavigation')">
          <RouterLink :to="localizedPublicPath('/home', locale)">{{ t('publicSite.market.home') }}</RouterLink>
          <RouterLink v-if="showModels" :to="localizedPublicPath('/model-plaza', locale)" @click="mobileOpen = false">{{ t('publicSite.market.heading') }}</RouterLink>
          <RouterLink v-if="showMonitor" to="/monitor" @click="mobileOpen = false">{{ t('nav.channelStatus') }}</RouterLink>
          <RouterLink :to="localizedPublicPath('/home#quick-start', locale)" @click="mobileOpen = false">{{ t('publicSite.nav.quickStart') }}</RouterLink>
          <RouterLink :to="localizedPublicPath('/docs', locale)" @click="mobileOpen = false">{{ t('docs.title') }}</RouterLink>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" @click="mobileOpen = false">{{ t('publicSite.nav.helpCenter') }} <Icon name="externalLink" size="xs" /></a>
        </nav>

        <div class="public-header-actions">
          <form v-if="showModels && !isLanding" class="public-search" role="search" @submit.prevent="searchModels">
            <Icon name="search" size="sm" />
            <input v-model="marketQuery" type="search" :aria-label="t('publicSite.models.searchLabel')" :placeholder="t('publicSite.models.searchLabel')" />
          </form>
          <AnnouncementBell v-if="authStore.isAuthenticated" class="public-notifications" />
          <LocaleSwitcher compact class="public-language" />
          <div v-if="isLanding" class="landing-themes" role="group" :aria-label="copy.systemTheme">
            <button v-for="choice in themeChoices" :key="choice.mode" type="button" :aria-label="choice.label" :title="choice.label" :aria-pressed="themeMode === choice.mode" @click="applyTheme(choice.mode)">
              <component :is="choice.icon" :size="15" aria-hidden="true" />
            </button>
          </div>
          <button v-else type="button" class="public-icon-button" :aria-label="isDark ? t('publicSite.nav.lightMode') : t('publicSite.nav.darkMode')" @click="applyTheme(isDark ? 'light' : 'dark')">
            <Icon :name="isDark ? 'moon' : 'sun'" size="sm" />
          </button>
          <ThemeConfigDrawer v-if="themeStore" :show-layout-options="false" />
          <Button v-if="authStore.isAuthenticated" as-child variant="default" size="sm" class="public-header-primary"><RouterLink :to="dashboardTarget">{{ t('publicSite.nav.enterConsole') }}</RouterLink></Button>
          <template v-else>
            <Button as-child variant="ghost" size="sm" class="public-header-login"><RouterLink :to="loginTarget">{{ t('publicSite.nav.login') }}</RouterLink></Button>
            <Button v-if="registrationEnabled" as-child variant="default" size="sm" class="public-header-primary"><RouterLink to="/register">{{ t('publicSite.nav.register') }}</RouterLink></Button>
            <Button v-else-if="isLanding" as-child variant="default" size="sm" class="public-header-primary"><RouterLink to="/keys">{{ t('publicHome.actions.getStarted') }}</RouterLink></Button>
          </template>
        </div>
      </div>
      <WebsiteStatusLink v-if="isLanding" live class="landing-website-status" />
      <button ref="menuButton" type="button" class="public-icon-button public-menu-toggle" :aria-expanded="mobileOpen" aria-controls="public-navigation" :aria-label="mobileOpen ? t('publicSite.nav.closeNavigation') : t('publicSite.nav.openNavigation')" @click="mobileOpen = !mobileOpen">
        <Icon :name="mobileOpen ? 'x' : 'menu'" size="sm" />
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { getActivePinia } from 'pinia'
import { Monitor, Moon, Sun } from '@lucide/vue'
import { DropdownMenuContent, DropdownMenuItem, DropdownMenuRoot, DropdownMenuTrigger } from 'reka-ui'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { localizedPublicPath } from '@/i18n/localeConfig'
import Icon from '@/components/icons/Icon.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import AnnouncementBell from '@/components/common/AnnouncementBell.vue'
import ThemeConfigDrawer from '@/components/common/ThemeConfigDrawer.vue'
import WebsiteStatusLink from '@/components/user/monitor/WebsiteStatusLink.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore, type ThemeMode } from '@/stores/theme'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { sanitizeUrl } from '@/utils/url'
import { landingCopy } from './home/landingCopy'

const props = defineProps<{ variant?: 'landing' }>()
const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const marketQuery = ref('')
const appStore = useAppStore()
const authStore = useAuthStore()
const mobileOpen = ref(false)
const menuButton = ref<HTMLButtonElement | null>(null)
const landingSearch = ref<HTMLInputElement | null>(null)
const isLanding = computed(() => props.variant === 'landing')
const copy = computed(() => landingCopy(locale.value))
const shortcut = /Mac|iPhone|iPad/.test(navigator.platform) ? '⌘K' : 'Ctrl K'
const themeStore = getActivePinia() ? useThemeStore() : null
const savedTheme = localStorage.getItem('theme')
const fallbackThemeMode = ref<ThemeMode>(savedTheme === 'dark' || savedTheme === 'light' ? savedTheme : 'system')
const fallbackIsDark = ref(document.documentElement.classList.contains('dark'))
const systemTheme = typeof window !== 'undefined' && typeof window.matchMedia === 'function'
  ? window.matchMedia('(prefers-color-scheme: dark)')
  : null
const themeMode = computed<ThemeMode>(() => (themeStore ? themeStore.mode : fallbackThemeMode.value))
const isDark = computed<boolean>(() => (themeStore ? themeStore.isDark : fallbackIsDark.value))
const settings = computed(() => appStore.cachedPublicSettings)
const siteName = computed(() => settings.value?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(settings.value?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const docUrl = computed(() => sanitizeUrl(settings.value?.doc_url || appStore.docUrl || ''))
const registrationEnabled = computed(() => settings.value?.registration_enabled === true)
const showModels = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza) && (authStore.isAuthenticated || settings.value?.model_plaza_require_auth !== true))
const showMonitor = computed(() => isFeatureFlagEnabled(FeatureFlags.channelMonitor))
const dashboardPath = computed(() => authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
const dashboardTarget = computed(() => authStore.isAuthenticated ? dashboardPath.value : { path: '/login', query: { redirect: '/dashboard' } })
const loginTarget = computed(() => ({ path: '/login', query: { redirect: route.name === 'ModelPlaza' ? route.fullPath : '/dashboard' } }))
const themeChoices = computed(() => [
  { mode: 'light', label: t('publicSite.nav.lightMode'), icon: Sun },
  { mode: 'dark', label: t('publicSite.nav.darkMode'), icon: Moon },
  { mode: 'system', label: copy.value.systemTheme, icon: Monitor },
] as const)
const landingMenus = computed(() => [
  { label: copy.value.products, items: [
    { label: t('publicHome.services.title'), to: '#features' },
    { label: t('publicHome.quickstart.createKey.title'), to: '/keys' },
    { label: t('publicHome.actions.dashboard'), to: dashboardTarget.value },
  ] },
  { label: copy.value.resources, items: [
    { label: t('publicSite.nav.quickStart'), to: '#quick-start' },
    { label: t('publicSite.nav.apiDocs'), to: localizedPublicPath('/docs/api', locale.value) },
    ...(showMonitor.value ? [{ label: t('nav.channelStatus'), to: '/monitor' }] : []),
    { label: t('publicSite.footer.usage'), to: '/key-usage' },
  ] },
])

function searchModels() {
  const search = marketQuery.value.trim()
  void router.push({ path: '/model-plaza', query: { ...(search ? { search } : {}), ...(locale.value !== 'en' ? { lang: locale.value } : {}) } })
}

function synchronizeTheme() {
  if (themeStore) return
  fallbackIsDark.value =
    fallbackThemeMode.value === 'dark' ||
    (fallbackThemeMode.value === 'system' && Boolean(systemTheme?.matches))
  document.documentElement.classList.toggle('dark', fallbackIsDark.value)
}

function applyTheme(mode: ThemeMode) {
  if (themeStore) {
    themeStore.setMode(mode)
    return
  }
  fallbackThemeMode.value = mode
  if (mode === 'system') localStorage.removeItem('theme')
  else localStorage.setItem('theme', mode)
  synchronizeTheme()
}

function focusLandingSearch(event: KeyboardEvent) {
  const input = landingSearch.value
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k' && input?.offsetWidth) {
    event.preventDefault()
    input.focus()
  }
}

function closeMobileNavigation() {
  if (!mobileOpen.value) return
  mobileOpen.value = false
  menuButton.value?.focus()
}

watch(() => route.fullPath, () => { mobileOpen.value = false })
onMounted(() => {
  if (!appStore.publicSettingsLoaded) void appStore.fetchPublicSettings()
  if (!themeStore) systemTheme?.addEventListener?.('change', synchronizeTheme)
  if (isLanding.value) document.addEventListener('keydown', focusLandingSearch)
})
onBeforeUnmount(() => {
  if (!themeStore) systemTheme?.removeEventListener?.('change', synchronizeTheme)
  document.removeEventListener('keydown', focusLandingSearch)
})
</script>

<style scoped>
.public-header {
  --public-nav-bg: rgb(250 250 252 / 90%);
  --public-nav-panel: #fafafc;
  --public-nav-ink: #1d1d1f;
  --public-nav-muted: #56565b;
  --public-nav-line: rgb(0 0 0 / 9%);
  --public-nav-hover: rgb(0 0 0 / 5%);
  --public-nav-accent: #0066cc;
  position: sticky;
  top: 0;
  z-index: 40;
  border-bottom: 1px solid var(--public-nav-line);
  background: var(--public-nav-bg);
  color: var(--public-nav-ink);
  backdrop-filter: saturate(160%) blur(20px);
}
.dark .public-header {
  --public-nav-bg: rgb(22 22 23 / 90%);
  --public-nav-panel: #161617;
  --public-nav-ink: #f5f5f7;
  --public-nav-muted: #b5b5bd;
  --public-nav-line: rgb(255 255 255 / 12%);
  --public-nav-hover: rgb(255 255 255 / 8%);
  --public-nav-accent: #2997ff;
}
.public-header-inner {
  display: flex;
  align-items: center;
  gap: clamp(12px, 2vw, 30px);
  max-width: 1320px;
  min-height: 54px;
  margin-inline: auto;
  padding-inline: clamp(16px, 3vw, 36px);
}
.public-brand {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  gap: 9px;
  min-width: 0;
  font-size: 16px;
  font-weight: 600;
  letter-spacing: -.025em;
}
.public-brand img { width: 25px; height: 25px; flex: none; object-fit: contain; }
.public-brand > span { max-width: 185px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.public-navigation {
  display: flex;
  align-items: center;
  gap: clamp(7px, 1vw, 18px);
  margin-inline: auto;
}
.public-navigation a {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-height: 34px;
  padding: 4px;
  border-radius: 8px;
  color: var(--public-nav-muted);
  font-size: 13px;
  white-space: nowrap;
}
.public-navigation a:hover { color: var(--public-nav-ink); }
.public-navigation a.router-link-exact-active:not([href*="#"]) {
  color: var(--public-nav-accent);
  font-weight: 600;
}
.public-header-actions { display: flex; align-items: center; gap: 6px; }
.public-navigation-and-tools { display: contents; }
.public-search {
  display: flex;
  align-items: center;
  gap: 7px;
  width: 154px;
  padding-inline: 10px;
  border: 1px solid var(--public-nav-line);
  border-radius: 999px;
  color: var(--public-nav-muted);
}
.public-search input {
  width: 100%;
  min-width: 0;
  padding: 6px 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--public-nav-ink);
  font-size: 12px;
  box-shadow: none;
}
.public-search:focus-within,
.public-header :is(a, button):focus-visible {
  outline: 2px solid #0071e3;
  outline-offset: 2px;
}
.public-icon-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 32px;
  height: 34px;
  border-radius: 999px;
  color: var(--public-nav-muted);
}
.public-icon-button:hover { background: var(--public-nav-hover); }
.public-header-primary {
  border-radius: 999px;
  background: #0066cc;
  color: #fff;
  font-weight: 500;
}
.public-header-primary:hover { background: #0059b3; color: #fff; }
.public-header-login { border-radius: 999px; color: var(--public-nav-muted); }
.public-menu-toggle { display: none; }
.public-language :deep(button) { color: var(--public-nav-muted); }

@media (max-width: 1210px) {
  .public-search { display: none; }
}
@media (max-width: 960px) {
  .public-menu-toggle { display: inline-flex; }
  .public-header-actions { margin-inline-start: auto; }
  .public-navigation {
    position: absolute;
    top: 100%;
    inset-inline: 0;
    display: none;
    align-items: stretch;
    flex-direction: column;
    gap: 4px;
    max-height: calc(100dvh - 54px);
    overflow: auto;
    padding: 12px 16px 18px;
    border-bottom: 1px solid var(--public-nav-line);
    background: var(--public-nav-panel);
  }
  .public-navigation.is-open { display: flex; }
  .public-navigation a { min-height: 44px; padding-inline: 12px; }
  .public-navigation a:hover { background: var(--public-nav-hover); }
}
@media (max-width: 480px) {
  .public-header-inner { gap: 6px; padding-inline: 12px; }
  .public-brand { flex: 1 1 0; gap: 6px; }
  .public-brand > span { min-width: 0; }
  .public-header-actions { flex: none; gap: 2px; }
  .public-header-actions :is(.public-header-primary, .public-header-login) { padding-inline: 8px; }
  .public-notifications { display: none; }
}
.landing-header {
  top: 8px;
  margin: 8px 8px 0;
  border: 1px solid var(--gateway-line);
  border-radius: 17px;
  background: color-mix(in srgb, var(--gateway-surface) 88%, transparent);
  color: var(--gateway-ink);
  box-shadow: 0 1px 2px #00000006;
}
.landing-header .public-header-inner {
  gap: clamp(10px, 1.2vw, 16px);
  max-width: none;
  min-height: 68px;
  padding-inline: 19px;
}
.landing-header .public-brand { gap: 6px; font-size: 20px; font-weight: 700; }
.landing-header .public-brand img { width: 24px; height: 24px; }
.landing-header .public-navigation {
  gap: clamp(6px, 1.4vw, 23px);
  margin-inline: 0 auto;
}
.landing-header .public-navigation :is(a.router-link-exact-active, a, button) {
  color: var(--gateway-muted);
  font-weight: 450;
}
.landing-header .public-navigation :is(a, button):hover { color: var(--gateway-ink); }
.landing-menu-trigger {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-height: 34px;
  padding-inline: 4px;
  border-radius: 8px;
  font-size: 13px;
}
.landing-menu-items {
  z-index: 45;
  min-width: 165px;
  padding: 5px;
  border: 1px solid var(--gateway-line);
  border-radius: 10px;
  background: var(--gateway-surface);
  color: var(--gateway-ink);
  box-shadow: 0 10px 24px #00000013;
  font-size: 13px;
}
.landing-menu-items a { display: block; padding: 8px 11px; border-radius: 6px; }
.landing-menu-items a[data-highlighted] { outline: none; background: var(--gateway-soft); }
.landing-header .public-search {
  width: clamp(125px, 13vw, 183px);
  min-height: 34px;
  border-color: var(--gateway-line);
  background: var(--gateway-surface);
  box-shadow: 0 1px 5px #0000000d;
}
.landing-search kbd {
  flex: none;
  padding: 2px 4px;
  border: 1px solid var(--gateway-line);
  border-radius: 5px;
  background: var(--gateway-soft);
  font-size: 10px;
  line-height: 1.25;
}
.landing-header .public-header-actions { gap: 8px; }
.landing-themes {
  display: flex;
  align-items: center;
  padding: 2px;
  border: 1px solid var(--gateway-line);
  border-radius: 999px;
  background: var(--gateway-surface);
}
.landing-themes button {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  color: var(--gateway-muted);
}
.landing-themes button[aria-pressed=true],
.landing-themes button:hover { background: var(--gateway-soft); color: var(--gateway-ink); }
.landing-header .public-header-primary {
  min-height: 36px;
  border-radius: 8px;
  background: var(--gateway-ink);
  color: var(--gateway-surface);
}
.landing-header .public-header-primary:hover { opacity: .88; }
.landing-header :is(a,button):focus-visible,
.landing-header .public-search:focus-within { outline-color: var(--gateway-accent); }
@media (max-width: 1220px) {
  .landing-header .public-search { display: none; }
}
@media (max-width: 1120px) {
  .landing-header .public-menu-toggle { display: inline-flex; }
  .landing-header .public-header-inner { min-height: 48px; }
  .landing-website-status { margin-inline-start: auto; }
  .landing-header .public-navigation-and-tools {
    position: absolute;
    top: calc(100% + 6px);
    inset-inline: 0;
    display: none;
    max-height: calc(100dvh - 76px);
    overflow: auto;
    padding: 9px;
    border: 1px solid var(--gateway-line);
    border-radius: 14px;
    background: var(--gateway-surface);
  }
  .landing-header .public-navigation-and-tools.is-open { display: block; }
  .landing-header .public-navigation {
    position: static;
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: 3px;
    padding: 0;
    margin: 0;
    border: 0;
    background: transparent;
  }
  .landing-header .public-navigation :is(a,.landing-menu-trigger) {
    justify-content: flex-start;
    min-height: 42px;
    padding-inline: 11px;
  }
  .landing-header .public-header-actions {
    justify-content: flex-end;
    flex-wrap: wrap;
    gap: 8px;
    margin-top: 8px;
    padding: 9px 3px 2px;
    border-top: 1px solid var(--gateway-line);
  }
}
@media (max-width: 600px) {
  .landing-header { border-radius: 13px; }
  .landing-header .public-header-inner { gap: 8px; padding-inline: 13px; }
  .landing-header .public-brand { font-size: 18px; }
  .landing-header .public-brand > span { max-width: 65vw; }
}
@media (max-width: 370px) {
  .landing-header .public-header-login { padding-inline: 4px; }
  .landing-header .public-header-actions { gap: 3px; }
}
</style>
