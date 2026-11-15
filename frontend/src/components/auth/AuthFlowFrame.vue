<script setup lang="ts">
import SiteHeader from '@/components/site-shell/SiteHeader.vue'
import SiteLayout from '@/components/site-shell/SiteLayout.vue'
import SiteThemeToggle from '@/components/site-shell/SiteThemeToggle.vue'
</script>

<template>
  <SiteLayout surface="canvas">
    <template #header>
      <SiteHeader>
        <div class="mx-auto flex min-h-14 max-w-7xl items-center justify-between gap-3 px-4 sm:px-7">
          <div class="inline-flex items-center gap-2 text-sm font-semibold">
            <img src="/logo.svg" alt="" class="size-7 rounded-md object-contain" />
            Sub2API
          </div>
          <SiteThemeToggle />
        </div>
      </SiteHeader>
    </template>
    <main class="mx-auto w-full min-w-0 max-w-2xl flex-1 px-4 py-6 sm:px-6 sm:py-10">
      <slot />
    </main>
  </SiteLayout>
</template>
