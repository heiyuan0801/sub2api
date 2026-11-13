<template>
  <footer class="public-footer" :class="{ 'public-footer-home': variant === 'home' }">
    <div class="public-footer-main" :style="variant === 'home' ? { '--footer-columns': footerColumns } : undefined">
      <div class="public-footer-brand">
        <RouterLink to="/home" class="footer-logo"><img :src="siteLogo || '/logo.svg'" alt="" /><span>{{ siteName }}</span></RouterLink>
        <p>{{ siteSubtitle }}</p>
      </div>
      <div class="public-footer-column">
        <h2>{{ t('publicSite.footer.help') }}</h2>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ t('publicSite.nav.helpCenter') }}</a>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ t('publicSite.nav.apiDocs') }}</a>
        <RouterLink v-else-if="variant === 'home'" to="/home#api-example">{{ t('publicSite.nav.apiDocs') }}</RouterLink>
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
  </footer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
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
.public-footer { border-top: 1px solid #f0f0f0; color: #737373; background: #fff; }
.public-footer-main { display: flex; justify-content: space-between; flex-wrap: wrap; gap: 42px; max-width: 1280px; margin: 0 auto; padding: 64px 32px 48px; }
.public-footer-brand { flex: 1 1 270px; max-width: 365px; }
.footer-logo { display: inline-flex; align-items: center; gap: 10px; color: #171717; font-size: 24px; font-weight: 700; text-decoration: none; letter-spacing: -.7px; }
.footer-logo img { width: 29px; height: 29px; object-fit: contain; }
.public-footer-brand p { margin-top: 18px; font-size: 13px; line-height: 1.9; }
.public-footer-column { display: flex; flex: 0 1 auto; flex-direction: column; align-items: flex-start; gap: 13px; min-width: 85px; font-size: 13px; }
.public-footer-column h2 { margin-bottom: 5px; color: #171717; font-size: 13px; font-weight: 600; }
.public-footer-column a { text-decoration: none; }
.public-footer-column a:hover { color: #171717; }
.public-footer-column a:focus-visible { outline: 2px solid #737373; outline-offset: 4px; }
.public-footer-contact { max-width: 180px; }
.public-footer-contact p { overflow-wrap: anywhere; line-height: 1.9; white-space: pre-line; }
.public-footer-bottom { display: flex; justify-content: space-between; gap: 20px; max-width: 1280px; margin: 0 auto; padding: 22px 32px 30px; border-top: 1px solid #f0f0f0; color: #969696; font-size: 11px; line-height: 1.8; }
.dark .public-footer { border-color: #262626; color: #999; background: #0f0f0f; }
.dark .footer-logo, .dark .public-footer-column h2, .dark .public-footer-column a:hover { color: #f5f5f5; }
.dark .public-footer-bottom { border-color: #262626; color: #808080; }
@media (max-width: 640px) { .public-footer-main { gap: 30px; padding: 42px 24px 32px; } .public-footer-brand { flex-basis: 100%; max-width: none; } .public-footer-column { flex: 1 1 30%; min-width: 80px; } .public-footer-bottom { flex-direction: column; gap: 8px; padding: 20px 24px; } }

.public-footer.public-footer-home {
  --footer-background: #f5f7fc;
  --footer-line: #e0e6f2;
  --footer-heading: #151d2d;
  --footer-text: #546179;
  --footer-accent: #4236cd;
  --footer-gutter: 58px;
  border-color: var(--footer-line);
  background: var(--footer-background);
  color: var(--footer-text);
}
.public-footer-home .public-footer-main {
  display: grid;
  grid-template-columns: minmax(0, 2fr) repeat(var(--footer-columns, 3), minmax(0, 1fr));
  gap: 26px;
  max-width: 1536px;
  padding: 45px var(--footer-gutter) 34px;
}
.public-footer-home .footer-logo { font-size: 20px; }
.public-footer-home .footer-logo,
.public-footer-home .public-footer-column h2 { color: var(--footer-heading); }
.public-footer-home .public-footer-column { gap: 10px; min-width: 0; }
.public-footer-home .public-footer-column h2 { margin-bottom: 4px; font-size: 14px; }
.public-footer-home .public-footer-column a:hover { color: var(--footer-accent); text-decoration: underline; text-underline-offset: 3px; }
.public-footer-home .public-footer-brand p { margin-top: 12px; }
.public-footer-home .public-footer-brand, .public-footer-home .public-footer-column { max-width: none; overflow-wrap: anywhere; }
.public-footer-home .public-footer-bottom {
  width: calc(100% - 2 * var(--footer-gutter));
  max-width: calc(1536px - 2 * var(--footer-gutter));
  padding: 17px 0 22px;
  border-color: var(--footer-line);
  color: var(--footer-text);
  font-size: 12px;
  overflow-wrap: anywhere;
}
.dark .public-footer.public-footer-home {
  --footer-background: #151b2b;
  --footer-line: #303a52;
  --footer-heading: #f3f5ff;
  --footer-text: #abb8d0;
  --footer-accent: #c0c8ff;
}
@media (max-width: 1050px) { .public-footer.public-footer-home { --footer-gutter: 32px; } }
@media (max-width: 760px) {
  .public-footer.public-footer-home { --footer-gutter: 22px; }
  .public-footer-home .public-footer-main { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 24px 18px; padding-block: 32px 24px; }
  .public-footer-home .public-footer-brand { grid-column: 1 / -1; }
}
</style>
