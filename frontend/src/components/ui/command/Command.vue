<script setup lang="ts">
import { computed, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import { Search } from '@lucide/vue'
import {
  ListboxContent, ListboxFilter, ListboxGroup, ListboxGroupLabel, ListboxItem, ListboxRoot, useFilter
} from 'reka-ui'
import type { CommandEntry, CommandGroup } from './types'

const props = defineProps<{
  groups: readonly CommandGroup[]
  placeholder?: string
  label?: string
  emptyMessage?: string
  autoFocus?: boolean
}>()
const emit = defineEmits<{ select: [entry: CommandEntry] }>()
const query = defineModel<string>('query', { default: '' })
const { t } = useI18n()
const { contains } = useFilter({ sensitivity: 'base' })
const listId = useId()
const filterLabel = computed(() => props.label ?? t('design.command.label'))
const visibleGroups = computed(() => {
  const term = query.value.trim()
  return props.groups.map(group => ({
    ...group,
    items: group.items.filter(item =>
      !term || [item.label, item.description, ...(item.keywords ?? [])].some(value => value && contains(value, term))
    )
  })).filter(group => group.items.length > 0)
})

function select(item: CommandEntry, event: Event) {
  // Palette entries are actions. Keep focus on the search field rather than
  // giving the action a persistent selected value.
  event.preventDefault()
  if (!item.disabled) emit('select', item)
}
</script>

<template>
  <ListboxRoot data-slot="command" highlight-on-hover class="min-w-0 overflow-hidden rounded-lg bg-popover text-popover-foreground">
    <div class="flex min-w-0 items-center gap-2.5 border-b border-border px-3">
      <Search aria-hidden="true" class="size-4 shrink-0 text-muted-foreground" />
      <ListboxFilter
        v-model="query"
        :auto-focus="autoFocus"
        :placeholder="placeholder ?? t('design.command.placeholder')"
        :aria-label="filterLabel"
        :aria-controls="listId"
        aria-autocomplete="list"
        aria-expanded="true"
        autocomplete="off"
        role="combobox"
        class="h-11 w-full min-w-0 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
      />
      <slot name="input-actions" />
    </div>
    <ListboxContent
      :id="listId"
      :aria-label="filterLabel"
      class="max-h-[min(55vh,22rem)] overflow-y-auto overscroll-contain p-1 empty:p-0"
    >
      <ListboxGroup v-for="group in visibleGroups" :key="group.id" class="p-1">
        <ListboxGroupLabel :class="group.label ? 'px-2 py-1.5 text-xs font-medium text-muted-foreground' : 'sr-only'">
          {{ group.label ?? filterLabel }}
        </ListboxGroupLabel>
        <ListboxItem
          v-for="item in group.items" :key="item.id" :value="item.id"
          :disabled="item.disabled" :aria-disabled="item.disabled ? true : undefined"
          class="flex min-w-0 cursor-default items-center gap-2 rounded-md px-2 py-2 text-sm outline-none data-[disabled]:opacity-50 data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground"
          @select="select(item, $event)"
        >
          <component :is="item.icon" v-if="item.icon" class="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
          <span class="min-w-0 flex-1">
            <span class="block truncate">{{ item.label }}</span>
            <span v-if="item.description" class="block truncate text-xs text-muted-foreground">{{ item.description }}</span>
          </span>
        </ListboxItem>
      </ListboxGroup>
    </ListboxContent>
    <p v-if="!visibleGroups.length" role="status" class="px-3 py-8 text-center text-sm text-muted-foreground">
      {{ emptyMessage ?? t('design.command.empty') }}
    </p>
  </ListboxRoot>
</template>
