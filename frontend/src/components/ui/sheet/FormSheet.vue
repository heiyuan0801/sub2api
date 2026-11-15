<script setup lang="ts">
import { computed, onBeforeUnmount, provide, useId, watch } from 'vue'
import { legacyPortalTargetKey } from '@/components/ui/dialog/legacyPortalTarget'
import Sheet from './Sheet.vue'
import SheetContent from './SheetContent.vue'
import SheetFooter from './SheetFooter.vue'
import SheetHeader from './SheetHeader.vue'
import SheetTitle from './SheetTitle.vue'

type Width = 'narrow' | 'normal' | 'wide' | 'extra-wide' | 'full'
const props = withDefaults(defineProps<{
  show: boolean
  title: string
  width?: Width
  panelClass?: string
  closeOnEscape?: boolean
  closeOnClickOutside?: boolean
  showCloseButton?: boolean
  zIndex?: number
}>(), { width: 'normal', closeOnEscape: true, closeOnClickOutside: false, showCloseButton: true })

const emit = defineEmits<{ close: [] }>()
const widths: Record<Width, string> = {
  narrow: 'sm:max-w-md',
  normal: 'sm:max-w-[600px]',
  wide: 'sm:max-w-3xl',
  'extra-wide': 'sm:max-w-5xl',
  full: 'sm:max-w-7xl'
}
const zIndex = computed(() => props.zIndex === 50 ? undefined : props.zIndex)
const portalId = 'form-sheet-overlays-' + useId()
// Legacy selects and tooltips need their portals inside the modal's focus boundary.
provide(legacyPortalTargetKey, '#' + portalId)

let returnFocus: HTMLElement | null = null
watch(() => props.show, open => {
  if (open) returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
}, { immediate: true })
function restoreFocus(event?: Event) {
  event?.preventDefault()
  const target = returnFocus
  returnFocus = null
  queueMicrotask(() => {
    if (target?.isConnected && (document.activeElement === document.body || !document.activeElement?.isConnected)) {
      target.focus({ preventScroll: true })
    }
  })
}
onBeforeUnmount(() => restoreFocus())
</script>

<template>
  <Sheet :open="show" @update:open="!$event && emit('close')">
    <SheetContent
      :class="['isolate overflow-visible motion-reduce:animate-none', widths[width], panelClass]"
      :aria-describedby="undefined" :show-close-button="showCloseButton" :z-index="zIndex"
      @close-auto-focus="restoreFocus"
      @escape-key-down="!closeOnEscape && $event.preventDefault()"
      @interact-outside="!closeOnClickOutside && $event.preventDefault()"
    >
      <SheetHeader><SheetTitle>{{ title }}</SheetTitle></SheetHeader>
      <div class="min-h-0 min-w-0 flex-1 overflow-y-auto overscroll-contain px-4 py-4 sm:px-6">
        <slot />
      </div>
      <SheetFooter v-if="$slots.footer" class="sm:rounded-b-xl sm:px-6"><slot name="footer" /></SheetFooter>
      <div :id="portalId" />
    </SheetContent>
  </Sheet>
</template>
