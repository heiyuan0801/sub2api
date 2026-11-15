<script setup lang="ts">
import PublicSiteHeader from '@/components/public/PublicSiteHeader.vue'
import PublicSiteFooter from '@/components/public/PublicSiteFooter.vue'
import SiteLayout from './SiteLayout.vue'

defineProps<{ footerVariant?: 'home' }>()
</script>

<template>
  <SiteLayout>
    <template #header><PublicSiteHeader /></template>
    <slot />
    <template #footer><PublicSiteFooter :variant="footerVariant" /></template>
  </SiteLayout>
</template>
