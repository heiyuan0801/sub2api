export const ADMIN_SETTINGS_INDEX_PATH = '/admin/settings'
export const ADMIN_SETTINGS_ROUTE_NAME = 'AdminSettings'
export const SETTINGS_SECTION_QUERY_KEY = 'section'

export const SETTINGS_SECTIONS = [
  { key: 'general', labelKey: 'admin.settings.tabs.general', icon: 'home' },
  { key: 'agreement', labelKey: 'admin.settings.tabs.agreement', icon: 'document' },
  { key: 'features', labelKey: 'admin.settings.tabs.features', icon: 'bolt' },
  { key: 'security', labelKey: 'admin.settings.tabs.security', icon: 'shield' },
  { key: 'users', labelKey: 'admin.settings.tabs.users', icon: 'user' },
  { key: 'gateway', labelKey: 'admin.settings.tabs.gateway', icon: 'server' },
  { key: 'payment', labelKey: 'admin.settings.tabs.payment', icon: 'creditCard' },
  { key: 'email', labelKey: 'admin.settings.tabs.email', icon: 'mail' },
  { key: 'backup', labelKey: 'admin.settings.tabs.backup', icon: 'database' },
] as const

export type SettingsSection = (typeof SETTINGS_SECTIONS)[number]
export type SettingsSectionKey = SettingsSection['key']

export const DEFAULT_SETTINGS_SECTION = SETTINGS_SECTIONS[0].key

export function getSettingsSection(value: unknown): SettingsSection | undefined {
  if (typeof value !== 'string') return undefined
  return SETTINGS_SECTIONS.find(section => section.key === value)
}

export function settingsSectionLocation(section: SettingsSectionKey) {
  return { name: ADMIN_SETTINGS_ROUTE_NAME, params: { section } } as const
}

export function settingsSectionPath(section: SettingsSectionKey) {
  return `${ADMIN_SETTINGS_INDEX_PATH}/${section}` as const
}
