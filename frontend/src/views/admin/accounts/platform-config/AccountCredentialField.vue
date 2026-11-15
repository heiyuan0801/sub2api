<script setup lang="ts">
import { computed, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AccountPlatform } from '@/types'
import { getApiKeyField } from './registry'
import type { AccountCredentialKey, AccountFormMode } from './types'

const props = withDefaults(defineProps<{
  field: AccountCredentialKey
  mode: AccountFormMode
  platform?: AccountPlatform
  inputId?: string
  labelId?: string
  showLabel?: boolean
  disabled?: boolean
  baseUrlHint?: string
}>(), { showLabel: true, platform: 'anthropic' })
const value = defineModel<string>({ required: true })
const { t } = useI18n()
const generatedId = useId()
const id = computed(() => props.inputId || generatedId)
const definition = computed(() => getApiKeyField(props.platform, props.field, props.mode))
const placeholder = computed(() => {
  const field = definition.value
  const key = props.mode === 'bulk' ? field?.bulk?.placeholderKey : undefined
  return key ? t(key) : field?.placeholder
})
const hint = computed(() => {
  if (props.field === 'base_url' && props.baseUrlHint !== undefined) return props.baseUrlHint
  const key = definition.value?.[props.mode]?.hintKey
  return key ? t(key) : undefined
})
const label = computed(() => {
  const field = definition.value
  if (!field) return ''
  return t(props.mode === 'create' ? field.create?.labelKey ?? field.labelKey : field.labelKey)
})
</script>

<template>
  <div v-if="definition">
    <label v-if="showLabel" :id="labelId" :for="id" class="input-label">{{ label }}</label>
    <input
      :id="id"
      v-model="value"
      :type="definition.type"
      :placeholder="placeholder"
      :required="mode === 'create' && definition.create?.required === true"
      :disabled="disabled"
      :autocomplete="definition.type === 'password' ? 'new-password' : undefined"
      :data-1p-ignore="definition.type === 'password' ? '' : undefined"
      :data-lpignore="definition.type === 'password' ? 'true' : undefined"
      :data-bwignore="definition.type === 'password' ? 'true' : undefined"
      :aria-labelledby="labelId"
      :aria-describedby="hint ? id + '-hint' : undefined"
      class="input disabled:cursor-not-allowed disabled:opacity-50"
      :class="definition.type === 'password' && 'font-mono'"
    />
    <p v-if="hint" :id="id + '-hint'" class="input-hint">{{ hint }}</p>
  </div>
</template>
