<template>
  <div
    class="console-shell flex min-h-screen text-foreground"
    :class="sidebarVariant === 'inset' ? 'bg-sidebar' : 'bg-background'"
  >
    <!-- Full-height Left Sidebar -->
    <AppSidebar />

    <!-- Right Main Pane (Airy Header + Content) -->
    <div
      class="console-main relative flex min-w-0 flex-1 flex-col transition-all duration-200"
      :class="mainPanelClasses"
    >
      <AppHeader />

      <main class="console-content flex-1 p-4 md:px-7 md:py-5">
        <slot />
      </main>
    </div>

    <!-- Global Command Menu (⌘K / Ctrl+K) -->
    <CommandMenu />
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onBeforeUnmount, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'
import CommandMenu from '@/components/common/CommandMenu.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const themeStore = useThemeStore()
const sidebarVariant = computed(() => themeStore.sidebarVariant)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const mainPanelClasses = computed(() => {
  if (sidebarVariant.value === 'inset') {
    return 'bg-background lg:my-2 lg:mr-2 lg:min-h-[calc(100vh-1rem)] lg:rounded-xl lg:border lg:border-border lg:shadow-sm'
  }
  return 'bg-background'
})

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

function handleSidebarShortcut(e: KeyboardEvent) {
  if (!(e.metaKey || e.ctrlKey) || e.altKey || e.shiftKey) return
  if (e.key.toLowerCase() !== 'b') return
  const target = e.target as HTMLElement | null
  if (
    target &&
    (target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.tagName === 'SELECT' ||
      target.isContentEditable)
  ) {
    return
  }
  e.preventDefault()
  appStore.toggleSidebar()
}

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
  window.addEventListener('keydown', handleSidebarShortcut)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleSidebarShortcut)
})

defineExpose({ replayTour })
</script>
