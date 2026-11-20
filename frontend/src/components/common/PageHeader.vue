<template>
  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
    <div class="min-w-0 space-y-1">
      <div class="flex flex-wrap items-center gap-2.5">
        <h2 class="truncate text-xl font-semibold tracking-tight text-foreground">
          {{ title }}
        </h2>
        <slot name="badge" />
      </div>
      <p v-if="description" class="text-sm text-muted-foreground">
        {{ description }}
      </p>
    </div>
    <div v-if="$slots.actions" class="flex shrink-0 flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()
</script>
