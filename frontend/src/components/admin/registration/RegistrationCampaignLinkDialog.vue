<template>
  <BaseDialog :show="show" :title="t('operations.campaignLink.title')" @close="emit('close')">
    <form id="registration-campaign-link" class="space-y-4" @submit.prevent="copyLink">
      <div>
        <label for="campaign-source" class="input-label">{{ t('operations.utmSource') }}</label>
        <select id="campaign-source" v-model="source" class="input" required>
          <option value="" disabled>{{ t('operations.campaignLink.chooseSource') }}</option>
          <option v-for="option in sourceOptions" :key="option" :value="option">
            {{ t(`operations.sourceLabels.${option}`) }}
          </option>
          <option value="custom">{{ t('operations.campaignLink.customSource') }}</option>
        </select>
      </div>
      <div v-if="source === 'custom'">
        <label for="campaign-custom-source" class="input-label">{{ t('operations.campaignLink.customSource') }}</label>
        <input
          id="campaign-custom-source"
          v-model="customSource"
          class="input"
          type="text"
          :maxlength="registrationUTMValueLimit"
          :placeholder="t('operations.campaignLink.customSourcePlaceholder')"
          :aria-invalid="utmSource === 'unknown'"
          :aria-describedby="utmSource === 'unknown' ? 'campaign-source-error' : undefined"
          autocomplete="off"
          required
        />
        <p v-if="utmSource === 'unknown'" id="campaign-source-error" class="input-hint text-red-600 dark:text-red-400">
          {{ t('operations.campaignLink.reservedSource') }}
        </p>
      </div>
      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label for="campaign-medium" class="input-label">
            {{ t('operations.campaignLink.medium') }}
            <span class="font-normal text-muted-foreground">({{ t('operations.optional') }})</span>
          </label>
          <input id="campaign-medium" v-model="medium" class="input" type="text" :maxlength="registrationUTMValueLimit" placeholder="social" autocomplete="off" />
        </div>
        <div>
          <label for="campaign-name" class="input-label">
            {{ t('operations.campaign') }}
            <span class="font-normal text-muted-foreground">({{ t('operations.optional') }})</span>
          </label>
          <input id="campaign-name" v-model="campaign" class="input" type="text" :maxlength="registrationUTMValueLimit" autocomplete="off" />
        </div>
      </div>
      <div>
        <label for="campaign-preview" class="input-label">{{ t('operations.campaignLink.preview') }}</label>
        <textarea
          id="campaign-preview"
          :value="link"
          :placeholder="t('operations.campaignLink.chooseSource')"
          class="input resize-none break-all font-mono text-xs"
          rows="3"
          readonly
          @focus="($event.target as HTMLTextAreaElement).select()"
        ></textarea>
      </div>
    </form>
    <template #footer>
      <div class="flex flex-wrap justify-end gap-2">
        <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.close') }}</button>
        <button type="submit" form="registration-campaign-link" class="btn btn-primary gap-2" :disabled="!link">
          <Icon name="copy" size="sm" aria-hidden="true" />
          {{ t('operations.copyRegistrationLink') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { registrationSources } from '@/types/operations'
import { normalizeRegistrationUTMValue, registrationUTMValueLimit } from '@/utils/registrationAttribution'

defineProps<{ show: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const router = useRouter()
const { copyToClipboard } = useClipboard()

const sourceOptions = registrationSources.filter(option => option !== 'other')
const source = ref('')
const customSource = ref('')
const medium = ref('')
const campaign = ref('')
const utmSource = computed(() => {
  const value = source.value === 'custom' ? customSource.value : source.value
  return normalizeRegistrationUTMValue(value.toLowerCase())
})
const link = computed(() => {
  if (!utmSource.value || utmSource.value === 'unknown') return ''
  const { href } = router.resolve({
    name: 'Register',
    query: {
      utm_source: utmSource.value,
      utm_medium: normalizeRegistrationUTMValue(medium.value) || undefined,
      utm_campaign: normalizeRegistrationUTMValue(campaign.value) || undefined,
    },
  })
  return new URL(href, window.location.origin).href
})

function copyLink() {
  if (link.value) void copyToClipboard(link.value)
}
</script>
