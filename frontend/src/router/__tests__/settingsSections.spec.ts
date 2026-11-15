import { describe, expect, it } from 'vitest'
import { SETTINGS_SECTIONS, settingsSectionLocation } from '@/components/admin-shell/settingsSections'
import router from '@/router'
import { redirectSettingsIndex, resolveSettingsSectionRedirect } from '../settingsSections'

describe('admin settings routes', () => {
  it('keeps every section on the same protected route record', () => {
    const resolved = SETTINGS_SECTIONS.map(({ key }) => router.resolve(settingsSectionLocation(key)))
    for (const [index, section] of resolved.entries()) {
      expect(section.path).toBe(`/admin/settings/${SETTINGS_SECTIONS[index].key}`)
      expect(section.name).toBe('AdminSettings')
      expect(section.meta).toMatchObject({ requiresAuth: true, requiresAdmin: true })
      expect(section.matched[0]).toBe(resolved[0].matched[0])
    }
  })

  it.each([
    ['/admin/settings', '/admin/settings/general'],
    ['/admin/settings?tab=email', '/admin/settings/email'],
    ['/admin/settings?section=payment&preview=1#billing-panel', '/admin/settings/payment?preview=1#billing-panel'],
    ['/admin/settings#backup', '/admin/settings/backup'],
    ['/admin/settings#settings-tab-security', '/admin/settings/security'],
    ['/admin/settings?section=unknown&tab=users#security', '/admin/settings/users'],
    ['/admin/settings?section=email&section=backup#unknown-anchor', '/admin/settings/general#unknown-anchor']
  ])('redirects legacy URL %s to %s', (legacy, canonical) => {
    const target = redirectSettingsIndex(router.resolve(legacy))
    expect(router.resolve(target).fullPath).toBe(canonical)
  })

  it.each([
    ['/admin/settings/unknown#features', '/admin/settings/features'],
    ['/admin/settings/unknown?foo=bar#extra', '/admin/settings/general?foo=bar#extra'],
    ['/admin/settings/email?tab=backup#template-editor', '/admin/settings/email#template-editor']
  ])('normalizes a noncanonical section URL %s to %s', (input, canonical) => {
    const target = resolveSettingsSectionRedirect(router.resolve(input))
    expect(target && router.resolve(target).fullPath).toBe(canonical)
  })

  it('leaves a canonical section and unrelated anchor in place', () => {
    const route = router.resolve('/admin/settings/email?preview=1#template-editor')
    expect(resolveSettingsSectionRedirect(route)).toBeUndefined()
  })
})
