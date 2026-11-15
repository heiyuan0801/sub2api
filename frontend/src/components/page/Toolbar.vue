<script setup lang="ts">
defineProps<{ label?: string }>()
</script>

<template>
  <div
    data-slot="page-toolbar"
    :role="label ? 'group' : undefined"
    :aria-label="label"
    class="flex min-w-0 flex-wrap items-center justify-between gap-2"
  >
    <div class="flex min-w-0 flex-1 flex-wrap items-center gap-2"><slot /></div>
    <div v-if="$slots.actions" class="ml-auto flex min-w-0 flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
