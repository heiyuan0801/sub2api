<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import { sanitizeUrl } from '@/utils/url'

const props = defineProps<{ name: string; logo?: string; hideNameOnMobile?: boolean }>()
const { t } = useI18n()
const safeLogo = computed(() => sanitizeUrl(props.logo || '', { allowRelative: true, allowDataUrl: true }) || '/logo.svg')
</script>

<template>
  <RouterLink
    to="/home"
    class="inline-flex min-w-0 items-center gap-2 rounded-md text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    :aria-label="t('publicSite.nav.homeLabel', { siteName: name })"
  >
    <img :src="safeLogo" alt="" class="size-7 shrink-0 rounded-md object-contain" />
    <span class="min-w-0 max-w-48 truncate text-sm font-semibold sm:max-w-52" :class="hideNameOnMobile && 'hidden sm:inline'">
      {{ name }}
    </span>
  </RouterLink>
</template>
