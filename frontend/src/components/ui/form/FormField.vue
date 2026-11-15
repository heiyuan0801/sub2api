<script setup lang="ts">
import { computed, useId } from 'vue'

const props = defineProps<{
  id?: string
  label: string
  description?: string
  error?: string
  required?: boolean
}>()

const generatedId = useId()
const fieldId = computed(() => props.id ?? generatedId)
const descriptionId = computed(() => `${fieldId.value}-description`)
const errorId = computed(() => `${fieldId.value}-error`)
const controlAttrs = computed(() => ({
  id: fieldId.value,
  required: props.required,
  'aria-invalid': props.error ? 'true' as const : undefined,
  'aria-describedby': [
    props.description ? descriptionId.value : '',
    props.error ? errorId.value : ''
  ].filter(Boolean).join(' ') || undefined
}))
</script>

<template>
  <div data-slot="form-field" class="min-w-0 space-y-1.5">
    <label :for="fieldId" class="block text-sm font-medium text-foreground">
      {{ label }} <span v-if="required" class="text-danger" aria-hidden="true">*</span>
    </label>
    <slot :control-attrs="controlAttrs" />
    <p v-if="description" :id="descriptionId" class="text-xs text-muted-foreground">{{ description }}</p>
    <p v-if="error" :id="errorId" role="alert" class="text-xs text-danger">{{ error }}</p>
  </div>
</template>
