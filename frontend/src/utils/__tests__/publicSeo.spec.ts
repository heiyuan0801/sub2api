import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { indexablePublicQuery, writePageHead } from '@/utils/publicSeo'

vi.mock('@/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }))

const originTag = '<meta name="public-site-origin" content="https://gateway.example/portal">'
const canonical = () => document.querySelector<HTMLLinkElement>('link[rel="canonical"]')?.href
const metadata = (name: string) => document.querySelector<HTMLMetaElement>(`meta[name="${name}"]`)?.content

let initialHead: string
beforeEach(() => {
  initialHead = document.head.innerHTML
  document.head.innerHTML = originTag
})
afterEach(() => { document.head.innerHTML = initialHead })

describe('public page SEO', () => {
  it('canonicalizes the translated market and advertises every localized URL', () => {
    writePageHead({
      title: 'モデルマーケット', description: 'AIモデルの料金を比較', siteName: 'Gateway',
      path: '/model-plaza', type: 'CollectionPage', locale: 'ja', urlLocale: 'ja', translated: true,
    })
    expect(canonical()).toBe('https://gateway.example/portal/model-plaza?lang=ja')
    const alternates = [...document.querySelectorAll<HTMLLinkElement>('link[hreflang]')]
    expect(alternates.map(link => link.hreflang)).toEqual(['en', 'zh-Hans', 'zh-Hant', 'ja', 'ko', 'x-default'])
    expect(alternates.find(link => link.hreflang === 'zh-Hant')?.href).toBe('https://gateway.example/portal/model-plaza?lang=zh-TW')
    const script = document.getElementById('public-seo-jsonld')
    expect(script?.getAttribute('type')).toBe('application/ld+json')
    expect(JSON.parse(script?.textContent ?? '{}')['@graph'][1]).toMatchObject({ '@type': 'CollectionPage', url: canonical() })
    expect(metadata('robots')).toBeUndefined()
  })

  it('clears metadata and structured data when a navigation switches to a private route', () => {
    writePageHead({ title: 'Home', siteName: 'Gateway', path: '/home', translated: true, type: 'WebPage' })
    writePageHead({ title: 'Dashboard', siteName: 'Gateway', noIndex: true })
    expect(canonical()).toBeUndefined()
    expect(document.querySelectorAll('link[hreflang], meta[property^="og:"], #public-seo-jsonld')).toHaveLength(0)
    expect(metadata('robots')).toBe('noindex')
    expect(metadata('public-site-origin')).toBe('https://gateway.example/portal')
  })

  it('limits indexable URLs to clean pages and known language parameters', () => {
    expect(indexablePublicQuery({})).toBe(true)
    expect(indexablePublicQuery({ lang: 'ko' })).toBe(true)
    expect(indexablePublicQuery({ lang: 'unknown' })).toBe(false)
    expect(indexablePublicQuery({ lang: ['zh', 'en'] })).toBe(false)
    expect(indexablePublicQuery({ embedded: '1' })).toBe(false)
    expect(indexablePublicQuery({ lang: 'ja', search: 'gpt' })).toBe(false)
  })
})
