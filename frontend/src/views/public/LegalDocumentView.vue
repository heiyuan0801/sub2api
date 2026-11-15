<template>
  <SiteLayout surface="canvas">
    <template #header>
      <SiteHeader>
        <div class="mx-auto flex min-h-14 max-w-5xl items-center justify-between gap-4 px-4 sm:px-6">
          <SiteBrand v-if="settings" :name="siteName" :logo="siteLogo" />
          <RouterLink v-else to="/home" class="flex min-w-0 items-center gap-2" :aria-label="t('design.backHome')">
            <span class="size-7 shrink-0 animate-pulse rounded-lg bg-muted" aria-hidden="true"></span>
            <span class="h-4 w-24 animate-pulse rounded bg-muted" aria-hidden="true"></span>
          </RouterLink>
          <div class="flex shrink-0 items-center gap-1.5">
            <SiteThemeToggle />
            <Button as-child size="sm"><RouterLink to="/login">{{ t('home.login') }}</RouterLink></Button>
          </div>
        </div>
      </SiteHeader>
    </template>

    <main class="mx-auto w-full max-w-4xl flex-1 px-4 py-8 sm:px-6 lg:py-10">
      <div v-if="loading" class="flex min-h-[320px] items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-brand"></div>
      </div>

      <SectionCard v-else-if="loadError" class="border-danger/25 bg-danger/5" role="alert">
        <PageHeader :title="t('legal.loadFailed')" :description="t('legal.retryLater')" />
      </SectionCard>

      <SectionCard v-else-if="!currentDocument">
        <div class="flex items-start gap-3">
          <span class="grid size-9 shrink-0 place-items-center rounded-md border border-border bg-muted text-muted-foreground">
            <Icon name="document" size="sm" aria-hidden="true" />
          </span>
          <PageHeader :title="t('legal.notFound')" :description="t('legal.notFoundDescription')" />
        </div>
      </SectionCard>

      <article v-else class="rounded-lg border border-border bg-card p-4 sm:p-7 md:p-9">
        <div class="mb-7 flex items-start gap-3 border-b border-border pb-6">
          <span class="grid size-10 shrink-0 place-items-center rounded-md border border-border bg-brand/10 text-brand">
            <Icon :name="documentIcon" size="md" aria-hidden="true" />
          </span>
          <PageHeader :title="currentDocument.title" :eyebrow="documentTypeLabel"
            :description="updatedAt ? t('legal.updatedAt', { date: updatedAt }) : undefined"
          />
        </div>

        <div
          v-if="hasContent"
          class="legal-document-content"
          v-html="renderedHtml"
        ></div>
        <EmptyState v-else :title="t('legal.empty')" class="rounded-lg border border-dashed border-border" />
      </article>
    </main>
    <template #footer><SiteMetaFooter :site-name="siteName" /></template>
  </SiteLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { getLocale } from '@/i18n'
import { Button } from '@/components/ui/button'
import { EmptyState, PageHeader, SectionCard } from '@/components/page'
import SiteBrand from '@/components/site-shell/SiteBrand.vue'
import SiteHeader from '@/components/site-shell/SiteHeader.vue'
import SiteLayout from '@/components/site-shell/SiteLayout.vue'
import SiteMetaFooter from '@/components/site-shell/SiteMetaFooter.vue'
import SiteThemeToggle from '@/components/site-shell/SiteThemeToggle.vue'
import { useAppStore } from '@/stores/app'
import type { LoginAgreementDocument } from '@/types'
import zhAdminCompliance from '../../../../docs/legal/admin-compliance.zh.md?raw'
import enAdminCompliance from '../../../../docs/legal/admin-compliance.en.md?raw'

type LegalDocumentIcon = 'document' | 'shield' | 'globe' | 'cog'

const route = useRoute()
const { t } = useI18n()
const appStore = useAppStore()
const settings = computed(() => appStore.cachedPublicSettings)
const loading = ref(!settings.value)
const loadError = ref(false)

marked.setOptions({
  breaks: true,
  gfm: true,
})

const documentId = computed(() => String(route.params.documentId || ''))
const isAdminComplianceDocument = computed(() => documentId.value === 'admin-compliance')
const documents = computed(() => settings.value?.login_agreement_documents ?? [])
const siteName = computed(() => settings.value?.site_name || 'Sub2API')
const siteLogo = computed(() => settings.value?.site_logo || '')
const updatedAt = computed(() =>
  isAdminComplianceDocument.value ? '' : settings.value?.login_agreement_updated_at || ''
)
const documentTypeLabel = computed(() =>
  isAdminComplianceDocument.value ? t('legal.adminCompliance') : t('legal.loginAgreement')
)

const currentDocument = computed<LoginAgreementDocument | null>(() => {
  if (isAdminComplianceDocument.value) {
    return {
      id: 'admin-compliance',
      title: t('adminCompliance.title'),
      content_md: getLocale() === 'zh' ? zhAdminCompliance : enAdminCompliance
    }
  }
  const id = documentId.value
  if (!id) {
    return null
  }
  return documents.value.find((doc) => doc.id === id) ?? null
})

const hasContent = computed(() => Boolean(currentDocument.value?.content_md?.trim()))

const renderedHtml = computed(() => {
  const content = currentDocument.value?.content_md?.trim() || ''
  if (!content) {
    return ''
  }
  const html = marked.parse(content) as string
  return DOMPurify.sanitize(html)
})

const documentIcon = computed<LegalDocumentIcon>(() => {
  const title = currentDocument.value?.title || ''
  if (title.includes('政策') || title.includes('隐私')) {
    return 'shield'
  }
  if (title.includes('国家') || title.includes('地区')) {
    return 'globe'
  }
  if (title.includes('特定')) {
    return 'cog'
  }
  return 'document'
})

onMounted(async () => {
  loadError.value = false
  const loadedSettings = await appStore.fetchPublicSettings()
  if (!loadedSettings) {
    loadError.value = true
  }
  loading.value = false
})
</script>

<style scoped>
.legal-document-content {
  line-height: 1.75;
  overflow-wrap: anywhere;
  color: inherit;
}

.legal-document-content :deep(h1) {
  @apply mb-4 mt-8 border-b border-border pb-3 text-3xl font-bold;
}

.legal-document-content :deep(h2) {
  @apply mb-3 mt-7 text-2xl font-bold;
}

.legal-document-content :deep(h3) {
  @apply mb-2 mt-6 text-xl font-semibold;
}

.legal-document-content :deep(h4) {
  @apply mb-2 mt-5 text-lg font-semibold;
}

.legal-document-content :deep(p) {
  @apply mb-4 text-foreground;
}

.legal-document-content :deep(a) {
  @apply text-brand underline underline-offset-4 hover:text-brand/80;
}

.legal-document-content :deep(ul) {
  @apply mb-4 list-disc pl-6;
}

.legal-document-content :deep(ol) {
  @apply mb-4 list-decimal pl-6;
}

.legal-document-content :deep(li) {
  @apply mb-1 text-foreground;
}

.legal-document-content :deep(blockquote) {
  @apply my-5 border-l-2 border-brand bg-muted/40 py-2 pl-4 text-muted-foreground;
}

.legal-document-content :deep(code) {
  @apply rounded bg-muted px-1.5 py-0.5 font-mono text-sm;
}

.legal-document-content :deep(pre) {
  @apply my-5 overflow-x-auto rounded-lg border border-border bg-muted p-4 text-foreground;
}

.legal-document-content :deep(pre code) {
  @apply bg-transparent p-0 text-inherit;
}

.legal-document-content :deep(table) {
  @apply my-5 block w-full overflow-x-auto border-collapse;
}

.legal-document-content :deep(th) {
  @apply border border-border bg-muted px-3 py-2 text-left font-semibold;
}

.legal-document-content :deep(td) {
  @apply border border-border px-3 py-2;
}

.legal-document-content :deep(img) {
  @apply my-5 h-auto max-w-full rounded-lg;
}

.legal-document-content :deep(hr) {
  @apply my-7 border-border;
}
</style>
