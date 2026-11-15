<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-4">
      <PageHeader
        :title="t('admin.settings.title')"
        :description="t('admin.settings.description')"
        as="h2"
      >
        <template #badge>
          <StatusBadge v-if="hasUnsavedChanges" tone="warning" :label="t('design.settings.unsavedChanges')" />
        </template>
      </PageHeader>
      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <div
          class="h-8 w-8 animate-spin rounded-full border-b-2 border-brand"
        ></div>
      </div>

      <!-- Settings Form -->
      <form v-else @submit.prevent="saveSettings" class="space-y-6" novalidate>
        <!-- Tab Navigation -->
        <div class="sticky top-12 z-20 rounded-lg border border-border bg-card p-1.5 shadow-sm">
          <nav
            class="overflow-x-auto overscroll-contain"
            role="tablist"
            :aria-label="t('admin.settings.title')"
          >
            <div class="flex min-w-max items-center gap-1">
              <button
                v-for="tab in settingsTabs"
                :key="tab.key"
                :id="`settings-tab-${tab.key}`"
                type="button"
                role="tab"
                :aria-selected="activeTab === tab.key"
                :aria-controls="`settings-panel-${tab.key}`"
                :tabindex="activeTab === tab.key ? 0 : -1"
                class="inline-flex h-8 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-md border px-2.5 text-xs font-medium focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring"
                :class="activeTab === tab.key
                  ? 'border-border bg-muted text-foreground'
                  : 'border-transparent text-muted-foreground hover:bg-accent hover:text-foreground'"
                @click="selectSettingsTab(tab.key)"
                @keydown="handleSettingsTabKeydown($event, tab.key)"
              >
                <span :class="activeTab === tab.key ? 'text-brand' : ''">
                  <Icon :name="tab.icon" size="sm" aria-hidden="true" />
                </span>
                <span>{{
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
          :id="`settings-panel-${key}`"
          :aria-labelledby="`settings-tab-${key}`"
          v-show="activeTab === key"
          role="tabpanel"
        />

        <!-- Save dock — sticky so long tabs never require scrolling to the end -->
        <div v-show="activeTab !== 'backup'" class="sticky bottom-3 z-20 ml-auto flex w-fit rounded-lg border border-border bg-card p-1.5 shadow-sm">
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
import { provide, type Component } from 'vue'
import type { SettingsSectionKey } from '@/components/admin-shell/settingsSections'
import { PageHeader, StatusBadge } from '@/components/page'
import { useSettingsSectionNavigation } from './settings/useSettingsSectionNavigation'
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
  hasUnsavedChanges,
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

const { activeTab, settingsTabs, selectSettingsTab, handleSettingsTabKeydown } = useSettingsSectionNavigation()
</script>
