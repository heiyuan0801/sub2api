<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <div
          class="h-8 w-8 animate-spin rounded-full border-b-2 border-brand"
        ></div>
      </div>

      <!-- Settings Form -->
      <form v-else @submit.prevent="saveSettings" class="space-y-6" novalidate>
        <!-- Tab Navigation -->
        <div class="settings-tabs-shell">
          <nav
            class="settings-tabs-scroll"
            role="tablist"
            :aria-label="t('admin.settings.title')"
          >
            <div class="settings-tabs">
              <button
                v-for="tab in settingsTabs"
                :key="tab.key"
                :id="`settings-tab-${tab.key}`"
                type="button"
                role="tab"
                :aria-selected="activeTab === tab.key"
                :tabindex="activeTab === tab.key ? 0 : -1"
                :class="[
                  'settings-tab',
                  activeTab === tab.key && 'settings-tab-active',
                ]"
                @click="selectSettingsTab(tab.key)"
                @keydown="handleSettingsTabKeydown($event, tab.key)"
              >
                <span class="settings-tab-icon">
                  <Icon :name="tab.icon" size="sm" />
                </span>
                <span class="settings-tab-label">{{
                  t(tab.labelKey)
                }}</span>
              </button>
            </div>
          </nav>
        </div>

        <component
          v-for="(panel, key) in settingsPanels"
          :is="panel"
          :key="key"
          v-show="activeTab === key"
        />

        <!-- Save dock — sticky so long tabs never require scrolling to the end -->
        <div v-show="activeTab !== 'backup'" class="settings-save-dock">
          <button
            type="submit"
            :disabled="saving || loadFailed"
            class="btn btn-primary"
          >
            <svg
              v-if="saving"
              class="h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{
              saving
                ? t("admin.settings.saving")
                : t("admin.settings.saveSettings")
            }}
          </button>
        </div>
      </form>

      <!-- Provider dialogs placed outside the settings form to prevent form submission bubbling -->
      <PaymentProviderDialog
        ref="providerDialogRef"
        :show="showProviderDialog"
        :saving="providerSaving"
        :editing="editingProvider"
        :all-key-options="providerKeyOptions"
        :enabled-key-options="enabledProviderKeyOptions"
        :all-payment-types="allPaymentTypes"
        :redirect-label="t('admin.settings.payment.easypayRedirect')"
        @close="showProviderDialog = false"
        @save="handleSaveProvider"
      />
      <ConfirmDialog
        :show="showDeleteProviderDialog"
        :title="t('admin.settings.payment.deleteProvider')"
        :message="t('admin.settings.payment.deleteProviderConfirm')"
        :confirm-text="t('common.delete')"
        danger
        @confirm="handleDeleteProvider"
        @cancel="showDeleteProviderDialog = false"
      />
      <ConfirmDialog
        :show="affiliateConfirmDialog.show"
        :title="affiliateConfirmDialog.title"
        :message="affiliateConfirmDialog.message"
        :confirm-text="affiliateConfirmDialog.confirmText"
        danger
        @confirm="handleAffiliateConfirm"
        @cancel="cancelAffiliateConfirm"
      />
      <!-- 关闭 step-up 开关等敏感保存操作触发的 TOTP 二次验证 -->
      <TotpStepUpDialog :controller="settingsStepUp" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, inject, provide, ref, watch, type Component } from 'vue'
import { routerKey, type Router } from 'vue-router'
import { getSettingsSection, SETTINGS_SECTIONS, SETTINGS_SECTION_QUERY_KEY, type SettingsSectionKey } from '@/components/admin-shell/settingsSections'
import { settingsContextKey } from './settings/settingsContext'
import { useSettingsForm } from './settings/useSettingsForm'
import AppLayout from '@/components/layout/AppLayout.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import PaymentProviderDialog from '@/components/payment/PaymentProviderDialog.vue'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'
import SecuritySection from './settings/sections/SecuritySection.vue'
import GatewaySection from './settings/sections/GatewaySection.vue'
import UsersSection from './settings/sections/UsersSection.vue'
import GeneralSection from './settings/sections/GeneralSection.vue'
import AgreementSection from './settings/sections/AgreementSection.vue'
import FeaturesSection from './settings/sections/FeaturesSection.vue'
import PaymentSection from './settings/sections/PaymentSection.vue'
import EmailSection from './settings/sections/EmailSection.vue'
import BackupSection from './settings/sections/BackupSection.vue'

const state = useSettingsForm()
provide(settingsContextKey, state)
const {
  affiliateConfirmDialog,
  allPaymentTypes,
  cancelAffiliateConfirm,
  editingProvider,
  enabledProviderKeyOptions,
  handleAffiliateConfirm,
  handleDeleteProvider,
  handleSaveProvider,
  loadFailed,
  loading,
  providerDialogRef,
  providerKeyOptions,
  providerSaving,
  saveSettings,
  saving,
  settingsStepUp,
  showDeleteProviderDialog,
  showProviderDialog,
  t
} = state

const settingsPanels = {
  security: SecuritySection,
  gateway: GatewaySection,
  users: UsersSection,
  general: GeneralSection,
  agreement: AgreementSection,
  features: FeaturesSection,
  payment: PaymentSection,
  email: EmailSection,
  backup: BackupSection
} satisfies Record<SettingsSectionKey, Component>

const router = inject<Router | null>(routerKey, null);
const settingsTabs = SETTINGS_SECTIONS;
const linkedSection = computed(() =>
  getSettingsSection(router?.currentRoute.value.query[SETTINGS_SECTION_QUERY_KEY]),
);
const activeTab = ref<SettingsSectionKey>(linkedSection.value?.key ?? "general");

watch(linkedSection, (section) => {
  if (section) activeTab.value = section.key;
});

const settingsTabKeyboardActions = {
  ArrowLeft: -1,
  ArrowUp: -1,
  ArrowRight: 1,
  ArrowDown: 1,
  Home: "first",
  End: "last",
} as const;

function selectSettingsTab(tab: SettingsSectionKey): void {
  activeTab.value = tab;

  // Tab changes stay local unless a deep link is now stale. Clear that
  // query without adding browser history so the same palette entry can reopen it.
  if (!router || !linkedSection.value || linkedSection.value.key === tab) return;
  const { path, query, hash } = router.currentRoute.value;
  const nextQuery = { ...query };
  delete nextQuery[SETTINGS_SECTION_QUERY_KEY];
  void router.replace({ path, query: nextQuery, hash });
}

function focusSettingsTab(tab: SettingsSectionKey): void {
  window.requestAnimationFrame(() => {
    document.getElementById(`settings-tab-${tab}`)?.focus();
  });
}

function handleSettingsTabKeydown(event: KeyboardEvent, tab: SettingsSectionKey): void {
  const action =
    settingsTabKeyboardActions[
      event.key as keyof typeof settingsTabKeyboardActions
    ];
  if (action === undefined) {
    return;
  }

  event.preventDefault();
  const currentIndex = settingsTabs.findIndex((item) => item.key === tab);
  let nextIndex = currentIndex < 0 ? 0 : currentIndex;

  if (action === "first") {
    nextIndex = 0;
  } else if (action === "last") {
    nextIndex = settingsTabs.length - 1;
  } else {
    nextIndex =
      (nextIndex + action + settingsTabs.length) % settingsTabs.length;
  }

  const nextTab = settingsTabs[nextIndex]?.key;
  if (!nextTab) {
    return;
  }

  selectSettingsTab(nextTab);
  focusSettingsTab(nextTab);
}
</script>

<style scoped>
/* ============ 系统设置 Tab 导航 ============ */
.settings-tabs-shell {
  @apply sticky z-20 -mx-1 rounded-2xl border border-white/80 bg-white/90 p-1.5 backdrop-blur-xl;
  top: 4.75rem;
  box-shadow:
    0 12px 28px rgb(15 23 42 / 0.07),
    0 1px 0 rgb(255 255 255 / 0.9) inset;
}

.settings-tabs-scroll {
  @apply overflow-x-auto;
  -ms-overflow-style: none;
  scrollbar-width: none;
}

.settings-tabs-scroll::-webkit-scrollbar {
  display: none;
}

.settings-tabs {
  @apply flex min-w-max items-center gap-1;
}

.settings-tab {
  @apply relative isolate flex h-10 min-w-[6.75rem] shrink-0 items-center justify-center gap-1.5 whitespace-nowrap rounded-xl border border-transparent px-3 text-sm font-medium text-gray-600 outline-none transition-colors duration-200 ease-out dark:text-gray-300;
}

@media (min-width: 768px) {
  .settings-tabs {
    @apply min-w-full;
  }

  .settings-tab {
    @apply min-w-0 flex-1 basis-0 overflow-hidden px-2 text-[13px];
  }

  .settings-tab-icon {
    @apply h-6 w-6;
  }
}

.settings-tab::before {
  @apply absolute inset-0 -z-10 rounded-xl opacity-0 transition-opacity duration-200;
  content: "";
  background: linear-gradient(135deg, rgb(248 250 252 / 0.95), rgb(241 245 249 / 0.8));
}

.settings-tab:hover::before,
.settings-tab:focus-visible::before {
  opacity: 1;
}

.settings-tab:focus-visible {
  @apply ring-2 ring-primary-500/40 ring-offset-2 ring-offset-white dark:ring-offset-dark-900;
}

.settings-tab-active {
  @apply border-primary-200/80 bg-white text-primary-700 shadow-sm dark:border-primary-400/30 dark:bg-dark-700/95 dark:text-primary-200;
  box-shadow:
    0 8px 18px rgb(15 23 42 / 0.08),
    0 1px 0 rgb(255 255 255 / 0.92) inset;
}

.settings-tab-active::before {
  opacity: 0;
}

.settings-tab-active::after {
  position: absolute;
  right: 0.75rem;
  bottom: 0.25rem;
  left: 0.75rem;
  height: 2px;
  border-radius: 9999px;
  content: "";
  background: linear-gradient(90deg, #14b8a6, #0ea5e9);
}

.settings-tab-icon {
  @apply flex h-7 w-7 shrink-0 items-center justify-center rounded-lg text-muted-foreground transition-colors duration-200;
}

.settings-tab:hover .settings-tab-icon,
.settings-tab:focus-visible .settings-tab-icon {
  @apply text-gray-700 dark:text-gray-200;
}

.settings-tab-active .settings-tab-icon {
  @apply bg-primary-50 text-primary-600 dark:bg-primary-400/10 dark:text-primary-300;
}

.settings-tab-label {
  @apply min-w-0 overflow-hidden text-ellipsis whitespace-nowrap leading-none;
}

/* ============ 系统设置 保存浮标 ============ */
.settings-save-dock {
  @apply sticky bottom-3 z-20 ml-auto flex w-fit items-center rounded-full border border-white/80 bg-white/90 p-1.5 backdrop-blur-xl;
  box-shadow:
    0 10px 26px rgb(15 23 42 / 0.12),
    0 1px 0 rgb(255 255 255 / 0.9) inset;
}
</style>

<style>
/* Dark-mode overrides for the settings tabs shell. Kept in an UNSCOPED block
   because Vue's scoped-CSS compiler was dropping the `:global(.dark) ...`
   rules in the production build, leaving inactive tabs unreadable on dark. */
.dark .settings-tabs-shell {
  border-color: rgb(51 65 85 / 0.65);
  background: rgb(15 23 42 / 0.86);
  box-shadow:
    0 16px 36px rgb(0 0 0 / 0.28),
    0 1px 0 rgb(255 255 255 / 0.06) inset;
}

.dark .settings-tab::before {
  background: linear-gradient(135deg, rgb(30 41 59 / 0.9), rgb(51 65 85 / 0.62));
}

.dark .settings-tab-active {
  box-shadow:
    0 12px 26px rgb(0 0 0 / 0.22),
    0 1px 0 rgb(255 255 255 / 0.08) inset;
}

.dark .settings-save-dock {
  border-color: rgb(51 65 85 / 0.65);
  background: rgb(15 23 42 / 0.86);
  box-shadow:
    0 14px 32px rgb(0 0 0 / 0.34),
    0 1px 0 rgb(255 255 255 / 0.06) inset;
}
</style>
