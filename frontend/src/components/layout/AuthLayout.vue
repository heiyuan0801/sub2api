<template>
  <SiteLayout surface="canvas" class="auth-page">
    <template #header>
      <SiteHeader plain class="border-b border-border/60">
        <div class="mx-auto flex min-h-14 max-w-[1536px] items-center justify-between gap-3 px-4 sm:px-7">
          <SiteBrand :name="siteName" :logo="siteLogo" />
          <SiteThemeToggle />
        </div>
      </SiteHeader>
    </template>

    <div class="auth-shell">
      <div class="auth-layout" :class="`auth-layout--${variant}`">
        <aside v-if="variant !== 'compact'" class="auth-aside" :class="{ 'auth-aside--intro': !$slots.providers }">
          <div v-if="$slots.providers" class="auth-provider-panel">
            <h2>{{ t('publicSite.auth.thirdPartyTitle') }}</h2>
            <div class="auth-social-providers"><slot name="providers" /></div>
            <p class="auth-provider-hint">{{ t('publicSite.auth.thirdPartyHint') }}</p>
          </div>
          <div v-else class="auth-intro">
            <div class="auth-intro-symbol" aria-hidden="true">
              <MailCheck v-if="variant === 'recovery'" :size="42" :stroke-width="1.4" />
              <img v-else :src="siteLogo || '/logo.svg'" alt="" />
            </div>
            <h2>{{ variant === 'recovery' ? t('publicSite.auth.recoveryTitle') : siteName }}</h2>
            <p>{{ t(variant === 'recovery' ? 'publicSite.auth.recoveryDescription' : 'publicSite.auth.welcomeDescription') }}</p>
            <router-link to="/home" class="auth-home-link"><ArrowLeft :size="15" />{{ t('design.backHome') }}</router-link>
          </div>
        </aside>
        <main class="auth-main">
          <div class="auth-card" :class="`auth-card--${variant}`">
            <slot />
            <div v-if="$slots.footer" class="auth-footer"><slot name="footer" /></div>
          </div>
        </main>
      </div>
    </div>

    <template #footer>
      <SiteFooter plain class="px-4 pb-6 pt-1 text-center sm:pb-8">
        <RouterLink to="/home" class="inline-flex items-center gap-1.5 rounded-sm text-sm hover:text-foreground">
          {{ t('design.backHome') }}<ArrowUpRight :size="14" aria-hidden="true" />
        </RouterLink>
      </SiteFooter>
    </template>
  </SiteLayout>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowLeft, ArrowUpRight, MailCheck } from '@lucide/vue'
import SiteBrand from '@/components/site-shell/SiteBrand.vue'
import SiteFooter from '@/components/site-shell/SiteFooter.vue'
import SiteHeader from '@/components/site-shell/SiteHeader.vue'
import SiteLayout from '@/components/site-shell/SiteLayout.vue'
import SiteThemeToggle from '@/components/site-shell/SiteThemeToggle.vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

withDefaults(defineProps<{
  variant?: 'default' | 'login' | 'register' | 'recovery' | 'compact'
}>(), { variant: 'default' })
const { t } = useI18n()
const appStore = useAppStore()
const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
onMounted(() => { appStore.fetchPublicSettings() })
</script>

<style scoped>
.auth-page {
  --auth-card-background: hsl(var(--card));
  --auth-text: hsl(var(--foreground));
  --auth-muted: hsl(var(--muted-foreground));
  --auth-border: hsl(var(--border));
  --auth-input-background: hsl(var(--background));
  --auth-social-background: hsl(var(--secondary));
  --auth-accent: hsl(var(--brand));
}

.auth-shell { display: grid; flex: 1; min-width: 0; place-items: center; padding: 32px 24px; }
.auth-layout {
  display: grid; grid-template-columns: minmax(0, .8fr) minmax(0, 1.2fr); align-items: stretch;
  width: min(100%, 900px); padding: 32px; border: 1px solid var(--auth-border);
  border-radius: 12px; background: var(--auth-card-background);
}
.auth-layout--compact { grid-template-columns: minmax(0, 1fr); max-width: 600px; }
.auth-aside { display: grid; place-items: center; padding-right: 32px; border-right: 1px solid var(--auth-border); }
.auth-main { display: flex; align-items: center; min-width: 0; padding-left: 32px; }
.auth-layout--compact .auth-main { padding-left: 0; }
.auth-card { width: 100%; min-width: 0; }
.auth-intro { max-width: 260px; text-align: left; }
.auth-intro-symbol {
  display: grid; place-items: center; width: 44px; height: 44px; margin-bottom: 16px;
  border: 1px solid var(--auth-border); border-radius: 8px; background: hsl(var(--muted)); color: var(--auth-accent);
}
.auth-intro-symbol img, .auth-intro-symbol svg { width: 24px; height: 24px; object-fit: contain; }
.auth-intro h2, .auth-provider-panel h2 { color: var(--auth-text); font-size: 16px; font-weight: 600; }
.auth-intro p, .auth-provider-hint { color: var(--auth-muted); margin-top: 8px; font-size: 13px; line-height: 1.6; }
.auth-home-link { display: inline-flex; gap: 6px; align-items: center; margin-top: 20px; font-size: 13px; color: var(--auth-accent); }
.auth-provider-panel { width: 100%; }
.auth-provider-panel h2 { margin-bottom: 12px; }
.auth-social-providers, .auth-social-providers :deep(.grid) { display: grid; grid-template-columns: minmax(0, 1fr); gap: 8px; }
.auth-social-providers :deep(.btn) {
  min-height: 40px; padding: 8px 12px; border: 1px solid var(--auth-border);
  border-radius: 6px; background: var(--auth-social-background); color: var(--auth-text);
  font-size: 14px; font-weight: 500; line-height: 1.35; box-shadow: none; gap: 8px;
}
.auth-social-providers :deep(.btn:hover:not(:disabled)) { background: hsl(var(--accent)); }
.auth-social-providers :deep(svg) { width: 18px; height: 18px; flex-shrink: 0; }

.auth-card :deep(.input-label) { margin: 0 0 6px; color: var(--auth-text); font-size: 13px; font-weight: 500; }
.auth-card :deep(:is(.input, [data-slot="input"])) {
  width: 100%; min-height: 42px; height: 42px; padding: 9px 12px;
  border: 1px solid var(--auth-border); border-radius: 6px; background: var(--auth-input-background);
  color: var(--auth-text); font-size: 16px; line-height: 22px; box-shadow: none;
}
.auth-card :deep(.input.pr-11), .auth-card :deep(.input.pr-10) { padding-right: 44px; }
.auth-card :deep(.auth-field .input) { padding-left: 40px; }
.auth-card :deep(.auth-password-field.has-recovery .input) { padding-right: 145px; }
.auth-card :deep(.auth-field-icon) {
  pointer-events: none; position: absolute; left: 12px; top: 50%; transform: translateY(-50%);
  width: 16px; height: 16px; color: var(--auth-muted);
}
.auth-card :deep(.input::placeholder) { color: var(--auth-muted); }
.auth-card :deep(.input:focus) { border-color: var(--auth-accent); outline: none; box-shadow: 0 0 0 2px hsl(var(--brand) / .18); }
.auth-card :deep(.input:disabled) { opacity: .65; }
.auth-card :deep(.input.input-error), .auth-card :deep(.input.border-danger) { border-color: hsl(var(--danger)); }
.auth-card :deep(.input.border-success) { border-color: hsl(var(--success)); }
.auth-card :deep(.input-hint) { color: var(--auth-muted); font-size: 12px; line-height: 1.6; }
.auth-card :deep(:is(.btn, [data-slot="button"])) {
  min-height: 40px; height: auto; padding: 8px 12px; border-radius: 6px; font-size: 14px;
  font-weight: 500; line-height: 20px; box-shadow: none; white-space: normal;
}
.auth-card :deep(:is(.btn-primary, [data-variant="default"])) {
  border: 1px solid transparent; background: var(--auth-accent); color: hsl(var(--brand-foreground)); font-weight: 600;
}
.auth-card :deep(:is(.btn-primary, [data-variant="default"]):hover:not(:disabled)) { background: hsl(var(--brand) / .9); }
.auth-card :deep(:is(.btn-secondary, [data-variant="outline"])) {
  border: 1px solid var(--auth-border); background: var(--auth-card-background); color: var(--auth-text);
}
.auth-card :deep(.auth-separator) { display: flex; align-items: center; gap: 12px; margin-top: 16px; color: var(--auth-muted); font-size: 12px; }
.auth-card :deep(.auth-separator::before), .auth-card :deep(.auth-separator::after) { content: ''; height: 1px; flex: 1; background: var(--auth-border); }
.auth-footer { margin-top: 20px; padding-top: 16px; border-top: 1px solid var(--auth-border); color: var(--auth-muted); font-size: 13px; line-height: 1.6; }
.auth-footer :deep(a) { color: var(--auth-accent); font-weight: 500; }
.auth-footer :deep(a:hover) { text-decoration: underline; text-underline-offset: 3px; }
.auth-card :deep(.auth-footer-row) { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 8px 16px; }
.auth-card :deep(.auth-footer-centered) { text-align: center; }
.auth-card :deep(#login-agreement-consent) { accent-color: var(--auth-accent); color: var(--auth-accent); }
.auth-card :deep(.auth-security-note) {
  display: flex; align-items: center; justify-content: center; gap: 6px; margin-top: 20px;
  color: var(--auth-muted); font-size: 12px; line-height: 1.5;
}
.auth-card :deep(.auth-actions) { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
.auth-card :deep(.auth-secondary-action) {
  display: flex; align-items: center; justify-content: center; min-height: 40px; padding: 8px 12px;
  border: 1px solid var(--auth-border); border-radius: 6px; color: var(--auth-text);
  font-size: 14px; text-align: center;
}
.auth-card :deep(a.auth-secondary-action:hover) { color: var(--auth-accent); border-color: var(--auth-accent); }
.auth-card :deep(.auth-secondary-action[aria-disabled="true"]) { color: var(--auth-muted); opacity: .65; }
.auth-page :deep(a:focus-visible), .auth-page :deep(button:focus-visible) { outline: 2px solid var(--auth-accent); outline-offset: 2px; }

@media (max-width: 850px) {
  .auth-card :deep(.auth-password-field.has-recovery .input) { padding-right: 44px; }
  .auth-layout { padding: 24px; }
  .auth-aside { padding-right: 24px; }
  .auth-main { padding-left: 24px; }
}
@media (max-width: 640px) {
  .auth-shell { padding: 16px 12px; }
  .auth-layout { display: flex; flex-direction: column; padding: 20px; max-width: 520px; }
  .auth-main { order: 0; padding: 0; }
  .auth-aside { order: 1; margin-top: 20px; padding: 20px 0 0; border-right: 0; border-top: 1px solid var(--auth-border); }
  .auth-aside--intro { display: none; }
}
@media (prefers-reduced-motion: reduce) { .auth-page :deep(*) { transition: none !important; } }
</style>
