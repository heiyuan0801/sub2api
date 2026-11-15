<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Toolbar from './Toolbar.vue'

defineProps<{ label?: string }>()
const emit = defineEmits<{ submit: [event: SubmitEvent] }>()
const { t } = useI18n()
</script>

<template>
  <form
    data-slot="page-filter-bar"
    role="search"
    :aria-label="label ?? t('design.page.filters')"
    class="min-w-0 space-y-2"
    @submit.prevent="emit('submit', $event)"
  >
    <Toolbar>
      <slot />
      <template v-if="$slots.actions" #actions><slot name="actions" /></template>
    </Toolbar>
    <div v-if="$slots.secondary" class="flex flex-wrap items-center gap-2">
      <slot name="secondary" />
    </div>
  </form>
</template>
