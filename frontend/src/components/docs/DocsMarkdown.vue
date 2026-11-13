<template>
  <div ref="root" class="docs-markdown" @click="handleClick" v-html="rendered.html"></div>
</template>

<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { docsAPI } from '@/api/docs'
import { renderDocMarkdown, type DocHeading } from './markdown'
import { useClipboard } from '@/composables/useClipboard'

const props = defineProps<{ markdown: string; previewArticle?: number; preview?: boolean }>()
const emit = defineEmits<{ headings: [headings: DocHeading[]] }>()
const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const router = useRouter()
const route = useRoute()
const root = ref<HTMLElement>()
const rendered = computed(() => renderDocMarkdown(props.markdown, t('docs.copy'), {
  NOTE: t('docs.note'), TIP: t('docs.tip'), IMPORTANT: t('docs.important'),
  WARNING: t('docs.warning'), CAUTION: t('docs.caution'),
}, props.preview ? 'doc-preview-' : 'doc-'))

watch(rendered, value => emit('headings', value.headings), { immediate: true })

watch(() => [rendered.value.html, props.previewArticle], async (_, __, cleanup) => {
  const controller = new AbortController()
  const urls: string[] = []
  cleanup(() => {
    controller.abort()
    urls.forEach(url => URL.revokeObjectURL(url))
  })
  await nextTick()
  const images = root.value?.querySelectorAll<HTMLImageElement>('img[data-docs-image]')
  if (!images || controller.signal.aborted) return
  const pending = Array.from(images)
  // Keep page load and search responsive when a document includes many images.
  await Promise.all(Array.from({ length: Math.min(4, pending.length) }, async () => {
    let image: HTMLImageElement | undefined
    while (!controller.signal.aborted && (image = pending.shift())) {
      try {
        const blob = await docsAPI.image(image.dataset.docsImage!, controller.signal, props.previewArticle)
        if (controller.signal.aborted) return
        const url = URL.createObjectURL(blob)
        urls.push(url)
        image.src = url
      } catch {
        if (!controller.signal.aborted) image.classList.add('docs-image-missing')
      }
    }
  }))
}, { immediate: true, flush: 'post' })

async function handleClick(event: MouseEvent) {
  if (!(event.target instanceof Element)) return
  const copy = event.target.closest<HTMLButtonElement>('.docs-copy')
  if (copy) {
    const source = copy.closest('.docs-code')?.querySelector('code')?.textContent || ''
    const ok = await copyToClipboard(source)
    copy.textContent = t(ok ? 'docs.copied' : 'docs.copyFailed')
    setTimeout(() => { copy.textContent = t('docs.copy') }, 1500)
    return
  }
  const link = event.target.closest('a[href]')
  if (!link || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return
  const href = link.getAttribute('href') || ''
  if (href.startsWith('#') && !props.preview) {
    event.preventDefault()
    void router.push({ path: route.path, query: route.query, hash: href })
  } else if (/^\/(?!\/)/.test(href) && !props.preview) {
    event.preventDefault()
    void router.push(href)
  } else if (props.preview) {
    event.preventDefault()
  }
}
</script>

<style>
.docs-markdown { overflow-wrap: anywhere; font-size: .9375rem; line-height: 1.8; }
.docs-markdown :where(p, ul, ol, blockquote, .docs-code, .docs-table) { margin: 1rem 0; }
.docs-markdown :where(h1, h2, h3, h4, h5, h6) { color: hsl(var(--foreground)); margin: 1.8em 0 .45em; font-weight: 650; line-height: 1.3; scroll-margin-top: 7rem; }
.docs-markdown :where(h1, h2) { font-size: 1.48rem; letter-spacing: -.025em; }
.docs-markdown h3 { font-size: 1.2rem; }
.docs-markdown :where(ol, ul) { padding-inline-start: 1.6em; }
.docs-markdown ol { list-style: decimal; }
.docs-markdown ul { list-style: disc; }
.docs-markdown a { text-decoration: underline; text-underline-offset: .2em; font-weight: 500; }
.docs-markdown a:hover { color: hsl(var(--muted-foreground)); }
.docs-markdown img { max-width: 100%; height: auto; border-radius: .65rem; margin: .75rem 0; }
.docs-markdown .docs-image-missing { opacity: .6; }
.docs-markdown :not(pre) > code { background: hsl(var(--muted)); padding: .18em .36em; border-radius: .3rem; font-size: .875em; }
.docs-markdown blockquote { border-inline-start: 3px solid hsl(var(--border)); padding: .55rem 1rem; color: hsl(var(--muted-foreground)); }
.docs-markdown blockquote p { margin: .35rem 0; }
.docs-markdown .docs-callout { border: 1px solid hsl(var(--border)); border-inline-start: 3px solid #38a584; border-radius: .65rem; background: hsl(var(--muted) / .55); color: hsl(var(--foreground)); }
.docs-markdown .docs-callout-warning, .docs-markdown .docs-callout-important { border-inline-start-color: #d99025; }
.docs-markdown .docs-callout-caution { border-inline-start-color: #e96564; }
.docs-callout-label { font-size: .85rem; }
.docs-code, .docs-table { min-width: 0; border: 1px solid hsl(var(--border)); border-radius: .7rem; overflow: hidden; }
.docs-code { background: #131820; color: #dde5ef; }
.docs-code-header { display: flex; align-items: center; justify-content: space-between; padding: .45rem .85rem; border-bottom: 1px solid #2c3340; color: #a5b3c4; font: .75rem/1.5 ui-monospace, monospace; }
.docs-copy { border: 1px solid #465366; border-radius: .35rem; padding: .15rem .45rem; color: #ebf1f7; }
.docs-copy:hover { background: #2c3440; }
.docs-code pre { overflow: auto; padding: .85rem; font-size: .83rem; line-height: 1.65; }
.docs-code .hljs-keyword, .docs-code .hljs-selector-tag { color: #c5a4fc; }
.docs-code .hljs-string, .docs-code .hljs-regexp { color: #9ce0aa; }
.docs-code .hljs-number, .docs-code .hljs-literal { color: #ffba8c; }
.docs-code .hljs-comment { color: #a2adbc; }
.docs-code .hljs-title, .docs-code .hljs-attr, .docs-code .hljs-built_in, .docs-code .hljs-variable { color: #a2d4ff; }
.docs-table { overflow-x: auto; }
.docs-table table { width: 100%; border-collapse: collapse; }
.docs-table th, .docs-table td { padding: .55rem .9rem; text-align: start; border-bottom: 1px solid hsl(var(--border)); }
.docs-table th { background: hsl(var(--muted)); }
.docs-table tr:last-child td { border-bottom: 0; }
</style>
