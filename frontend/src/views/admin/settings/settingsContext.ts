import { inject, type InjectionKey } from 'vue'
import type { useSettingsForm } from './useSettingsForm'

type SettingsContext = ReturnType<typeof useSettingsForm>

export const settingsContextKey: InjectionKey<SettingsContext> = Symbol('admin settings form')

export function useSettingsContext() {
  const state = inject(settingsContextKey)
  if (!state) throw new Error('Settings sections require the shared settings form')
  return state
}
