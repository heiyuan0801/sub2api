<script setup lang="ts">
withDefaults(defineProps<{
  title: string
  description?: string
  eyebrow?: string
  as?: 'h1' | 'h2'
}>(), { as: 'h1' })
</script>

<template>
  <header data-slot="page-header" class="flex flex-wrap items-end justify-between gap-x-5 gap-y-3">
    <div class="min-w-0 space-y-1">
      <p v-if="eyebrow" class="text-xs font-medium text-muted-foreground">{{ eyebrow }}</p>
      <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
        <component :is="as" class="min-w-0 break-words text-xl font-semibold leading-tight tracking-tight text-foreground sm:text-2xl">
          {{ title }}
        </component>
        <slot name="badge" />
      </div>
      <p v-if="description || $slots.description" class="max-w-3xl text-sm text-muted-foreground">
        <slot name="description">{{ description }}</slot>
      </p>
    </div>
    <div v-if="$slots.actions" class="flex min-w-0 flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </header>
</template>
