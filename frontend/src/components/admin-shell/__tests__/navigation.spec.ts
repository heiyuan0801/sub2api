import { describe, expect, it } from 'vitest'
import router from '@/router'
import { selectConsoleLinks } from '../navigation'

const links = [
  { path: '/keys', label: 'Keys' },
  { path: '/keys', label: 'Keys (duplicated)' },
  { path: '/subscriptions', label: 'Subscriptions' },
  { path: '/admin/settings', label: 'Settings' },
  { path: '/admin/orders', label: 'Orders' },
  { path: '/admin/risk-control', label: 'Risk control' },
  { path: '/unregistered', label: 'Unregistered' }
]

describe('console command access', () => {
  it('removes admin, unknown and duplicate entries for ordinary users', () => {
    const visible = selectConsoleLinks(links, {
      authenticated: true, admin: false, backendOnly: false
    }, router.resolve)
    expect(visible.map(link => link.path)).toEqual(['/keys', '/subscriptions'])
  })

  it('does not suggest routes blocked by explicitly disabled features', () => {
    const visible = selectConsoleLinks(links, {
      authenticated: true, admin: true, backendOnly: false,
      settings: { payment_enabled: false, risk_control_enabled: false, subscription_enabled: false }
    }, router.resolve)
    expect(visible.map(link => link.path)).toEqual(['/keys', '/admin/settings'])
  })

  it('omits destinations when access to the user console is unavailable', () => {
    expect(selectConsoleLinks(links, {
      authenticated: false, admin: false, backendOnly: false
    }, router.resolve)).toEqual([])
    expect(selectConsoleLinks(links, {
      authenticated: true, admin: false, backendOnly: true
    }, router.resolve)).toEqual([])
  })
})
