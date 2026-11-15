<script setup lang="ts">
withDefaults(defineProps<{
  surface?: 'background' | 'canvas' | 'plain'
}>(), { surface: 'background' })
</script>

<template>
  <div
    class="flex min-h-screen min-h-[100dvh] min-w-0 flex-col text-foreground"
    :class="{ 'bg-background': surface === 'background', 'bg-canvas': surface === 'canvas' }"
  >
    <slot name="header" />
    <slot />
    <slot name="footer" />
  </div>
</template>
