<script setup lang="ts">
import { Inbox } from '@lucide/vue'
import type { Component } from 'vue'
import { useI18n } from 'vue-i18n'

defineProps<{
  title?: string
  description?: string
  icon?: Component
}>()
const { t } = useI18n()
</script>

<template>
  <div data-slot="page-empty-state" class="flex min-w-0 flex-col items-center gap-3 px-4 py-12 text-center">
    <slot name="icon">
      <div class="grid size-10 place-items-center rounded-lg border border-border bg-muted/60 text-muted-foreground" aria-hidden="true">
        <component :is="icon ?? Inbox" class="size-5" />
      </div>
    </slot>
    <div class="space-y-1">
      <p class="text-sm font-semibold text-foreground">{{ title ?? t('common.noData') }}</p>
      <p v-if="description || $slots.description" class="max-w-md text-sm text-muted-foreground">
        <slot name="description">{{ description }}</slot>
      </p>
    </div>
    <div v-if="$slots.actions" class="flex flex-wrap justify-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
