<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { X } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'

const props = withDefaults(defineProps<{
  show: boolean
  title: string
  showCloseButton?: boolean
  closeOnEscape?: boolean
  closeOnClickOutside?: boolean
  width?: 'normal' | 'wide'
}>(), { showCloseButton: true, closeOnEscape: true, closeOnClickOutside: false, width: 'normal' })
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const widthClass = computed(() => props.width === 'wide' ? 'sm:max-w-3xl' : 'sm:max-w-lg')
let returnFocus: HTMLElement | null = null
watch(() => props.show, (open) => {
  if (open) returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
}, { immediate: true })
function restoreFocus(event: Event) {
  event.preventDefault()
  if (returnFocus?.isConnected) returnFocus.focus()
}
</script>

<template>
  <Dialog :open="show" @update:open="!$event && emit('close')">
    <DialogContent :class="[widthClass, 'max-h-[85dvh] overflow-y-auto']" :show-close-button="false" :aria-describedby="undefined"
      @close-auto-focus="restoreFocus"
      @escape-key-down="!closeOnEscape && $event.preventDefault()"
      @interact-outside="!closeOnClickOutside && $event.preventDefault()">
      <DialogHeader class="pr-8"><DialogTitle>{{ title }}</DialogTitle></DialogHeader>
      <Button v-if="showCloseButton" variant="ghost" size="icon-sm" type="button" class="absolute right-4 top-4" :aria-label="t('common.close')" @click="emit('close')"><X :size="16" /></Button>
      <div class="min-w-0"><slot /></div>
      <DialogFooter v-if="$slots.footer"><slot name="footer" /></DialogFooter>
    </DialogContent>
  </Dialog>
</template>
