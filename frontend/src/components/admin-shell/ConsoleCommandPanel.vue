<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, type RouteLocationRaw } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { CircleHelp, LogOut, Moon, PanelLeft, Settings2, Sun } from '@lucide/vue'
import { useAppStore, useAuthStore } from '@/stores'
import { CommandDialog, type CommandEntry, type CommandGroup } from '@/components/ui/command'
import { selectConsoleLinks, type SidebarCommandLink } from './navigation'
import { ADMIN_SETTINGS_INDEX_PATH, SETTINGS_SECTIONS, settingsSectionLocation } from './settingsSections'
import { getShellShortcutHint } from './shortcuts'

const props = defineProps<{ links: SidebarCommandLink[]; dark: boolean }>()
const open = defineModel<boolean>('open', { required: true })
const query = defineModel<string>('query', { required: true })
const emit = defineEmits<{
  'toggle-theme': []
  'toggle-sidebar': []
  'close-auto-focus': [event: Event]
  logout: []
  replay: []
}>()
const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const canReplay = computed(() => authStore.isAdmin && !authStore.isSimpleMode)

interface Destination extends CommandEntry {
  to: RouteLocationRaw
}

const destinations = computed<Destination[]>(() => (
  selectConsoleLinks(props.links, {
    authenticated: authStore.isAuthenticated,
    admin: authStore.isAdmin,
    backendOnly: appStore.backendModeEnabled,
    settings: appStore.publicSettingsLoaded ? appStore.cachedPublicSettings : null
  }, router.resolve).map(link => ({ ...link, id: 'page:' + link.path, to: link.path, keywords: [link.path] }))
))

const sectionDestinations = computed<Destination[]>(() => {
  if (!authStore.isAdmin || !destinations.value.some(item => item.to === ADMIN_SETTINGS_INDEX_PATH)) return []
  return SETTINGS_SECTIONS.map(section => ({
    id: 'settings:' + section.key,
    label: t(section.labelKey),
    description: t('admin.settings.title'),
    keywords: [section.key, ADMIN_SETTINGS_INDEX_PATH],
    icon: Settings2,
    to: settingsSectionLocation(section.key)
  }))
})

const groups = computed<CommandGroup[]>(() => {
  if (!authStore.isAuthenticated) return []
  const actions: CommandEntry[] = [
    { id: 'theme', label: props.dark ? t('nav.lightMode') : t('nav.darkMode'), icon: props.dark ? Sun : Moon, shortcut: getShellShortcutHint('theme') },
    { id: 'sidebar', label: t('design.shell.toggleSidebar'), icon: PanelLeft, shortcut: getShellShortcutHint('sidebar') },
    { id: 'logout', label: t('nav.logout'), icon: LogOut }
  ]
  if (canReplay.value) actions.splice(2, 0, { id: 'replay', label: t('onboarding.restartTour'), icon: CircleHelp })
  return [
    { id: 'pages', label: t('design.shell.pages'), items: destinations.value },
    { id: 'settings', label: t('admin.settings.title'), items: sectionDestinations.value },
    { id: 'actions', label: t('design.shell.actions'), items: actions }
  ]
})

function activate(entry: CommandEntry) {
  const destination = destinations.value.find(item => item.id === entry.id)
    ?? sectionDestinations.value.find(item => item.id === entry.id)
  if (destination) {
    void router.push(destination.to)
    return
  }

  switch (entry.id) {
    case 'theme': emit('toggle-theme'); break
    case 'sidebar': emit('toggle-sidebar'); break
    case 'logout': emit('logout'); break
    case 'replay': if (canReplay.value) emit('replay'); break
  }
}
</script>

<template>
  <CommandDialog v-model:open="open" v-model:query="query" :groups="groups" @select="activate" @close-auto-focus="emit('close-auto-focus', $event)">
    <template #footer>
      <div class="flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-border bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
        <span><kbd class="rounded border border-border px-1" aria-hidden="true">↑ ↓</kbd> {{ t('design.shell.choose') }}</span>
        <span><kbd class="rounded border border-border px-1">Enter</kbd> {{ t('design.shell.openItem') }}</span>
        <span><kbd class="rounded border border-border px-1">Esc</kbd> {{ t('common.close') }}</span>
        <span class="ml-auto"><kbd class="rounded border border-border px-1">{{ getShellShortcutHint('command') }}</kbd> {{ t('design.shell.togglePanel') }}</span>
      </div>
    </template>
  </CommandDialog>
</template>
