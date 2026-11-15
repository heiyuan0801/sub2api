<template>
  <SiteLayout surface="canvas">
    <template #header>
      <SiteHeader>
        <div class="mx-auto flex h-14 max-w-5xl items-center justify-between gap-4 px-4 sm:px-6">
          <RouterLink to="/home" class="flex items-center gap-2 rounded-md text-sm font-medium text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
            <Home :size="16" aria-hidden="true" />{{ t('design.backHome') }}
          </RouterLink>
          <SiteThemeToggle />
        </div>
      </SiteHeader>
    </template>

    <main class="mx-auto grid w-full max-w-lg flex-1 place-items-center px-4 py-12 sm:px-6">
      <SectionCard class="w-full">
        <div class="px-2 py-7 sm:px-5 sm:py-9">
          <div class="mb-4 flex items-center gap-2">
            <span class="grid size-9 place-items-center rounded-md border border-border bg-warning/10 text-warning">
              <TriangleAlert :size="18" aria-hidden="true" />
            </span>
            <span class="rounded-md border border-border bg-muted px-2 py-1 font-mono text-xs tabular-nums text-muted-foreground">404</span>
          </div>
          <PageHeader :title="t('errors.pageNotFound')" :description="t('errors.pageNotFoundDescription')" />
          <div class="mt-6 flex flex-wrap gap-2">
            <Button type="button" variant="outline" @click="goBack"><ArrowLeft />{{ t('common.back') }}</Button>
            <Button as-child><RouterLink to="/dashboard">{{ t('home.goToDashboard') }}</RouterLink></Button>
          </div>
        </div>
      </SectionCard>
    </main>
  </SiteLayout>
</template>

<script setup lang="ts">
import { ArrowLeft, Home, TriangleAlert } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { RouterLink, useRouter } from 'vue-router'
import { PageHeader, SectionCard } from '@/components/page'
import { Button } from '@/components/ui/button'
import SiteHeader from '@/components/site-shell/SiteHeader.vue'
import SiteLayout from '@/components/site-shell/SiteLayout.vue'
import SiteThemeToggle from '@/components/site-shell/SiteThemeToggle.vue'

const { t } = useI18n()
const router = useRouter()

function goBack(): void {
  router.back()
}
</script>
