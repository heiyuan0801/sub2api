import {
  ADMIN_SETTINGS_INDEX_PATH,
  ADMIN_SETTINGS_ROUTE_NAME,
  DEFAULT_SETTINGS_SECTION,
  getSettingsSection,
  settingsSectionPath
} from './settingsSections'

export const ADMIN_SECTIONS = ['operations', 'resources', 'billing', 'system'] as const
export type AdminSection = (typeof ADMIN_SECTIONS)[number]

export interface AdminRouteEntry {
  readonly name: string
  readonly path: `/admin/${string}`
  readonly sidebarPath?: `/admin/${string}`
}

export interface AdminRouteGroup {
  readonly section: AdminSection
  readonly routes: readonly AdminRouteEntry[]
}

// Presentation inventory for the sidebar, breadcrumb and command palette.
// Router records and guards remain the source of truth for authorization.
export const ADMIN_ROUTE_GROUPS: readonly AdminRouteGroup[] = [
  {
    section: 'operations',
    routes: [
      { name: 'AdminDashboard', path: '/admin/dashboard' },
      { name: 'AdminUsage', path: '/admin/usage' },
      { name: 'AdminOps', path: '/admin/ops' },
      { name: 'AdminCluster', path: '/admin/cluster' },
    ],
  },
  {
    section: 'resources',
    routes: [
      { name: 'AdminAccounts', path: '/admin/accounts' },
      { name: 'AdminChannels', path: '/admin/channels/pricing' },
      { name: 'AdminChannelMonitor', path: '/admin/channels/monitor' },
      { name: 'AdminGroups', path: '/admin/groups' },
      { name: 'AdminProxies', path: '/admin/proxies' },
    ],
  },
  {
    section: 'billing',
    routes: [
      { name: 'AdminUsers', path: '/admin/users' },
      { name: 'AdminSubscriptions', path: '/admin/subscriptions' },
      { name: 'AdminPaymentDashboard', path: '/admin/orders/dashboard' },
      { name: 'AdminOrders', path: '/admin/orders' },
      { name: 'AdminPaymentPlans', path: '/admin/orders/plans' },
      { name: 'AdminRedeem', path: '/admin/redeem' },
      { name: 'AdminPromoCodes', path: '/admin/promo-codes' },
      { name: 'AdminAffiliateInvites', path: '/admin/affiliates/invites' },
      { name: 'AdminAffiliateRebates', path: '/admin/affiliates/rebates' },
      { name: 'AdminAffiliateTransfers', path: '/admin/affiliates/transfers' },
      { name: 'AdminRewardActivities', path: '/admin/reward-activities' },
    ],
  },
  {
    section: 'system',
    routes: [
      { name: 'AdminDocs', path: '/admin/docs' },
      { name: 'AdminAnnouncements', path: '/admin/announcements' },
      {
        name: ADMIN_SETTINGS_ROUTE_NAME,
        path: '/admin/settings/:section',
        sidebarPath: ADMIN_SETTINGS_INDEX_PATH,
      },
      { name: 'AdminPlugins', path: '/admin/plugins' },
      { name: 'AdminAuditLogs', path: '/admin/audit-logs' },
      { name: 'AdminRiskControl', path: '/admin/risk-control' },
      { name: 'AdminPromptAudit', path: '/admin/prompt-audit' },
    ],
  },
]

export const ADMIN_ROUTE_REDIRECTS: Readonly<Record<string, AdminRouteEntry['path']>> = {
  '/admin': '/admin/dashboard',
  '/admin/channels': '/admin/channels/pricing',
  '/admin/affiliates': '/admin/affiliates/invites',
  [ADMIN_SETTINGS_INDEX_PATH]: settingsSectionPath(DEFAULT_SETTINGS_SECTION),
}

export const ADMIN_SECTION_LABEL_KEYS: Readonly<Record<AdminSection, string>> = {
  operations: 'design.shell.operations',
  resources: 'design.shell.resources',
  billing: 'design.shell.billing',
  system: 'design.shell.system',
}

export function getAdminRouteSection(name: string | symbol | null | undefined, path?: string): AdminSection | undefined {
  const normalizedPath = path?.replace(/\/+$/, '')
  const canonicalPath = normalizedPath && (ADMIN_ROUTE_REDIRECTS[normalizedPath] ?? normalizedPath)
  const settingsPrefix = `${ADMIN_SETTINGS_INDEX_PATH}/`
  const settingsSection = canonicalPath?.startsWith(settingsPrefix)
    ? getSettingsSection(canonicalPath.slice(settingsPrefix.length))
    : undefined
  return ADMIN_ROUTE_GROUPS.find(group => group.routes.some(route =>
    route.name === name || route.path === canonicalPath ||
    (route.name === ADMIN_SETTINGS_ROUTE_NAME && settingsSection !== undefined)
  ))?.section
}
