import { onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore, useAuthStore } from '@/stores'
import { useDocumentTheme } from '@/composables/useDocumentTheme'
import type { SidebarCommandLink } from './navigation'
import { getShellShortcut } from './shortcuts'

const SIDEBAR_PREF = 'sub2api.console.sidebar.collapsed'
const DESKTOP = '(min-width: 1024px)'

export function useConsoleShell() {
  const appStore = useAppStore()
  const authStore = useAuthStore()
  const route = useRoute()
  const router = useRouter()
  const { isDark, toggleTheme } = useDocumentTheme()
  const commandOpen = ref(false)
  const commandQuery = ref('')
  const links = shallowRef<SidebarCommandLink[]>([])
  let previousFocus: HTMLElement | null = null

  try {
    const saved = window.localStorage.getItem(SIDEBAR_PREF)
    if (saved === '1' || saved === '0') appStore.setSidebarCollapsed(saved === '1')
  } catch {
    // Leave this session's default intact when storage is unavailable.
  }
  watch(() => appStore.sidebarCollapsed, collapsed => {
    try {
      window.localStorage.setItem(SIDEBAR_PREF, collapsed ? '1' : '0')
    } catch {
      // The sidebar still works with a session-only preference.
    }
  }, { flush: 'sync' })

  function setNavigation(next: SidebarCommandLink[]) {
    links.value = next
  }

  function openCommand() {
    if (commandOpen.value || !authStore.isAuthenticated) return
    previousFocus = appStore.mobileOpen
      ? document.querySelector<HTMLElement>('[data-console-mobile-trigger]')
      : document.activeElement instanceof HTMLElement ? document.activeElement : null
    if (appStore.mobileOpen) appStore.setMobileOpen(false)
    commandQuery.value = ''
    commandOpen.value = true
  }

  function restoreCommandFocus(event: Event) {
    event.preventDefault()
    if (previousFocus?.isConnected) previousFocus.focus({ preventScroll: true })
    previousFocus = null
  }

  function toggleSidebar() {
    if (window.matchMedia(DESKTOP).matches) {
      appStore.toggleSidebar()
    } else {
      commandOpen.value = false
      appStore.toggleMobileSidebar()
    }
  }

  function onKeydown(event: KeyboardEvent) {
    if (!authStore.isAuthenticated) return
    switch (getShellShortcut(event)) {
      case 'command':
        event.preventDefault()
        if (commandOpen.value) commandOpen.value = false
        else openCommand()
        break
      case 'theme':
        event.preventDefault()
        toggleTheme()
        break
      case 'sidebar':
        event.preventDefault()
        toggleSidebar()
        break
    }
  }

  async function logout() {
    try {
      await authStore.logout()
    } catch {
      // Clear the local session and redirect behavior are owned by the auth store
      // and router; a failed network request must not block the login screen.
    }
    await router.push('/login')
  }

  watch(() => route.path, () => {
    commandOpen.value = false
    appStore.setMobileOpen(false)
  })
  onMounted(() => window.addEventListener('keydown', onKeydown))
  onBeforeUnmount(() => {
    window.removeEventListener('keydown', onKeydown)
    appStore.setMobileOpen(false)
  })

  return {
    isDark, toggleTheme, commandOpen, commandQuery, links, setNavigation,
    openCommand, restoreCommandFocus, toggleSidebar, logout
  }
}
