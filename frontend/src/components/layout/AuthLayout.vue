<template>
  <SiteLayout surface="plain" class="auth-page">
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
        <aside class="auth-aside" :class="{ 'auth-aside--intro': !$slots.providers }">
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
  variant?: 'default' | 'login' | 'register' | 'recovery'
}>(), { variant: 'default' })
const { t } = useI18n()
const appStore = useAppStore()
const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
onMounted(() => { appStore.fetchPublicSettings() })
</script>

<style scoped>
.auth-page {
  --auth-background: hsl(var(--canvas));
  --auth-card-background: hsl(var(--card));
  --auth-text: hsl(var(--foreground));
  --auth-muted: hsl(var(--muted-foreground));
  --auth-border: hsl(var(--border));
  --auth-input-background: hsl(var(--muted));
  --auth-social-background: hsl(var(--secondary));
  --auth-accent: hsl(var(--brand));
  background: radial-gradient(ellipse at 10% 12%, hsl(var(--brand) / .09) 0, transparent 47%), radial-gradient(ellipse at 94% 96%, hsl(var(--brand) / .06) 0, transparent 43%), var(--auth-background);
}
.auth-shell { flex: 1; min-width: 0; display: flex; flex-direction: column; justify-content: center; align-items: center; padding: 36px 32px; }
.auth-layout { display: grid; grid-template-columns: minmax(0, .86fr) minmax(0, 1.14fr); align-items: stretch; width: min(100%, 1230px); min-height: 600px; padding: 72px; border-radius: 36px; background: var(--auth-card-background); }
.auth-aside { display: flex; align-items: center; justify-content: center; padding-right: 60px; border-right: 1px solid var(--auth-border); }
.auth-provider-panel { width: 100%; text-align: center; }
.auth-provider-panel h2 { font-size: 18px; font-weight: 500; margin-bottom: 18px; }
.auth-provider-hint { font-size: 14px; line-height: 1.7; color: var(--auth-muted); margin-top: 18px; }
.auth-main { display: flex; align-items: center; min-width: 0; padding-left: 60px; }
.auth-card { width: 100%; min-width: 0; }
.auth-intro { text-align: center; max-width: 320px; }
.auth-intro-symbol { display: grid; place-items: center; width: 104px; height: 104px; margin: 0 auto 28px; border-radius: 32px; background: var(--auth-input-background); color: var(--auth-accent); }
.auth-intro-symbol img { width: 52px; height: 52px; object-fit: contain; }
.auth-intro h2 { font-size: 26px; font-weight: 600; letter-spacing: -.6px; }
.auth-intro p { color: var(--auth-muted); margin-top: 16px; font-size: 15px; line-height: 1.9; text-wrap: balance; }
.auth-home-link { display: inline-flex; gap: 6px; align-items: center; margin-top: 32px; font-size: 13px; color: var(--auth-accent); }
.auth-card :deep(.auth-heading) { text-align: left; }
.auth-card :deep(.auth-eyebrow) { color: var(--auth-accent); font-size: 12px; letter-spacing: .08em; }
.auth-card :deep(.auth-heading h2) { margin: 10px 0 0; color: var(--auth-text); font-size: 28px; font-weight: 650; line-height: 1.3; letter-spacing: -.8px; }
.auth-card :deep(.auth-heading p) { margin-top: 10px; color: var(--auth-muted); font-size: 14px; line-height: 1.8; }
.auth-card :deep(.input-label) { margin: 0 0 8px 18px; color: var(--auth-text); font-size: 13px; font-weight: 500; }
.auth-card :deep(:is(.input, [data-slot="input"])) { width: 100%; min-height: 60px; height: 60px; padding: 16px 24px; border: 1px solid transparent; border-radius: 999px; background: var(--auth-input-background); color: var(--auth-text); font-size: 16px; line-height: 24px; box-shadow: none; }
.auth-card :deep(.input.pr-11), .auth-card :deep(.input.pr-10) { padding-right: 52px; }
.auth-card :deep(.auth-field .input) { padding-left: 56px; }
.auth-card :deep(.auth-password-field.has-recovery .input) { padding-right: 164px; }
.auth-card :deep(.auth-field-icon) { pointer-events: none; position: absolute; left: 24px; top: 50%; transform: translateY(-50%); width: 20px; height: 20px; color: var(--auth-muted); }
.auth-card :deep(.input::placeholder) { color: var(--auth-muted); opacity: .8; }
.auth-card :deep(.input:focus) { border-color: var(--auth-accent); outline: none; box-shadow: 0 0 0 3px color-mix(in srgb, var(--auth-accent) 12%, transparent); }
.auth-card :deep(.input:disabled) { opacity: .65; }
.auth-card :deep(.input.input-error), .auth-card :deep(.input.border-red-500) { border-color: hsl(var(--danger)); }
.auth-card :deep(.input.border-green-500) { border-color: hsl(var(--success)); }
.auth-card :deep(.input-hint) { color: var(--auth-muted); font-size: 12px; line-height: 1.6; }
.auth-card :deep(:is(.btn, [data-slot="button"])) { min-height: 56px; height: auto; padding: 14px 24px; border-radius: 999px; font-size: 16px; font-weight: 500; line-height: 24px; box-shadow: none; white-space: normal; }
.auth-card :deep(:is(.btn-primary, [data-variant="default"])) { border: 1px solid transparent; background: var(--auth-accent); color: hsl(var(--brand-foreground)); font-weight: 600; }
.auth-card :deep(:is(.btn-primary, [data-variant="default"]):hover:not(:disabled)) { background: hsl(var(--brand) / .9); }
.auth-card :deep(:is(.btn-secondary, [data-variant="outline"])) { border: 1px solid var(--auth-border); background: var(--auth-card-background); color: var(--auth-text); }
.auth-card :deep(:is(.btn-primary, [data-variant="default"]) > svg:not(.animate-spin)) { display: none; }
.auth-social-providers { display: grid; gap: 16px; }
.auth-social-providers :deep(.grid) { grid-template-columns: minmax(0, 1fr); gap: 16px; }
.auth-social-providers :deep(.btn) { min-height: 60px; padding: 14px 20px; border: 0; border-radius: 999px; background: var(--auth-social-background); color: var(--auth-text); font-size: 16px; font-weight: 500; box-shadow: none; gap: 12px; }
.auth-social-providers :deep(.btn:hover:not(:disabled)) { background: var(--auth-input-background); }
.auth-social-providers :deep(svg) { width: 24px; height: 24px; flex-shrink: 0; }
.auth-card :deep(.auth-separator) { display: flex; align-items: center; gap: 16px; margin-top: 20px; color: var(--auth-muted); font-size: 12px; }
.auth-card :deep(.auth-separator::before), .auth-card :deep(.auth-separator::after) { content: ''; height: 1px; flex: 1; background: var(--auth-border); }
.auth-footer { padding-top: 26px; color: var(--auth-muted); font-size: 14px; line-height: 1.7; }
.auth-footer :deep(p), .auth-footer :deep(.auth-footer-row) { color: var(--auth-muted); }
.auth-footer :deep(a), .auth-card :deep(a[class*='text-primary-']) { color: var(--auth-accent); font-size: inherit; font-weight: 500; }
.auth-footer :deep(a:hover) { text-decoration: underline; text-underline-offset: 4px; }
.auth-card :deep(.auth-footer-row) { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 8px 16px; }
.auth-card :deep(.auth-footer-centered) { text-align: center; }
.auth-card :deep(#login-agreement-consent) { accent-color: var(--auth-accent); color: var(--auth-accent); }
.auth-card :deep(.auth-security-note) { display: flex; align-items: center; justify-content: center; gap: 6px; margin-top: 28px; color: var(--auth-muted); font-size: 13px; line-height: 1.6; }
.auth-card :deep(.auth-actions) { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.auth-card :deep(.auth-secondary-action) { display: flex; align-items: center; justify-content: center; min-height: 56px; padding: 14px 16px; border: 1px solid var(--auth-border); border-radius: 999px; color: var(--auth-muted); font-size: 16px; text-align: center; }
.auth-card :deep(a.auth-secondary-action:hover) { color: var(--auth-accent); border-color: var(--auth-accent); }
.auth-card :deep(.auth-secondary-action[aria-disabled="true"]) { opacity: .65; }
.auth-page :deep(a:focus-visible), .auth-page :deep(button:focus-visible) { outline: 2px solid var(--auth-accent); outline-offset: 4px; }
@media (min-width: 1500px) {
  .auth-card :deep(:is(.input, [data-slot="input"])) { min-height: 72px; height: 72px; }
  .auth-card :deep(:is(.btn, [data-slot="button"], .auth-secondary-action)) { min-height: 72px; }
  .auth-social-providers :deep(.btn) { min-height: 66px; font-size: 18px; }
}
@media (max-width: 1100px) {
  .auth-card :deep(.auth-password-field.has-recovery .input) { padding-right: 60px; }
  .auth-layout { padding: 48px; min-height: 540px; }
  .auth-aside { padding-right: 32px; }
  .auth-main { padding-left: 32px; }
}
@media (max-width: 760px) {
  .auth-shell { padding: 24px 16px; }
  .auth-layout { display: flex; flex-direction: column; padding: 32px 24px; min-height: 0; border-radius: 28px; max-width: 520px; }
  .auth-main { order: 0; padding: 0; }
  .auth-aside { order: 1; margin-top: 28px; padding: 26px 0 0; border-right: 0; border-top: 1px solid var(--auth-border); }
  .auth-aside--intro { display: none; }
  .auth-provider-panel h2 { font-size: 14px; }
  .auth-social-providers, .auth-social-providers :deep(.grid) { gap: 12px; }
  .auth-social-providers :deep(.btn) { min-height: 52px; font-size: 15px; }
  .auth-card :deep(.auth-heading h2) { font-size: 25px; }
}
@media (prefers-reduced-motion: reduce) { .auth-page :deep(*) { transition: none !important; } }
</style>
