import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import router from '@/router'
import {
  ADMIN_ROUTE_GROUPS,
  ADMIN_ROUTE_REDIRECTS,
  ADMIN_SECTIONS,
  getAdminRouteSection,
} from '../adminRoutes'
import {
  ADMIN_SETTINGS_ROUTE_NAME,
  DEFAULT_SETTINGS_SECTION,
  settingsSectionLocation,
  settingsSectionPath
} from '../settingsSections'

describe('admin route navigation inventory', () => {
  it('includes every registered admin page once in one of the four sections', () => {
    const adminRecords = router.getRoutes().filter(route => route.meta.requiresAdmin && !route.aliasOf)
    const configuredRoutes = ADMIN_ROUTE_GROUPS.flatMap(group => group.routes)
    expect(ADMIN_ROUTE_GROUPS.map(group => group.section)).toEqual(ADMIN_SECTIONS)
    expect(configuredRoutes.map(route => route.name).sort()).toEqual(adminRecords.map(route => route.name).sort())
    expect(configuredRoutes.map(route => route.path).sort()).toEqual(adminRecords.map(route => route.path).sort())

    for (const group of ADMIN_ROUTE_GROUPS) {
      for (const entry of group.routes) {
        const settingsRoute = entry.name === ADMIN_SETTINGS_ROUTE_NAME
        const resolved = router.resolve(settingsRoute
          ? settingsSectionLocation(DEFAULT_SETTINGS_SECTION)
          : { name: entry.name })
        expect(resolved.path).toBe(settingsRoute ? settingsSectionPath(DEFAULT_SETTINGS_SECTION) : entry.path)
        expect(resolved.meta.requiresAuth).toBe(true)
        expect(getAdminRouteSection(resolved.name, resolved.path)).toBe(group.section)
      }
    }
  })

  it('groups actual admin redirects with their destinations without duplicate menu entries', () => {
    const redirects = router.getRoutes().filter(route => /^\/admin(?:\/|$)/.test(route.path) && route.redirect)
    expect(redirects.map(route => route.path).sort()).toEqual(Object.keys(ADMIN_ROUTE_REDIRECTS).sort())
    for (const route of redirects) {
      const target = ADMIN_ROUTE_REDIRECTS[route.path]
      const redirectTarget = typeof route.redirect === 'function'
        ? route.redirect(router.resolve(route.path), router.currentRoute.value)
        : route.redirect
      expect(redirectTarget && router.resolve(redirectTarget).path).toBe(target)
      expect(getAdminRouteSection(undefined, route.path)).toBeDefined()
      expect(getAdminRouteSection(undefined, route.path)).toBe(getAdminRouteSection(undefined, target))
    }
    expect(getAdminRouteSection('Dashboard', '/dashboard')).toBeUndefined()
  })

  it('keeps every grouped destination in the permission-filtered sidebar source', () => {
    const sidebar = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../../layout/AppSidebar.vue'), 'utf8')
    const sidebarPaths = [...sidebar.matchAll(/path: '(\/admin\/[^']+)'/g)].map(match => match[1])
    for (const entry of ADMIN_ROUTE_GROUPS.flatMap(group => group.routes)) {
      expect(sidebarPaths).toContain(entry.sidebarPath ?? entry.path)
    }
  })
})
