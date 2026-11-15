import type { RegistrationAttribution, RegistrationSource } from '@/types/operations'

const storageKey = 'sub2api_registration_utm'
const utmKeys = ['utm_source', 'utm_medium', 'utm_campaign', 'utm_content', 'utm_term'] as const
type Campaign = Pick<RegistrationAttribution, typeof utmKeys[number]>

function clean(value: string | null | undefined, length: number): string {
  return [...(value ?? '').replace(/\p{Cc}/gu, '').trim()].slice(0, length).join('')
}

export const registrationUTMValueLimit = 64

export function normalizeRegistrationUTMValue(value: string | null | undefined): string {
  return clean(value, registrationUTMValueLimit)
}

export function captureRegistrationCampaign(search: string): void {
  try {
    if (sessionStorage.getItem(storageKey)) return
    const params = new URLSearchParams(search)
    const campaign: Campaign = {}
    for (const key of utmKeys) {
      const value = normalizeRegistrationUTMValue(params.get(key))
      if (value) campaign[key] = value
    }
    if (Object.keys(campaign).length) sessionStorage.setItem(storageKey, JSON.stringify(campaign))
  } catch { /* attribution is optional */ }
}

export function buildRegistrationAttribution(
  source?: RegistrationSource | '',
  note?: string
): RegistrationAttribution | undefined {
  captureRegistrationCampaign(window.location.search)
  const result: RegistrationAttribution = {}
  try {
    const cached: Campaign = JSON.parse(sessionStorage.getItem(storageKey) || '{}')
    for (const key of utmKeys) {
      const value = typeof cached[key] === 'string' ? normalizeRegistrationUTMValue(cached[key]) : ''
      if (value) result[key] = value
    }
  } catch { /* still accept the manually selected source */ }
  if (source) {
    result.selected_source = source
    const sourceNote = source === 'other' ? clean(note, 140) : ''
    if (sourceNote) result.source_note = sourceNote
  }
  return Object.keys(result).length ? result : undefined
}

export function clearRegistrationCampaign(): void {
  try {
    sessionStorage.removeItem(storageKey)
    sessionStorage.removeItem(oauthStorageKey)
  } catch { /* optional */ }
}

const oauthStorageKey = 'sub2api_oauth_registration_attribution'

export function storeOAuthRegistrationAttribution(value?: RegistrationAttribution) {
  try {
    if (value) sessionStorage.setItem(oauthStorageKey, JSON.stringify(value))
    else sessionStorage.removeItem(oauthStorageKey)
  } catch { /* optional */ }
}

export function oauthRegistrationAttributionPayload(): { registration_attribution?: RegistrationAttribution } {
  try {
    const attribution = sessionStorage.getItem(oauthStorageKey)
    if (attribution) return { registration_attribution: JSON.parse(attribution) }
  } catch { /* optional */ }
  return {}
}
