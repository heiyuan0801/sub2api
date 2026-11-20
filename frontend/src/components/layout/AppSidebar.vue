<template>
  <aside
    class="sidebar"
    :class="[
      'w-[272px] border-r border-sidebar-border/80 lg:sticky lg:z-40 lg:shrink-0 lg:translate-x-0',
      sidebarCollapsed ? 'lg:w-[56px]' : 'lg:w-[220px]',
      { '-translate-x-full': !mobileOpen },
      themeStore.sidebarVariant === 'floating'
        ? 'lg:my-2 lg:ml-2 lg:top-2 lg:h-[calc(100vh-1rem)] lg:rounded-xl lg:border lg:border-sidebar-border lg:shadow-sm'
        : themeStore.sidebarVariant === 'inset'
          ? 'lg:top-0 lg:h-screen lg:border-r-0'
          : 'lg:top-0 lg:h-screen lg:border-r lg:border-sidebar-border/80'
    ]"
  >
    <!-- Brand Header (compact, borderless, collapses smoothly to centered logo) -->
    <div
      class="sidebar-header"
      :class="effectiveCollapsed ? 'justify-center px-0' : ''"
    >
      <router-link
        :to="homePath"
        class="sidebar-logo flex size-6 items-center justify-center overflow-hidden rounded-md transition-opacity hover:opacity-80"
        :title="effectiveCollapsed ? siteName : undefined"
        @click="handleMenuItemClick(homePath)"
      >
        <img
          v-if="settingsLoaded"
          :src="siteLogo || '/logo.svg'"
          alt="Logo"
          class="h-full w-full object-contain"
        />
      </router-link>
      <div v-if="!effectiveCollapsed" class="sidebar-brand">
        <router-link
          :to="homePath"
          class="sidebar-brand-title text-foreground transition-colors hover:text-foreground/80"
          @click="handleMenuItemClick(homePath)"
        >
          {{ siteName }}
        </router-link>
        <VersionBadge :version="siteVersion" />
      </div>
    </div>

    <!-- Navigation -->
    <nav ref="sidebarNavRef" class="sidebar-nav scrollbar-hide">
      <TooltipProvider :delay-duration="120">
        <!-- Admin View: Grouped admin menu first, then collapsible personal menu -->
        <template v-if="isAdmin">
          <div
            v-for="section in adminNavGroups"
            :key="section.key"
            class="sidebar-section"
          >
            <div
              class="sidebar-section-title"
              :class="{ 'sidebar-section-title-collapsed': effectiveCollapsed }"
              :aria-hidden="effectiveCollapsed ? 'true' : 'false'"
            >
              <span
                class="sidebar-section-title-text"
                :class="{ 'sidebar-section-title-text-collapsed': effectiveCollapsed }"
              >
                {{ section.title }}
              </span>
            </div>

            <div class="space-y-0.5">
              <template v-for="item in section.items" :key="item.path">
                <!-- Collapsible group (has children) -->
                <template v-if="item.children?.length">
                  <!-- Collapsed desktop dropdown menu -->
                  <DropdownMenuRoot v-if="effectiveCollapsed">
                    <DropdownMenuTrigger as-child>
                      <button
                        type="button"
                        class="group sidebar-link mx-auto size-8 justify-center px-0"
                        :class="{ 'sidebar-link-active': isGroupActive(item) }"
                        :title="item.label"
                        :aria-label="item.label"
                      >
                        <component
                          :is="item.icon"
                          :stroke-width="1.75"
                          class="size-[15px] shrink-0 transition-colors"
                          :class="
                            isGroupActive(item)
                              ? 'text-foreground'
                              : 'text-muted-foreground/80 group-hover:text-foreground'
                          "
                        />
                      </button>
                    </DropdownMenuTrigger>
                    <DropdownMenuPortal>
                      <DropdownMenuContent
                        side="right"
                        align="start"
                        :side-offset="8"
                        class="z-50 min-w-[180px] rounded-lg border border-border bg-popover p-1 text-popover-foreground shadow-lg"
                      >
                        <DropdownMenuLabel class="px-2 py-1.5 text-xs font-semibold text-muted-foreground">
                          {{ item.label }}
                        </DropdownMenuLabel>
                        <DropdownMenuSeparator class="my-1 h-px bg-border" />
                        <DropdownMenuItem
                          v-for="child in item.children"
                          :key="child.path"
                          as-child
                        >
                          <router-link
                            :to="child.path"
                            class="group flex cursor-pointer items-center gap-2 rounded-lg border border-transparent px-2 py-1.5 text-[13px] text-popover-foreground outline-none transition-colors hover:bg-muted/70 focus:bg-muted/70"
                            :class="{
                              'border-border bg-muted/85 font-medium text-foreground hover:bg-muted/85 focus:bg-muted/85 dark:border-white/10 dark:bg-white/[0.08] dark:text-white':
                                route.path === child.path
                            }"
                            @click="handleMenuItemClick(child.path)"
                          >
                            <component
                              :is="child.icon"
                              :stroke-width="1.75"
                              class="size-[14px] shrink-0 transition-colors"
                              :class="
                                route.path === child.path
                                  ? 'text-foreground'
                                  : 'text-muted-foreground/80 group-hover:text-foreground'
                              "
                            />
                            <span class="truncate">{{ child.label }}</span>
                          </router-link>
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenuPortal>
                  </DropdownMenuRoot>

                  <!-- Expanded collapsible parent + children -->
                  <template v-else>
                    <button
                      type="button"
                      class="group sidebar-link w-full"
                      :class="{ 'sidebar-link-active': isGroupActive(item) && !isGroupExpanded(item) }"
                      @click="handleGroupClick(item)"
                    >
                      <component
                        :is="item.icon"
                        :stroke-width="1.75"
                        class="size-[15px] shrink-0 transition-colors"
                        :class="
                          isGroupActive(item) && !isGroupExpanded(item)
                            ? 'text-foreground'
                            : 'text-muted-foreground/80 group-hover:text-foreground'
                        "
                      />
                      <span class="sidebar-label sidebar-label-flex">
                        <span class="min-w-0 truncate">{{ item.label }}</span>
                        <ChevronRight
                          :size="14"
                          :stroke-width="1.75"
                          class="shrink-0 text-muted-foreground/60 transition-transform duration-200"
                          :class="{ 'rotate-90': isGroupExpanded(item) }"
                        />
                      </span>
                    </button>
                    <div
                      v-if="isGroupExpanded(item)"
                      class="mx-3.5 mt-0.5 flex flex-col gap-0.5 border-l border-sidebar-border py-0.5 pl-2.5"
                    >
                      <router-link
                        v-for="child in item.children"
                        :key="child.path"
                        :to="child.path"
                        class="group flex h-7 items-center gap-2 rounded-lg border border-transparent px-2.5 text-[13px] text-foreground/80 transition-colors hover:bg-muted/70 hover:text-foreground"
                        :class="{
                          'border-border bg-muted/85 font-medium text-foreground shadow-[0_1px_2px_rgba(0,0,0,0.03)] hover:bg-muted/85 hover:text-foreground dark:border-white/10 dark:bg-white/[0.08] dark:text-white dark:hover:bg-white/[0.08] dark:hover:text-white':
                            route.path === child.path
                        }"
                        @click="handleMenuItemClick(child.path)"
                      >
                        <component
                          :is="child.icon"
                          :stroke-width="1.75"
                          class="size-[14px] shrink-0 transition-colors"
                          :class="
                            route.path === child.path
                              ? 'text-foreground'
                              : 'text-muted-foreground/80 group-hover:text-foreground'
                          "
                        />
                        <span class="min-w-0 truncate">{{ child.label }}</span>
                      </router-link>
                    </div>
                  </template>
                </template>

                <!-- Normal item (no children) -->
                <TooltipRoot v-else :disabled="!effectiveCollapsed">
                  <TooltipTrigger as-child>
                    <router-link
                      :to="item.path"
                      class="group sidebar-link"
                      :class="[
                        isActive(item.path) && 'sidebar-link-active',
                        effectiveCollapsed && 'mx-auto size-8 justify-center px-0'
                      ]"
                      :aria-label="effectiveCollapsed ? item.label : undefined"
                      :id="
                        item.path === '/admin/accounts'
                          ? 'sidebar-channel-manage'
                          : item.path === '/admin/groups'
                            ? 'sidebar-group-manage'
                            : item.path === '/admin/redeem'
                              ? 'sidebar-wallet'
                              : undefined
                      "
                      :data-tour="item.path === '/keys' ? 'sidebar-my-keys' : undefined"
                      @click="handleMenuItemClick(item.path)"
                    >
                      <span
                        v-if="item.iconSvg"
                        class="size-[15px] shrink-0 sidebar-svg-icon transition-colors"
                        :class="
                          isActive(item.path)
                            ? 'text-foreground'
                            : 'text-muted-foreground/80 group-hover:text-foreground'
                        "
                        v-html="sanitizeSvg(item.iconSvg)"
                      ></span>
                      <component
                        v-else
                        :is="item.icon"
                        :stroke-width="1.75"
                        class="size-[15px] shrink-0 transition-colors"
                        :class="
                          isActive(item.path)
                            ? 'text-foreground'
                            : 'text-muted-foreground/80 group-hover:text-foreground'
                        "
                      />
                      <span v-if="!effectiveCollapsed" class="sidebar-label">{{ item.label }}</span>
                    </router-link>
                  </TooltipTrigger>
                  <TooltipPortal>
                    <TooltipContent
                      side="right"
                      :side-offset="8"
                      class="z-50 rounded-md border border-border bg-popover px-2.5 py-1 text-xs font-medium text-popover-foreground shadow-md"
                    >
                      {{ item.label }}
                    </TooltipContent>
                  </TooltipPortal>
                </TooltipRoot>
              </template>
            </div>
          </div>

          <!-- Collapsible Personal Section for Admin (hidden in simple mode) -->
          <div v-if="!authStore.isSimpleMode && personalNavItems.length" class="sidebar-section">
            <button
              v-if="!effectiveCollapsed"
              type="button"
              class="sidebar-section-title w-full justify-between rounded-md transition-colors hover:text-foreground"
              :aria-expanded="isPersonalSectionExpanded"
              @click="togglePersonalSection"
            >
              <span class="sidebar-section-title-text">
                {{ t('nav.myAccount') }}
              </span>
              <ChevronRight
                :size="13"
                :stroke-width="1.75"
                class="shrink-0 text-muted-foreground/60 transition-transform duration-200"
                :class="{ 'rotate-90': isPersonalSectionExpanded }"
              />
            </button>
            <div
              v-else
              class="sidebar-section-title sidebar-section-title-collapsed"
              aria-hidden="true"
            >
              <span class="sidebar-section-title-text sidebar-section-title-text-collapsed">
                {{ t('nav.myAccount') }}
              </span>
            </div>

            <div v-if="effectiveCollapsed || isPersonalSectionExpanded" class="space-y-0.5">
              <TooltipRoot
                v-for="item in personalNavItems"
                :key="item.path"
                :disabled="!effectiveCollapsed"
              >
                <TooltipTrigger as-child>
                  <router-link
                    :to="item.path"
                    class="group sidebar-link"
                    :class="[
                      isActive(item.path) && 'sidebar-link-active',
                      effectiveCollapsed && 'mx-auto size-8 justify-center px-0'
                    ]"
                    :aria-label="effectiveCollapsed ? item.label : undefined"
                    :data-tour="item.path === '/keys' ? 'sidebar-my-keys' : undefined"
                    @click="handleMenuItemClick(item.path)"
                  >
                    <span
                      v-if="item.iconSvg"
                      class="size-[15px] shrink-0 sidebar-svg-icon transition-colors"
                      :class="
                        isActive(item.path)
                          ? 'text-foreground'
                          : 'text-muted-foreground/80 group-hover:text-foreground'
                      "
                      v-html="sanitizeSvg(item.iconSvg)"
                    ></span>
                    <component
                      v-else
                      :is="item.icon"
                      :stroke-width="1.75"
                      class="size-[15px] shrink-0 transition-colors"
                      :class="
                        isActive(item.path)
                          ? 'text-foreground'
                          : 'text-muted-foreground/80 group-hover:text-foreground'
                      "
                    />
                    <span v-if="!effectiveCollapsed" class="sidebar-label">{{ item.label }}</span>
                  </router-link>
                </TooltipTrigger>
                <TooltipPortal>
                  <TooltipContent
                    side="right"
                    :side-offset="8"
                    class="z-50 rounded-md border border-border bg-popover px-2.5 py-1 text-xs font-medium text-popover-foreground shadow-md"
                  >
                    {{ item.label }}
                  </TooltipContent>
                </TooltipPortal>
              </TooltipRoot>
            </div>
          </div>
        </template>

        <!-- Regular User View -->
        <template v-else-if="!appStore.backendModeEnabled">
          <div
            v-for="section in userNavGroups"
            :key="section.key"
            class="sidebar-section"
          >
            <div
              class="sidebar-section-title"
              :class="{ 'sidebar-section-title-collapsed': effectiveCollapsed }"
              :aria-hidden="effectiveCollapsed ? 'true' : 'false'"
            >
              <span
                class="sidebar-section-title-text"
                :class="{ 'sidebar-section-title-text-collapsed': effectiveCollapsed }"
              >
                {{ section.title }}
              </span>
            </div>

            <div class="space-y-0.5">
              <TooltipRoot
                v-for="item in section.items"
                :key="item.path"
                :disabled="!effectiveCollapsed"
              >
                <TooltipTrigger as-child>
                  <router-link
                    :to="item.path"
                    class="group sidebar-link"
                    :class="[
                      isActive(item.path) && 'sidebar-link-active',
                      effectiveCollapsed && 'mx-auto size-8 justify-center px-0'
                    ]"
                    :aria-label="effectiveCollapsed ? item.label : undefined"
                    :data-tour="item.path === '/keys' ? 'sidebar-my-keys' : undefined"
                    @click="handleMenuItemClick(item.path)"
                  >
                    <span
                      v-if="item.iconSvg"
                      class="size-[15px] shrink-0 sidebar-svg-icon transition-colors"
                      :class="
                        isActive(item.path)
                          ? 'text-foreground'
                          : 'text-muted-foreground/80 group-hover:text-foreground'
                      "
                      v-html="sanitizeSvg(item.iconSvg)"
                    ></span>
                    <component
                      v-else
                      :is="item.icon"
                      :stroke-width="1.75"
                      class="size-[15px] shrink-0 transition-colors"
                      :class="
                        isActive(item.path)
                          ? 'text-foreground'
                          : 'text-muted-foreground/80 group-hover:text-foreground'
                      "
                    />
                    <span v-if="!effectiveCollapsed" class="sidebar-label">{{ item.label }}</span>
                  </router-link>
                </TooltipTrigger>
                <TooltipPortal>
                  <TooltipContent
                    side="right"
                    :side-offset="8"
                    class="z-50 rounded-md border border-border bg-popover px-2.5 py-1 text-xs font-medium text-popover-foreground shadow-md"
                  >
                    {{ item.label }}
                  </TooltipContent>
                </TooltipPortal>
              </TooltipRoot>
            </div>
          </div>
        </template>
      </TooltipProvider>
    </nav>

    <!-- Interactive SidebarRail (desktop toggle edge) -->
    <button
      type="button"
      tabindex="-1"
      aria-label="Toggle Sidebar"
      :title="sidebarCollapsed ? t('nav.expand') : t('nav.collapse')"
      class="group absolute inset-y-0 -right-1.5 z-20 hidden w-3 cursor-ew-resize items-center justify-center lg:flex"
      @click="toggleSidebar"
    >
      <span class="h-full w-px bg-transparent transition-colors group-hover:bg-sidebar-border" />
    </button>
  </aside>

  <!-- Mobile Overlay -->
  <transition name="fade">
    <div
      v-if="mobileOpen"
      class="fixed inset-0 z-[45] bg-black/50 lg:hidden"
      @click="closeMobile"
    ></div>
  </transition>
</template>

<script setup lang="ts">
import { computed, h, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  Activity,
  ArrowLeftRight,
  BadgePercent,
  Bell,
  BookOpen,
  ChevronRight,
  Coins,
  Cpu,
  CreditCard,
  FileSearch,
  FileText,
  Gauge,
  Globe,
  History,
  Image,
  KeyRound,
  Layers,
  Network,
  Package,
  PieChart,
  Puzzle,
  Radio,
  Receipt,
  Settings,
  Share2,
  ShieldAlert,
  ShieldCheck,
  Signal,
  Sparkles,
  Tag,
  Ticket,
  TrendingUp,
  User,
  UserPlus,
  Users,
  Wallet,
} from '@lucide/vue'
import {
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuRoot,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  TooltipContent,
  TooltipPortal,
  TooltipProvider,
  TooltipRoot,
  TooltipTrigger,
} from 'reka-ui'
import { useAdminSettingsStore, useAppStore, useAuthStore, useOnboardingStore } from '@/stores'
import { useThemeStore } from '@/stores/theme'
import VersionBadge from '@/components/common/VersionBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeSvg } from '@/utils/sanitize'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, makeSidebarFlag } from '@/utils/featureFlags'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'

type UserNavGroupKey = 'workspace' | 'services' | 'billing' | 'account'

interface NavItem {
  path: string
  label: string
  icon: unknown
  iconSvg?: string
  group?: UserNavGroupKey
  hideInSimpleMode?: boolean
  children?: NavItem[]
  /**
   * When true, the parent item only toggles the expand/collapse state and
   * does NOT navigate to its `path`. The `path` is purely a stable key.
   */
  expandOnly?: boolean
  /**
   * 可选的功能开关 getter。返回 false 时菜单项被隐藏；返回 undefined/true 时显示。
   * 宽容策略（undefined → 显示）避免 public settings 未加载完成时菜单闪烁消失。
   * Getter 里访问的 reactive 来源（store / composable）会被 computed 自动追踪，
   * 开关切换时菜单自动更新。
   */
  featureFlag?: () => boolean | undefined
}

interface NavSection {
  key: string
  title: string
  items: NavItem[]
}

// applyFeatureFlags 递归过滤掉 featureFlag() === false 的节点（含子节点）。
// 使用 `!== false` 宽容语义：undefined（设置未加载）或 true 都视为显示。
function applyFeatureFlags(items: NavItem[]): NavItem[] {
  const out: NavItem[] = []
  for (const item of items) {
    if (item.featureFlag && item.featureFlag() === false) continue
    if (item.children) {
      out.push({ ...item, children: applyFeatureFlags(item.children) })
    } else {
      out.push(item)
    }
  }
  return out
}

const { t } = useI18n()

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const onboardingStore = useOnboardingStore()
const adminSettingsStore = useAdminSettingsStore()
const themeStore = useThemeStore()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()

const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const mobileOpen = computed(() => appStore.mobileOpen)
const effectiveCollapsed = computed(() => sidebarCollapsed.value && !mobileOpen.value)
const isAdmin = computed(() => authStore.isAdmin)
const sidebarNavRef = ref<HTMLElement | null>(null)

const homePath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))

// Per-group expand/collapse overrides. A group with no entry follows the
// automatic behavior (expanded while the active route is one of its children);
// a chevron click records the user's choice, which wins over the automatic
// state so an active group can still be collapsed manually.
const groupExpandOverrides = ref<Map<string, boolean>>(new Map())

// Collapsible personal section for admins (collapsed by default unless a personal
// route is active or the onboarding tour is running; manual click wins).
const personalSectionOverride = ref<boolean | null>(null)

// Site settings from appStore (cached, no flicker)
const siteName = computed(() => appStore.siteName)
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteVersion = computed(() => appStore.siteVersion)
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const MCPIcon = {
  render: () => h(Icon, { name: 'mcp' })
}

// Public-settings flags go through the registry in utils/featureFlags.ts,
// which handles the opt-in vs opt-out fallback when settings haven't loaded
// yet. Admin-only flags (not in public settings) stay inline below.
const flagChannelMonitor = makeSidebarFlag(FeatureFlags.channelMonitor)
const flagPayment = makeSidebarFlag(FeatureFlags.payment)
const flagAvailableChannels = makeSidebarFlag(FeatureFlags.availableChannels)
const flagSubscription = makeSidebarFlag(FeatureFlags.subscription)

// 购买入口文案随站点计费模式切换：仅充值 → 「充值」，仅订阅 → 「订阅」，否则「充值/订阅」。
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
const flagAffiliate = makeSidebarFlag(FeatureFlags.affiliate)
const flagRiskControl = makeSidebarFlag(FeatureFlags.riskControl)
const flagPluginManagement = makeSidebarFlag(FeatureFlags.pluginManagement)
const flagMCP = makeSidebarFlag(FeatureFlags.mcp)
const flagOpsMonitoring = () => adminSettingsStore.opsMonitoringEnabled
const flagAdminPayment = () => adminSettingsStore.paymentEnabled
const flagBatchImageAccess = () => canUseBatchImage.value

// buildSelfNavItems 构造用户自己的导航项（用户端主菜单和管理员的"我的账户"子菜单共享这组声明）。
// withDashboard=true 时包含仪表盘（用户端），false 时不含（管理员的个人区已经有独立仪表盘入口）。
//
// 每项带有 group 标签，用户端按工作台 / 服务与渠道 / 账单与福利 / 账号设置分组渲染；
// 管理员的「我的账户」保留平铺紧凑列表，不重复维护功能开关。
function buildSelfNavItems(withDashboard: boolean): NavItem[] {
  const items: NavItem[] = []
  if (withDashboard) {
    items.push({ path: '/dashboard', label: t('nav.dashboard'), icon: Activity, group: 'workspace' })
  }
  items.push(
    { path: '/keys', label: t('nav.apiKeys'), icon: KeyRound, group: 'workspace' },
    { path: '/usage', label: t('nav.usage'), icon: FileText, hideInSimpleMode: true, group: 'workspace' },
    { path: '/batch-image', label: t('nav.batchImage'), icon: Image, hideInSimpleMode: true, featureFlag: flagBatchImageAccess, group: 'workspace' },
    { path: '/available-channels', label: t('nav.availableChannels'), icon: Radio, hideInSimpleMode: true, featureFlag: flagAvailableChannels, group: 'services' },
    { path: '/monitor', label: t('nav.channelStatus'), icon: Signal, featureFlag: flagChannelMonitor, group: 'services' },
    { path: '/notifications', label: t('operations.notifications'), icon: Bell, group: 'services' },
    { path: '/subscriptions', label: t('nav.mySubscriptions'), icon: CreditCard, hideInSimpleMode: true, featureFlag: flagSubscription, group: 'billing' },
    { path: '/purchase', label: purchaseNavLabel.value, icon: Wallet, hideInSimpleMode: true, featureFlag: flagPayment, group: 'billing' },
    { path: '/orders', label: t('nav.myOrders'), icon: Receipt, hideInSimpleMode: true, featureFlag: flagPayment, group: 'billing' },
    { path: '/reward-activities', label: t('rewards.title'), icon: Sparkles, hideInSimpleMode: true, group: 'billing' },
    { path: '/redeem', label: t('nav.redeem'), icon: Ticket, hideInSimpleMode: true, group: 'billing' },
    { path: '/affiliate', label: t('nav.affiliate'), icon: Share2, hideInSimpleMode: true, featureFlag: flagAffiliate, group: 'billing' },
    { path: '/profile', label: t('nav.profile'), icon: User, group: 'account' },
    { path: '/settings/mcp', label: t('mcp.title'), icon: MCPIcon, featureFlag: flagMCP, group: 'account' },
    ...customMenuItemsForUser.value.map((item): NavItem => ({
      path: `/custom/${item.id}`,
      label: item.label,
      icon: null,
      iconSvg: item.icon_svg,
      group: 'account',
    })),
  )
  return items
}

// finalizeNav 合并三重过滤：featureFlag 过滤 + simple 模式过滤。
function finalizeNav(items: NavItem[]): NavItem[] {
  const visible = applyFeatureFlags(items)
  return authStore.isSimpleMode ? visible.filter(item => !item.hideInSimpleMode) : visible
}

// User navigation items (for regular users)
const userNavItems = computed((): NavItem[] => finalizeNav(buildSelfNavItems(true)))

// Grouped user navigation sections (only non-empty sections are rendered)
const userNavGroups = computed((): NavSection[] => {
  const items = userNavItems.value
  const sections: NavSection[] = [
    {
      key: 'workspace',
      title: t('nav.sections.workspace'),
      items: items.filter(item => item.group === 'workspace'),
    },
    {
      key: 'services',
      title: t('nav.sections.services'),
      items: items.filter(item => item.group === 'services'),
    },
    {
      key: 'billing',
      title: t('nav.sections.billing'),
      items: items.filter(item => item.group === 'billing'),
    },
    {
      key: 'account',
      title: t('nav.sections.account'),
      items: items.filter(item => item.group === 'account'),
    },
  ]
  return sections.filter(section => section.items.length > 0)
})

// Personal navigation items (for admin's "My Account" section, without Dashboard).
// Admins access 可用渠道 from this section just like regular users — there is no
// separate admin entry, since the page is purely a user-facing view.
const personalNavItems = computed((): NavItem[] => finalizeNav(buildSelfNavItems(false)))

const isPersonalSectionExpanded = computed(() => {
  if (personalSectionOverride.value !== null) return personalSectionOverride.value
  return true
})

function togglePersonalSection() {
  personalSectionOverride.value = !isPersonalSectionExpanded.value
}

// Custom menu items filtered by visibility
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

// Grouped admin navigation sections (only non-empty sections are rendered)
const adminNavGroups = computed((): NavSection[] => {
  const overviewItems: NavItem[] = [
    { path: '/admin/dashboard', label: t('nav.dashboard'), icon: Activity },
    { path: '/admin/cluster', label: t('cluster.title'), icon: Cpu },
    { path: '/admin/ops', label: t('nav.ops'), icon: Gauge, featureFlag: flagOpsMonitoring },
  ]

  const resourcesItems: NavItem[] = [
    { path: '/admin/users', label: t('nav.users'), icon: Users, hideInSimpleMode: true },
    { path: '/admin/registration-sources', label: t('operations.registrationSources'), icon: TrendingUp, hideInSimpleMode: true },
    { path: '/admin/groups', label: t('nav.groups'), icon: Layers },
    {
      path: '/admin/channels',
      label: t('nav.channelManagement'),
      icon: Radio,
      hideInSimpleMode: true,
      expandOnly: true,
      children: [
        { path: '/admin/channels/pricing', label: t('nav.channelPricing'), icon: Tag },
        { path: '/admin/channels/monitor', label: t('nav.channelMonitor'), icon: Signal, featureFlag: flagChannelMonitor },
      ],
    },
    // 「仅充值」站点连管理端的「订阅管理」入口也一并收起（路由本身不拦截）。
    { path: '/admin/subscriptions', label: t('nav.subscriptions'), icon: CreditCard, hideInSimpleMode: true, featureFlag: flagSubscription },
    { path: '/admin/accounts', label: t('nav.accounts'), icon: Globe },
    { path: '/admin/proxies', label: t('nav.proxies'), icon: Network },
  ]

  const operationsItems: NavItem[] = [
    {
      path: '/admin/orders',
      label: t('nav.orderManagement'),
      icon: Receipt,
      hideInSimpleMode: true,
      expandOnly: true,
      featureFlag: flagAdminPayment,
      children: [
        { path: '/admin/orders/dashboard', label: t('nav.paymentDashboard'), icon: PieChart },
        { path: '/admin/orders', label: t('nav.orderManagement'), icon: Receipt },
        { path: '/admin/orders/plans', label: t('nav.paymentPlans'), icon: Package },
      ],
    },
    { path: '/admin/redeem', label: t('nav.redeemCodes'), icon: Ticket, hideInSimpleMode: true },
    { path: '/admin/promo-codes', label: t('nav.promoCodes'), icon: BadgePercent, hideInSimpleMode: true },
    { path: '/admin/reward-activities', label: t('rewards.adminTitle'), icon: Sparkles, hideInSimpleMode: true },
    {
      path: '/admin/affiliates',
      label: t('nav.affiliateManagement'),
      icon: Share2,
      hideInSimpleMode: true,
      expandOnly: true,
      featureFlag: flagAffiliate,
      children: [
        { path: '/admin/affiliates/invites', label: t('nav.affiliateInviteRecords'), icon: UserPlus },
        { path: '/admin/affiliates/rebates', label: t('nav.affiliateRebateRecords'), icon: Coins },
        { path: '/admin/affiliates/transfers', label: t('nav.affiliateTransferRecords'), icon: ArrowLeftRight },
      ],
    },
    { path: '/admin/announcements', label: t('operations.notifications'), icon: Bell },
    { path: '/admin/docs', label: t('docs.adminTitle'), icon: BookOpen, hideInSimpleMode: true },
  ]

  const systemBaseItems: NavItem[] = [
    { path: '/admin/usage', label: t('nav.usage'), icon: FileText },
    {
      path: '/admin/security-audit',
      label: t('nav.securityAudit'),
      icon: ShieldCheck,
      expandOnly: true,
      featureFlag: flagRiskControl,
      children: [
        { path: '/admin/risk-control', label: t('nav.contentModeration'), icon: ShieldAlert },
        { path: '/admin/prompt-audit', label: t('nav.promptAudit'), icon: FileSearch },
      ],
    },
    { path: '/admin/audit-logs', label: t('nav.auditLogs'), icon: History, hideInSimpleMode: true },
    { path: '/admin/plugins', label: t('nav.plugins'), icon: Puzzle, featureFlag: flagPluginManagement },
  ]

  const visibleSystem = applyFeatureFlags(systemBaseItems)
  let systemItems: NavItem[]

  // 简单模式下，在系统设置前插入 API密钥
  if (authStore.isSimpleMode) {
    const filtered = visibleSystem.filter(item => !item.hideInSimpleMode)
    filtered.push({ path: '/keys', label: t('nav.apiKeys'), icon: KeyRound })
    if (flagMCP()) filtered.push({ path: '/settings/mcp', label: t('mcp.title'), icon: MCPIcon })
    filtered.push({ path: '/admin/settings', label: t('nav.settings'), icon: Settings })
    for (const cm of customMenuItemsForAdmin.value) {
      filtered.push({ path: `/custom/${cm.id}`, label: cm.label, icon: null, iconSvg: cm.icon_svg })
    }
    systemItems = filtered
  } else {
    visibleSystem.push({ path: '/admin/settings', label: t('nav.settings'), icon: Settings })
    for (const cm of customMenuItemsForAdmin.value) {
      visibleSystem.push({ path: `/custom/${cm.id}`, label: cm.label, icon: null, iconSvg: cm.icon_svg })
    }
    systemItems = visibleSystem
  }

  const sections: NavSection[] = [
    { key: 'overview', title: t('nav.sections.overview'), items: finalizeNav(overviewItems) },
    { key: 'resources', title: t('nav.sections.resources'), items: finalizeNav(resourcesItems) },
    { key: 'operations', title: t('nav.sections.operations'), items: finalizeNav(operationsItems) },
    { key: 'system', title: t('nav.sections.system'), items: systemItems },
  ]

  return sections.filter(section => section.items.length > 0)
})

function toggleSidebar() {
  appStore.toggleSidebar()
}

function closeMobile() {
  appStore.setMobileOpen(false)
}

function handleMenuItemClick(itemPath: string) {
  if (mobileOpen.value) {
    setTimeout(() => {
      appStore.setMobileOpen(false)
    }, 150)
  }

  // Map paths to tour selectors
  const pathToSelector: Record<string, string> = {
    '/admin/groups': '#sidebar-group-manage',
    '/admin/accounts': '#sidebar-channel-manage',
    '/keys': '[data-tour="sidebar-my-keys"]'
  }

  const selector = pathToSelector[itemPath]
  if (selector && onboardingStore.isCurrentStep(selector)) {
    onboardingStore.nextStep(500)
  }
}

function isActive(path: string): boolean {
  return route.path === path || route.path.startsWith(path + '/')
}

function isGroupActive(item: NavItem): boolean {
  if (!item.children) return false
  return item.children.some(child => route.path === child.path)
}

function isGroupExpanded(item: NavItem): boolean {
  const override = groupExpandOverrides.value.get(item.path)
  if (override !== undefined) return override
  return isGroupActive(item)
}

function toggleGroup(item: NavItem) {
  groupExpandOverrides.value.set(item.path, !isGroupExpanded(item))
}

/**
 * Click handler for collapsible parent items.
 * - When the sidebar shows icons only, the dropdown handles its children.
 * - When `expandOnly` is true: only toggle expand state.
 * - Otherwise (default, e.g. /admin/orders): navigate to the parent path
 *   (router-link semantics) and ensure the group is expanded.
 */
function handleGroupClick(item: NavItem) {
  if (effectiveCollapsed.value) return
  if (item.expandOnly) {
    toggleGroup(item)
    return
  }
  // Push to path and ensure expanded
  if (route.path !== item.path) {
    router.push(item.path)
  }
  groupExpandOverrides.value.set(item.path, true)
}

// Fetch admin settings (for feature-gated nav items like Ops).
watch(
  isAdmin,
  (v) => {
    if (v) {
      adminSettingsStore.fetch()
    }
  },
  { immediate: true }
)

onMounted(() => {
  void refreshBatchImageAccess()
  if (isAdmin.value) {
    adminSettingsStore.fetch()
  }
  // Restore sidebar scroll position after route change re-mounts the component
  if (appStore.sidebarScrollTop > 0 && sidebarNavRef.value) {
    void nextTick(() => {
      if (sidebarNavRef.value) {
        sidebarNavRef.value.scrollTop = appStore.sidebarScrollTop
      }
    })
  }
})

onBeforeUnmount(() => {
  if (sidebarNavRef.value) {
    appStore.sidebarScrollTop = sidebarNavRef.value.scrollTop
  }
})
</script>

<style scoped>
.sidebar-logo {
  flex: 0 0 1.5rem;
  min-width: 1.5rem;
}

.sidebar-brand {
  display: flex;
  min-width: 0;
  flex: 1 1 auto;
  align-items: center;
  gap: 0.5rem;
  white-space: nowrap;
}

.sidebar-brand-title {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-section-title {
  position: relative;
  display: flex;
  align-items: center;
  min-height: 1.5rem;
  overflow: hidden;
  white-space: nowrap;
}

.sidebar-section:first-child .sidebar-section-title-collapsed {
  min-height: 0;
  height: 0;
  margin-bottom: 0;
  padding-top: 0;
  padding-bottom: 0;
}

.sidebar-section-title-text {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition:
    opacity 0.16s ease,
    transform 0.16s ease;
}

.sidebar-section-title::after {
  content: '';
  position: absolute;
  left: 0.5rem;
  right: 0.5rem;
  top: 50%;
  height: 1px;
  background: hsl(var(--sidebar-border));
  opacity: 0;
  transform: translateY(-50%);
  transition: opacity 0.18s ease;
}

.sidebar-section-title-text-collapsed {
  opacity: 0;
  transform: translateX(-4px);
}

.sidebar-section-title-collapsed::after {
  opacity: 1;
  transition-delay: 0.08s;
}

.sidebar-section:first-child .sidebar-section-title-collapsed::after {
  opacity: 0;
}

.sidebar-label {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-label-flex {
  display: flex;
  flex: 1 1 auto;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

/* Custom SVG icon in sidebar: constrain size without overriding uploaded SVG colors */
.sidebar-svg-icon {
  color: currentColor;
}

.sidebar-svg-icon :deep(svg) {
  display: block;
  width: 0.9375rem;
  height: 0.9375rem;
}
</style>
