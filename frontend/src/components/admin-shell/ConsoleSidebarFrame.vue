<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useMediaQuery } from '@vueuse/core'
import { watch } from 'vue'
import { Sheet, SheetContent, SheetDescription, SheetTitle } from '@/components/ui/sheet'

const props = defineProps<{
  collapsed: boolean
  mobileOpen: boolean
  admin: boolean
}>()
const emit = defineEmits<{ 'update:mobileOpen': [open: boolean] }>()
const { t } = useI18n()
const desktop = useMediaQuery('(min-width: 1024px)')

watch(desktop, wide => {
  if (wide && props.mobileOpen) emit('update:mobileOpen', false)
})

function restoreMobileFocus(event: Event) {
  event.preventDefault()
  if (!document.querySelector('[data-console-command-trigger][aria-expanded=true]')) {
    document.querySelector<HTMLElement>('[data-console-mobile-trigger]')?.focus()
  }
}
</script>

<template>
  <aside
    v-if="!mobileOpen"
    class="console-sidebar fixed inset-y-0 left-0 z-40 hidden flex-col bg-canvas transition-[width] duration-200 motion-reduce:transition-none lg:flex" id="console-desktop-navigation"
    :class="collapsed ? 'w-[64px]' : admin ? 'w-[232px]' : 'w-[216px]'"
  >
    <slot :collapsed="collapsed" />
  </aside>
  <Sheet :open="mobileOpen" @update:open="emit('update:mobileOpen', $event)">
    <SheetContent
      side="left"
      class="console-sidebar w-[min(88vw,20rem)] overflow-visible border-border bg-canvas"
      @close-auto-focus="restoreMobileFocus"
    >
      <SheetTitle class="sr-only">{{ t('design.shell.navigation') }}</SheetTitle>
      <SheetDescription class="sr-only">{{ t('design.shell.navigationDescription') }}</SheetDescription>
      <slot v-if="mobileOpen" :collapsed="false" />
    </SheetContent>
  </Sheet>
</template>

<style scoped>
.console-sidebar :deep(.sidebar-header) {
  @apply h-14 shrink-0 px-3;
}
.console-sidebar :deep(.sidebar-nav) {
  @apply min-h-0 px-2 py-2;
}
.console-sidebar :deep(.sidebar-link) {
  @apply min-h-8 rounded-md border border-transparent px-2 py-1.5 text-[13px] text-muted-foreground;
}
.console-sidebar :deep(.sidebar-link-active) {
  @apply border-border bg-card text-foreground shadow-none ring-0;
}
.console-sidebar :deep(.sidebar-link-active > svg) {
  @apply text-brand;
}
.console-sidebar :deep(.sidebar-link-collapsed) {
  @apply justify-center;
}
.console-sidebar :deep(.sidebar-section) {
  @apply mb-3;
}
.console-sidebar :deep(.sidebar-section-title) {
  @apply mb-1 px-2 normal-case text-muted-foreground;
}
</style>
