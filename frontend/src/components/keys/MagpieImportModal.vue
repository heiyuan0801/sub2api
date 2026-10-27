<template>
  <BaseDialog :show="true" :title="t('keys.magpie.title')" @close="$emit('close')">
    <div class="space-y-4">
      <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('keys.magpie.description') }}</p>
      <p class="font-medium">{{ apiKey.name }}</p>
      <dl v-if="config" class="space-y-2 rounded-xl border border-gray-200 p-4 text-sm dark:border-dark-600">
        <div v-for="(endpoint, protocol) in config.endpoints" :key="protocol">
          <dt class="font-medium">{{ protocol }}</dt><dd class="break-all text-gray-500">{{ endpoint }}</dd>
        </div>
      </dl>
      <p v-else role="alert" class="text-sm text-red-600">{{ t('keys.magpie.invalidUrl') }}</p>
      <p class="text-xs text-gray-500">{{ t('keys.magpie.keyNotice') }}</p>
    </div>
    <template #footer>
      <button class="btn btn-secondary" @click="$emit('close')">{{ t('common.cancel') }}</button>
      <a v-if="config" :href="config.fallback" target="_blank" rel="noopener noreferrer" referrerpolicy="no-referrer" class="btn btn-secondary">{{ t('keys.magpie.fallback') }}</a>
      <a v-if="config" :href="config.deeplink" class="btn btn-primary">{{ t('keys.magpie.open') }}</a>
    </template>
  </BaseDialog>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { buildMagpieImport } from '@/utils/magpieImport'
import type { ApiKey } from '@/types'
const props = defineProps<{ apiKey: ApiKey; baseUrl: string; siteName: string }>()
defineEmits<{ close: [] }>()
const { t } = useI18n()
const config = computed(() => {
  try { return buildMagpieImport({ baseUrl: props.baseUrl, providerName: `${props.siteName} · ${props.apiKey.name}`, apiKey: props.apiKey.key, keyId: props.apiKey.id, platform: props.apiKey.is_aggregate ? 'composite' : props.apiKey.group?.platform }) }
  catch { return null }
})
</script>
