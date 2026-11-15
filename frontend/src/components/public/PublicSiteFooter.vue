<template>
  <SiteFooter plain class="public-footer" :class="{ 'public-footer-home': variant === 'home' }">
    <div class="public-footer-main" :style="variant === 'home' ? { '--footer-columns': footerColumns } : undefined">
      <div class="public-footer-brand">
        <RouterLink to="/home" class="footer-logo"><img :src="siteLogo || '/logo.svg'" alt="" /><span>{{ siteName }}</span></RouterLink>
        <p>{{ siteSubtitle }}</p>
      </div>
      <div class="public-footer-column">
        <h2>{{ t('publicSite.footer.help') }}</h2>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ t('publicSite.nav.helpCenter') }}</a>
        <RouterLink v-if="variant === 'home'" to="/docs">{{ t('docs.title') }}</RouterLink>
        <RouterLink to="/docs/api">{{ t('publicSite.nav.apiDocs') }}</RouterLink>
        <RouterLink to="/home#quick-start">{{ t('publicSite.nav.quickStart') }}</RouterLink>
      </div>
      <div class="public-footer-column">
        <h2>{{ t('publicSite.footer.platform') }}</h2>
        <RouterLink v-if="showModels" to="/model-plaza">{{ t(variant === 'home' ? 'publicSite.market.heading' : 'publicSite.nav.models') }}</RouterLink>
        <RouterLink v-if="variant === 'home' && showMonitor" to="/monitor">{{ t('nav.channelStatus') }}</RouterLink>
        <RouterLink :to="dashboardTarget">{{ t('publicSite.nav.console') }}</RouterLink>
        <RouterLink to="/key-usage">{{ t('publicSite.footer.usage') }}</RouterLink>
      </div>
      <div v-if="documents.length" class="public-footer-column">
        <h2>{{ t('publicSite.footer.agreements') }}</h2>
        <RouterLink v-for="document in documents" :key="document.id" :to="`/legal/${encodeURIComponent(document.id)}`">{{ document.title }}</RouterLink>
      </div>
      <div v-if="contactInfo" class="public-footer-column public-footer-contact">
        <h2>{{ t('publicSite.footer.contact') }}</h2>
        <p>{{ contactInfo }}</p>
      </div>
    </div>
    <div class="public-footer-bottom">
      <p>{{ t('publicSite.footer.copyright', { year: currentYear, siteName }) }}</p>
      <p>{{ t('publicSite.footer.notice') }}</p>
    </div>
  </SiteFooter>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import SiteFooter from '@/components/site-shell/SiteFooter.vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()
defineProps<{ variant?: 'home' }>()
const appStore = useAppStore()
const authStore = useAuthStore()
const settings = computed(() => appStore.cachedPublicSettings)
const siteName = computed(() => settings.value?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(settings.value?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => settings.value?.site_subtitle || t('publicSite.footer.subtitle'))
const docUrl = computed(() => sanitizeUrl(settings.value?.doc_url || appStore.docUrl || ''))
const contactInfo = computed(() => settings.value?.contact_info?.trim() || '')
const documents = computed(() => settings.value?.login_agreement_documents ?? [])
const footerColumns = computed(() => 2 + Number(documents.value.length > 0) + Number(Boolean(contactInfo.value)))
const showModels = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza) && (authStore.isAuthenticated || settings.value?.model_plaza_require_auth !== true))
const showMonitor = computed(() => isFeatureFlagEnabled(FeatureFlags.channelMonitor))
const dashboardTarget = computed(() => authStore.isAuthenticated ? (authStore.isAdmin ? '/admin/dashboard' : '/dashboard') : '/login')
const currentYear = new Date().getFullYear()
</script>

<style scoped>
.public-footer {
  --footer-line: hsl(var(--border));
  --footer-heading: hsl(var(--foreground));
  --footer-text: hsl(var(--muted-foreground));
  --footer-accent: hsl(var(--brand));
  border-top: 1px solid var(--footer-line);
  background: hsl(var(--background));
  color: var(--footer-text);
}
.public-footer-main { display: flex; justify-content: space-between; flex-wrap: wrap; gap: 32px; max-width: 1440px; margin: 0 auto; padding: 24px 24px 20px; }
.public-footer-brand { flex: 1 1 250px; max-width: 350px; min-width: 0; }
.footer-logo { display: inline-flex; align-items: center; gap: 8px; color: var(--footer-heading); font-size: 16px; font-weight: 600; }
.footer-logo img { width: 26px; height: 26px; object-fit: contain; }
.public-footer-brand p { margin-top: 12px; font-size: 13px; line-height: 1.7; }
.public-footer-column { display: flex; min-width: 85px; flex: 0 1 auto; flex-direction: column; align-items: flex-start; gap: 9px; font-size: 13px; }
.public-footer-column h2 { margin-bottom: 2px; color: var(--footer-heading); font-size: 13px; font-weight: 600; }
.public-footer-column a:hover { color: var(--footer-heading); text-decoration: underline; text-underline-offset: 3px; }
.public-footer :is(a, button):focus-visible { outline: 2px solid hsl(var(--ring)); outline-offset: 3px; }
.public-footer-contact { max-width: 180px; }
.public-footer-brand p, .public-footer-column, .public-footer-bottom { overflow-wrap: anywhere; }
.public-footer-contact p { line-height: 1.7; white-space: pre-line; }
.public-footer-bottom { display: flex; justify-content: space-between; gap: 12px; max-width: 1440px; margin: 0 auto; padding: 12px 24px 18px; border-top: 1px solid var(--footer-line); font-size: 12px; line-height: 1.7; }
.public-footer-home { --footer-gutter: 40px; background: hsl(var(--canvas)); }
.public-footer-home .public-footer-main { display: grid; grid-template-columns: minmax(0, 2fr) repeat(var(--footer-columns, 3), minmax(0, 1fr)); gap: 24px; max-width: 1440px; padding-inline: var(--footer-gutter); }
.public-footer-home .public-footer-brand, .public-footer-home .public-footer-column { max-width: none; min-width: 0; }
.public-footer-home .public-footer-column a:hover { color: var(--footer-accent); }
.public-footer-home .public-footer-bottom { width: calc(100% - 2 * var(--footer-gutter)); max-width: calc(1440px - 2 * var(--footer-gutter)); padding-inline: 0; }
@media (max-width: 760px) {
  .public-footer-home { --footer-gutter: 16px; }
  .public-footer-home .public-footer-main { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 24px 18px; }
  .public-footer-home .public-footer-brand { grid-column: 1 / -1; }
}
@media (max-width: 640px) {
  .public-footer-main { gap: 22px; padding-inline: 16px; }
  .public-footer-brand { flex-basis: 100%; max-width: none; }
  .public-footer-column { flex: 1 1 30%; }
  .public-footer-bottom { flex-direction: column; gap: 5px; padding-inline: 16px; }
}
</style>
