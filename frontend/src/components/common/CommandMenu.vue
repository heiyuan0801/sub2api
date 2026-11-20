<template>
  <DialogRoot :open="themeStore.commandMenuOpen" @update:open="handleOpenChange">
    <DialogPortal>
      <DialogOverlay
        class="fixed inset-0 z-50 bg-black/50 backdrop-blur-[2px] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0"
      />
      <DialogContent
        class="fixed left-1/2 top-[15vh] z-50 w-[calc(100%-2rem)] max-w-lg -translate-x-1/2 overflow-hidden rounded-xl border border-border bg-popover text-popover-foreground shadow-2xl outline-none duration-150 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95"
        @keydown="handleDialogKeydown"
      >
        <DialogTitle class="sr-only">
          {{ t('commandMenu.title') }}
        </DialogTitle>
        <DialogDescription class="sr-only">
          {{ t('commandMenu.placeholder') }}
        </DialogDescription>

        <!-- Search Bar -->
        <div class="flex items-center gap-2.5 border-b border-border px-3.5 py-2.5">
          <Search class="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
          <input
            ref="inputRef"
            v-model="query"
            type="text"
            autocomplete="off"
            spellcheck="false"
            class="h-6 w-full bg-transparent text-sm text-popover-foreground placeholder:text-muted-foreground focus:outline-none"
            :placeholder="t('commandMenu.placeholder')"
            :aria-label="t('commandMenu.placeholder')"
          />
          <kbd
            class="hidden shrink-0 rounded border border-border bg-muted px-1.5 py-0.5 font-mono text-[10px] font-medium text-muted-foreground sm:inline-block"
          >
            ESC
          </kbd>
        </div>

        <!-- Results List -->
        <div
          ref="listRef"
          role="listbox"
          class="max-h-[min(60vh,380px)] overflow-y-auto overscroll-contain p-1.5"
        >
          <div
            v-if="groupedItems.length === 0"
            class="px-4 py-10 text-center text-sm text-muted-foreground"
          >
            {{ t('commandMenu.empty') }}
          </div>

          <template v-else>
            <div
              v-for="group in groupedItems"
              :key="group.key"
              class="mb-1.5 last:mb-0"
            >
              <div class="px-2.5 py-1.5 text-[11px] font-medium text-muted-foreground">
                {{ group.title }}
              </div>
              <div class="space-y-0.5">
                <button
                  v-for="item in group.items"
                  :key="item.id"
                  type="button"
                  role="option"
                  :data-command-index="item.flatIndex"
                  :aria-selected="activeIndex === item.flatIndex"
                  class="flex w-full items-center justify-between gap-3 rounded-lg px-2.5 py-2 text-left text-sm transition-colors"
                  :class="
                    activeIndex === item.flatIndex
                      ? 'bg-accent text-accent-foreground'
                      : 'text-popover-foreground hover:bg-accent/60'
                  "
                  @mouseenter="activeIndex = item.flatIndex"
                  @click="executeItem(item)"
                >
                  <span class="flex min-w-0 items-center gap-2.5">
                    <component
                      :is="item.icon"
                      class="size-4 shrink-0"
                      :class="
                        activeIndex === item.flatIndex
                          ? 'text-accent-foreground'
                          : 'text-muted-foreground'
                      "
                      aria-hidden="true"
                    />
                    <span class="truncate font-medium">{{ item.label }}</span>
                  </span>
                  <span
                    v-if="item.path"
                    class="hidden shrink-0 font-mono text-[11px] text-muted-foreground sm:inline"
                  >
                    {{ item.path }}
                  </span>
                  <CornerDownLeft
                    v-else-if="activeIndex === item.flatIndex"
                    class="size-3.5 shrink-0 text-muted-foreground"
                    aria-hidden="true"
                  />
                </button>
              </div>
            </div>
          </template>
        </div>

        <!-- Keyboard Hints Footer -->
        <div
          class="hidden items-center justify-between border-t border-border bg-muted/30 px-3.5 py-2 text-[11px] text-muted-foreground sm:flex"
        >
          <div class="flex items-center gap-3">
            <span class="inline-flex items-center gap-1">
              <kbd class="rounded border border-border bg-background px-1 py-0.5 font-mono text-[10px]">
                ↑↓
              </kbd>
              <span>{{ t('commandMenu.hints.navigate') }}</span>
            </span>
            <span class="inline-flex items-center gap-1">
              <kbd class="rounded border border-border bg-background px-1 py-0.5 font-mono text-[10px]">
                ↵
              </kbd>
              <span>{{ t('commandMenu.hints.select') }}</span>
            </span>
          </div>
          <span class="inline-flex items-center gap-1">
            <kbd class="rounded border border-border bg-background px-1 py-0.5 font-mono text-[10px]">
              ESC
            </kbd>
            <span>{{ t('commandMenu.hints.close') }}</span>
          </span>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch, type Component } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle
} from 'reka-ui'
import {
  Activity,
  BarChart3,
  Bell,
  BookOpen,
  CornerDownLeft,
  Cpu,
  CreditCard,
  Folder,
  Gift,
  Globe,
  Image,
  Key,
  Layers,
  LayoutDashboard,
  Monitor,
  Moon,
  Palette,
  Plug,
  Receipt,
  Search,
  Server,
  Settings,
  Shield,
  Sparkles,
  Sun,
  Ticket,
  User,
  Users,
  Wallet
} from '@lucide/vue'
import { useAdminSettingsStore, useAppStore, useAuthStore } from '@/stores'
import { useThemeStore } from '@/stores/theme'
import { FeatureFlags, makeSidebarFlag } from '@/utils/featureFlags'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'

interface CommandEntry {
  id: string
  label: string
  groupKey: string
  groupTitle: string
  icon: Component
  path?: string
  keywords?: string[]
  action: () => void
}

interface IndexedCommandEntry extends CommandEntry {
  flatIndex: number
}

interface CommandGroup {
  key: string
  title: string
  items: IndexedCommandEntry[]
}

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const adminSettingsStore = useAdminSettingsStore()
const themeStore = useThemeStore()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()

const query = ref('')
const activeIndex = ref(0)
const inputRef = ref<HTMLInputElement | null>(null)
const listRef = ref<HTMLElement | null>(null)

const flagChannelMonitor = makeSidebarFlag(FeatureFlags.channelMonitor)
const flagPayment = makeSidebarFlag(FeatureFlags.payment)
const flagAvailableChannels = makeSidebarFlag(FeatureFlags.availableChannels)
const flagSubscription = makeSidebarFlag(FeatureFlags.subscription)
const flagAffiliate = makeSidebarFlag(FeatureFlags.affiliate)
const flagRiskControl = makeSidebarFlag(FeatureFlags.riskControl)
const flagPluginManagement = makeSidebarFlag(FeatureFlags.pluginManagement)
const flagMCP = makeSidebarFlag(FeatureFlags.mcp)

const purchaseNavLabel = computed(() => {
  switch (resolveSiteBillingMode(appStore.cachedPublicSettings)) {
    case 'recharge_only':
      return t('nav.recharge')
    case 'subscription_only':
      return t('nav.subscribe')
    default:
      return t('nav.buySubscription')
  }
})

const customMenuItemsForUser = computed(() => {
  const items = appStore.cachedPublicSettings?.custom_menu_items ?? []
  return items
    .filter((item) => item.visibility === 'user')
    .sort((a, b) => a.sort_order - b.sort_order)
})

const customMenuItemsForAdmin = computed(() => {
  return adminSettingsStore.customMenuItems
    .filter((item) => item.visibility === 'admin')
    .sort((a, b) => a.sort_order - b.sort_order)
})

function isFlagVisible(flag?: () => boolean | undefined): boolean {
  return !flag || flag() !== false
}

function navigateTo(path: string): void {
  void router.push(path)
}

function buildSelfNavEntries(withDashboard: boolean, groupOverride?: { key: string; title: string }): CommandEntry[] {
  const isSimple = authStore.isSimpleMode
  const entries: Array<{
    path: string
    label: string
    icon: Component
    groupKey: string
    groupTitle: string
    hideInSimpleMode?: boolean
    featureFlag?: () => boolean | undefined
  }> = []

  if (withDashboard) {
    entries.push({
      path: '/dashboard',
      label: t('nav.dashboard'),
      icon: LayoutDashboard,
      groupKey: 'workspace',
      groupTitle: t('nav.sections.workspace')
    })
  }

  entries.push(
    {
      path: '/keys',
      label: t('nav.apiKeys'),
      icon: Key,
      groupKey: 'workspace',
      groupTitle: t('nav.sections.workspace')
    },
    {
      path: '/usage',
      label: t('nav.usage'),
      icon: BarChart3,
      groupKey: 'workspace',
      groupTitle: t('nav.sections.workspace'),
      hideInSimpleMode: true
    },
    {
      path: '/batch-image',
      label: t('nav.batchImage'),
      icon: Image,
      groupKey: 'workspace',
      groupTitle: t('nav.sections.workspace'),
      hideInSimpleMode: true,
      featureFlag: () => canUseBatchImage.value
    },
    {
      path: '/available-channels',
      label: t('nav.availableChannels'),
      icon: Layers,
      groupKey: 'services',
      groupTitle: t('nav.sections.services'),
      hideInSimpleMode: true,
      featureFlag: flagAvailableChannels
    },
    {
      path: '/monitor',
      label: t('nav.channelStatus'),
      icon: Activity,
      groupKey: 'services',
      groupTitle: t('nav.sections.services'),
      featureFlag: flagChannelMonitor
    },
    {
      path: '/notifications',
      label: t('operations.notifications'),
      icon: Bell,
      groupKey: 'services',
      groupTitle: t('nav.sections.services')
    },
    {
      path: '/subscriptions',
      label: t('nav.mySubscriptions'),
      icon: CreditCard,
      groupKey: 'billing',
      groupTitle: t('nav.sections.billing'),
      hideInSimpleMode: true,
      featureFlag: flagSubscription
    },
    {
      path: '/purchase',
      label: purchaseNavLabel.value,
      icon: Wallet,
      groupKey: 'billing',
      groupTitle: t('nav.sections.billing'),
      hideInSimpleMode: true,
      featureFlag: flagPayment
    },
    {
      path: '/orders',
      label: t('nav.myOrders'),
      icon: Receipt,
      groupKey: 'billing',
      groupTitle: t('nav.sections.billing'),
      hideInSimpleMode: true,
      featureFlag: flagPayment
    },
    {
      path: '/reward-activities',
      label: t('rewards.title'),
      icon: Gift,
      groupKey: 'billing',
      groupTitle: t('nav.sections.billing'),
      hideInSimpleMode: true
    },
    {
      path: '/redeem',
      label: t('nav.redeem'),
      icon: Ticket,
      groupKey: 'billing',
      groupTitle: t('nav.sections.billing'),
      hideInSimpleMode: true
    },
    {
      path: '/affiliate',
      label: t('nav.affiliate'),
      icon: Users,
      groupKey: 'billing',
      groupTitle: t('nav.sections.billing'),
      hideInSimpleMode: true,
      featureFlag: flagAffiliate
    },
    {
      path: '/profile',
      label: t('nav.profile'),
      icon: User,
      groupKey: 'account',
      groupTitle: t('nav.sections.account')
    },
    {
      path: '/settings/mcp',
      label: t('mcp.title'),
      icon: Plug,
      groupKey: 'account',
      groupTitle: t('nav.sections.account'),
      featureFlag: flagMCP
    },
    ...customMenuItemsForUser.value.map((item) => ({
      path: `/custom/${item.id}`,
      label: item.label,
      icon: Sparkles,
      groupKey: 'account',
      groupTitle: t('nav.sections.account')
    }))
  )

  return entries
    .filter((entry) => (!isSimple || !entry.hideInSimpleMode) && isFlagVisible(entry.featureFlag))
    .map((entry) => ({
      id: `nav:${groupOverride ? groupOverride.key : entry.groupKey}:${entry.path}`,
      label: entry.label,
      groupKey: groupOverride ? groupOverride.key : entry.groupKey,
      groupTitle: groupOverride ? groupOverride.title : entry.groupTitle,
      icon: entry.icon,
      path: entry.path,
      action: () => navigateTo(entry.path)
    }))
}

const navigationEntries = computed<CommandEntry[]>(() => {
  const isSimple = authStore.isSimpleMode

  if (authStore.isAdmin) {
    const rawAdmin: Array<{
      path: string
      label: string
      icon: Component
      groupKey: string
      groupTitle: string
      hideInSimpleMode?: boolean
      featureFlag?: () => boolean | undefined
    }> = [
      // Overview
      {
        path: '/admin/dashboard',
        label: t('nav.dashboard'),
        icon: LayoutDashboard,
        groupKey: 'overview',
        groupTitle: t('nav.sections.overview')
      },
      {
        path: '/admin/cluster',
        label: t('cluster.title'),
        icon: Cpu,
        groupKey: 'overview',
        groupTitle: t('nav.sections.overview')
      },
      {
        path: '/admin/ops',
        label: t('nav.ops'),
        icon: BarChart3,
        groupKey: 'overview',
        groupTitle: t('nav.sections.overview'),
        featureFlag: () => adminSettingsStore.opsMonitoringEnabled
      },
      // Resources
      {
        path: '/admin/users',
        label: t('nav.users'),
        icon: Users,
        groupKey: 'resources',
        groupTitle: t('nav.sections.resources'),
        hideInSimpleMode: true
      },
      {
        path: '/admin/registration-sources',
        label: t('operations.registrationSources'),
        icon: BarChart3,
        groupKey: 'resources',
        groupTitle: t('nav.sections.resources'),
        hideInSimpleMode: true
      },
      {
        path: '/admin/groups',
        label: t('nav.groups'),
        icon: Folder,
        groupKey: 'resources',
        groupTitle: t('nav.sections.resources')
      },
      {
        path: '/admin/channels/pricing',
        label: t('nav.channelPricing'),
        icon: Layers,
        groupKey: 'resources',
        groupTitle: t('nav.sections.resources'),
        hideInSimpleMode: true
      },
      {
        path: '/admin/channels/monitor',
        label: t('nav.channelMonitor'),
        icon: Activity,
        groupKey: 'resources',
        groupTitle: t('nav.sections.resources'),
        hideInSimpleMode: true,
        featureFlag: flagChannelMonitor
      },
      {
        path: '/admin/subscriptions',
        label: t('nav.subscriptions'),
        icon: CreditCard,
        groupKey: 'resources',
        groupTitle: t('nav.sections.resources'),
        hideInSimpleMode: true,
        featureFlag: flagSubscription
      },
      {
        path: '/admin/accounts',
        label: t('nav.accounts'),
        icon: Globe,
        groupKey: 'resources',
        groupTitle: t('nav.sections.resources')
      },
      {
        path: '/admin/proxies',
        label: t('nav.proxies'),
        icon: Server,
        groupKey: 'resources',
        groupTitle: t('nav.sections.resources')
      },
      // Operations
      {
        path: '/admin/orders/dashboard',
        label: t('nav.paymentDashboard'),
        icon: BarChart3,
        groupKey: 'operations',
        groupTitle: t('nav.sections.operations'),
        hideInSimpleMode: true,
        featureFlag: () => adminSettingsStore.paymentEnabled
      },
      {
        path: '/admin/orders',
        label: t('nav.orderManagement'),
        icon: Receipt,
        groupKey: 'operations',
        groupTitle: t('nav.sections.operations'),
        hideInSimpleMode: true,
        featureFlag: () => adminSettingsStore.paymentEnabled
      },
      {
        path: '/admin/orders/plans',
        label: t('nav.paymentPlans'),
        icon: CreditCard,
        groupKey: 'operations',
        groupTitle: t('nav.sections.operations'),
        hideInSimpleMode: true,
        featureFlag: () => adminSettingsStore.paymentEnabled
      },
      {
        path: '/admin/redeem',
        label: t('nav.redeemCodes'),
        icon: Ticket,
        groupKey: 'operations',
        groupTitle: t('nav.sections.operations'),
        hideInSimpleMode: true
      },
      {
        path: '/admin/promo-codes',
        label: t('nav.promoCodes'),
        icon: Gift,
        groupKey: 'operations',
        groupTitle: t('nav.sections.operations'),
        hideInSimpleMode: true
      },
      {
        path: '/admin/reward-activities',
        label: t('rewards.adminTitle'),
        icon: Gift,
        groupKey: 'operations',
        groupTitle: t('nav.sections.operations'),
        hideInSimpleMode: true
      },
      {
        path: '/admin/affiliates/invites',
        label: t('nav.affiliateInviteRecords'),
        icon: Users,
        groupKey: 'operations',
        groupTitle: t('nav.sections.operations'),
        hideInSimpleMode: true,
        featureFlag: flagAffiliate
      },
      {
        path: '/admin/affiliates/rebates',
        label: t('nav.affiliateRebateRecords'),
        icon: Receipt,
        groupKey: 'operations',
        groupTitle: t('nav.sections.operations'),
        hideInSimpleMode: true,
        featureFlag: flagAffiliate
      },
      {
        path: '/admin/affiliates/transfers',
        label: t('nav.affiliateTransferRecords'),
        icon: CreditCard,
        groupKey: 'operations',
        groupTitle: t('nav.sections.operations'),
        hideInSimpleMode: true,
        featureFlag: flagAffiliate
      },
      {
        path: '/admin/announcements',
        label: t('operations.notifications'),
        icon: Bell,
        groupKey: 'operations',
        groupTitle: t('nav.sections.operations')
      },
      {
        path: '/admin/docs',
        label: t('docs.adminTitle'),
        icon: BookOpen,
        groupKey: 'operations',
        groupTitle: t('nav.sections.operations'),
        hideInSimpleMode: true
      },
      // System
      {
        path: '/admin/usage',
        label: t('nav.usage'),
        icon: BarChart3,
        groupKey: 'system',
        groupTitle: t('nav.sections.system')
      },
      {
        path: '/admin/risk-control',
        label: t('nav.contentModeration'),
        icon: Shield,
        groupKey: 'system',
        groupTitle: t('nav.sections.system'),
        featureFlag: flagRiskControl
      },
      {
        path: '/admin/prompt-audit',
        label: t('nav.promptAudit'),
        icon: Shield,
        groupKey: 'system',
        groupTitle: t('nav.sections.system'),
        featureFlag: flagRiskControl
      },
      {
        path: '/admin/audit-logs',
        label: t('nav.auditLogs'),
        icon: Shield,
        groupKey: 'system',
        groupTitle: t('nav.sections.system'),
        hideInSimpleMode: true
      },
      {
        path: '/admin/plugins',
        label: t('nav.plugins'),
        icon: Plug,
        groupKey: 'system',
        groupTitle: t('nav.sections.system'),
        featureFlag: flagPluginManagement
      }
    ]

    if (isSimple) {
      rawAdmin.push({
        path: '/keys',
        label: t('nav.apiKeys'),
        icon: Key,
        groupKey: 'system',
        groupTitle: t('nav.sections.system')
      })
      if (isFlagVisible(flagMCP)) {
        rawAdmin.push({
          path: '/settings/mcp',
          label: t('mcp.title'),
          icon: Plug,
          groupKey: 'system',
          groupTitle: t('nav.sections.system')
        })
      }
    }

    rawAdmin.push({
      path: '/admin/settings',
      label: t('nav.settings'),
      icon: Settings,
      groupKey: 'system',
      groupTitle: t('nav.sections.system')
    })

    for (const cm of customMenuItemsForAdmin.value) {
      rawAdmin.push({
        path: `/custom/${cm.id}`,
        label: cm.label,
        icon: Sparkles,
        groupKey: 'system',
        groupTitle: t('nav.sections.system')
      })
    }

    const adminEntries: CommandEntry[] = rawAdmin
      .filter((entry) => (!isSimple || !entry.hideInSimpleMode) && isFlagVisible(entry.featureFlag))
      .map((entry) => ({
        id: `nav:admin:${entry.path}`,
        label: entry.label,
        groupKey: entry.groupKey,
        groupTitle: entry.groupTitle,
        icon: entry.icon,
        path: entry.path,
        action: () => navigateTo(entry.path)
      }))

    if (!isSimple) {
      adminEntries.push(
        ...buildSelfNavEntries(false, { key: 'personal', title: t('nav.myAccount') })
      )
    }

    return adminEntries
  }

  if (!appStore.backendModeEnabled) {
    return buildSelfNavEntries(true)
  }

  return []
})

const themeActionEntries = computed<CommandEntry[]>(() => [
  {
    id: 'action:theme-light',
    label: t('commandMenu.actions.switchToLight'),
    groupKey: 'theme',
    groupTitle: t('commandMenu.groups.theme'),
    icon: Sun,
    keywords: ['light', 'theme', 'appearance', 'day', 'bright'],
    action: () => themeStore.setMode('light')
  },
  {
    id: 'action:theme-dark',
    label: t('commandMenu.actions.switchToDark'),
    groupKey: 'theme',
    groupTitle: t('commandMenu.groups.theme'),
    icon: Moon,
    keywords: ['dark', 'theme', 'appearance', 'night'],
    action: () => themeStore.setMode('dark')
  },
  {
    id: 'action:theme-system',
    label: t('commandMenu.actions.switchToSystem'),
    groupKey: 'theme',
    groupTitle: t('commandMenu.groups.theme'),
    icon: Monitor,
    keywords: ['system', 'auto', 'theme', 'appearance'],
    action: () => themeStore.setMode('system')
  },
  {
    id: 'action:theme-settings',
    label: t('commandMenu.actions.openThemeSettings'),
    groupKey: 'theme',
    groupTitle: t('commandMenu.groups.theme'),
    icon: Palette,
    keywords: ['theme', 'preset', 'color', 'radius', 'font', 'sidebar', 'appearance', 'config'],
    action: () => themeStore.setConfigDrawerOpen(true)
  }
])

const allEntries = computed<CommandEntry[]>(() => [
  ...navigationEntries.value,
  ...themeActionEntries.value
])

const filteredEntries = computed<IndexedCommandEntry[]>(() => {
  const normalized = query.value.trim().toLowerCase()
  const matched = normalized
    ? allEntries.value.filter((entry) => {
        if (entry.label.toLowerCase().includes(normalized)) return true
        if (entry.groupTitle.toLowerCase().includes(normalized)) return true
        if (entry.path && entry.path.toLowerCase().includes(normalized)) return true
        if (entry.keywords?.some((kw) => kw.includes(normalized))) return true
        return false
      })
    : allEntries.value

  return matched.map((entry, idx) => ({ ...entry, flatIndex: idx }))
})

const groupedItems = computed<CommandGroup[]>(() => {
  const groups = new Map<string, CommandGroup>()
  for (const item of filteredEntries.value) {
    const existing = groups.get(item.groupKey)
    if (existing) {
      existing.items.push(item)
    } else {
      groups.set(item.groupKey, {
        key: item.groupKey,
        title: item.groupTitle,
        items: [item]
      })
    }
  }
  return Array.from(groups.values())
})

function handleOpenChange(open: boolean): void {
  themeStore.setCommandMenuOpen(open)
}

function executeItem(item: CommandEntry): void {
  themeStore.setCommandMenuOpen(false)
  item.action()
}

function scrollActiveItemIntoView(): void {
  void nextTick(() => {
    const container = listRef.value
    if (!container) return
    const el = container.querySelector<HTMLElement>(`[data-command-index="${activeIndex.value}"]`)
    if (el && typeof el.scrollIntoView === 'function') {
      el.scrollIntoView({ block: 'nearest' })
    }
  })
}

function handleDialogKeydown(event: KeyboardEvent): void {
  if (event.isComposing || event.keyCode === 229) return

  const total = filteredEntries.value.length
  if (total === 0) return

  if (event.key === 'ArrowDown') {
    event.preventDefault()
    activeIndex.value = (activeIndex.value + 1) % total
    scrollActiveItemIntoView()
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    activeIndex.value = (activeIndex.value - 1 + total) % total
    scrollActiveItemIntoView()
  } else if (event.key === 'Enter') {
    event.preventDefault()
    const target = filteredEntries.value[activeIndex.value]
    if (target) {
      executeItem(target)
    }
  }
}

function handleGlobalKeydown(event: KeyboardEvent): void {
  if (!(event.metaKey || event.ctrlKey) || event.key.toLowerCase() !== 'k') return
  if (event.defaultPrevented) return

  // Do not hijack Cmd/Ctrl+K if PublicSiteHeader's landing search is focused.
  const activeEl = document.activeElement
  if (activeEl instanceof HTMLElement && activeEl.closest('.landing-search')) {
    return
  }

  event.preventDefault()
  themeStore.setCommandMenuOpen(!themeStore.commandMenuOpen)
}

watch(query, () => {
  activeIndex.value = 0
})

watch(
  () => themeStore.commandMenuOpen,
  (open) => {
    if (open) {
      activeIndex.value = 0
      void nextTick(() => {
        inputRef.value?.focus()
      })
    } else {
      query.value = ''
      activeIndex.value = 0
    }
  }
)

onMounted(() => {
  document.addEventListener('keydown', handleGlobalKeydown)
  void refreshBatchImageAccess()
  if (authStore.isAdmin) {
    adminSettingsStore.fetch()
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleGlobalKeydown)
})
</script>
