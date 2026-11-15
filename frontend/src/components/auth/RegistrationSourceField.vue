<template>
  <div class="space-y-2">
    <label for="registration-source" class="input-label">{{ t('operations.sourceQuestion') }} <span class="font-normal text-muted-foreground">({{ t('operations.optional') }})</span></label>
    <select id="registration-source" v-model="source" class="input" :disabled="disabled">
      <option value="">{{ t('operations.noSourceSelected') }}</option>
      <option v-for="item in registrationSources" :key="item" :value="item">{{ t(`operations.sourceLabels.${item}`) }}</option>
    </select>
    <div v-if="source === 'other'">
      <label for="registration-source-note" class="input-label">{{ t('operations.sourceNote') }}</label>
      <input id="registration-source-note" v-model="note" maxlength="140" class="input" :disabled="disabled" type="text" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { registrationSources, type RegistrationSource } from '@/types/operations'

defineProps<{ disabled?: boolean }>()
const source = defineModel<RegistrationSource | ''>({ required: true })
const note = defineModel<string>('note', { required: true })
const { t } = useI18n()
</script>
