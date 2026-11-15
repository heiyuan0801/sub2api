import { computed, inject, ref } from 'vue'
import { routerKey, type Router } from 'vue-router'
import {
  DEFAULT_SETTINGS_SECTION,
  getSettingsSection,
  SETTINGS_SECTIONS,
  settingsSectionLocation,
  type SettingsSectionKey
} from '@/components/admin-shell/settingsSections'

const keyboardActions = {
  ArrowLeft: -1,
  ArrowUp: -1,
  ArrowRight: 1,
  ArrowDown: 1,
  Home: 'first',
  End: 'last'
} as const

export function useSettingsSectionNavigation() {
  const router = inject<Router | null>(routerKey, null)
  // Standalone mounts use the same panel contract without installing a router.
  const localSection = ref<SettingsSectionKey>(DEFAULT_SETTINGS_SECTION)
  const activeTab = computed(() =>
    getSettingsSection(router?.currentRoute.value.params.section)?.key ?? localSection.value
  )

  async function selectSettingsTab(key: SettingsSectionKey) {
    if (!router) {
      localSection.value = key
    } else if (key !== activeTab.value) {
      await router.push({
        ...settingsSectionLocation(key),
        query: router.currentRoute.value.query
      })
    }
  }

  async function handleSettingsTabKeydown(event: KeyboardEvent, key: SettingsSectionKey) {
    const action = keyboardActions[event.key as keyof typeof keyboardActions]
    if (action === undefined) return
    event.preventDefault()
    const index = Math.max(SETTINGS_SECTIONS.findIndex(section => section.key === key), 0)
    const count = SETTINGS_SECTIONS.length
    const target = action === 'first' ? 0 : action === 'last' ? count - 1 : (index + action + count) % count
    const section = SETTINGS_SECTIONS[target]
    if (!section) return
    await selectSettingsTab(section.key)
    window.requestAnimationFrame(() => {
      document.getElementById(`settings-tab-${activeTab.value}`)?.focus()
    })
  }

  return { activeTab, settingsTabs: SETTINGS_SECTIONS, selectSettingsTab, handleSettingsTabKeydown }
}
