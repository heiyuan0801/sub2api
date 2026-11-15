import { i18n } from '@/i18n'
import { availableLocales, DEFAULT_LOCALE, isLocaleCode, localizedPublicPath, type LocaleCode } from '@/i18n/localeConfig'

const pages = {
  Home: { key: 'home', path: '/home', type: 'WebPage' },
  ModelPlaza: { key: 'market', path: '/model-plaza', type: 'CollectionPage' },
  DocsIndex: { key: 'docs', path: '/docs', type: 'CollectionPage' },
} as const

type PublicPage = typeof pages[keyof typeof pages]

// These descriptions mirror the initial HTML rendered by backend/internal/web.
export function publicRouteSeo(name: unknown): PublicPage | undefined {
  return typeof name === 'string' && Object.prototype.hasOwnProperty.call(pages, name)
    ? pages[name as keyof typeof pages]
    : undefined
}

export function indexablePublicQuery(query: Readonly<Record<string, unknown>>): boolean {
  return Object.keys(query).every(key => key === 'lang' && isLocaleCode(query.lang))
}

interface PageHead {
  title: string
  siteName: string
  description?: string
  siteLogo?: string
  path?: string
  locale?: LocaleCode
  urlLocale?: LocaleCode
  type?: PublicPage['type'] | 'TechArticle'
  translated?: boolean
  noIndex?: boolean
}

function siteOrigin(): string | undefined {
  const configured = document.querySelector<HTMLMetaElement>('meta[name="public-site-origin"]')?.content
  try {
    const url = new URL(configured || (import.meta.env.DEV ? window.location.origin : ''))
    if ((url.protocol === 'https:' || url.protocol === 'http:') && !url.username && !url.password && !url.search && !url.hash) {
      return url.href.replace(/\/+$/, '')
    }
  } catch {
    // The embedded server sets the public origin only when a valid frontend URL is configured.
  }
  return undefined
}

function socialImage(base: string | undefined, logo: string | undefined): string | undefined {
  if (!logo) return undefined
  try {
    const url = new URL(logo.trim(), base)
    if (['http:', 'https:'].includes(url.protocol) && !url.username && !url.password && !url.search && !url.hash) return url.href
  } catch {
    // Ignore missing or invalid configured images.
  }
  return undefined
}

function setMetadata(attribute: 'name' | 'property', name: string, content?: string) {
  const nodes = [...document.head.querySelectorAll<HTMLMetaElement>(`meta[${attribute}="${name}"]`)]
  const found = nodes.shift()
  nodes.forEach(node => node.remove())
  if (!content) {
    found?.remove()
    return
  }
  const tag = found ?? document.createElement('meta')
  tag.setAttribute(attribute, name)
  tag.setAttribute('data-public-seo', '')
  tag.content = content
  if (!found) document.head.append(tag)
}

function addLink(rel: string, href: string, hreflang?: string) {
  const link = document.createElement('link')
  link.rel = rel
  link.href = href
  if (hreflang) link.hreflang = hreflang
  link.setAttribute('data-public-seo', '')
  document.head.append(link)
}

export function writePageHead({ title, siteName, description, siteLogo, path, locale, urlLocale = DEFAULT_LOCALE, type, translated = false, noIndex = false }: PageHead) {
  document.title = title
  const origin = path && !noIndex ? siteOrigin() : undefined
  const url = origin && path ? origin + (translated ? localizedPublicPath(path, urlLocale) : path) : undefined
  const language = availableLocales.find(item => item.code === locale)
  const isPublic = !noIndex && Boolean(path)
  const image = isPublic ? socialImage(origin, siteLogo) : undefined

  setMetadata('name', 'robots', noIndex ? 'noindex' : undefined)
  setMetadata('name', 'description', isPublic ? description : undefined)
  setMetadata('property', 'og:title', isPublic ? title : undefined)
  setMetadata('property', 'og:description', isPublic ? description : undefined)
  setMetadata('property', 'og:type', isPublic ? (type === 'TechArticle' ? 'article' : 'website') : undefined)
  setMetadata('property', 'og:site_name', isPublic ? siteName : undefined)
  setMetadata('property', 'og:url', url)
  setMetadata('property', 'og:image', image)
  setMetadata('property', 'og:locale', isPublic ? language?.socialTag : undefined)
  setMetadata('name', 'twitter:card', isPublic ? 'summary' : undefined)
  setMetadata('name', 'twitter:title', isPublic ? title : undefined)
  setMetadata('name', 'twitter:description', isPublic ? description : undefined)
  setMetadata('name', 'twitter:image', image)

  document.head.querySelectorAll('link[rel="canonical"], link[rel="alternate"][hreflang]').forEach(node => node.remove())
  if (url) addLink('canonical', url)
  if (translated && origin && path) {
    for (const option of availableLocales) addLink('alternate', origin + localizedPublicPath(path, option.code), option.tag)
    addLink('alternate', origin + path, 'x-default')
  }

  document.getElementById('public-seo-jsonld')?.remove()
  if (url && origin && type) {
    const structuredData = document.createElement('script')
    structuredData.type = 'application/ld+json'
    structuredData.id = 'public-seo-jsonld'
    structuredData.setAttribute('data-public-seo', '')
    structuredData.textContent = JSON.stringify({
      '@context': 'https://schema.org',
      '@graph': [
        { '@type': 'WebSite', '@id': `${origin}/#website`, url: `${origin}/home`, name: siteName },
        {
          '@type': type, '@id': `${url}#webpage`, url, name: title, description,
          ...(language && type !== 'TechArticle' ? { inLanguage: language.tag } : {}),
          isPartOf: { '@id': `${origin}/#website` },
        },
      ],
    })
    document.head.append(structuredData)
  }
}

export function localizedSeoCopy(page: PublicPage) {
  return {
    title: i18n.global.t(`seo.${page.key}.title`),
    description: i18n.global.t(`seo.${page.key}.description`),
  }
}
