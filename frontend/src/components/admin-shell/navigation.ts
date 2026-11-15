import type { Component } from 'vue'
import type { Router } from 'vue-router'
import type { PublicSettings } from '@/types'

/** Commands use the same filtered links already rendered in the sidebar. */
export interface SidebarCommandLink {
  path: string
  label: string
  icon?: Component
  description?: string
}

interface ConsoleAccess {
  authenticated: boolean
  admin: boolean
  backendOnly: boolean
  settings?: Pick<PublicSettings, 'payment_enabled' | 'risk_control_enabled' | 'subscription_enabled'> | null
}

/** Resolve actual destinations and reject stale sidebar entries; router guards still apply. */
export function selectConsoleLinks(
  links: SidebarCommandLink[],
  access: ConsoleAccess,
  resolve: Router['resolve']
): SidebarCommandLink[] {
  if (!access.authenticated || (!access.admin && access.backendOnly)) return []
  const seen = new Set<string>()
  return links.filter(link => {
    if (seen.has(link.path)) return false
    const route = resolve(link.path)
    if (!route.matched.length || route.name === 'NotFound') return false
    if (!access.admin && (route.meta.requiresAdmin || route.path === '/admin' || route.path.startsWith('/admin/'))) return false
    if (route.meta.requiresPayment && access.settings?.payment_enabled === false) return false
    if (route.meta.requiresRiskControl && access.settings?.risk_control_enabled === false) return false
    if (route.meta.requiresSubscription && access.settings?.subscription_enabled === false) return false
    seen.add(link.path)
    return true
  })
}
