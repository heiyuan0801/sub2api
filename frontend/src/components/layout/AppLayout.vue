<template>
  <TooltipProvider :delay-duration="250">
    <div class="console-shell min-h-screen bg-canvas text-foreground" :data-console="isAdmin ? 'admin' : 'user'">
      <AppSidebar @navigation-change="setNavigation" />
      <div
        class="console-main relative min-h-screen min-w-0 bg-background transition-[margin] duration-200 motion-reduce:transition-none lg:my-2 lg:mr-2 lg:min-h-[calc(100vh-1rem)] lg:rounded-xl lg:border lg:border-border"
        :class="sidebarCollapsed ? 'lg:ml-[64px]' : isAdmin ? 'lg:ml-[232px]' : 'lg:ml-[216px]'"
      >
        <AppHeader
          :dark="isDark" :command-open="commandOpen"
          @open-command="openCommand" @toggle-theme="toggleTheme"
          @logout="logout" @replay="onboardingStore.replay()"
        />
        <main class="console-content p-4 md:p-5" :class="isAdmin ? 'lg:p-6' : 'lg:px-6'">
          <slot />
        </main>
      </div>
      <ConsoleCommandPanel
        v-if="authStore.isAuthenticated"
        v-model:open="commandOpen" v-model:query="commandQuery"
        :links="links" :dark="isDark"
        @toggle-theme="toggleTheme" @toggle-sidebar="toggleSidebar"
        @logout="logout" @replay="onboardingStore.replay()"
        @close-auto-focus="restoreCommandFocus"
      />
    </div>
  </TooltipProvider>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'
import { TooltipProvider } from '@/components/ui/tooltip'
import ConsoleCommandPanel from '@/components/admin-shell/ConsoleCommandPanel.vue'
import { useConsoleShell } from '@/components/admin-shell/useConsoleShell'

const appStore = useAppStore()
const authStore = useAuthStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.isAdmin)
const {
  isDark, toggleTheme, commandOpen, commandQuery, links, setNavigation,
  openCommand, restoreCommandFocus, toggleSidebar, logout
} = useConsoleShell()

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>
