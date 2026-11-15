import type { RouteLocationNormalized, RouteLocationRaw } from 'vue-router'
import {
  DEFAULT_SETTINGS_SECTION,
  getSettingsSection,
  SETTINGS_SECTION_QUERY_KEY,
  settingsSectionLocation
} from '@/components/admin-shell/settingsSections'

type SettingsRedirectInput = Pick<RouteLocationNormalized, 'params' | 'query' | 'hash'>

function settingsSectionFromHash(hash: string) {
  return getSettingsSection(hash.slice(1).replace(/^(?:settings-tab-|tab-|section=|tab=)/, ''))
}

/** Canonical links and old query/hash links share the same section list. */
export function resolveSettingsSectionRedirect(to: SettingsRedirectInput): RouteLocationRaw | undefined {
  const pathSection = getSettingsSection(to.params.section)
  const hasLegacyQuery = SETTINGS_SECTION_QUERY_KEY in to.query || 'tab' in to.query
  const hashSection = settingsSectionFromHash(to.hash)
  if (pathSection && !hasLegacyQuery && !hashSection) return undefined

  const { [SETTINGS_SECTION_QUERY_KEY]: section, tab, ...query } = to.query
  const destination = pathSection ?? getSettingsSection(section) ?? getSettingsSection(tab) ?? hashSection
  return {
    ...settingsSectionLocation(destination?.key ?? DEFAULT_SETTINGS_SECTION),
    query,
    hash: hashSection ? '' : to.hash
  }
}

export function redirectSettingsIndex(to: SettingsRedirectInput): RouteLocationRaw {
  return resolveSettingsSectionRedirect({ ...to, params: {} }) ?? settingsSectionLocation(DEFAULT_SETTINGS_SECTION)
}
