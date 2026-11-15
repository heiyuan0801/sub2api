<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import SiteFooter from './SiteFooter.vue'

defineProps<{ siteName: string }>()
const { t } = useI18n()
const year = new Date().getFullYear()
</script>

<template>
  <SiteFooter>
    <div class="mx-auto flex max-w-[1500px] flex-wrap items-center justify-between gap-x-4 gap-y-2 px-4 py-3 text-xs sm:px-7">
      <span>{{ t('publicSite.footer.copyright', { year, siteName }) }}</span>
      <RouterLink to="/home" class="rounded-sm hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
        {{ t('design.backHome') }}
      </RouterLink>
    </div>
  </SiteFooter>
</template>
