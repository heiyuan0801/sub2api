<script setup lang="ts">
import { X } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog'
import Command from './Command.vue'
import type { CommandEntry, CommandGroup } from './types'

defineProps<{
  groups: readonly CommandGroup[]
  title?: string
  description?: string
  placeholder?: string
}>()
const open = defineModel<boolean>('open', { default: false })
const query = defineModel<string>('query', { default: '' })
const emit = defineEmits<{ select: [entry: CommandEntry] }>()
const { t } = useI18n()

function select(item: CommandEntry) {
  open.value = false
  emit('select', item)
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent :show-close-button="false" class="gap-0 overflow-hidden p-0 sm:max-w-[600px]">
      <DialogTitle class="sr-only">{{ title ?? t('design.command.label') }}</DialogTitle>
      <DialogDescription class="sr-only">{{ description ?? t('design.command.description') }}</DialogDescription>
      <Command v-model:query="query" :groups="groups" :placeholder="placeholder" :label="title" @select="select">
        <template #input-actions>
          <DialogClose as-child>
            <Button variant="ghost" size="icon" type="button" :aria-label="t('common.close')" class="size-7 shrink-0">
              <X aria-hidden="true" />
            </Button>
          </DialogClose>
        </template>
      </Command>
    </DialogContent>
  </Dialog>
</template>
