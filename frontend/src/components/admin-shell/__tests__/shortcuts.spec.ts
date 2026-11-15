import { describe, expect, it } from 'vitest'
import { getShellShortcut, getShellShortcutAriaKey, getShellShortcutHint } from '../shortcuts'

function shortcutAt(target: Element, init: KeyboardEventInit) {
  let shortcut: ReturnType<typeof getShellShortcut>
  target.addEventListener('keydown', event => {
    shortcut = getShellShortcut(event as KeyboardEvent)
  }, { once: true })
  target.dispatchEvent(new KeyboardEvent('keydown', { ...init, bubbles: true, composed: true }))
  return shortcut
}

describe('shell shortcuts', () => {
  it('recognizes the command, theme and sidebar combinations', () => {
    expect(getShellShortcut(new KeyboardEvent('keydown', { key: 'k', metaKey: true }))).toBe('command')
    expect(getShellShortcut(new KeyboardEvent('keydown', { key: 'J', ctrlKey: true }))).toBe('theme')
    expect(getShellShortcut(new KeyboardEvent('keydown', { key: 'b', metaKey: true }))).toBe('sidebar')
  })

  it('allows the command palette from editors but leaves their editing shortcuts alone', () => {
    const parent = document.createElement('div')
    parent.innerHTML = '<input><textarea></textarea><div role="textbox"><span></span></div><div contenteditable="plaintext-only"><em></em></div>'

    for (const editor of parent.querySelectorAll('input, textarea, span, em')) {
      expect(shortcutAt(editor, { key: 'j', ctrlKey: true })).toBeUndefined()
      expect(shortcutAt(editor, { key: 'b', metaKey: true })).toBeUndefined()
      expect(shortcutAt(editor, { key: 'k', ctrlKey: true })).toBe('command')
    }

    const editingHost = document.createElement('div')
    editingHost.setAttribute('contenteditable', 'true')
    const nonEditable = editingHost.appendChild(document.createElement('div'))
    nonEditable.setAttribute('contenteditable', 'false')
    const button = nonEditable.appendChild(document.createElement('button'))
    expect(shortcutAt(button, { key: 'j', ctrlKey: true })).toBe('theme')
  })

  it('does not intercept denied, repeated, composing or unrelated shortcuts', () => {
    for (const options of [
      { key: 'k' },
      { key: 'x', ctrlKey: true },
      { key: 'k', ctrlKey: true, altKey: true },
      { key: 'k', metaKey: true, shiftKey: true },
      { key: 'k', metaKey: true, ctrlKey: true },
      { key: 'k', ctrlKey: true, repeat: true },
      { key: 'j', ctrlKey: true, isComposing: true }
    ]) {
      expect(getShellShortcut(new KeyboardEvent('keydown', options))).toBeUndefined()
    }

    const cancelled = new KeyboardEvent('keydown', { key: 'b', metaKey: true, cancelable: true })
    cancelled.preventDefault()
    expect(getShellShortcut(cancelled)).toBeUndefined()
  })

  it('formats the visible and accessibility labels for the primary modifier', () => {
    expect(getShellShortcutHint('command', 'macOS')).toBe('⌘K')
    expect(getShellShortcutAriaKey('sidebar', 'MacIntel')).toBe('Meta+B')
    expect(getShellShortcutHint('theme', 'Win32')).toBe('Ctrl+J')
    expect(getShellShortcutAriaKey('command', 'Linux')).toBe('Control+K')
  })
})
