<script setup lang="ts">
const emit = defineEmits<{ submit: [event: SubmitEvent] }>()
</script>

<template>
  <form data-slot="form" @submit.prevent="emit('submit', $event)">
    <slot />
  </form>
</template>
