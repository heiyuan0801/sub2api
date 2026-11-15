<template>
  <dialog
    ref="dialog"
    class="model-detail"
    aria-labelledby="plaza-model-detail-title"
    @cancel.prevent="emit('close')"
    @pointerdown="startedOnBackdrop = isBackdrop($event)"
    @click="closeBackdrop"
  >
    <button type="button" class="model-close" :aria-label="t('publicSite.models.closeDetails')" autofocus @click="emit('close')">
      <X :size="19" aria-hidden="true" />
    </button>
    <Tabs v-model="activeTab" class="model-tabs" activation-mode="manual">
      <div class="model-topbar">
        <header class="model-header">
          <div class="model-brand">
            <PlatformIcon :platform="model.platform as GroupPlatform" size="lg" aria-hidden="true" />
          </div>
          <div class="model-identity">
            <div class="model-heading">
              <h2 id="plaza-model-detail-title">{{ model.name }}</h2>
              <button type="button" class="model-copy" :class="{ 'is-copied': copiedKey === 'name' }" :aria-label="copiedKey === 'name' ? t('common.copied') : t('publicSite.models.copyName')" @click="copyValue(model.name, 'name')">
                <Transition name="copy-icon" mode="out-in">
                  <Check v-if="copiedKey === 'name'" :size="17" aria-hidden="true" />
                  <Copy v-else :size="17" aria-hidden="true" />
                </Transition>
              </button>
            </div>
            <div class="model-meta">
              <span>{{ platformLabel(model.platform) }}</span>
              <span class="meta-dot" aria-hidden="true" />
              <span class="model-billing" :class="{ 'dynamic-price': dynamicPricing }">{{ billingLabel }}</span>
              <span class="model-group-count">{{ t('publicSite.modelDetail.availableGroups') }} <strong>{{ groups.length }}</strong></span>
            </div>
          </div>
        </header>
        <div class="model-navigation">
          <TabsList class="model-tab-list" :style="{ '--active-tab': ['overview', 'performance', 'api'].indexOf(activeTab) }" :aria-label="t('publicSite.models.modelDetails')">
            <span class="model-tab-pill" aria-hidden="true" />
            <TabsTrigger value="overview"><Info aria-hidden="true" />{{ t('publicSite.modelDetail.overview') }}</TabsTrigger>
            <TabsTrigger value="performance"><HeartPulse aria-hidden="true" />{{ t('publicSite.modelDetail.performance') }}</TabsTrigger>
            <TabsTrigger value="api"><CodeXml aria-hidden="true" />{{ t('publicSite.modelDetail.api') }}</TabsTrigger>
          </TabsList>
          <RouterLink :to="apiDocsPath" class="model-docs" @click="emit('close')">{{ t('publicSite.nav.apiDocs') }}<ArrowUpRight :size="15" aria-hidden="true" /></RouterLink>
        </div>
      </div>
      <TabsContent value="overview" class="model-tab-panel">
        <PlazaModelOverview :model="model" :groups="groups" :initial-group-id="initialGroupId" />
      </TabsContent>
      <TabsContent value="performance" class="model-tab-panel">
        <PlazaModelPerformance v-if="activeTab === 'performance'" :model="model.name" :platform="model.platform" />
      </TabsContent>
      <TabsContent value="api" class="model-tab-panel">
        <header class="detail-page-heading">
          <h3>{{ t('publicSite.modelDetail.api') }}</h3>
          <p>{{ t('publicSite.modelDetail.apiDescription') }}</p>
        </header>
        <dl class="api-fields">
          <div v-for="field in apiFields" :key="field.key" class="api-field">
            <dt>{{ field.label }}</dt>
            <dd>
              <code>{{ field.value }}</code>
              <button type="button" class="model-copy" :aria-label="t('publicSite.modelDetail.copy', { item: field.label })" @click="copyValue(field.value, field.key)">
                <Check v-if="copiedKey === field.key" :size="16" aria-hidden="true" />
                <Copy v-else :size="16" aria-hidden="true" />
              </button>
            </dd>
          </div>
        </dl>
        <RouterLink :to="apiDocsPath" class="api-guide" @click="emit('close')">{{ t('publicSite.nav.apiDocs') }} <ArrowUpRight :size="16" aria-hidden="true" /></RouterLink>
      </TabsContent>
    </Tabs>
  </dialog>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, Check, CodeXml, Copy, HeartPulse, Info, X } from '@lucide/vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { buildGatewayUrl } from '@/api/url'
import { localizedPublicPath } from '@/i18n/localeConfig'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'
import { useClipboard } from '@/composables/useClipboard'
import type { GroupPlatform } from '@/types'
import { platformLabel } from '@/utils/platformColors'
import PlazaModelOverview from './PlazaModelOverview.vue'

const PlazaModelPerformance = defineAsyncComponent(() => import('./PlazaModelPerformance.vue'))
const props = defineProps<{ model: PlazaModel; groups: ModelPlazaGroup[]; initialGroupId?: number }>()
const emit = defineEmits<{ close: [] }>()
const { t, locale } = useI18n()
const apiDocsPath = computed(() => localizedPublicPath('/docs/api', locale.value))
const { copyToClipboard } = useClipboard()
const dialog = ref<HTMLDialogElement | null>(null)
const activeTab = ref('overview')
const copiedKey = ref<string | null>(null)
let copyReset: ReturnType<typeof setTimeout> | undefined
// Keep in a local ref so the native dialog can prevent page scrolling and restore it on teardown.
let previousOverflow = ''
let startedOnBackdrop = false

const dynamicPricing = computed(() => props.groups.some(group =>
  group.peak_rate_enabled || group.models.some(model => Boolean(model.time_pricing?.periods.length)),
))
const billingLabel = computed(() => {
  if (dynamicPricing.value) return t('publicSite.modelDetail.dynamicBilling')
  switch (props.model.pricing?.billing_mode) {
    case 'image': return t('publicSite.models.imageBilling')
    case 'video': return t('publicSite.models.videoBilling')
    case 'per_request': return t('publicSite.models.requestBilling')
    default: return t('publicSite.models.tokenBilling')
  }
})
const apiFields = computed(() => [
  { key: 'model', label: t('publicSite.modelDetail.modelId'), value: props.model.name },
  { key: 'base', label: t('publicSite.modelDetail.apiBase'), value: buildGatewayUrl('/v1') },
])

async function copyValue(value: string, key: string) {
  if (!await copyToClipboard(value)) return
  if (copyReset) clearTimeout(copyReset)
  copiedKey.value = key
  copyReset = setTimeout(() => { copiedKey.value = null }, 2000)
}

function isBackdrop(event: MouseEvent) {
  const element = dialog.value
  if (!element || event.target !== element) return false
  const bounds = element.getBoundingClientRect()
  return event.clientX < bounds.left || event.clientX > bounds.right || event.clientY < bounds.top || event.clientY > bounds.bottom
}

function closeBackdrop(event: MouseEvent) {
  if (startedOnBackdrop && isBackdrop(event)) emit('close')
  startedOnBackdrop = false
}

onMounted(() => {
  dialog.value?.showModal()
  previousOverflow = document.documentElement.style.overflow
  document.documentElement.style.overflow = 'hidden'
})
onBeforeUnmount(() => {
  if (copyReset) clearTimeout(copyReset)
  if (dialog.value?.open) dialog.value.close()
  document.documentElement.style.overflow = previousOverflow
})
</script>

<style scoped>
/* Motion references: DesignBookmark Tab pill and Stat card (https://designbookmark.com/motion).
   Independent Vue implementation: a shared sliding capsule, soft surfaces and short spring transitions. */
.model-detail {
  --detail-bg: #fff;
  --detail-card: #fff;
  --detail-ink: #171717;
  --detail-muted: #747474;
  --detail-border: #ececec;
  --detail-subtle: #f8f8f7;
  --detail-accent: #171717;
  --detail-accent-soft: #f3f3f2;
  --detail-success: #168252;
  --detail-warning: #a46912;
  --detail-warning-soft: #fff5e5;
  --detail-danger: #ca3d50;
  --detail-danger-soft: #fff0f2;
  --detail-gutter: clamp(20px, 3vw, 36px);
  position: fixed;
  inset: clamp(12px, 4dvh, 36px) 0 auto;
  width: min(calc(100% - 40px), 1180px);
  height: min(920px, calc(100dvh - 2 * clamp(12px, 4dvh, 36px)));
  max-height: none;
  margin: 0 auto;
  padding: 0;
  overflow: hidden;
  border: none;
  border-radius: 28px;
  background: var(--detail-bg);
  color: var(--detail-ink);
  box-shadow: 0 30px 90px rgb(0 0 0 / 18%);
}
.model-detail[open] { display: flex; animation: detail-arrive 360ms cubic-bezier(.22, 1, .36, 1) both; }
.dark .model-detail {
  --detail-bg: #171717;
  --detail-card: #202020;
  --detail-ink: #f5f5f5;
  --detail-muted: #aaa;
  --detail-border: #343434;
  --detail-subtle: #262626;
  --detail-accent: #f5f5f5;
  --detail-accent-soft: #303030;
  --detail-success: #34d17a;
  --detail-warning: #efb869;
  --detail-warning-soft: #352d20;
  --detail-danger: #ff8796;
  --detail-danger-soft: #392126;
}
.model-detail::backdrop { background: rgb(0 0 0 / 42%); backdrop-filter: blur(4px); }
.model-tabs { display: flex; flex: 1; flex-direction: column; gap: 0; min-width: 0; min-height: 0; }
.model-topbar { flex: none; padding: 30px var(--detail-gutter) 0; background: var(--detail-bg); }
.model-header { display: flex; align-items: center; gap: 16px; padding-inline-end: 42px; }
.model-brand { display: grid; flex: none; width: 58px; height: 58px; place-items: center; border-radius: 19px; background: var(--detail-subtle); }
.model-brand :deep(svg) { width: 34px; height: 34px; }
.model-identity { min-width: 0; }
.model-heading { display: flex; align-items: center; gap: 10px; }
.model-heading h2 { min-width: 0; font-size: clamp(1.25rem, 2.3vw, 1.9rem); font-weight: 650; line-height: 1.3; letter-spacing: -.045em; overflow-wrap: anywhere; }
.model-meta { display: flex; flex-wrap: wrap; gap: 9px; align-items: center; margin-top: 9px; color: var(--detail-muted); font-size: .78rem; }
.meta-dot { width: 3px; height: 3px; background: currentColor; border-radius: 50%; }
.dynamic-price { color: var(--detail-warning); }
.model-group-count { display: inline-flex; gap: 7px; align-items: center; padding-inline-start: 4px; }
.model-group-count strong { padding: 2px 7px; color: var(--detail-ink); background: var(--detail-subtle); border-radius: 999px; font-size: .7rem; }
.model-close, .model-copy { display: grid; flex: none; place-items: center; color: var(--detail-muted); transition: background 180ms, color 180ms, transform 180ms; }
.model-copy { width: 32px; height: 32px; border-radius: 50%; }
.model-close { position: absolute; top: 20px; right: 20px; z-index: 2; width: 38px; height: 38px; border-radius: 50%; background: var(--detail-subtle); }
.model-close:hover, .model-copy:hover { background: var(--detail-accent-soft); color: var(--detail-ink); }
.model-close:active, .model-copy:active { transform: scale(.92); }
.model-copy.is-copied { color: var(--detail-success); }
.model-detail :is(button, a):focus-visible { outline: 2px solid var(--detail-accent); outline-offset: 3px; }
.model-navigation { display: flex; align-items: center; justify-content: space-between; gap: 20px; margin-top: 24px; padding-bottom: 20px; }
.model-tabs .model-tab-list { position: relative; isolation: isolate; display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); flex: 0 1 390px; width: 390px; height: auto; gap: 0; padding: 5px; border-radius: 999px; background: var(--detail-subtle); }
.model-tab-pill { position: absolute; z-index: -1; top: 5px; bottom: 5px; left: 5px; width: calc((100% - 10px) / 3); border-radius: 999px; background: var(--detail-ink); transform: translateX(calc(var(--active-tab) * 100%)); transition: transform 480ms cubic-bezier(.34, 1.38, .64, 1); }
.model-tab-list :deep(button) { position: relative; display: inline-flex; align-items: center; justify-content: center; min-width: 0; width: 100%; height: 40px; gap: 8px; border: 0; border-radius: 999px; padding: 0 12px; background: transparent; color: var(--detail-muted); font-size: .84rem; box-shadow: none; transition: color 180ms, transform 180ms; }
.model-tab-list :deep(button:hover) { color: var(--detail-ink); }
.model-tab-list :deep(button[data-state='active']) { background: transparent; color: var(--detail-bg); font-weight: 600; }
.model-tab-list :deep(button:active) { transform: scale(.96); }
.model-tab-list :deep(button:focus-visible) { outline: 2px solid var(--detail-accent); outline-offset: 3px; box-shadow: none; }
.model-docs { display: inline-flex; align-items: center; gap: 7px; padding: 8px 0 8px 12px; color: var(--detail-muted); font-size: .78rem; white-space: nowrap; }
.model-docs svg { transition: transform 180ms; }
.model-docs:hover { color: var(--detail-ink); }
.model-docs:hover svg { transform: translate(2px, -2px); }
.model-tab-panel { min-width: 0; min-height: 0; padding: 4px var(--detail-gutter) 30px; overflow: auto; overscroll-behavior: contain; scrollbar-gutter: stable; }
.model-tab-panel[hidden] { display: none; }
.model-tab-panel[data-state='active'] { animation: content-arrive 260ms cubic-bezier(.22, 1, .36, 1) both; }
.detail-page-heading { margin-bottom: 24px; }
.detail-page-heading h3 { font-size: 1.25rem; font-weight: 600; line-height: 1.4; letter-spacing: -.025em; }
.detail-page-heading p { margin-top: 6px; color: var(--detail-muted); font-size: .82rem; }
.api-fields { display: grid; gap: 12px; }
.api-field { padding: 22px; border-radius: 20px; background: var(--detail-subtle); }
.api-field dt { color: var(--detail-muted); font-size: .76rem; }
.api-field dd { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-top: 6px; }
.api-field code { overflow-wrap: anywhere; font-size: .92rem; }
.api-guide { display: inline-flex; align-items: center; gap: 8px; margin-top: 22px; padding: 12px 18px; border-radius: 999px; background: var(--detail-ink); color: var(--detail-bg); font-size: .8rem; }
.copy-icon-enter-active, .copy-icon-leave-active { transition: transform 140ms, opacity 140ms; }
.copy-icon-enter-from { opacity: 0; transform: scale(.6) rotate(-12deg); }
.copy-icon-leave-to { opacity: 0; transform: scale(.6) rotate(12deg); }
@keyframes detail-arrive { from { opacity: 0; transform: translateY(14px) scale(.985); } to { opacity: 1; transform: translateY(0) scale(1); } }
@keyframes content-arrive { from { opacity: .3; transform: translateY(7px); } to { opacity: 1; transform: translateY(0); } }
@media (max-width: 600px) {
  .model-detail { inset: max(8px, env(safe-area-inset-top)) 0 0; width: 100%; max-width: none; height: calc(100dvh - max(8px, env(safe-area-inset-top))); border-radius: 22px 22px 0 0; }
  .model-topbar { padding: 22px 18px 0; }
  .model-header { align-items: flex-start; gap: 10px; padding-inline-end: 32px; }
  .model-brand { width: 40px; height: 40px; border-radius: 13px; }
  .model-brand :deep(svg) { width: 27px; height: 27px; }
  .model-heading { gap: 3px; }
  .model-heading h2 { font-size: 1.1rem; }
  .model-copy { width: 27px; height: 27px; }
  .model-meta { margin-top: 6px; gap: 6px; font-size: .7rem; }
  .model-group-count { padding: 0; }
  .model-close { top: 14px; right: 12px; width: 34px; height: 34px; }
  .model-navigation { margin-top: 20px; padding-bottom: 18px; }
  .model-tabs .model-tab-list { flex: 1; width: 100%; }
  .model-tab-list :deep(button) { height: 42px; font-size: .8rem; }
  .model-docs { display: none; }
  .model-tab-panel { padding: 2px 18px calc(24px + env(safe-area-inset-bottom)); }
  .api-field { padding: 18px; }
}
@media (prefers-reduced-motion: reduce) {
  .model-detail[open], .model-tab-panel[data-state='active'] { animation: none; }
  .model-tab-pill, .model-tab-list :deep(button), .model-copy, .model-close, .model-docs svg, .copy-icon-enter-active, .copy-icon-leave-active { transition: none; }
}
</style>
