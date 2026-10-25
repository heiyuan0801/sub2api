<template>
  <header class="public-header" @keydown.esc="mobileOpen = false">
    <div class="public-header-inner">
      <RouterLink to="/home" class="public-brand" :aria-label="t('publicSite.nav.homeLabel', { siteName })">
        <img :src="siteLogo || '/logo.svg'" alt="" />
        <span>{{ siteName }}</span>
        <small>API</small>
      </RouterLink>

      <nav id="public-navigation" class="public-navigation" :class="{ 'is-open': mobileOpen }" :aria-label="t('publicSite.nav.mainNavigation')">
        <RouterLink :to="dashboardTarget" @click="mobileOpen = false">{{ t('publicSite.nav.console') }}</RouterLink>
        <RouterLink v-if="showModels" to="/model-plaza" :class="{ active: route.name === 'ModelPlaza' }" @click="mobileOpen = false">{{ t('publicSite.nav.models') }}</RouterLink>
        <RouterLink to="/home#quick-start" @click="mobileOpen = false">{{ t('publicSite.nav.quickStart') }}</RouterLink>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" @click="mobileOpen = false">{{ t('publicSite.nav.helpCenter') }}</a>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" @click="mobileOpen = false">
          {{ t('publicSite.nav.apiDocs') }} <Icon name="externalLink" size="xs" />
        </a>
      </nav>

      <div class="public-header-actions">
        <AnnouncementBell v-if="authStore.isAuthenticated" class="public-notifications" />
        <LocaleSwitcher compact class="public-language" />
        <button type="button" class="public-icon-button" :aria-label="isDark ? t('publicSite.nav.lightMode') : t('publicSite.nav.darkMode')" @click="toggleTheme">
          <Icon :name="isDark ? 'moon' : 'sun'" size="sm" />
        </button>
        <Button v-if="authStore.isAuthenticated" as-child variant="default" size="sm"><RouterLink :to="dashboardTarget">{{ t('publicSite.nav.enterConsole') }}</RouterLink></Button>
        <template v-else>
          <Button as-child variant="ghost" size="sm"><RouterLink :to="loginTarget">{{ t('publicSite.nav.login') }}</RouterLink></Button>
          <Button v-if="registrationEnabled" as-child variant="default" size="sm"><RouterLink to="/register">{{ t('publicSite.nav.register') }}</RouterLink></Button>
        </template>
        <button type="button" class="public-icon-button public-menu-toggle" :aria-expanded="mobileOpen" aria-controls="public-navigation" :aria-label="mobileOpen ? t('publicSite.nav.closeNavigation') : t('publicSite.nav.openNavigation')" @click="mobileOpen = !mobileOpen">
          <Icon :name="mobileOpen ? 'x' : 'menu'" size="sm" />
        </button>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import AnnouncementBell from '@/components/common/AnnouncementBell.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const mobileOpen = ref(false)
const isDark = ref(document.documentElement.classList.contains('dark'))
const settings = computed(() => appStore.cachedPublicSettings)
const siteName = computed(() => settings.value?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(settings.value?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const docUrl = computed(() => sanitizeUrl(settings.value?.doc_url || appStore.docUrl || ''))
const registrationEnabled = computed(() => settings.value?.registration_enabled === true)
const showModels = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza) && (authStore.isAuthenticated || settings.value?.model_plaza_require_auth !== true))
const dashboardPath = computed(() => authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
const dashboardTarget = computed(() => authStore.isAuthenticated ? dashboardPath.value : { path: '/login', query: { redirect: '/dashboard' } })
const loginTarget = computed(() => ({ path: '/login', query: { redirect: route.name === 'ModelPlaza' ? route.fullPath : '/dashboard' } }))

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

watch(() => route.fullPath, () => { mobileOpen.value = false })
onMounted(() => {
  if (!appStore.publicSettingsLoaded) void appStore.fetchPublicSettings()
})
</script>

<style scoped>
.public-header { position: sticky; top: 0; z-index: 40; height: 64px; border-bottom: 1px solid #f0f0f0; color: #171717; background: rgb(255 255 255 / 94%); backdrop-filter: blur(16px); }
.public-header-inner { display: flex; align-items: center; gap: 20px; max-width: 1536px; height: 100%; margin: 0 auto; padding: 0 32px; }
.public-brand { display: inline-flex; align-items: center; flex-shrink: 0; gap: 12px; min-width: 0; color: inherit; text-decoration: none; }
.public-brand img { width: 28px; height: 28px; object-fit: contain; }
.public-brand > span { max-width: 200px; overflow: hidden; font-size: 24px; font-weight: 700; line-height: 1; letter-spacing: -.8px; text-overflow: ellipsis; white-space: nowrap; }
.public-brand small { align-self: flex-start; margin: -1px 0 0 -7px; padding: 1px 4px; border: 1px solid #ededed; border-radius: 5px; color: #8b8b8b; font-size: 9px; font-weight: 500; line-height: 13px; }
.public-navigation { display: flex; align-items: center; gap: 5px; min-width: 0; margin-right: auto; margin-left: auto; }
.public-navigation a { display: inline-flex; align-items: center; justify-content: center; gap: 4px; padding: 7px 11px; border-radius: 999px; color: inherit; font-size: 13px; font-weight: 600; line-height: 20px; text-decoration: none; white-space: nowrap; transition: background .15s; }
.public-navigation a:hover, .public-navigation a.active { background: #f5f5f5; }
.public-header-actions { display: flex; align-items: center; gap: 9px; }
.public-icon-button { display: inline-flex; align-items: center; justify-content: center; width: 30px; height: 32px; flex-shrink: 0; border-radius: 8px; color: #404040; background: transparent; }
.public-icon-button:hover { background: #f5f5f5; }
.public-auth-button { display: inline-flex; justify-content: center; align-items: center; min-height: 32px; padding: 5px 11px; border: 1px solid #ededed; border-radius: 8px; font-size: 12px; font-weight: 600; white-space: nowrap; text-decoration: none; }
.public-auth-button:hover { background: #f6f6f6; }
.public-auth-button.is-primary { border-color: #181818; color: white; background: #181818; }
.public-auth-button.is-primary:hover { background: #333; }
.public-menu-toggle { display: none; }
.public-language :deep(button) { color: inherit; }
.public-header :is(a, button):focus-visible { outline: 2px solid #737373; outline-offset: 4px; }
.dark .public-header { border-color: #252525; color: #f5f5f5; background: rgb(15 15 15 / 94%); }
.dark .public-brand small { border-color: #333; color: #999; }
.dark .public-navigation a:hover, .dark .public-navigation a.active, .dark .public-icon-button:hover { background: #262626; }
.dark .public-icon-button { color: #d4d4d4; }
.dark .public-auth-button { border-color: #333; }
.dark .public-auth-button:hover { background: #262626; }
.dark .public-auth-button.is-primary { border-color: #f5f5f5; color: #171717; background: #f5f5f5; }
@media (max-width: 1150px) { .public-header-inner { gap: 12px; padding: 0 24px; } .public-navigation { gap: 0; } .public-navigation a { padding-right: 8px; padding-left: 8px; } .public-header-actions { gap: 5px; } .public-brand > span { max-width: 155px; font-size: 22px; } }
@media (max-width: 900px) { .public-header-inner { padding: 0 20px; } .public-menu-toggle { display: inline-flex; } .public-navigation { position: absolute; top: 63px; right: 0; left: 0; display: none; align-items: stretch; gap: 6px; padding: 16px 20px; border-bottom: 1px solid #eee; background: #fff; } .public-navigation.is-open { display: flex; flex-direction: column; } .public-navigation a { justify-content: flex-start; padding: 10px 12px; border-radius: 8px; } .dark .public-navigation { border-color: #262626; background: #111; } }
@media (max-width: 480px) { .public-header-inner { gap: 8px; padding: 0 16px; } .public-brand { flex: 1 1 0; gap: 7px; } .public-brand img { flex-shrink: 0; width: 24px; height: 24px; } .public-brand > span { min-width: 0; max-width: 100px; font-size: 19px; } .public-brand small, .public-notifications { display: none; } .public-header-actions { flex-shrink: 0; gap: 2px; margin-left: 0; } .public-auth-button { padding: 5px 8px; } }
@media (prefers-reduced-motion: reduce) { .public-navigation a { transition: none; } }
.public-header { height: 68px; border-color: hsl(var(--border)); background: hsl(var(--background) / .94); }
.public-header-inner { max-width: 1264px; }
.public-brand > span { font-size: 18px; letter-spacing: -.5px; }
.public-navigation a { border-radius: 6px; font-weight: 500; font-size: 13px; }
@media (max-width: 640px) { .public-header-inner { padding: 0 16px; gap: 12px; } .public-brand > span { max-width: 110px; font-size: 16px; } .public-header-actions { gap: 5px; } }
</style>
