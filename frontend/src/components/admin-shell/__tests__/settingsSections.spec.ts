import { describe, expect, it } from 'vitest'
import { getSettingsSection } from '../settingsSections'

describe('settings section deep links', () => {
  it('resolves direct links to real settings tabs', () => {
    expect(getSettingsSection('backup')?.labelKey).toBe('admin.settings.tabs.backup')
    expect(getSettingsSection('payment')?.labelKey).toBe('admin.settings.tabs.payment')
  })

  it('ignores empty, unknown and ambiguous section queries', () => {
    for (const value of [undefined, null, '', 'not-a-tab', ['security', 'backup']]) {
      expect(getSettingsSection(value)).toBeUndefined()
    }
  })
})
