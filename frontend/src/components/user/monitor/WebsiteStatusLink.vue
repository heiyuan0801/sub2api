<template>
  <a
    href="https://status.weilai.chat/status/weilai"
    target="_blank"
    rel="noopener noreferrer"
    class="btn btn-secondary btn-sm shrink-0 gap-1.5"
  >
    <Icon name="globe" size="sm" aria-hidden="true" />
    {{ t('channelStatus.websiteStatus') }}
    <Icon name="externalLink" size="xs" aria-hidden="true" />
    <span class="sr-only">{{ t('channelStatus.opensInNewTab') }}</span>
  </a>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
</script>
