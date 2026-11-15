<template>
  <SiteLayout surface="canvas" class="docs-page">
    <template #header>
      <SiteHeader sticky>
        <div class="mx-auto flex h-16 max-w-[1500px] items-center gap-2 px-4 sm:gap-4 sm:px-7">
          <SiteBrand :name="app.siteName" :logo="app.siteLogo" hide-name-on-mobile />
          <span class="text-muted-foreground" aria-hidden="true">/</span>
          <RouterLink to="/docs" class="shrink-0 text-sm font-semibold">{{ t('docs.title') }}</RouterLink>
          <div class="ml-auto flex items-center gap-1 sm:gap-2">
            <Button variant="outline" size="sm" class="sm:w-52 sm:justify-start" @click="searchOpen = true" :aria-label="t('docs.search')">
              <Search />
              <span class="hidden text-muted-foreground sm:inline">{{ t('docs.search') }}</span>
              <kbd class="ml-auto hidden rounded border px-1.5 text-[10px] text-muted-foreground sm:inline">⌘/Ctrl K</kbd>
            </Button>
            <SiteThemeToggle />
            <Button as-child variant="ghost" size="sm">
              <RouterLink :to="auth.isAuthenticated ? consolePath : loginPath">
                {{ t(auth.isAuthenticated ? 'docs.console' : 'home.login') }} <ArrowUpRight />
              </RouterLink>
            </Button>
          </div>
        </div>
        <div class="mx-auto flex max-w-[1500px] items-center justify-between gap-2 border-t border-border px-4 py-1.5 xl:hidden">
          <Button variant="ghost" size="sm" class="lg:hidden" @click="navigationOpen = true"><Menu /> {{ t('docs.categories') }}</Button>
          <Button v-if="headings.length" variant="ghost" size="sm" class="ml-auto" @click="outlineOpen = true"><ListTree /> {{ t('docs.onThisPage') }}</Button>
        </div>
      </SiteHeader>
    </template>

    <div class="docs-grid mx-auto grid w-full max-w-[1500px] flex-1 gap-5 px-4 py-6 sm:px-7 lg:py-8">
      <aside class="docs-left hidden rounded-lg border border-border bg-card p-3 lg:block" :aria-label="t('docs.categories')">
        <DocsNavigation :navigation="navigation" :active="article?.slug" />
      </aside>

      <main id="docs-content" class="mx-auto w-full min-w-0 max-w-[860px]">
        <SectionCard v-if="loading" class="docs-article" aria-busy="true">
          <div class="space-y-4">
            <div class="h-4 w-28 animate-pulse rounded bg-muted"></div>
            <div class="h-11 w-4/5 animate-pulse rounded bg-muted"></div>
            <div class="h-4 w-3/5 animate-pulse rounded bg-muted"></div>
          </div>
        </SectionCard>
        <SectionCard v-else-if="problem || !article" class="docs-article">
          <span class="mb-3 grid size-9 place-items-center rounded-md border border-border bg-muted text-muted-foreground">
            <BookOpen class="size-4" aria-hidden="true" />
          </span>
          <PageHeader :title="t(problem || 'docs.empty')" />
          <div class="mt-5 flex flex-wrap gap-2">
            <Button v-if="problem === 'docs.loadFailed'" variant="outline" @click="reload">{{ t('docs.retry') }}</Button>
            <Button v-if="!auth.isAuthenticated" as-child><RouterLink :to="loginPath">{{ t('home.login') }}</RouterLink></Button>
            <Button v-if="slug" as-child variant="outline"><RouterLink to="/docs">{{ t('docs.back') }}</RouterLink></Button>
          </div>
        </SectionCard>
        <article v-else class="docs-article rounded-lg border border-border bg-card">
          <PageHeader :title="article.title" :description="article.description" :eyebrow="categoryName">
            <template #badge>
              <StatusBadge v-if="article.visibility === 'authenticated'" :label="t('docs.authenticated')" :icon="LockKeyhole" />
            </template>
          </PageHeader>
          <div class="my-6 border-t border-border"></div>
          <DocsMarkdown :key="article.id" :markdown="article.markdown" @headings="headings = $event" />
          <nav v-if="previous || next" :aria-label="t('docs.more')" class="mt-12 grid gap-3 sm:grid-cols-2">
            <RouterLink v-if="previous" :to="articleRoute(previous.slug)" class="docs-neighbor">
              <span><ArrowLeft class="size-3.5" />{{ t('docs.previous') }}</span>
              <strong>{{ previous.title }}</strong>
            </RouterLink>
            <RouterLink v-if="next" :to="articleRoute(next.slug)" class="docs-neighbor sm:col-start-2 sm:text-right">
              <span class="sm:justify-end">{{ t('docs.next') }}<ArrowRight class="size-3.5" /></span>
              <strong>{{ next.title }}</strong>
            </RouterLink>
          </nav>
        </article>
      </main>

      <aside class="docs-right hidden rounded-lg border border-border bg-card p-3 xl:block" v-show="headings.length" :aria-label="t('docs.onThisPage')">
          <h2 class="mb-3 flex items-center gap-2 text-sm font-semibold"><ListTree class="size-4" /> {{ t('docs.onThisPage') }}</h2>
          <nav class="border-l border-border">
            <a v-for="heading in headings" :key="heading.id" :href="headingHref(heading.id)" :aria-current="activeHeading === heading.id ? 'location' : undefined"
              :class="['docs-heading-link', { 'is-active': activeHeading === heading.id }]" :style="{ paddingInlineStart: `${12 + Math.max(0, heading.level - 2) * 10}px` }"
              @click.prevent="jumpTo(heading.id)">{{ heading.text }}</a>
          </nav>
      </aside>
    </div>

    <BaseDialog :show="navigationOpen" :title="t('docs.categories')" panel-class="rounded-lg border border-border" close-on-click-outside @close="navigationOpen = false">
      <DocsNavigation :navigation="navigation" :active="article?.slug" @navigate="navigationOpen = false" />
    </BaseDialog>
    <BaseDialog :show="outlineOpen" :title="t('docs.onThisPage')" panel-class="rounded-lg border border-border" close-on-click-outside @close="outlineOpen = false">
      <nav>
        <a v-for="heading in headings" :key="heading.id" :href="headingHref(heading.id)" class="docs-heading-link"
          :style="{ paddingInlineStart: `${Math.max(0, heading.level - 2) * 12}px` }" @click.prevent="jumpTo(heading.id)">{{ heading.text }}</a>
      </nav>
    </BaseDialog>
    <DocsSearch :open="searchOpen" @close="searchOpen = false" />
    <template #footer><SiteMetaFooter :site-name="app.siteName" /></template>
  </SiteLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ArrowLeft, ArrowRight, ArrowUpRight, BookOpen, ListTree, LockKeyhole, Menu, Search } from '@lucide/vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { Button } from '@/components/ui/button'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { PageHeader, SectionCard, StatusBadge } from '@/components/page'
import { docsAPI, type DocNavigation, type DocPublication } from '@/api/docs'
import SiteBrand from '@/components/site-shell/SiteBrand.vue'
import SiteHeader from '@/components/site-shell/SiteHeader.vue'
import SiteLayout from '@/components/site-shell/SiteLayout.vue'
import SiteMetaFooter from '@/components/site-shell/SiteMetaFooter.vue'
import SiteThemeToggle from '@/components/site-shell/SiteThemeToggle.vue'
import DocsMarkdown from '@/components/docs/DocsMarkdown.vue'
import DocsNavigation from '@/components/docs/DocsNavigation.vue'
import DocsSearch from '@/components/docs/DocsSearch.vue'
import type { DocHeading } from '@/components/docs/markdown'

const { t, locale } = useI18n()
const app = useAppStore()
const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const navigation = ref<DocNavigation>({ categories: [], articles: [] })
const article = ref<DocPublication>()
const headings = ref<DocHeading[]>([])
const activeHeading = ref('')
const loading = ref(true)
const problem = ref('')
const refresh = ref(0)
const navigationOpen = ref(false)
const outlineOpen = ref(false)
const searchOpen = ref(false)
const slug = computed(() => typeof route.params.slug === 'string' ? route.params.slug : '')
const loginPath = computed(() => ({ path: '/login', query: { redirect: route.fullPath } }))
const consolePath = computed(() => auth.isAdmin ? '/admin/dashboard' : '/dashboard')
const categoryName = computed(() => navigation.value.categories.find(item => item.id === article.value?.category_id)?.name)
const index = computed(() => navigation.value.articles.findIndex(item => item.id === article.value?.id))
const previous = computed(() => index.value > 0 ? navigation.value.articles[index.value - 1] : undefined)
const next = computed(() => index.value >= 0 ? navigation.value.articles[index.value + 1] : undefined)
const articleRoute = (slug: string) => ({ name: 'DocsArticle', params: { slug } })
const reload = () => { refresh.value++ }

watch([slug, () => auth.isAuthenticated, refresh], async ([currentSlug], _, cleanup) => {
  const controller = new AbortController()
  cleanup(() => controller.abort())
  loading.value = true
  problem.value = ''
  article.value = undefined
  headings.value = []
  navigation.value = { categories: [], articles: [] }
  try {
    const nav = await docsAPI.navigation(controller.signal)
    if (controller.signal.aborted) return
    navigation.value = nav
    const target = currentSlug || nav.articles[0]?.slug
    if (target) {
      const result = await docsAPI.article(target, controller.signal)
      if (!controller.signal.aborted) article.value = result
    }
  } catch (error) {
    if (controller.signal.aborted) return
    const status = (error as { status?: number }).status
    problem.value = status === 404 ? 'docs.notFound' : status === 403 ? 'docs.unavailable' : 'docs.loadFailed'
  } finally {
    if (!controller.signal.aborted) loading.value = false
  }
}, { immediate: true })

function headingHref(id: string) {
  return router.resolve({ path: route.path, query: route.query, hash: '#' + id }).href
}
function jumpTo(id: string) {
  outlineOpen.value = false
  if (route.hash === '#' + id) scrollToHeading(id)
  else void router.push({ path: route.path, query: route.query, hash: '#' + id })
}

watch([() => route.hash, headings, loading], async ([hash]) => {
  await nextTick()
  if (hash) {
    scrollToHeading(hash.slice(1))
    activeHeading.value = hash.slice(1)
  } else {
    trackHeading()
  }
}, { flush: 'post' })

function headingElement(id: string) {
  return document.querySelector<HTMLElement>(`#docs-content .docs-markdown [id="${CSS.escape('doc-' + id)}"]`)
}
function scrollToHeading(id: string) {
  headingElement(id)?.scrollIntoView()
}
function trackHeading() {
  let current = ''
  for (const heading of headings.value) {
    const node = headingElement(heading.id)
    if (node && node.getBoundingClientRect().top <= 135) current = heading.id
    else break
  }
  activeHeading.value = current
}

function setMeta(name: string, value: string) {
  let tag = document.querySelector<HTMLMetaElement>(`meta[name="${name}"]`)
  if (!value) { tag?.remove(); return }
  if (!tag) {
    tag = document.createElement('meta')
    tag.name = name
    document.head.append(tag)
  }
  tag.content = value
}

watch([article, loading, locale, () => app.siteName, () => route.fullPath], ([item, busy]) => {
  if (route.name !== 'DocsIndex' && route.name !== 'DocsArticle') return
  if (busy && !item) return
  const isPublic = item?.visibility === 'public'
  document.title = slug.value && isPublic ? (item.seo_title || `${item.title} · ${app.siteName}`) : `${item?.title || t('docs.title')} · ${app.siteName}`
  setMeta('description', slug.value && isPublic ? item.seo_description || item.description : '')
  setMeta('robots', slug.value && !isPublic ? 'noindex' : '')
}, { immediate: true })

function onKey(event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
    event.preventDefault()
    searchOpen.value = !searchOpen.value
  }
}

onMounted(() => {
  void app.fetchPublicSettings()
  window.addEventListener('keydown', onKey)
  window.addEventListener('scroll', trackHeading, { passive: true })
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKey)
  window.removeEventListener('scroll', trackHeading)
  setMeta('description', '')
  setMeta('robots', '')
})
</script>

<style scoped>
.docs-grid { grid-template-columns: minmax(0, 1fr); }
@media (min-width: 1024px) {
  .docs-grid { grid-template-columns: 225px minmax(0, 1fr); }
}
@media (min-width: 1280px) {
  .docs-grid { grid-template-columns: 225px minmax(0, 1fr) 205px; }
}
.docs-left, .docs-right { position: sticky; top: 7rem; align-self: start; max-height: calc(100dvh - 8rem); overflow-y: auto; }
.docs-article { padding: clamp(1rem, 3vw, 2.5rem); }
.docs-article :deep([data-slot="page-section-body"]) { padding: 0; }
.docs-article :deep([data-slot="page-header"] h1) { font-size: clamp(1.45rem, 3vw, 2.1rem); }
@media (min-width: 1280px) {
  .docs-left, .docs-right { top: 5.5rem; max-height: calc(100dvh - 6.5rem); }
}
.docs-heading-link { display: block; padding: .36rem .4rem; font-size: .81rem; color: hsl(var(--muted-foreground)); overflow-wrap: anywhere; }
.docs-heading-link:hover, .docs-heading-link.is-active { color: hsl(var(--foreground)); }
.docs-heading-link.is-active { font-weight: 600; box-shadow: -2px 0 hsl(var(--brand)); }
.docs-neighbor { display: flex; min-width: 0; flex-direction: column; gap: .4rem; border: 1px solid hsl(var(--border)); border-radius: var(--radius); padding: .85rem 1rem; transition: background .15s; }
.docs-neighbor:hover { background: hsl(var(--muted)); }
.docs-neighbor span { display: flex; align-items: center; gap: .35rem; font-size: .75rem; color: hsl(var(--muted-foreground)); }
.docs-neighbor strong { font-size: .9rem; font-weight: 600; overflow-wrap: anywhere; }
</style>
