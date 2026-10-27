import { describe, it, expect } from 'vitest'
import { buildMagpieImport } from '../magpieImport'
const base = { baseUrl: 'https://gateway.example/v1/', providerName: '中文 & Provider', apiKey: 'sk-test+&?#=', keyId: 42 }
describe('Magpie provider import', () => {
  it('encodes the key and places web fallback parameters only in the fragment', () => {
    const result = buildMagpieImport({ ...base, platform: 'openai' })
    const direct = new URL(result.deeplink), fallback = new URL(result.fallback)
    expect(direct.protocol).toBe('magpie:')
    expect(direct.searchParams.get('key')).toBe(base.apiKey)
    expect(direct.searchParams.get('name')).toBe(base.providerName)
    expect(direct.searchParams.get('chat')).toBe('https://gateway.example/v1')
    expect(direct.searchParams.get('responses')).toBe('https://gateway.example/v1')
    expect(direct.searchParams.has('models')).toBe(false)
    expect(fallback.search).toBe('')
    expect(new URLSearchParams(fallback.hash.slice(1)).get('key')).toBe(base.apiKey)
  })
  it.each(['https://user:pass@gateway.example', 'https://gateway.example?key=bad', 'https://gateway.example/#bad', 'javascript:alert(1)', 'http://public.example'])('rejects unsafe base URLs %s', baseUrl => {
    expect(() => buildMagpieImport({ ...base, baseUrl })).toThrow()
  })
  it('supports local previews and preserves custom paths', () => {
    expect(buildMagpieImport({ ...base, baseUrl: 'http://127.0.0.1:28180/prefix/v1beta', platform: 'gemini' }).endpoints).toEqual({ gemini: 'http://127.0.0.1:28180/prefix/v1beta' })
  })
  it('uses provider-specific protocols without speculative probes', () => {
    expect(buildMagpieImport({ ...base, platform: 'anthropic' }).endpoints).toEqual({ anthropic: 'https://gateway.example' })
    expect(buildMagpieImport({ ...base, platform: 'antigravity' }).endpoints).toEqual({ anthropic: 'https://gateway.example/antigravity', gemini: 'https://gateway.example/antigravity/v1beta' })
    expect(buildMagpieImport({ ...base, platform: 'grok' }).endpoints).toEqual({ chat: 'https://gateway.example/v1' })
  })
})
