<script setup lang="ts">
defineProps<{ plain?: boolean }>()
</script>

<template>
  <footer
    class="shrink-0 text-muted-foreground"
    :class="!plain && 'border-t border-border bg-background'"
  >
    <slot />
  </footer>
</template>
