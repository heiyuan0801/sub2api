<template>
  <div class="model-overview">
    <header class="overview-heading">
      <h3>{{ t('publicSite.modelDetail.overviewTitle') }}</h3>
      <p>{{ t('publicSite.modelDetail.overviewIntro') }}</p>
    </header>

    <div v-if="availableGroups.length" class="group-picker">
      <p class="section-label">{{ t('publicSite.modelDetail.availableGroups') }}</p>
      <div class="group-options" role="group" :aria-label="t('publicSite.modelDetail.availableGroups')">
        <button
          v-for="group in availableGroups"
          :key="group.id"
          type="button"
          class="group-option"
          :aria-pressed="selectedGroup?.id === group.id"
          @click="selectedGroupId = group.id"
        >
          <span class="selection-dot" aria-hidden="true" />
          {{ group.name }}
          <GroupDisplayTags :tags="group.display_tags" :custom-tags="group.custom_display_tags" />
        </button>
      </div>
    </div>

    <template v-if="selectedGroup && selectedModel && prices">
      <section class="price-summary" :aria-label="t('publicSite.modelDetail.pricingSummary')">
        <div class="price-heading">
          <h4>{{ t('publicSite.modelDetail.pricingSummary') }}</h4>
          <span class="billing-label">{{ billingLabel }}</span>
        </div>
        <dl v-if="selectedModel.pricing" class="price-columns" :class="{ 'single-price': prices.mode !== BILLING_MODE_TOKEN }">
          <template v-if="prices.mode === BILLING_MODE_TOKEN">
            <div v-for="item in tokenPrices" :key="item.key" class="price-item">
              <dt>{{ item.label }}</dt>
              <dd>
                <strong>{{ formatCatalogPrice(item.value, 1_000_000) }}</strong>
                <span>{{ t('publicSite.modelDetail.perMillionTokens') }}</span>
              </dd>
            </div>
          </template>
          <div v-else class="price-item">
            <dt>{{ t('publicSite.modelDetail.requestUnitPrice') }}</dt>
            <dd>
              <strong>{{ formatCatalogPrice(prices.request) }}</strong>
              <span>{{ t(prices.mode === BILLING_MODE_IMAGE ? 'modelPlaza.table.perUnitImage' : 'modelPlaza.table.perUnitRequest') }}</span>
            </dd>
          </div>
        </dl>
        <p v-else class="no-pricing">{{ t('publicSite.modelDetail.pricingUnavailable') }}</p>
        <p class="price-note">{{ t('publicSite.modelDetail.pricingSummaryHint') }}</p>
      </section>

      <section v-if="selectedGroup.description || selectedGroup.is_exclusive || selectedGroup.subscription_type === 'subscription'" class="group-information">
        <h4>{{ t('publicSite.modelDetail.groupInformation') }}</h4>
        <p v-if="selectedGroup.description" class="group-description">{{ selectedGroup.description }}</p>
        <div v-if="selectedGroup.is_exclusive || selectedGroup.subscription_type === 'subscription'" class="group-flags">
          <span v-if="selectedGroup.is_exclusive"><LockKeyhole :size="13" aria-hidden="true" />{{ t('modelPlaza.badges.exclusive') }}</span>
          <span v-if="selectedGroup.subscription_type === 'subscription'"><Ticket :size="13" aria-hidden="true" />{{ t('modelPlaza.badges.subscription') }}</span>
        </div>
      </section>

      <section class="full-pricing" :class="{ 'is-open': fullPricingOpen }">
        <button type="button" class="pricing-toggle" :aria-expanded="fullPricingOpen" :aria-controls="pricingPanelId" @click="fullPricingOpen = !fullPricingOpen">
          <span><ListFilter :size="17" aria-hidden="true" />{{ t('publicSite.modelDetail.fullPricing') }}</span>
          <span class="expand-icon"><ChevronDown :size="17" class="details-chevron" aria-hidden="true" /></span>
        </button>
        <Transition name="pricing-expand">
          <div v-show="fullPricingOpen" :id="pricingPanelId" class="pricing-panel" :inert="!fullPricingOpen">
            <div class="pricing-panel-inner">
              <div class="full-pricing-content">
                <PlazaGroupSection :group="selectedGroup" prices-only />
              </div>
            </div>
          </div>
        </Transition>
      </section>
    </template>
    <p v-else class="empty-overview">{{ t('modelPlaza.detail.noModels') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import GroupDisplayTags from '@/components/common/GroupDisplayTags.vue'
import { ChevronDown, ListFilter, LockKeyhole, Ticket } from '@lucide/vue'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'
import { BILLING_MODE_IMAGE, BILLING_MODE_TOKEN, BILLING_MODE_VIDEO } from '@/constants/channel'
import PlazaGroupSection from './PlazaGroupSection.vue'
import { catalogPrices, formatCatalogPrice, groupsForPlazaModel } from './catalogPricing'

const props = defineProps<{ model: PlazaModel; groups: ModelPlazaGroup[]; initialGroupId?: number }>()
const { t } = useI18n()
const pricingPanelId = useId()
const fullPricingOpen = ref(false)
const selectedGroupId = ref(props.initialGroupId)
const availableGroups = computed(() => groupsForPlazaModel(props.groups, props.model))
const selectedGroup = computed(() => availableGroups.value.find(group => group.id === selectedGroupId.value) ?? availableGroups.value[0])
watch(() => selectedGroup.value?.id, () => { fullPricingOpen.value = false })
// Each group's matching model carries its own price, intervals and timed rules.
const selectedModel = computed(() => selectedGroup.value?.models[0])
const prices = computed(() => selectedGroup.value && selectedModel.value ? catalogPrices(selectedModel.value, selectedGroup.value) : null)
const tokenPrices = computed(() => [
  { key: 'input', label: t('publicSite.market.input'), value: prices.value?.input ?? null },
  { key: 'output', label: t('publicSite.market.output'), value: prices.value?.output ?? null },
  { key: 'cache', label: t('publicSite.market.cache'), value: prices.value?.cache ?? null },
])
const billingLabel = computed(() => {
  switch (prices.value?.mode) {
    case BILLING_MODE_IMAGE: return t('publicSite.models.imageBilling')
    case BILLING_MODE_VIDEO: return t('publicSite.models.videoBilling')
    case BILLING_MODE_TOKEN: return t('publicSite.models.tokenBilling')
    default: return t('publicSite.models.requestBilling')
  }
})
</script>

<style scoped>
.model-overview { color: var(--detail-ink); }
.overview-heading { margin-bottom: 25px; }
.overview-heading h3 { margin: 0; font-size: 1.5rem; line-height: 1.25; font-weight: 650; letter-spacing: -.045em; }
.overview-heading p { margin-top: 8px; color: var(--detail-muted); font-size: .82rem; line-height: 1.7; }
.group-picker { margin-bottom: 27px; }
.section-label { margin-bottom: 11px; color: var(--detail-muted); font-size: .73rem; font-weight: 500; }
.group-options { display: flex; flex-wrap: wrap; gap: 7px; }
.group-option { display: flex; align-items: center; gap: 7px; min-height: 40px; max-width: 100%; padding: 8px 15px; border: 1px solid var(--detail-border); border-radius: 24px; background: var(--detail-card); font-size: .76rem; line-height: 1.5; text-align: left; overflow-wrap: anywhere; transition: background-color 200ms, color 200ms, border-color 200ms, transform 200ms; }
.group-option:hover { background: var(--detail-accent-soft); transform: translateY(-1px); }
.group-option:active { transform: translateY(0); }
.group-option[aria-pressed='true'] { border-color: var(--detail-ink); background: var(--detail-ink); color: var(--detail-bg); font-weight: 550; }
.selection-dot { flex: none; width: 5px; height: 5px; border-radius: 50%; background: currentColor; opacity: .35; }
.group-option[aria-pressed='true'] .selection-dot { opacity: 1; }
.price-heading { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; margin-bottom: 12px; }
.price-heading h4, .group-information h4 { font-size: .83rem; font-weight: 600; }
.billing-label { padding: 3px 8px; border-radius: 7px; background: var(--detail-accent-soft); color: var(--detail-muted); font-size: .67rem; }
.price-columns { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.price-item { min-width: 0; padding: 24px 21px 22px; border: 1px solid var(--detail-border); border-radius: 22px; background: var(--detail-card); box-shadow: 0 2px 3px rgb(0 0 0 / 1%); }
.price-item dt { margin-bottom: 22px; color: var(--detail-muted); font-size: .74rem; }
.price-item dd { display: flex; flex-direction: column; gap: 10px; }
.price-item strong { font-size: clamp(1.65rem, 3vw, 2.25rem); line-height: 1.15; font-weight: 550; letter-spacing: -.055em; font-variant-numeric: tabular-nums; overflow-wrap: anywhere; }
.price-item dd > span { color: var(--detail-muted); font-size: .69rem; }
.single-price { grid-template-columns: minmax(0, 1fr); }
.single-price .price-item { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); align-items: center; }
.single-price .price-item dt { margin: 0; }
.single-price .price-item dd { text-align: right; }
.price-note { margin-top: 12px; color: var(--detail-muted); font-size: .71rem; line-height: 1.8; }
.no-pricing { padding: 32px 22px; border: 1px solid var(--detail-border); border-radius: 22px; background: var(--detail-card); color: var(--detail-muted); font-size: .88rem; }
.group-information { margin-top: 22px; padding: 20px 22px; border-radius: 18px; background: var(--detail-accent-soft); }
.group-description { margin-top: 8px; font-size: .78rem; line-height: 1.9; color: var(--detail-muted); white-space: pre-line; overflow-wrap: anywhere; }
.group-flags { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 12px; }
.group-flags span { display: inline-flex; align-items: center; gap: 5px; padding: 4px 9px; border-radius: 20px; background: var(--detail-card); font-size: .69rem; }
.full-pricing { margin-top: 16px; border: 1px solid var(--detail-border); border-radius: 20px; background: var(--detail-card); }
.pricing-toggle { display: flex; justify-content: space-between; align-items: center; gap: 12px; width: 100%; min-height: 68px; padding: 14px 20px; border-radius: inherit; cursor: pointer; font-size: .8rem; font-weight: 550; text-align: left; }
.pricing-toggle > span { display: flex; align-items: center; gap: 9px; }
.expand-icon { flex: none; justify-content: center; width: 30px; height: 30px; border-radius: 50%; background: var(--detail-subtle); transition: background-color 180ms; }
.pricing-toggle:hover .expand-icon { background: var(--detail-accent-soft); }
.details-chevron { transition: transform 260ms cubic-bezier(.22, 1, .36, 1); }
.is-open .details-chevron { transform: rotate(180deg); }
.pricing-panel { display: grid; grid-template-rows: 1fr; }
.pricing-panel-inner { min-height: 0; overflow: hidden; }
.pricing-expand-enter-active, .pricing-expand-leave-active { transition: grid-template-rows 320ms cubic-bezier(.22, 1, .36, 1), opacity 220ms ease; }
.pricing-expand-enter-from, .pricing-expand-leave-to { grid-template-rows: 0fr; opacity: 0; }
.full-pricing-content { padding: 0 15px 15px; }
.full-pricing-content :deep(section) { border-radius: 13px; border-color: var(--detail-border); background: var(--detail-card); box-shadow: none; }
.full-pricing-content :deep(.plaza-pricing-table) { --plaza-accent: var(--detail-ink) !important; --pz-bg: transparent; --pz-bg-hover: var(--detail-subtle); }
.empty-overview { padding: 36px 22px; border-radius: 22px; background: var(--detail-accent-soft); color: var(--detail-muted); font-size: .86rem; }
.group-option:focus-visible, .pricing-toggle:focus-visible { outline: 2px solid var(--detail-ink); outline-offset: 3px; }
@media (max-width: 600px) {
  .overview-heading h3 { font-size: 1.4rem; }
  .group-option { min-height: 44px; }
  .price-columns { gap: 9px; grid-template-columns: minmax(0, 1fr); }
  .price-item { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); align-items: center; padding: 20px; border-radius: 18px; }
  .price-item dt { margin-bottom: 0; }
  .price-item dd { text-align: right; }
  .price-item strong { font-size: 1.9rem; }
  .group-information { padding: 18px; }
  .pricing-toggle { padding-inline: 17px; }
  .full-pricing-content { padding: 0 8px 8px; }
}
@media (prefers-reduced-motion: reduce) {
  .group-option, .details-chevron, .expand-icon, .pricing-expand-enter-active, .pricing-expand-leave-active { transition: none; }
  .group-option:hover { transform: none; }
}
</style>
