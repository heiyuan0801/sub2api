<script setup lang="ts">
defineProps<{ sticky?: boolean; plain?: boolean }>()
</script>

<template>
  <header
    class="shrink-0 text-foreground"
    :class="[
      sticky && 'sticky top-0 z-30',
      !plain && 'border-b border-border bg-background/95 backdrop-blur'
    ]"
  >
    <slot />
  </header>
</template>
