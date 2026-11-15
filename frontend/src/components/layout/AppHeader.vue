<template>
  <header class="sticky top-0 z-30 border-b border-border bg-background/95 backdrop-blur-xl lg:rounded-t-xl">
    <div class="flex h-12 items-center justify-between gap-1 px-1 sm:gap-2 sm:px-3 lg:px-4">
      <ShellHeaderNavigation
        :title="pageTitle"
        :description="pageDescription"
        :section="sectionLabel"
        :collapsed="appStore.sidebarCollapsed"
        :mobile-open="appStore.mobileOpen"
        @toggle-desktop="appStore.toggleSidebar"
        @toggle-mobile="appStore.toggleMobileSidebar"
      />

      <div class="flex shrink-0 items-center gap-0.5 sm:gap-1.5 lg:gap-2">
        <ShellHeaderActions
          :dark="dark" :open="commandOpen"
          @open-command="emit('open-command')"
          @toggle-theme="emit('toggle-theme')"
        />
        <WebsiteStatusLink v-if="user" live class="max-sm:[&>span:last-child]:sr-only" />

        <!-- Announcement Bell -->
        <AnnouncementBell v-if="user" />

        <!-- Docs Link -->
        <a
          v-if="docUrl"
          :href="docUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="hidden items-center gap-1.5 rounded-md px-2 py-1.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground sm:flex"
        >
          <Icon name="book" size="sm" />
          <span class="hidden md:inline">{{ t('nav.docs') }}</span>
        </a>

        <RouterLink v-else to="/docs" :aria-label="t('docs.title')" class="btn-ghost btn-icon hidden sm:inline-flex">
          <Icon name="book" size="sm" />
        </RouterLink>

        <!-- Model Plaza Entry (icon only below sm) -->
        <router-link
          v-if="user && modelPlazaEnabled"
          :to="{ path: '/model-plaza', query: { embedded: '1' } }"
          :title="t('nav.modelPlaza')"
          :aria-label="t('nav.modelPlaza')"
          class="flex shrink-0 items-center gap-1.5 rounded-md px-1.5 py-1.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground sm:px-2"
        >
          <Icon name="grid" size="sm" />
          <span class="hidden md:inline">{{ t('nav.modelPlaza') }}</span>
        </router-link>

        <!-- Language Switcher -->
        <LocaleSwitcher />

        <!-- Subscription Progress (for users with active subscriptions; not mounted at all when the feature is off) -->
        <SubscriptionProgressMini v-if="user && subscriptionFeatureEnabled" />

        <!-- Balance Display -->
        <div
          v-if="user"
          class="group relative hidden shrink-0 items-center gap-1.5 rounded-md border border-border bg-card px-2 py-1 md:flex"
        >
          <svg
            class="h-4 w-4 text-muted-foreground"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="1.5"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z"
            />
          </svg>
          <span class="text-sm font-semibold text-foreground">
            {{ formatHeaderMoney(availableBalance) }}
          </span>
          <span
            v-if="frozenBalance > 0"
            class="rounded-full bg-amber-100 px-1.5 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/40 dark:text-amber-200"
          >
            {{ balanceFrozenLabel }}
          </span>
          <div
            class="pointer-events-none absolute right-0 top-full mt-2 hidden w-56 rounded-lg border border-border bg-card p-3 text-xs shadow-lg group-hover:block"
          >
            <div class="flex items-center justify-between">
              <span class="text-muted-foreground">{{ balanceAvailableText }}</span>
              <span class="font-medium text-foreground">{{ formatHeaderMoney(availableBalance) }}</span>
            </div>
            <div class="mt-2 flex items-center justify-between">
              <span class="text-muted-foreground">{{ balanceFrozenText }}</span>
              <span class="font-medium text-amber-700 dark:text-amber-200">{{ formatHeaderMoney(frozenBalance) }}</span>
            </div>
            <div class="mt-2 border-t border-gray-100 pt-2 dark:border-dark-700">
              <div class="flex items-center justify-between">
                <span class="text-muted-foreground">{{ balanceTotalText }}</span>
                <span class="font-semibold text-foreground">{{ formatHeaderMoney(totalBalance) }}</span>
              </div>
            </div>
          </div>
        </div>

        <ConsoleUserMenu
          v-if="user"
          :available-balance="formatHeaderMoney(availableBalance)"
          :frozen-balance="frozenBalance > 0 ? formatHeaderMoney(frozenBalance) : undefined"
          @logout="emit('logout')"
          @replay="emit('replay')"
        />
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore, useAuthStore } from '@/stores'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import SubscriptionProgressMini from '@/components/common/SubscriptionProgressMini.vue'
import AnnouncementBell from '@/components/common/AnnouncementBell.vue'
import WebsiteStatusLink from '@/components/user/monitor/WebsiteStatusLink.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { resolveRouteMetaKeys } from '@/router/title'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'

import ConsoleUserMenu from '@/components/admin-shell/ConsoleUserMenu.vue'
import ShellHeaderActions from '@/components/admin-shell/ShellHeaderActions.vue'
import ShellHeaderNavigation from '@/components/admin-shell/ShellHeaderNavigation.vue'
import { ADMIN_SECTION_LABEL_KEYS, getAdminRouteSection } from '@/components/admin-shell/adminRoutes'

defineProps<{ dark: boolean; commandOpen: boolean }>()
const emit = defineEmits<{
  'open-command': []
  'toggle-theme': []
  logout: []
  replay: []
}>()
const route = useRoute()
const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const adminSettingsStore = useAdminSettingsStore()

const user = computed(() => authStore.user)
const docUrl = computed(() => sanitizeUrl(appStore.docUrl))
const modelPlazaEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza))
const availableBalance = computed(() => Number(user.value?.balance || 0))
const frozenBalance = computed(() => Number(user.value?.frozen_balance || 0))
const totalBalance = computed(() => availableBalance.value + frozenBalance.value)
const balanceAvailableText = computed(() => t('common.availableBalance') === 'common.availableBalance' ? '可用余额' : t('common.availableBalance'))
const balanceFrozenText = computed(() => t('common.frozenBalance') === 'common.frozenBalance' ? '冻结金额' : t('common.frozenBalance'))
const balanceTotalText = computed(() => t('common.totalBalance') === 'common.totalBalance' ? '总余额' : t('common.totalBalance'))
const balanceFrozenLabel = computed(() => `${balanceFrozenText.value} ${formatHeaderMoney(frozenBalance.value)}`)

// 订阅功能关闭时不挂载顶栏订阅徽章（组件 onMounted 会拉取订阅接口）。
const subscriptionFeatureEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.subscription))

// /purchase 的标题/描述随站点计费模式切换，与 document.title 共用同一解析。
const routeMetaKeys = computed(() => resolveRouteMetaKeys(route, {
  billingMode: resolveSiteBillingMode(appStore.cachedPublicSettings),
}))

const pageTitle = computed(() => {
  // For custom pages, use the menu item's label instead of generic "自定义页面"
  if (route.name === 'CustomPage') {
    const id = route.params.id as string
    const publicItems = appStore.cachedPublicSettings?.custom_menu_items ?? []
    const menuItem = publicItems.find((item) => item.id === id)
      ?? (authStore.isAdmin ? adminSettingsStore.customMenuItems.find((item) => item.id === id) : undefined)
    if (menuItem?.label) return menuItem.label
  }
  const titleKey = routeMetaKeys.value.titleKey
  if (titleKey) {
    return t(titleKey)
  }
  return (route.meta.title as string) || ''
})

const pageDescription = computed(() => {
  const descKey = routeMetaKeys.value.descriptionKey
  if (descKey) {
    return t(descKey)
  }
  return (route.meta.description as string) || ''
})

const sectionLabel = computed(() => {
  const section = authStore.isAdmin ? getAdminRouteSection(route.name, route.path) : undefined
  return section ? t(ADMIN_SECTION_LABEL_KEYS[section]) : undefined
})

function formatHeaderMoney(value: number) {
  if (!Number.isFinite(value)) return '$0.00'
  return `$${value.toFixed(2)}`
}

</script>
