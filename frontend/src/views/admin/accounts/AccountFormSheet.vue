<script setup lang="ts">
import { onBeforeUnmount, provide, ref, useId, watch } from 'vue'
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { legacyPortalTargetKey } from '@/components/ui/dialog/legacyPortalTarget'

const props = defineProps<{
  show: boolean
  title: string
  width?: 'wide'
  description?: string
  sections?: readonly { id: string; label: string }[]
  childDialogOpen?: boolean
}>()
const emit = defineEmits<{ close: [] }>()
const body = ref<HTMLElement | null>(null)
const activeSection = ref('')
const portalId = 'account-form-overlay-' + useId()
provide(legacyPortalTargetKey, '#' + portalId)
let returnFocus: HTMLElement | null = null

watch(() => props.show, open => {
  if (open) {
    returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    activeSection.value = props.sections?.[0]?.id ?? ''
  }
}, { immediate: true })

function showSection(id: string) {
  const section = body.value?.querySelector<HTMLElement>(`[data-account-section="${id}"]`)
  if (!section) return
  activeSection.value = id
  const container = body.value
  if (!container) return
  const top = section.getBoundingClientRect().top - container.getBoundingClientRect().top + container.scrollTop - 12
  container.scrollTo({ top, behavior: 'auto' })
  section.focus({ preventScroll: true })
}

function restoreFocus(event?: Event) {
  event?.preventDefault()
  const target = returnFocus
  if (!target) return
  returnFocus = null
  // The drawer is mounted conditionally by the list, so it may be unmounted
  // immediately after save instead of receiving an animated close.
  queueMicrotask(() => {
    if (target.isConnected && (document.activeElement === document.body || !document.activeElement?.isConnected)) {
      target.focus({ preventScroll: true })
    }
  })
}
onBeforeUnmount(() => restoreFocus())
</script>

<template>
  <Sheet :open="show" @update:open="!$event && !childDialogOpen && emit('close')">
    <SheetContent
      class="isolate overflow-visible motion-reduce:animate-none"
      @close-auto-focus="restoreFocus"
      @interact-outside="$event.preventDefault()"
      @escape-key-down="childDialogOpen && $event.preventDefault()"
    >
      <SheetHeader>
        <SheetTitle>{{ title }}</SheetTitle>
        <SheetDescription :class="{ 'sr-only': !description }">{{ description || title }}</SheetDescription>
      </SheetHeader>
      <slot name="summary" />
      <nav
        v-if="sections?.length" :aria-label="title"
        class="flex shrink-0 flex-wrap gap-x-3 gap-y-0 border-b border-border px-4"
      >
        <button
          v-for="section in sections" :key="section.id" type="button"
          class="border-b-2 px-0.5 py-2.5 text-xs transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring"
          :class="(activeSection || sections[0]?.id) === section.id
            ? 'border-brand font-medium text-foreground'
            : 'border-transparent text-muted-foreground hover:text-foreground'"
          :aria-current="(activeSection || sections[0]?.id) === section.id ? 'location' : undefined"
          @click="showSection(section.id)"
        >{{ section.label }}</button>
      </nav>
      <div ref="body" class="min-h-0 min-w-0 flex-1 overflow-y-auto overscroll-contain px-4 py-4">
        <slot />
      </div>
      <SheetFooter v-if="$slots.footer" class="block rounded-b-xl bg-card">
        <slot name="footer" />
      </SheetFooter>
      <div :id="portalId" />
    </SheetContent>
  </Sheet>
</template>
