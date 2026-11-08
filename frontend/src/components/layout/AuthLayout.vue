<template>
  <div class="auth-page">
    <header class="auth-header">
      <router-link to="/home" class="auth-brand" :aria-label="siteName">
        <img :src="siteLogo || '/logo.svg'" :alt="siteName" /><span>{{ siteName }}</span>
      </router-link>
      <Button variant="ghost" size="icon" type="button" class="auth-theme-button"
        :aria-label="isDark ? t('nav.lightMode') : t('nav.darkMode')"
        :title="isDark ? t('nav.lightMode') : t('nav.darkMode')" @click="toggleTheme">
        <Icon :name="isDark ? 'moon' : 'sun'" size="md" />
      </Button>
    </header>
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
      <router-link to="/home" class="auth-bottom-link">{{ t('design.backHome') }}<ArrowUpRight :size="14" /></router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { ArrowLeft, ArrowUpRight, MailCheck } from '@lucide/vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

withDefaults(defineProps<{
  variant?: 'default' | 'login' | 'register' | 'recovery'
}>(), { variant: 'default' })
const { t } = useI18n()
const appStore = useAppStore()
const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const isDark = ref(document.documentElement.classList.contains('dark'))
function toggleTheme(): void {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}
onMounted(() => { appStore.fetchPublicSettings() })
</script>

<style scoped>
.auth-page {
  --auth-background: #edf3fb; --auth-card-background: #fff; --auth-text: #192130;
  --auth-muted: #788191; --auth-border: #e8ebf0; --auth-input-background: #f0f2f6;
  --auth-social-background: #ededf0; --auth-accent: #2860ef;
  min-height: 100dvh; color: var(--auth-text);
  background: radial-gradient(ellipse at 10% 12%, #dceaff 0, transparent 47%), radial-gradient(ellipse at 94% 96%, #e0ecfc 0, transparent 43%), var(--auth-background);
}
:global(.dark .auth-page) {
  --auth-background: #111827; --auth-card-background: #1a2333; --auth-text: #edf2fa;
  --auth-muted: #a4afbf; --auth-border: #303b4d; --auth-input-background: #253043;
  --auth-social-background: #293345; --auth-accent: #709aff;
  background: radial-gradient(ellipse at 10% 12%, #192d4b 0, transparent 47%), var(--auth-background);
}
.auth-header { position: absolute; z-index: 10; inset: 24px 32px auto; display: flex; align-items: center; justify-content: space-between; pointer-events: none; }
.auth-brand, .auth-theme-button { pointer-events: auto; }
.auth-brand { display: inline-flex; gap: 10px; align-items: center; font-size: 16px; font-weight: 600; }
.auth-brand img { width: 28px; height: 28px; object-fit: contain; }
.auth-theme-button { color: var(--auth-muted); border-radius: 50%; }
.auth-shell { min-height: 100dvh; display: flex; flex-direction: column; justify-content: center; align-items: center; padding: 104px 32px 44px; }
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
.auth-bottom-link { display: inline-flex; align-items: center; gap: 6px; margin-top: 30px; color: var(--auth-muted); font-size: 14px; }
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
.auth-card :deep(.input.input-error), .auth-card :deep(.input.border-red-500) { border-color: #ef4444; }
.auth-card :deep(.input.border-green-500) { border-color: #22c55e; }
.auth-card :deep(.input-hint) { color: var(--auth-muted); font-size: 12px; line-height: 1.6; }
.auth-card :deep(:is(.btn, [data-slot="button"])) { min-height: 56px; height: auto; padding: 14px 24px; border-radius: 999px; font-size: 16px; font-weight: 500; line-height: 24px; box-shadow: none; white-space: normal; }
.auth-card :deep(:is(.btn-primary, [data-variant="default"])) { border: 1px solid transparent; background: #2860ef; color: #fff; font-weight: 600; }
.auth-card :deep(:is(.btn-primary, [data-variant="default"]):hover:not(:disabled)) { background: #1e50d8; }
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
  .auth-header { inset: 20px 22px auto; }
  .auth-shell { padding: 88px 20px 28px; }
  .auth-layout { display: flex; flex-direction: column; padding: 32px 24px; min-height: 0; border-radius: 28px; max-width: 520px; }
  .auth-main { order: 0; padding: 0; }
  .auth-aside { order: 1; margin-top: 28px; padding: 26px 0 0; border-right: 0; border-top: 1px solid var(--auth-border); }
  .auth-aside--intro { display: none; }
  .auth-provider-panel h2 { font-size: 14px; }
  .auth-social-providers, .auth-social-providers :deep(.grid) { gap: 12px; }
  .auth-social-providers :deep(.btn) { min-height: 52px; font-size: 15px; }
  .auth-card :deep(.auth-heading h2) { font-size: 25px; }
  .auth-bottom-link { margin-top: 24px; }
}
@media (prefers-reduced-motion: reduce) { .auth-page :deep(*) { transition: none !important; } }
</style>
