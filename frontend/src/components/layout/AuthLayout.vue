<template>
  <div class="auth-page">
    <header class="auth-header">
      <router-link to="/home" class="auth-brand" :aria-label="siteName">
        <img :src="siteLogo || '/logo.svg'" :alt="siteName" /><span>{{ siteName }}</span>
      </router-link>
      <Button
        variant="ghost" size="icon" type="button"
        class="auth-theme-button"
        :aria-label="isDark ? t('nav.lightMode') : t('nav.darkMode')"
        :title="isDark ? t('nav.lightMode') : t('nav.darkMode')"
        @click="toggleTheme"
      >
        <Icon :name="isDark ? 'moon' : 'sun'" size="md" />
      </Button>
    </header>

    <div class="auth-layout">
      <aside class="auth-story" aria-labelledby="auth-story-title">
        <div class="auth-story-copy">
          <span class="auth-story-eyebrow"><span></span>{{ t('design.authEyebrow') }}</span>
          <h1 id="auth-story-title">{{ t('design.allSystems') }}</h1>
          <p>{{ t('design.authDescription') }}</p>
          <div class="auth-stack" aria-hidden="true">
            <div><Layers :size="18" /><span>{{ t('nav.modelPlaza') }}</span><code>01</code></div>
            <div><KeyRound :size="18" /><span>{{ t('nav.apiKeys') }}</span><code>02</code></div>
            <div><Terminal :size="18" /><span>{{ t('design.workspace') }}</span><code>03</code></div>
          </div>
        </div>
        <router-link to="/home" class="auth-story-link">{{ t('design.backHome') }}<ArrowUpRight :size="16" /></router-link>
      </aside>
      <main class="auth-main">
      <Card class="auth-card" :class="`auth-card--${variant}`">
        <slot />
        <div v-if="$slots.footer" class="auth-footer">
          <slot name="footer" />
        </div>
      </Card>
    </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { ArrowUpRight, Terminal, Layers, KeyRound } from '@lucide/vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

withDefaults(defineProps<{
  variant?: 'default' | 'login' | 'register' | 'recovery'
}>(), {
  variant: 'default'
})

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

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.auth-page { --auth-background: hsl(var(--background)); --auth-card-background: hsl(var(--card)); --auth-text: hsl(var(--foreground)); --auth-muted: hsl(var(--muted-foreground)); --auth-border: hsl(var(--border)); --auth-input-background: hsl(var(--background)); min-height: 100dvh; background: var(--auth-background); color: var(--auth-text); }
.auth-header { position: absolute; z-index: 10; inset: 28px 36px auto; display: flex; align-items: center; justify-content: space-between; pointer-events: none; }
.auth-brand, .auth-theme-button { pointer-events: auto; }
.auth-brand { display: inline-flex; gap: 10px; align-items: center; font-size: 17px; font-weight: 650; letter-spacing: -.4px; }
.auth-brand img { width: 30px; height: 30px; object-fit: contain; }
.auth-layout { display: grid; min-height: 100dvh; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); }
.auth-story { display: flex; flex-direction: column; justify-content: center; padding: 120px clamp(36px, 7vw, 110px) 40px; border-right: 1px solid var(--auth-border); background: hsl(var(--muted) / .55); }
.auth-story-copy { margin: auto 0; max-width: 500px; }
.auth-story-eyebrow { display: inline-flex; align-items: center; gap: 8px; color: var(--auth-muted); font-size: 12px; font-weight: 500; }
.auth-story-eyebrow span { width: 6px; height: 6px; border-radius: 50%; background: var(--auth-text); }
.auth-story h1 { margin-top: 22px; font-size: clamp(30px, 3.4vw, 52px); line-height: 1.2; letter-spacing: -.055em; font-weight: 600; text-wrap: balance; }
.auth-story p { margin-top: 24px; max-width: 390px; font-size: 15px; line-height: 1.9; color: var(--auth-muted); }
.auth-stack { display: grid; gap: 8px; margin-top: 48px; max-width: 360px; }
.auth-stack > div { display: flex; align-items: center; gap: 14px; border: 1px solid var(--auth-border); border-radius: 10px; background: var(--auth-background); padding: 16px 20px; font-size: 13px; box-shadow: 0 2px 5px #00000003; }
.auth-stack code { margin-left: auto; color: var(--auth-muted); font-size: 11px; }
.auth-story-link { display: inline-flex; align-items: center; gap: 8px; width: fit-content; margin-top: 48px; font-size: 13px; color: var(--auth-muted); }
.auth-main { display: flex; align-items: center; justify-content: center; padding: 100px 32px 48px; }
.auth-card { width: 100%; max-width: 430px; display: flex; flex-direction: column; padding: 32px; border: 0; border-radius: 14px; background: var(--auth-card-background); box-shadow: none; }
.auth-card--recovery { padding-top: 32px; }
@media (max-width: 900px) { .auth-layout { grid-template-columns: minmax(0,1fr); } .auth-story { display: none; } .auth-header { inset: 24px; } .auth-card { border: 1px solid var(--auth-border); } }

.auth-card :deep(.auth-heading) {
  text-align: left;
}

.auth-card :deep(.auth-eyebrow) {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 2px 11px;
  border: 1px solid var(--auth-border);
  border-radius: 6px;
  color: var(--auth-muted);
  font-size: 11px;
  font-weight: 500;
  line-height: 16px;
  letter-spacing: 0.16em;
}

.auth-card :deep(.auth-heading h2) {
  margin: 11px 0 0;
  color: var(--auth-text);
  font-size: 32px;
  font-weight: 650;
  line-height: 1.3;
  letter-spacing: -1.1px;
}

.auth-card :deep(.auth-heading p) {
  margin-top: 9px;
  color: var(--auth-muted);
  font-size: 14px;
  line-height: 22px;
}

.auth-card :deep(.input-label) {
  margin-bottom: 7px;
  color: var(--auth-text);
  font-size: 13px;
  font-weight: 550;
  line-height: 20px;
}

.auth-card :deep(:is(.input, [data-slot="input"])) {
  min-height: 42px;
  height: 42px;
  padding: 7px 15px;
  border: 1px solid var(--auth-border);
  border-radius: 6px;
  background: var(--auth-input-background);
  color: var(--auth-text);
  font-size: 14px;
  line-height: 20px;
  box-shadow: 0 1px 2px #00000004;
}

.auth-card :deep(.input.pr-11),
.auth-card :deep(.input.pr-10) {
  padding-right: 44px;
}

.auth-card :deep(.input::placeholder) {
  color: #a1a1a1;
}

.auth-card :deep(.input:focus) {
  border-color: #999;
  outline: none;
  box-shadow: 0 0 0 2px rgb(127 127 127 / 10%);
}

.auth-card :deep(.input:disabled) {
  background: var(--auth-border);
}

.auth-card :deep(.input.input-error),
.auth-card :deep(.input.border-red-500) {
  border-color: #ef4444;
}

.auth-card :deep(.input.border-green-500) {
  border-color: #22c55e;
}

.auth-card :deep(.input-hint) {
  color: var(--auth-muted);
  font-size: 11px;
  line-height: 18px;
}

.auth-card :deep(:is(.btn, [data-slot="button"])) {
  min-height: 36px;
  height: auto;
  padding: 7px 15px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  line-height: 20px;
  box-shadow: none;
}

.auth-card :deep(.btn-primary) {
  border: 1px solid var(--auth-text);
  background: var(--auth-text);
  color: var(--auth-background);
}

.auth-card :deep(.btn-primary:hover:not(:disabled)) {
  opacity: 0.83;
  box-shadow: none;
}

.auth-card :deep(.btn-secondary) {
  border: 1px solid var(--auth-border);
  background: var(--auth-input-background);
  color: var(--auth-text);
}

.auth-card :deep(.btn-secondary:hover:not(:disabled)) {
  background: var(--auth-border);
}

.auth-card :deep(.btn-primary > svg:not(.animate-spin)) {
  display: none;
}

.auth-card :deep(.btn-primary > svg.animate-spin) {
  color: var(--auth-background);
}

.auth-card :deep(.auth-social-providers) {
  display: grid;
  gap: 12px;
}

.auth-card :deep(.auth-social-providers .grid) {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.auth-card :deep(.auth-social-providers .grid-cols-1:not([class*='sm:grid-cols-2'])) {
  grid-template-columns: minmax(0, 1fr);
}

.auth-card :deep(.auth-social-providers svg) {
  width: 15px;
  height: 15px;
}

.auth-card :deep(.auth-separator) {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-top: 20px;
  color: var(--auth-muted);
  font-size: 12px;
  line-height: 20px;
}

.auth-card :deep(.auth-separator::before),
.auth-card :deep(.auth-separator::after) {
  content: '';
  height: 1px;
  flex: 1;
  background: var(--auth-border);
}

.auth-card :deep(.auth-passkey) {
  margin-top: 12px;
}

.auth-footer {
  margin-top: auto;
  padding-top: 23px;
  color: var(--auth-muted);
  font-size: 12px;
  line-height: 20px;
}

.auth-footer :deep(p),
.auth-footer :deep(.auth-footer-row) {
  color: var(--auth-muted);
}

.auth-footer :deep(a) {
  color: var(--auth-text);
  font-size: inherit;
  font-weight: 500;
  text-decoration: underline;
  text-underline-offset: 3px;
}

.auth-footer :deep(a:hover) {
  opacity: 0.65;
}

.auth-card :deep(.auth-footer-row) {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 16px;
}

.auth-card :deep(.auth-footer-row > a:last-child) {
  margin-left: auto;
}

.auth-card :deep(.auth-footer-centered) {
  text-align: center;
}

.auth-card :deep(#login-agreement-consent) {
  accent-color: var(--auth-text);
  color: var(--auth-text);
}

.auth-card :deep(a[class*='text-primary-']) {
  color: var(--auth-text);
}

@media (max-width: 600px) {
  .auth-header {
    inset: 19px 20px auto;
  }

  .auth-main {
    padding: 82px 20px 32px;
  }

  .auth-card {
    padding: 24px;
    border-radius: 22px;
  }

  .auth-card--login,
  .auth-card--register {
    min-height: 0;
  }

  .auth-card :deep(.auth-heading h2) {
    font-size: 29px;
  }
}
</style>
