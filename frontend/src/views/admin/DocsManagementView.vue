<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6">
      <PageHeader :title="t('docs.adminTitle')">
        <template #actions>
          <Button as-child variant="outline">
            <RouterLink to="/docs" target="_blank">
              {{ t('docs.title') }}
              <ArrowUpRight class="size-4" />
            </RouterLink>
          </Button>
          <Button :disabled="!selectedCategory || busy" @click="newArticle">
            <Plus class="size-4" />
            {{ t('docs.newArticle') }}
          </Button>
        </template>
      </PageHeader>

      <div
        v-if="error"
        class="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-destructive/30 bg-destructive/10 px-4 py-2.5 text-sm text-destructive"
        role="alert"
      >
        <span>{{ error }}</span>
        <Button variant="outline" size="sm" @click="load">{{ t('docs.retry') }}</Button>
      </div>

      <div class="grid gap-5 lg:grid-cols-[260px_minmax(0,1fr)]">
        <aside class="self-start rounded-2xl border border-border/80 bg-card shadow-sm">
          <div class="flex items-center justify-between gap-2 border-b border-border/80 px-4 py-3">
            <h3 class="text-sm font-semibold tracking-tight text-foreground">{{ t('docs.categories') }}</h3>
            <Button
              variant="outline"
              size="icon-sm"
              :aria-label="t('docs.newCategory')"
              :title="t('docs.newCategory')"
              :disabled="busy"
              @click="editCategory()"
            >
              <Plus />
            </Button>
          </div>
          <div v-if="loading && !categories.length" class="space-y-1.5 p-2.5">
            <Skeleton v-for="i in 4" :key="`cat-sk-${i}`" class="h-9 w-full rounded-lg" />
          </div>
          <VueDraggable
            v-else
            v-model="categories"
            handle=".category-drag"
            :animation="150"
            :disabled="busy"
            class="space-y-1 p-2"
            @end="reorderCategories"
          >
            <div
              v-for="category in categories"
              :key="category.id"
              :class="[
                'flex min-w-0 items-center rounded-lg transition-colors',
                selected === category.id
                  ? 'bg-muted font-medium text-foreground'
                  : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'
              ]"
            >
              <span
                class="category-drag cursor-grab px-1.5 py-2 text-muted-foreground/60 transition-colors hover:text-foreground"
                :title="t('docs.ordering')"
              >
                <GripVertical class="size-4" />
              </span>
              <button
                type="button"
                :aria-pressed="selected === category.id"
                class="min-w-0 flex-1 break-words py-2 pr-1 text-left text-sm"
                @click="selected = category.id"
              >
                {{ category.name }}
              </button>
              <Button
                variant="ghost"
                size="icon-xs"
                :aria-label="t('docs.edit') + ' ' + category.name"
                :disabled="busy"
                @click="editCategory(category)"
              >
                <Pencil />
              </Button>
              <Button
                variant="ghost"
                size="icon-xs"
                :aria-label="t('docs.deleteCategory') + ' ' + category.name"
                :disabled="busy"
                @click="confirm = { kind: 'category', id: category.id, name: category.name, revision: 0 }"
              >
                <Trash2 />
              </Button>
            </div>
          </VueDraggable>
          <div v-if="selectedCategory" class="flex justify-end gap-1 border-t border-border/80 px-2.5 py-2">
            <Button variant="ghost" size="icon-xs" :disabled="busy || categoryIndex <= 0" :aria-label="t('docs.moveUp')" @click="moveCategory(-1)"><ChevronUp /></Button>
            <Button variant="ghost" size="icon-xs" :disabled="busy || categoryIndex >= categories.length - 1" :aria-label="t('docs.moveDown')" @click="moveCategory(1)"><ChevronDown /></Button>
          </div>
        </aside>

        <section class="min-w-0 overflow-hidden rounded-2xl border border-border/80 bg-card shadow-sm">
          <div class="flex items-center justify-between gap-3 border-b border-border/80 px-5 py-3.5">
            <h3 class="min-w-0 break-words text-sm font-semibold tracking-tight text-foreground">
              {{ selectedCategory?.name || t('docs.articles') }}
            </h3>
            <span class="rounded-full border border-border bg-muted px-2 py-0.5 text-xs font-medium tabular-nums text-muted-foreground">
              {{ currentArticles.length }}
            </span>
          </div>
          <div v-if="loading" class="divide-y divide-border/70">
            <div v-for="i in 4" :key="`art-sk-${i}`" class="flex items-center gap-3 px-4 py-3.5">
              <Skeleton class="size-4 rounded" />
              <div class="min-w-0 flex-1 space-y-1.5">
                <Skeleton class="h-4 w-48" />
                <Skeleton class="h-3.5 w-32" />
              </div>
              <Skeleton class="h-5 w-16 rounded-md" />
            </div>
          </div>
          <p v-else-if="!currentArticles.length" class="py-12 text-center text-sm text-muted-foreground">
            {{ t(selectedCategory ? 'docs.noArticles' : 'docs.selectCategory') }}
          </p>
          <VueDraggable
            v-else
            v-model="currentArticles"
            handle=".article-drag"
            :animation="150"
            :disabled="busy"
            class="divide-y divide-border/70"
            @end="reorderArticles"
          >
            <div
              v-for="(article, index) in currentArticles"
              :key="article.id"
              class="flex flex-wrap items-center gap-x-3 gap-y-2 px-3.5 py-3.5 transition-colors hover:bg-muted/30 sm:flex-nowrap sm:px-5"
            >
              <span
                class="article-drag cursor-grab text-muted-foreground/60 transition-colors hover:text-foreground"
                :title="t('docs.ordering')"
              >
                <GripVertical class="size-4" />
              </span>
              <button type="button" class="min-w-0 flex-1 break-words text-left" :disabled="busy" @click="editArticle(article.id)">
                <span class="block text-sm font-medium text-foreground hover:underline">{{ article.title }}</span>
                <code class="mt-1 inline-block max-w-full truncate rounded border border-border/60 bg-muted/70 px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground">
                  /docs/{{ article.slug }}
                </code>
              </button>
              <div class="flex flex-wrap items-center gap-1.5">
                <span
                  :class="[
                    'badge',
                    article.status === 'published' ? 'badge-success' : 'badge-gray'
                  ]"
                >
                  {{ t('docs.' + article.status) }}
                </span>
                <span
                  v-if="article.published_revision && article.published_revision !== article.revision"
                  class="badge badge-warning"
                >
                  {{ t('docs.unpublishedChanges') }}
                </span>
                <LockKeyhole
                  v-if="article.visibility === 'authenticated'"
                  class="size-3.5 text-muted-foreground"
                  :aria-label="t('docs.authenticated')"
                />
              </div>
              <div class="ml-auto flex shrink-0 items-center gap-0.5">
                <Button variant="ghost" size="icon-xs" :disabled="busy || index === 0" :aria-label="t('docs.moveUp') + ' ' + article.title" @click="moveArticle(index, -1)"><ChevronUp /></Button>
                <Button variant="ghost" size="icon-xs" :disabled="busy || index === currentArticles.length - 1" :aria-label="t('docs.moveDown') + ' ' + article.title" @click="moveArticle(index, 1)"><ChevronDown /></Button>
                <Button v-if="article.published_slug" variant="ghost" size="icon-sm" :disabled="busy" :aria-label="t('docs.takeOffline') + ' ' + article.title" :title="t('docs.takeOffline')" @click="setConfirm('offline', article)"><EyeOff /></Button>
                <Button variant="ghost" size="icon-sm" :disabled="busy" :aria-label="t('docs.deleteArticle') + ' ' + article.title" :title="t('docs.deleteArticle')" @click="setConfirm('article', article)"><Trash2 /></Button>
              </div>
            </div>
          </VueDraggable>
        </section>
      </div>
    </div>

    <DocsEditor v-if="editorOpen" :key="editorKey" :article="editingArticle" :category="selected" :categories="categories" @close="editorOpen = false" @changed="load" />

    <BaseDialog :show="categoryOpen" :title="t(categoryForm.id ? 'docs.edit' : 'docs.newCategory')" width="narrow" :close-on-escape="!busy" @close="!busy && (categoryOpen = false)">
      <form id="docs-category-form" @submit.prevent="saveCategory">
        <label class="block">
          <span class="input-label">{{ t('docs.categoryName') }}</span>
          <input v-model.trim="categoryForm.name" class="input w-full" maxlength="100" required />
        </label>
        <p v-if="dialogError" role="alert" class="mt-3 text-sm text-destructive">{{ dialogError }}</p>
      </form>
      <template #footer>
        <Button variant="outline" :disabled="busy" @click="categoryOpen = false">{{ t('common.cancel') }}</Button>
        <Button type="submit" form="docs-category-form" :disabled="busy || !categoryForm.name">{{ t('common.save') }}</Button>
      </template>
    </BaseDialog>

    <BaseDialog :show="!!confirm" :title="t(confirm?.kind === 'category' ? 'docs.deleteCategory' : confirm?.kind === 'offline' ? 'docs.takeOffline' : 'docs.deleteArticle')" width="narrow" :close-on-escape="!busy" @close="!busy && (confirm = null)">
      <p v-if="confirm" class="text-sm text-foreground">{{ t(confirm.kind === 'category' ? 'docs.confirmCategory' : confirm.kind === 'offline' ? 'docs.confirmOffline' : 'docs.confirmArticle', { name: confirm.name }) }}</p>
      <p v-if="dialogError" role="alert" class="mt-3 text-sm text-destructive">{{ dialogError }}</p>
      <template #footer>
        <Button variant="outline" :disabled="busy" @click="confirm = null">{{ t('common.cancel') }}</Button>
        <Button variant="destructive" :disabled="busy" @click="applyConfirm">{{ t('common.confirm') }}</Button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, ChevronDown, ChevronUp, EyeOff, GripVertical, LockKeyhole, Pencil, Plus, Trash2 } from '@lucide/vue'
import { VueDraggable } from 'vue-draggable-plus'
import { docsAPI, type DocAdminSummary, type DocArticle, type DocCategory } from '@/api/docs'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import PageHeader from '@/components/common/PageHeader.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import DocsEditor from '@/components/docs/DocsEditor.vue'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const app = useAppStore()
const categories = ref<DocCategory[]>([])
const articles = ref<DocAdminSummary[]>([])
const currentArticles = ref<DocAdminSummary[]>([])
const selected = ref(0)
const categoryIndex = computed(() => categories.value.findIndex(category => category.id === selected.value))
const selectedCategory = computed(() => categories.value[categoryIndex.value])
const loading = ref(true)
const busy = ref(false)
const error = ref('')
const dialogError = ref('')
const editorOpen = ref(false)
const editingArticle = ref<DocArticle>()
const editorKey = ref(0)
const categoryOpen = ref(false)
const categoryForm = ref({ id: 0, name: '' })
const confirm = ref<{ kind: 'category' | 'article' | 'offline'; id: number; name: string; revision: number } | null>(null)
let loadSequence = 0

watch([articles, selected], () => {
  currentArticles.value = articles.value.filter(article => article.category_id === selected.value)
}, { immediate: true })

function failure(err: unknown) {
  const reason = (err as { reason?: string }).reason
  return t(reason === 'DOC_CONFLICT' ? 'docs.conflict' : reason === 'DOC_INVALID' ? 'docs.invalid'
    : reason === 'DOC_CATEGORY_NOT_EMPTY' ? 'docs.categoryNotEmpty' : 'docs.changeFailed')
}

async function load() {
  const sequence = ++loadSequence
  error.value = ''
  try {
    const nav = await docsAPI.adminNavigation()
    if (sequence !== loadSequence) return
    categories.value = nav.categories
    articles.value = nav.articles
    if (!nav.categories.some(category => category.id === selected.value)) selected.value = nav.categories[0]?.id ?? 0
  } catch {
    if (sequence === loadSequence) error.value = t('docs.loadFailed')
  } finally {
    if (sequence === loadSequence) loading.value = false
  }
}

function newArticle() {
  editingArticle.value = undefined
  editorKey.value++
  editorOpen.value = true
}

async function editArticle(id: number) {
  busy.value = true
  error.value = ''
  try {
    editingArticle.value = await docsAPI.adminArticle(id)
    editorKey.value++
    editorOpen.value = true
  } catch (err) {
    error.value = failure(err)
  } finally {
    busy.value = false
  }
}

function editCategory(category?: DocCategory) {
  dialogError.value = ''
  categoryForm.value = category ? { ...category } : { id: 0, name: '' }
  categoryOpen.value = true
}

async function saveCategory() {
  if (busy.value) return
  busy.value = true
  dialogError.value = ''
  try {
    const result = await docsAPI.saveCategory(categoryForm.value.name, categoryForm.value.id || undefined)
    categoryOpen.value = false
    selected.value = result.id
    await load()
  } catch (err) {
    dialogError.value = failure(err)
  } finally {
    busy.value = false
  }
}

function setConfirm(kind: 'article' | 'offline', article: DocAdminSummary) {
  dialogError.value = ''
  confirm.value = { kind, id: article.id, revision: article.revision, name: article.title }
}
watch(confirm, () => { dialogError.value = '' })

async function applyConfirm() {
  const action = confirm.value
  if (!action || busy.value) return
  busy.value = true
  dialogError.value = ''
  try {
    if (action.kind === 'category') await docsAPI.deleteCategory(action.id)
    else if (action.kind === 'article') await docsAPI.deleteArticle(action.id, action.revision)
    else {
      await docsAPI.publish(action.id, action.revision, false)
      app.showSuccess(t('docs.offlineSuccess'))
    }
    confirm.value = null
    await load()
  } catch (err) {
    dialogError.value = failure(err)
  } finally {
    busy.value = false
  }
}

async function persistOrder(write: () => Promise<unknown>) {
  if (busy.value) return
  busy.value = true
  try {
    await write()
    await load()
  } catch (err) {
    await load()
    error.value = failure(err)
  } finally {
    busy.value = false
  }
}
function reorderCategories() {
  return persistOrder(() => docsAPI.sortCategories(categories.value.map(item => item.id)))
}
function reorderArticles() {
  return persistOrder(() => docsAPI.sortArticles(selected.value, currentArticles.value.map(item => item.id)))
}
function move<T>(items: T[], from: number, delta: number) {
  const to = from + delta
  if (from < 0 || to < 0 || to >= items.length) return false
  const item = items[from]
  items.splice(from, 1)
  items.splice(to, 0, item)
  return true
}
function moveCategory(delta: number) {
  if (move(categories.value, categoryIndex.value, delta)) void reorderCategories()
}
function moveArticle(index: number, delta: number) {
  if (move(currentArticles.value, index, delta)) void reorderArticles()
}
onMounted(load)
</script>
