import type { GroupPlatform } from '@/types'

export interface MagpieImportInput {
  baseUrl: string
  providerName: string
  apiKey: string
  keyId: number
  platform?: GroupPlatform | null
}

/** Official protocol: https://usemagpie.ai/docs/import. Never probe inference endpoints. */
export function buildMagpieImport(input: MagpieImportInput) {
  const url = new URL(input.baseUrl)
  const loopback = ['localhost', '127.0.0.1', '[::1]'].includes(url.hostname)
  if ((url.protocol !== 'https:' && !(url.protocol === 'http:' && loopback)) || url.username || url.password || url.search || url.hash) {
    throw new Error('Invalid provider URL')
  }
  if (!input.apiKey.trim()) throw new Error('Missing API key')
  const root = url.href.replace(/\/+$/, '').replace(/\/(v1|v1beta)$/, '')
  const params = new URLSearchParams({
    name: input.providerName.trim().slice(0, 80) || 'Sub2API',
    id: `sub2api-${url.hostname.replace(/[^a-z0-9-]/gi, '-')}-${input.keyId}`,
    key: input.apiKey
  })
  const endpoints: Record<string, string> = {}
  switch (input.platform) {
    case 'anthropic':
    case 'kiro': endpoints.anthropic = root; break
    case 'gemini': endpoints.gemini = `${root}/v1beta`; break
    case 'antigravity':
      endpoints.anthropic = `${root}/antigravity`
      endpoints.gemini = `${root}/antigravity/v1beta`
      break
    case 'openai':
      endpoints.chat = `${root}/v1`
      endpoints.responses = `${root}/v1`
      break
    case 'composite':
      endpoints.chat = `${root}/v1`
      endpoints.responses = `${root}/v1`
      endpoints.anthropic = root
      endpoints.gemini = `${root}/v1beta`
      break
    default: endpoints.chat = `${root}/v1`
  }
  for (const [protocol, endpoint] of Object.entries(endpoints)) params.set(protocol, endpoint)
  // HTTPS fallback parameters stay in the fragment, not server query/access logs.
  return { deeplink: `magpie://import?${params}`, fallback: `https://usemagpie.ai/import#${params}`, endpoints }
}
