<script setup lang="ts">
import { useId } from 'vue'

withDefaults(defineProps<{
  title?: string
  description?: string
  heading?: 'h2' | 'h3' | 'h4'
  flush?: boolean
}>(), { heading: 'h2' })

const titleId = useId()
</script>

<template>
  <section
    data-slot="page-section-card"
    :aria-labelledby="title || $slots.title ? titleId : undefined"
    class="min-w-0 rounded-lg border border-border bg-card text-card-foreground"
  >
    <header
      v-if="title || description || $slots.title || $slots.actions"
      class="flex min-h-11 flex-wrap items-center justify-between gap-x-3 gap-y-1.5 border-b border-border px-3.5 py-2"
    >
      <div class="flex min-w-0 flex-wrap items-baseline gap-x-2 gap-y-1">
        <component :is="heading" v-if="title || $slots.title" :id="titleId" class="text-sm font-semibold">
          <slot name="title">{{ title }}</slot>
        </component>
        <p v-if="description" class="text-xs text-muted-foreground">{{ description }}</p>
      </div>
      <div v-if="$slots.actions" class="ml-auto flex flex-wrap items-center gap-2"><slot name="actions" /></div>
    </header>
    <div data-slot="page-section-body" :class="flush ? 'min-w-0' : 'min-w-0 p-3.5'"><slot /></div>
    <footer v-if="$slots.footer" class="border-t border-border px-3.5 py-2"><slot name="footer" /></footer>
  </section>
</template>
