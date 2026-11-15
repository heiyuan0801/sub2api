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

export function getSettingsSection(value: unknown): SettingsSection | undefined {
  if (typeof value !== 'string') return undefined
  return SETTINGS_SECTIONS.find(section => section.key === value)
}
