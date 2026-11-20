/**
 * Theme System & Presets Store
 * Manages color mode (system/light/dark), theme presets, border radius,
 * font family, sidebar layout variant, and theme/command overlays.
 */

import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

export type ThemeMode = 'system' | 'light' | 'dark'
export type ThemePreset =
  | 'default'
  | 'anthropic'
  | 'rose-garden'
  | 'lake-view'
  | 'sunset-glow'
  | 'forest-whisper'
  | 'ocean-breeze'
  | 'lavender-dream'
  | 'neutral'
export type ThemeRadius = 'none' | 'sm' | 'md' | 'lg'
export type ThemeFont = 'system' | 'sans' | 'serif'
export type SidebarVariant = 'inset' | 'floating' | 'sidebar'

export const THEME_STORAGE_KEYS = {
  mode: 'theme',
  preset: 'theme_preset',
  radius: 'theme_radius',
  font: 'theme_font',
  sidebarVariant: 'sidebar_variant'
} as const

export const VALID_THEME_PRESETS: readonly ThemePreset[] = [
  'default',
  'anthropic',
  'rose-garden',
  'lake-view',
  'sunset-glow',
  'forest-whisper',
  'ocean-breeze',
  'lavender-dream',
  'neutral'
] as const

export const VALID_THEME_RADII: readonly ThemeRadius[] = ['none', 'sm', 'md', 'lg'] as const
export const VALID_THEME_FONTS: readonly ThemeFont[] = ['system', 'sans', 'serif'] as const
export const VALID_SIDEBAR_VARIANTS: readonly SidebarVariant[] = [
  'sidebar',
  'inset',
  'floating'
] as const

export const DEFAULT_SIDEBAR_VARIANT: SidebarVariant = 'sidebar'

function safeGetStorageItem(key: string): string | null {
  try {
    if (typeof localStorage === 'undefined') return null
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

function safeSetStorageItem(key: string, value: string): void {
  try {
    if (typeof localStorage === 'undefined') return
    localStorage.setItem(key, value)
  } catch {
    // Ignore storage errors (private mode / quota / jsdom restrictions)
  }
}

function safeRemoveStorageItem(key: string): void {
  try {
    if (typeof localStorage === 'undefined') return
    localStorage.removeItem(key)
  } catch {
    // Ignore storage errors
  }
}

function getColorSchemeMediaQuery(): MediaQueryList | null {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return null
  }
  try {
    return window.matchMedia('(prefers-color-scheme: dark)') || null
  } catch {
    return null
  }
}

function readSystemPrefersDark(): boolean {
  const mql = getColorSchemeMediaQuery()
  return Boolean(mql && mql.matches)
}

function readStoredMode(): ThemeMode {
  const saved = safeGetStorageItem(THEME_STORAGE_KEYS.mode)
  if (saved === 'light' || saved === 'dark') {
    return saved
  }
  return 'system'
}

function readStoredPreset(): ThemePreset {
  const saved = safeGetStorageItem(THEME_STORAGE_KEYS.preset)
  if (saved && (VALID_THEME_PRESETS as readonly string[]).includes(saved)) {
    return saved as ThemePreset
  }
  return 'default'
}

function readStoredRadius(): ThemeRadius {
  const saved = safeGetStorageItem(THEME_STORAGE_KEYS.radius)
  if (saved && (VALID_THEME_RADII as readonly string[]).includes(saved)) {
    return saved as ThemeRadius
  }
  return 'md'
}

function readStoredFont(): ThemeFont {
  const saved = safeGetStorageItem(THEME_STORAGE_KEYS.font)
  if (saved && (VALID_THEME_FONTS as readonly string[]).includes(saved)) {
    return saved as ThemeFont
  }
  return 'system'
}

function readStoredSidebarVariant(): SidebarVariant {
  const saved = safeGetStorageItem(THEME_STORAGE_KEYS.sidebarVariant)
  if (saved && (VALID_SIDEBAR_VARIANTS as readonly string[]).includes(saved)) {
    return saved as SidebarVariant
  }
  return DEFAULT_SIDEBAR_VARIANT
}

export function applyStoredThemeToDom(): void {
  if (typeof document === 'undefined' || !document.documentElement) return

  const savedMode = readStoredMode()
  const shouldUseDark =
    savedMode === 'dark' || (savedMode === 'system' && readSystemPrefersDark())

  const root = document.documentElement
  root.classList.toggle('dark', shouldUseDark)
  root.setAttribute('data-theme-preset', readStoredPreset())
  root.setAttribute('data-theme-radius', readStoredRadius())
  root.setAttribute('data-theme-font', readStoredFont())
}

export const useThemeStore = defineStore('theme', () => {
  // ==================== State ====================

  const mode = ref<ThemeMode>(readStoredMode())
  const preset = ref<ThemePreset>(readStoredPreset())
  const radius = ref<ThemeRadius>(readStoredRadius())
  const font = ref<ThemeFont>(readStoredFont())
  const sidebarVariant = ref<SidebarVariant>(readStoredSidebarVariant())
  const commandMenuOpen = ref<boolean>(false)
  const configDrawerOpen = ref<boolean>(false)

  const systemPrefersDark = ref<boolean>(readSystemPrefersDark())

  let mediaQueryList: MediaQueryList | null = null
  let mediaQueryListenerBound = false

  // ==================== Computed ====================

  const isDark = computed<boolean>(
    () => mode.value === 'dark' || (mode.value === 'system' && systemPrefersDark.value)
  )

  // ==================== Helpers ====================

  function syncDom(): void {
    if (typeof document === 'undefined' || !document.documentElement) return
    const root = document.documentElement
    root.classList.toggle('dark', isDark.value)
    root.setAttribute('data-theme-preset', preset.value)
    root.setAttribute('data-theme-radius', radius.value)
    root.setAttribute('data-theme-font', font.value)
  }

  function handleSystemSchemeChange(event: MediaQueryListEvent): void {
    systemPrefersDark.value = Boolean(event.matches)
    if (mode.value === 'system') {
      syncDom()
    }
  }

  function bindMediaQueryListener(): void {
    if (mediaQueryListenerBound) return
    const mql = getColorSchemeMediaQuery()
    if (!mql) return

    mediaQueryList = mql
    systemPrefersDark.value = Boolean(mql.matches)

    if (typeof mql.addEventListener === 'function') {
      mql.addEventListener('change', handleSystemSchemeChange)
      mediaQueryListenerBound = true
    } else if (typeof mql.addListener === 'function') {
      mql.addListener(handleSystemSchemeChange)
      mediaQueryListenerBound = true
    }
  }

  // ==================== Actions ====================

  function init(): void {
    mode.value = readStoredMode()
    preset.value = readStoredPreset()
    radius.value = readStoredRadius()
    font.value = readStoredFont()
    sidebarVariant.value = readStoredSidebarVariant()
    systemPrefersDark.value = mediaQueryList
      ? Boolean(mediaQueryList.matches)
      : readSystemPrefersDark()

    bindMediaQueryListener()
    syncDom()
  }

  function setMode(m: ThemeMode): void {
    mode.value = m
    if (m === 'system') {
      safeRemoveStorageItem(THEME_STORAGE_KEYS.mode)
    } else {
      safeSetStorageItem(THEME_STORAGE_KEYS.mode, m)
    }
    syncDom()
  }

  function toggleDark(): void {
    setMode(isDark.value ? 'light' : 'dark')
  }

  function setPreset(p: ThemePreset): void {
    preset.value = p
    safeSetStorageItem(THEME_STORAGE_KEYS.preset, p)
    syncDom()
  }

  function setRadius(r: ThemeRadius): void {
    radius.value = r
    safeSetStorageItem(THEME_STORAGE_KEYS.radius, r)
    syncDom()
  }

  function setFont(f: ThemeFont): void {
    font.value = f
    safeSetStorageItem(THEME_STORAGE_KEYS.font, f)
    syncDom()
  }

  function setSidebarVariant(v: SidebarVariant): void {
    sidebarVariant.value = v
    safeSetStorageItem(THEME_STORAGE_KEYS.sidebarVariant, v)
  }

  function setCommandMenuOpen(v: boolean): void {
    commandMenuOpen.value = v
  }

  function setConfigDrawerOpen(v: boolean): void {
    configDrawerOpen.value = v
  }

  function resetAll(): void {
    mode.value = 'system'
    preset.value = 'default'
    radius.value = 'md'
    font.value = 'system'
    sidebarVariant.value = DEFAULT_SIDEBAR_VARIANT

    safeRemoveStorageItem(THEME_STORAGE_KEYS.mode)
    safeRemoveStorageItem(THEME_STORAGE_KEYS.preset)
    safeRemoveStorageItem(THEME_STORAGE_KEYS.radius)
    safeRemoveStorageItem(THEME_STORAGE_KEYS.font)
    safeRemoveStorageItem(THEME_STORAGE_KEYS.sidebarVariant)

    syncDom()
  }

  return {
    // State
    mode,
    preset,
    radius,
    font,
    sidebarVariant,
    commandMenuOpen,
    configDrawerOpen,

    // Computed
    isDark,

    // Actions
    init,
    setMode,
    toggleDark,
    setPreset,
    setRadius,
    setFont,
    setSidebarVariant,
    setCommandMenuOpen,
    setConfigDrawerOpen,
    resetAll
  }
})
