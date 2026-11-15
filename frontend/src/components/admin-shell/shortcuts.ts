export type ShellShortcut = 'command' | 'theme' | 'sidebar'

const shortcutKeys: Record<ShellShortcut, string> = {
  command: 'K',
  theme: 'J',
  sidebar: 'B'
}

const textControl = 'input, textarea, select, [role~="textbox"], [role~="searchbox"], [role~="combobox"], [role~="spinbutton"]'

function isElement(target: EventTarget | null): target is Element {
  return typeof Element !== 'undefined' && target instanceof Element
}

function isEditing(event: KeyboardEvent): boolean {
  const path = event.composedPath().filter(isElement)
  const target = path[0] ?? (isElement(event.target) ? event.target : null)
  if (!target) return false

  // The composed path detects controls inside an open shadow root. Checking
  // ancestors also handles custom inputs that expose roles on a wrapper.
  if (path.some(element => element.matches(textControl)) || target.closest(textControl)) return true
  if ((target as HTMLElement).isContentEditable || target.ownerDocument.designMode === 'on') return true

  // Reflect the nearest explicit editing state, including false islands.
  for (let element: Element | null = target; element; element = element.parentElement) {
    const value = element.getAttribute('contenteditable')?.toLowerCase()
    if (value === 'false') return false
    if (value === '' || value === 'true' || value === 'plaintext-only') return true
  }
  return false
}

/** Recognizes a shortcut without cancelling the event. The caller owns handling. */
export function getShellShortcut(event: KeyboardEvent): ShellShortcut | undefined {
  if (
    event.defaultPrevented || event.repeat || event.isComposing ||
    event.altKey || event.shiftKey || event.metaKey === event.ctrlKey ||
    event.getModifierState('AltGraph')
  ) return undefined

  const key = event.key.toUpperCase()
  if (key === shortcutKeys.command) return 'command'
  if (key !== shortcutKeys.theme && key !== shortcutKeys.sidebar) return undefined
  if (isEditing(event)) return undefined
  return key === shortcutKeys.theme ? 'theme' : 'sidebar'
}

function getPlatform(): string {
  if (typeof navigator === 'undefined') return ''
  const browser = navigator as Navigator & { userAgentData?: { platform?: string } }
  return browser.userAgentData?.platform || browser.platform
}

function usesCommandKey(platform: string): boolean {
  return /mac|iphone|ipad|ipod/i.test(platform)
}

export function getShellShortcutHint(shortcut: ShellShortcut, platform = getPlatform()): string {
  return (usesCommandKey(platform) ? '⌘' : 'Ctrl+') + shortcutKeys[shortcut]
}

export function getShellShortcutAriaKey(shortcut: ShellShortcut, platform = getPlatform()): string {
  return (usesCommandKey(platform) ? 'Meta+' : 'Control+') + shortcutKeys[shortcut]
}
