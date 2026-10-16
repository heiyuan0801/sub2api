<template>
  <div class="auth-page">
    <header class="auth-header">
      <router-link to="/home" class="auth-brand" :aria-label="siteName">
        <img :src="siteLogo || '/logo.svg'" :alt="siteName" />
      </router-link>
      <button
        type="button"
        class="auth-theme-button"
        :aria-label="isDark ? t('nav.lightMode') : t('nav.darkMode')"
        :title="isDark ? t('nav.lightMode') : t('nav.darkMode')"
        @click="toggleTheme"
      >
        <Icon :name="isDark ? 'moon' : 'sun'" size="md" />
      </button>
    </header>

    <main class="auth-main">
      <section class="auth-card" :class="`auth-card--${variant}`">
        <slot />
        <div v-if="$slots.footer" class="auth-footer">
          <slot name="footer" />
        </div>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
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
.auth-page {
  --auth-background: #fff;
  --auth-card-background: #fefefe;
  --auth-text: #171717;
  --auth-muted: #8a8a8a;
  --auth-border: #ededed;
  --auth-input-background: #fff;
  min-height: 100dvh;
  background: var(--auth-background);
  color: var(--auth-text);
}

.dark .auth-page {
  --auth-background: #111;
  --auth-card-background: #171717;
  --auth-text: #f5f5f5;
  --auth-muted: #939393;
  --auth-border: #303030;
  --auth-input-background: #1b1b1b;
}

.auth-header {
  position: absolute;
  z-index: 10;
  inset: 20px 32px auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  pointer-events: none;
}

.auth-brand,
.auth-theme-button {
  pointer-events: auto;
}

.auth-brand {
  display: inline-flex;
  align-items: center;
  height: 30px;
  max-width: 140px;
}

.auth-brand img {
  width: auto;
  height: 30px;
  max-width: 140px;
  object-fit: contain;
}

.auth-theme-button {
  display: inline-flex;
  width: 30px;
  height: 30px;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: var(--auth-text);
  transition: background-color 0.15s;
}

.auth-theme-button:hover {
  background: var(--auth-border);
}

.auth-theme-button:focus-visible,
.auth-brand:focus-visible {
  outline: 2px solid var(--auth-text);
  outline-offset: 4px;
}

.auth-main {
  display: flex;
  min-height: 100dvh;
  align-items: center;
  justify-content: center;
  padding: 54px 24px;
}

.auth-card {
  display: flex;
  width: 100%;
  max-width: 506px;
  flex-direction: column;
  padding: 39px 39px 30px;
  border: 1px solid var(--auth-border);
  border-radius: 24px;
  background: var(--auth-card-background);
}

.auth-card--login {
  min-height: 608px;
}

.auth-card--register {
  min-height: 592px;
}

.auth-card--recovery {
  min-height: 364px;
  padding-top: 41px;
}

.auth-card :deep(.auth-heading) {
  text-align: center;
}

.auth-card :deep(.auth-eyebrow) {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 2px 11px;
  border: 1px solid var(--auth-border);
  border-radius: 999px;
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

.auth-card :deep(.input) {
  min-height: 36px;
  height: 36px;
  padding: 7px 15px;
  border: 1px solid var(--auth-border);
  border-radius: 999px;
  background: var(--auth-input-background);
  color: var(--auth-text);
  font-size: 12px;
  line-height: 20px;
  box-shadow: none;
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

.auth-card :deep(.btn) {
  min-height: 36px;
  height: auto;
  padding: 7px 15px;
  border-radius: 999px;
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
