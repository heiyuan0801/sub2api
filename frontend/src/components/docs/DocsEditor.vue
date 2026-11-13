<template>
  <BaseDialog :show="true" :title="form.id ? t('docs.edit') + ' · ' + (form.title || t('docs.articles')) : t('docs.newArticle')" width="full" :close-on-escape="!busy" :show-close-button="!busy" @close="close">
    <form class="space-y-5" @submit.prevent="save(false)">
      <div v-if="error" class="rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-destructive" role="alert">{{ error }}</div>
      <fieldset :disabled="busy" class="grid gap-3 md:grid-cols-2">
        <label class="docs-field">{{ t('docs.articleTitle') }}<input v-model="form.title" class="input" maxlength="160" required /></label>
        <label class="docs-field">{{ t('docs.slug') }}<input v-model.trim="form.slug" class="input" maxlength="96" required placeholder="example-guide" /><small>{{ t('docs.slugHint') }}</small></label>
        <label class="docs-field">{{ t('docs.categories') }}
          <select v-model.number="form.category_id" class="input" required>
            <option :value="0" disabled>{{ t('docs.selectCategory') }}</option>
            <option v-for="category in categories" :key="category.id" :value="category.id">{{ category.name }}</option>
          </select>
        </label>
        <label class="docs-field">{{ t('docs.visibility') }}
          <select v-model="form.visibility" class="input">
            <option value="public">{{ t('docs.public') }}</option>
            <option value="authenticated">{{ t('docs.authenticated') }}</option>
          </select>
        </label>
        <label class="docs-field md:col-span-2">{{ t('docs.description') }}<input v-model="form.description" class="input" maxlength="320" /></label>
      </fieldset>
      <details class="rounded-lg border border-border px-3 py-2">
        <summary class="cursor-pointer text-sm font-medium">SEO</summary>
        <fieldset :disabled="busy" class="mt-3 grid gap-3 md:grid-cols-2">
          <label class="docs-field">{{ t('docs.seoTitle') }}<input v-model="form.seo_title" class="input" maxlength="160" /></label>
          <label class="docs-field">{{ t('docs.seoDescription') }}<input v-model="form.seo_description" class="input" maxlength="320" /></label>
        </fieldset>
      </details>
      <div class="overflow-hidden rounded-xl border border-border">
        <div class="flex flex-wrap items-center justify-between gap-2 border-b border-border bg-muted/40 px-3 py-2">
          <div class="flex items-center gap-1">
            <Button type="button" size="sm" class="lg:hidden" :variant="!mobilePreview ? 'secondary' : 'ghost'" @click="mobilePreview = false">{{ t('docs.markdown') }}</Button>
            <Button type="button" size="sm" class="lg:hidden" :variant="mobilePreview ? 'secondary' : 'ghost'" @click="mobilePreview = true">{{ t('docs.preview') }}</Button>
            <span class="hidden text-sm font-semibold lg:inline">{{ t('docs.markdown') }}</span>
          </div>
          <div class="flex items-center gap-2">
            <span v-if="dirty" class="text-xs text-muted-foreground">{{ t('docs.unsaved') }}</span>
            <Button type="button" variant="outline" size="sm" :disabled="busy || !form.id" :title="form.id ? t('docs.imageHint') : t('docs.saveForUpload')" @click="fileInput?.click()"><ImagePlus /> {{ t('docs.uploadImage') }}</Button>
            <input ref="fileInput" type="file" class="hidden" accept="image/png,image/jpeg,image/webp,image/gif" @change="upload" />
          </div>
        </div>
        <div class="grid lg:grid-cols-2">
          <textarea ref="editor" v-model="form.markdown" :readonly="busy" :class="['min-h-80 w-full resize-y bg-background p-4 font-mono text-sm leading-6 outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring/50', mobilePreview && 'hidden lg:block']"
            :aria-label="t('docs.markdown')" :placeholder="t('docs.editorHint')" maxlength="262144" spellcheck="false"></textarea>
          <section :class="['min-h-80 min-w-0 border-border px-4 pb-4 lg:max-h-[500px] lg:overflow-y-auto lg:border-l', !mobilePreview && 'hidden lg:block']">
            <h4 class="sticky top-0 z-10 border-b border-border bg-background py-2 text-xs font-semibold text-muted-foreground">{{ t('docs.preview') }}</h4>
            <DocsMarkdown v-if="previewMarkdown" preview :preview-article="form.id || undefined" :markdown="previewMarkdown" />
          </section>
        </div>
      </div>
      <p class="text-xs text-muted-foreground">{{ form.id ? t('docs.imageHint') : t('docs.saveForUpload') }}</p>
    </form>
    <template #footer>
      <div class="flex w-full flex-wrap items-center justify-end gap-2">
        <RouterLink v-if="form.published_slug" :to="{ name: 'DocsArticle', params: { slug: form.published_slug } }" target="_blank" class="mr-auto inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground">
          {{ t('docs.openPublished') }} <ArrowUpRight class="size-3.5" />
        </RouterLink>
        <Button type="button" variant="outline" :disabled="busy" @click="close">{{ t('common.cancel') }}</Button>
        <Button type="button" variant="outline" :disabled="busy" @click="save(false)">{{ t('docs.saveDraft') }}</Button>
        <Button type="button" :disabled="busy || !form.markdown.trim()" @click="save(true)">{{ t(form.published_slug ? 'docs.republish' : 'docs.publish') }}</Button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, ImagePlus } from '@lucide/vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { Button } from '@/components/ui/button'
import { docsAPI, type DocArticle, type DocCategory, type DocInput } from '@/api/docs'
import { useAppStore } from '@/stores/app'
import DocsMarkdown from './DocsMarkdown.vue'

const props = defineProps<{ article?: DocArticle; category: number; categories: DocCategory[] }>()
const emit = defineEmits<{ close: []; changed: [] }>()
const { t } = useI18n()
const app = useAppStore()
const form = ref(props.article ? { ...props.article } : {
  id: 0, category_id: props.category, slug: '', title: '', description: '', markdown: '',
  seo_title: '', seo_description: '', visibility: 'public' as const, revision: 0,
  published_slug: '', published_revision: 0, status: 'draft' as const, updated_at: '',
})
const saved = ref(JSON.stringify(form.value))
const dirty = computed(() => JSON.stringify(form.value) !== saved.value)
const busy = ref(false)
const error = ref('')
const mobilePreview = ref(false)
const editor = ref<HTMLTextAreaElement>()
const fileInput = ref<HTMLInputElement>()
const previewMarkdown = ref(form.value.markdown)

watch(() => form.value.markdown, (text, _, cleanup) => {
  const timer = setTimeout(() => { previewMarkdown.value = text }, 120)
  cleanup(() => clearTimeout(timer))
})

function close() {
  if (!busy.value && (!dirty.value || window.confirm(t('docs.discard')))) emit('close')
}

function setError(err: unknown) {
  const reason = (err as { reason?: string }).reason
  error.value = t(reason === 'DOC_CONFLICT' ? 'docs.conflict' : reason === 'DOC_INVALID' ? 'docs.invalid' : 'docs.changeFailed')
}

async function save(publish: boolean) {
  if (busy.value) return
  const { category_id, slug, title, description, markdown, seo_title, seo_description, visibility, revision } = form.value
  if (!title.trim() || !category_id || !/^[a-z0-9]+(-[a-z0-9]+)*$/.test(slug) || slug === 'batch-image' || new Blob([markdown]).size > 256 * 1024) {
    error.value = t('docs.invalid')
    return
  }
  const input: DocInput = { category_id, slug, title, description, markdown, seo_title, seo_description, visibility, revision }
  busy.value = true
  error.value = ''
  try {
    if (!form.value.id || dirty.value) {
      form.value = await docsAPI.saveArticle(input, form.value.id || undefined)
      saved.value = JSON.stringify(form.value)
      emit('changed')
    }
    if (publish) {
      form.value = await docsAPI.publish(form.value.id, form.value.revision, true)
      saved.value = JSON.stringify(form.value)
      emit('changed')
    }
    app.showSuccess(t(publish ? 'docs.publishSuccess' : 'docs.saved'))
  } catch (err) {
    setError(err)
  } finally {
    busy.value = false
  }
}

async function upload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file || !form.value.id || busy.value) return
  if (file.size > 5 * 1024 * 1024) { error.value = t('docs.imageHint'); return }
  busy.value = true
  error.value = ''
  const start = editor.value?.selectionStart ?? form.value.markdown.length
  try {
    const path = await docsAPI.uploadImage(form.value.id, file)
    const snippet = `\n\n![${t('docs.imageAlt')}](${path})\n`
    const value = form.value.markdown
    form.value.markdown = value.slice(0, start) + snippet + value.slice(start)
    mobilePreview.value = false
    await nextTick()
    editor.value?.focus()
    editor.value?.setSelectionRange(start + snippet.length, start + snippet.length)
  } catch (err) {
    setError(err)
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.docs-field { display: flex; flex-direction: column; gap: .35rem; color: hsl(var(--foreground)); font-size: .875rem; font-weight: 500; }
.docs-field small { color: hsl(var(--muted-foreground)); font-size: .75rem; font-weight: 400; }
</style>
