import { createI18n } from 'vue-i18n'
import { availableLocales, DEFAULT_LOCALE, isLocaleCode, localeHasPublicUrl, matchBrowserLocale, type LocaleCode } from './localeConfig'

export { availableLocales } from './localeConfig'

type LocaleMessages = Record<string, any>
const LOCALE_KEY = 'sub2api_locale'

const localeLoaders: Record<LocaleCode, () => Promise<{ default: LocaleMessages }>> = {
  en: () => import('./locales/en'),
  zh: () => import('./locales/zh'),
  'zh-TW': () => import('./locales/zh-TW'),
  ja: () => import('./locales/ja'),
  ko: () => import('./locales/ko')
}

function getDefaultLocale(): LocaleCode {
  const requested = new URLSearchParams(window.location.search).get('lang')
  if (localeHasPublicUrl(window.location.pathname) && isLocaleCode(requested)) return requested

  const saved = localStorage.getItem(LOCALE_KEY)
  if (isLocaleCode(saved)) return saved

  for (const language of navigator.languages?.length ? navigator.languages : [navigator.language]) {
    const match = matchBrowserLocale(language)
    if (match) return match
  }
  return DEFAULT_LOCALE
}

export const i18n = createI18n({
  legacy: false,
  locale: getDefaultLocale(),
  fallbackLocale: { 'zh-TW': [DEFAULT_LOCALE], default: [DEFAULT_LOCALE] },
  messages: {},
  // Additional languages cover the public site. Untranslated console strings fall back to English.
  missingWarn: false,
  fallbackWarn: false,
  warnHtmlMessage: false
})

const loadedLocales = new Set<LocaleCode>()

export async function loadLocaleMessages(locale: LocaleCode): Promise<void> {
  if (loadedLocales.has(locale)) return
  const module = await localeLoaders[locale]()
  i18n.global.setLocaleMessage(locale, module.default)
  loadedLocales.add(locale)
}

export async function initI18n(): Promise<void> {
  await setLocale(getLocale(), { persist: false })
}

export async function setLocale(locale: string, options: { persist?: boolean } = {}): Promise<void> {
  if (!isLocaleCode(locale)) return
  await Promise.all([loadLocaleMessages(locale), loadLocaleMessages(DEFAULT_LOCALE)])
  i18n.global.locale.value = locale
  if (options.persist !== false) localStorage.setItem(LOCALE_KEY, locale)
  document.documentElement.lang = availableLocales.find(item => item.code === locale)?.tag ?? locale
}

export function getLocale(): LocaleCode {
  const current = i18n.global.locale.value
  return isLocaleCode(current) ? current : DEFAULT_LOCALE
}

export default i18n
