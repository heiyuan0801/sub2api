<template>
  <footer class="public-footer" :class="{ 'landing-footer': isLanding }">
    <div class="public-footer-main" :style="{ '--footer-columns': footerColumns }">
      <div class="public-footer-brand">
        <RouterLink :to="localizedPublicPath('/home', locale)" class="footer-logo"><img :src="siteLogo || '/logo.svg'" alt="" /><span>{{ siteName }}</span></RouterLink>
        <p>{{ siteSubtitle }}</p>
        <RouterLink v-if="isLanding && showMonitor" to="/monitor" class="footer-status">
          <Activity :size="13" aria-hidden="true" />{{ t('nav.channelStatus') }}
        </RouterLink>
      </div>
      <div class="public-footer-column">
        <h2>{{ isLanding ? copy.resources : t('publicSite.footer.help') }}</h2>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ t('publicSite.nav.helpCenter') }}</a>
        <RouterLink v-if="isLanding" :to="localizedPublicPath('/docs', locale)">{{ t('docs.title') }}</RouterLink>
        <RouterLink :to="localizedPublicPath('/docs/api', locale)">{{ t('publicSite.nav.apiDocs') }}</RouterLink>
        <RouterLink :to="localizedPublicPath('/home#quick-start', locale)">{{ t('publicSite.nav.quickStart') }}</RouterLink>
        <a v-if="isLanding" href="#faq">{{ copy.faq }}</a>
      </div>
      <div class="public-footer-column">
        <h2>{{ isLanding ? copy.products : t('publicSite.footer.platform') }}</h2>
        <a v-if="isLanding" href="#features">{{ t('publicHome.services.title') }}</a>
        <RouterLink v-if="showModels" :to="localizedPublicPath('/model-plaza', locale)">{{ t(isLanding ? 'publicSite.market.heading' : 'publicSite.nav.models') }}</RouterLink>
        <RouterLink v-if="isLanding" to="/keys">{{ t('keys.title') }}</RouterLink>
        <RouterLink v-if="isLanding && showMonitor" to="/monitor">{{ t('nav.channelStatus') }}</RouterLink>
        <RouterLink :to="dashboardTarget">{{ t('publicSite.nav.console') }}</RouterLink>
        <RouterLink to="/key-usage">{{ t('publicSite.footer.usage') }}</RouterLink>
      </div>
      <div v-if="documents.length" class="public-footer-column">
        <h2>{{ t('publicSite.footer.agreements') }}</h2>
        <RouterLink v-for="document in documents" :key="document.id" :to="`/legal/${encodeURIComponent(document.id)}`">{{ document.title }}</RouterLink>
      </div>
      <div v-if="isLanding && showModels" class="public-footer-column">
        <h2>{{ t('publicSite.nav.models') }}</h2>
        <RouterLink :to="localizedPublicPath('/model-plaza', locale)">{{ t('publicSite.market.explore') }}</RouterLink>
        <RouterLink :to="localizedPublicPath('/model-plaza', locale)">{{ copy.pricing }}</RouterLink>
        <a href="#routing">{{ t('publicHome.services.unifiedApi.title') }}</a>
      </div>
      <div v-if="isLanding && showModels && providers?.length" class="public-footer-column">
        <h2>{{ t('publicSite.market.providers') }}</h2>
        <RouterLink v-for="provider in providers" :key="provider.name" :to="providerLink(provider.name)">{{ platformLabel(provider.name) }}</RouterLink>
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
  </footer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { localizedPublicPath } from '@/i18n/localeConfig'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { sanitizeUrl } from '@/utils/url'
import { platformLabel } from '@/utils/platformColors'
import { Activity } from '@lucide/vue'
import { landingCopy } from './home/landingCopy'

const { t, locale } = useI18n()
const props = defineProps<{ variant?: 'landing'; providers?: { name: string }[] }>()
const appStore = useAppStore()
const authStore = useAuthStore()
const settings = computed(() => appStore.cachedPublicSettings)
const siteName = computed(() => settings.value?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(settings.value?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => settings.value?.site_subtitle || t('publicSite.footer.subtitle'))
const docUrl = computed(() => sanitizeUrl(settings.value?.doc_url || appStore.docUrl || ''))
const contactInfo = computed(() => settings.value?.contact_info?.trim() || '')
const documents = computed(() => settings.value?.login_agreement_documents ?? [])
const isLanding = computed(() => props.variant === 'landing')
const copy = computed(() => landingCopy(locale.value))
const showModels = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza) && (authStore.isAuthenticated || settings.value?.model_plaza_require_auth !== true))
const showMonitor = computed(() => isFeatureFlagEnabled(FeatureFlags.channelMonitor))
const footerColumns = computed(() =>
  2 + Number(documents.value.length > 0) + Number(Boolean(contactInfo.value)) +
  Number(isLanding.value && showModels.value) +
  Number(isLanding.value && showModels.value && Boolean(props.providers?.length)),
)
const dashboardTarget = computed(() => authStore.isAuthenticated ? (authStore.isAdmin ? '/admin/dashboard' : '/dashboard') : '/login')
const currentYear = new Date().getFullYear()
const providerLink = (name: string) => localizedPublicPath(`/model-plaza?${new URLSearchParams({ search: platformLabel(name) })}`, locale.value)
</script>

<style scoped>
.public-footer {
  --footer-background: #f5f5f7;
  --footer-line: #d9d9de;
  --footer-heading: #1d1d1f;
  --footer-text: #56565b;
  --footer-accent: #0066cc;
  --footer-gutter: clamp(16px, 3vw, 36px);
  border-top: 1px solid var(--footer-line);
  background: var(--footer-background);
  color: var(--footer-text);
}
.dark .public-footer {
  --footer-background: #161617;
  --footer-line: #36363b;
  --footer-heading: #f5f5f7;
  --footer-text: #b5b5bd;
  --footer-accent: #2997ff;
}
.public-footer-main,
.public-footer-bottom {
  width: calc(100% - 2 * var(--footer-gutter));
  max-width: 1248px;
  margin-inline: auto;
  overflow-wrap: anywhere;
}
.public-footer-main {
  display: grid;
  grid-template-columns: minmax(0, 1.8fr) repeat(var(--footer-columns, 2), minmax(0, 1fr));
  gap: 28px;
  padding-block: 40px 32px;
}
.footer-logo {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  color: var(--footer-heading);
  font-size: 18px;
  font-weight: 600;
  letter-spacing: -.025em;
}
.footer-logo img { width: 24px; height: 24px; object-fit: contain; }
.public-footer-brand p { max-width: 33ch; margin-top: 12px; font-size: 13px; line-height: 1.6; }
.public-footer-column { display: flex; flex-direction: column; align-items: flex-start; gap: 9px; font-size: 13px; }
.public-footer-column h2 { margin-bottom: 3px; color: var(--footer-heading); font-size: 13px; font-weight: 600; }
.public-footer a:hover { color: var(--footer-accent); text-decoration: underline; text-underline-offset: 3px; }
.public-footer a:focus-visible { outline: 2px solid #0071e3; outline-offset: 3px; }
.public-footer-contact p { line-height: 1.55; white-space: pre-line; }
.public-footer-bottom {
  display: flex;
  justify-content: space-between;
  gap: 8px 24px;
  padding-block: 14px 20px;
  border-top: 1px solid var(--footer-line);
  font-size: 12px;
  line-height: 1.5;
}
@media (max-width: 760px) {
  .public-footer-main { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 22px 16px; padding-block: 28px 24px; }
  .public-footer-brand { grid-column: 1 / -1; }
  .public-footer-bottom { flex-direction: column; }
}
.landing-footer {
  --footer-background: var(--gateway-surface);
  --footer-line: var(--gateway-line);
  --footer-heading: var(--gateway-ink);
  --footer-text: var(--gateway-muted);
  --footer-accent: var(--gateway-accent);
  --footer-gutter: var(--gateway-gutter);
}
.landing-footer :is(.public-footer-main, .public-footer-bottom) { max-width: var(--gateway-width); }
.landing-footer .public-footer-main {
  grid-template-columns: minmax(140px, 1.3fr) repeat(var(--footer-columns), minmax(0, 1fr));
  gap: 24px;
  padding-block: 64px 56px;
}
.landing-footer .public-footer-column { gap: 12px; font-size: 14px; }
.landing-footer .public-footer-column h2 { font-size: 14px; }
.footer-status {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  margin-top: 17px;
  padding: 5px 10px;
  border: 1px solid var(--footer-line);
  border-radius: 999px;
  font-size: 12px;
}
@media (max-width: 900px) {
  .landing-footer .public-footer-main { grid-template-columns: repeat(3, minmax(0, 1fr)); padding-block: 38px; }
  .landing-footer .public-footer-brand { grid-column: 1 / -1; }
}
@media (max-width: 520px) {
  .landing-footer .public-footer-main { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 25px 16px; }
}
</style>
