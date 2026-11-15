<template>
  <SiteHeader sticky class="public-header" @keydown.esc="mobileOpen = false">
    <div class="public-header-inner">
      <SiteBrand class="public-brand" :name="siteName" :logo="siteLogo" />

      <nav id="public-navigation" class="public-navigation" :class="{ 'is-open': mobileOpen }" :aria-label="t('publicSite.nav.mainNavigation')">
        <RouterLink to="/home">{{ t('publicSite.market.home') }}</RouterLink>
        <RouterLink v-if="showModels" to="/model-plaza" @click="mobileOpen = false">{{ t('publicSite.market.heading') }}</RouterLink>
        <RouterLink v-if="showMonitor" to="/monitor" @click="mobileOpen = false">{{ t('nav.channelStatus') }}</RouterLink>
        <RouterLink to="/home#quick-start" @click="mobileOpen = false">{{ t('publicSite.nav.quickStart') }}</RouterLink>
        <RouterLink to="/docs" @click="mobileOpen = false">{{ t('docs.title') }}</RouterLink>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" @click="mobileOpen = false">{{ t('publicSite.nav.helpCenter') }} <Icon name="externalLink" size="xs" /></a>
      </nav>

      <div class="public-header-actions">
        <form v-if="showModels" class="public-search" role="search" @submit.prevent="searchModels">
          <Icon name="search" size="sm" />
          <input v-model="marketQuery" type="search" :aria-label="t('publicSite.models.searchLabel')" :placeholder="t('publicSite.market.search')" />
        </form>
        <AnnouncementBell v-if="authStore.isAuthenticated" class="public-notifications" />
        <LocaleSwitcher compact class="public-language" />
        <SiteThemeToggle />
        <Button v-if="authStore.isAuthenticated" as-child variant="default" size="sm"><RouterLink :to="dashboardTarget">{{ t('publicSite.nav.enterConsole') }}</RouterLink></Button>
        <template v-else>
          <Button as-child variant="ghost" size="sm"><RouterLink :to="loginTarget">{{ t('publicSite.nav.login') }}</RouterLink></Button>
          <Button v-if="registrationEnabled" as-child variant="default" size="sm"><RouterLink to="/register">{{ t('publicSite.nav.register') }}</RouterLink></Button>
        </template>
        <Button variant="ghost" size="icon-sm" type="button" class="public-menu-toggle" :aria-expanded="mobileOpen" aria-controls="public-navigation" :aria-label="mobileOpen ? t('publicSite.nav.closeNavigation') : t('publicSite.nav.openNavigation')" @click="mobileOpen = !mobileOpen">
          <Icon :name="mobileOpen ? 'x' : 'menu'" size="sm" aria-hidden="true" />
        </Button>
      </div>
    </div>
  </SiteHeader>
</template>

<script setup lang="ts">
import { Button } from '@/components/ui/button'
import SiteBrand from '@/components/site-shell/SiteBrand.vue'
import SiteHeader from '@/components/site-shell/SiteHeader.vue'
import SiteThemeToggle from '@/components/site-shell/SiteThemeToggle.vue'
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
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
const router = useRouter()
const marketQuery = ref('')
const appStore = useAppStore()
const authStore = useAuthStore()
const mobileOpen = ref(false)
const settings = computed(() => appStore.cachedPublicSettings)
const siteName = computed(() => settings.value?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => settings.value?.site_logo || appStore.siteLogo || '')
const docUrl = computed(() => sanitizeUrl(settings.value?.doc_url || appStore.docUrl || ''))
const registrationEnabled = computed(() => settings.value?.registration_enabled === true)
const showModels = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza) && (authStore.isAuthenticated || settings.value?.model_plaza_require_auth !== true))
const showMonitor = computed(() => isFeatureFlagEnabled(FeatureFlags.channelMonitor))
const dashboardPath = computed(() => authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
const dashboardTarget = computed(() => authStore.isAuthenticated ? dashboardPath.value : { path: '/login', query: { redirect: '/dashboard' } })
const loginTarget = computed(() => ({ path: '/login', query: { redirect: route.name === 'ModelPlaza' ? route.fullPath : '/dashboard' } }))

function searchModels() {
  const search = marketQuery.value.trim()
  void router.push({ path: '/model-plaza', query: search ? { search } : {} })
}

watch(() => route.fullPath, () => { mobileOpen.value = false })
onMounted(() => {
  if (!appStore.publicSettingsLoaded) void appStore.fetchPublicSettings()
})
</script>

<style scoped>
.public-header { min-height: 60px; z-index: 40; }
.public-header-inner { display: flex; align-items: center; gap: 16px; max-width: 1536px; min-height: 59px; margin: 0 auto; padding: 0 24px; }
.public-brand { flex-shrink: 0; }
.public-navigation { display: flex; align-items: center; gap: 2px; min-width: 0; margin-right: auto; }
.public-navigation a { display: inline-flex; align-items: center; gap: 4px; padding: 6px 8px; border-radius: calc(var(--radius) - 2px); color: hsl(var(--muted-foreground)); font-size: 13px; font-weight: 500; line-height: 20px; white-space: nowrap; }
.public-navigation a:hover, .public-navigation a.router-link-active { color: hsl(var(--accent-foreground)); background: hsl(var(--accent)); }
.public-navigation a:focus-visible { outline: 2px solid hsl(var(--ring)); outline-offset: 2px; }
.public-header-actions { display: flex; align-items: center; gap: 6px; }
.public-language :deep(button) { color: inherit; }
.public-search { display: flex; align-items: center; gap: 7px; width: clamp(128px, 14vw, 220px); min-height: 32px; padding: 0 8px; border: 1px solid hsl(var(--border)); border-radius: calc(var(--radius) - 2px); color: hsl(var(--muted-foreground)); background: hsl(var(--background)); }
.public-search input { width: 100%; min-width: 0; padding: 4px 0; border: 0; background: transparent; color: hsl(var(--foreground)); font-size: 12px; }
.public-search input:focus { outline: none; box-shadow: none; }
.public-search:focus-within { outline: 2px solid hsl(var(--ring)); outline-offset: 1px; }
.public-menu-toggle { display: none; }
@media (max-width: 1150px) {
  .public-header-inner { gap: 10px; }
  .public-search { display: none; }
}
@media (max-width: 960px) {
  .public-header-inner { justify-content: space-between; padding-inline: 16px; }
  .public-menu-toggle { display: inline-flex; }
  .public-navigation { position: absolute; top: 100%; right: 0; left: 0; display: none; align-items: stretch; margin: 0; padding: 10px 16px; border-bottom: 1px solid hsl(var(--border)); background: hsl(var(--background)); }
  .public-navigation.is-open { display: flex; flex-direction: column; }
  .public-navigation a { padding: 9px 10px; }
}
@media (max-width: 480px) {
  .public-brand { flex: 1 1 0; }
  .public-notifications { display: none; }
  .public-header-inner { gap: 4px; padding-inline: 12px; }
  .public-header-actions { flex-shrink: 0; gap: 1px; }
  .public-header-actions > a { padding-inline: 7px; }
}
</style>
