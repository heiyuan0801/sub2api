export const availableLocales = [
  { code: 'en', tag: 'en', name: 'English', flag: '🇺🇸', socialTag: 'en_US' },
  { code: 'zh', tag: 'zh-Hans', name: '简体中文', flag: '🇨🇳', socialTag: 'zh_CN' },
  { code: 'zh-TW', tag: 'zh-Hant', name: '繁體中文', flag: '🇨🇳', socialTag: 'zh_TW' },
  { code: 'ja', tag: 'ja', name: '日本語', flag: '🇯🇵', socialTag: 'ja_JP' },
  { code: 'ko', tag: 'ko', name: '한국어', flag: '🇰🇷', socialTag: 'ko_KR' },
] as const

export type LocaleCode = typeof availableLocales[number]['code']
export const DEFAULT_LOCALE: LocaleCode = 'en'

export function isLocaleCode(value: unknown): value is LocaleCode {
  return availableLocales.some(locale => locale.code === value)
}

export function matchBrowserLocale(language: string): LocaleCode | undefined {
  const tag = language.toLowerCase()
  if (/^zh-(?:hant|tw|hk|mo)(?:-|$)/.test(tag)) return 'zh-TW'
  const primary = tag.split('-')[0]
  return isLocaleCode(primary) ? primary : undefined
}

export function localeHasPublicUrl(path: string): boolean {
  return /^\/(?:home|model-plaza|model-market|models|docs(?:\/[a-z0-9-]+)?)\/?$/.test(path)
}

/** Keeps existing query arguments and anchors when linking to a localized public page. */
export function localizedPublicPath(path: string, locale: string): string {
  const url = new URL(path, 'https://public.invalid')
  if (localeHasPublicUrl(url.pathname) && isLocaleCode(locale)) {
    if (locale === DEFAULT_LOCALE) url.searchParams.delete('lang')
    else url.searchParams.set('lang', locale)
  }
  return url.pathname + url.search + url.hash
}
