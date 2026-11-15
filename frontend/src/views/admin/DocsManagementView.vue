<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-5">
      <PageHeader :title="t('docs.adminTitle')">
        <template #actions>
          <Button as-child variant="outline"><RouterLink to="/docs" target="_blank">{{ t('docs.title') }} <ArrowUpRight /></RouterLink></Button>
          <Button :disabled="!selectedCategory || busy" @click="newArticle"><Plus />{{ t('docs.newArticle') }}</Button>
        </template>
      </PageHeader>

      <div v-if="error" class="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-danger/25 bg-danger/10 px-4 py-2 text-sm text-destructive" role="alert">
        {{ error }} <Button variant="outline" size="sm" @click="load">{{ t('docs.retry') }}</Button>
      </div>
      <div class="grid gap-4 lg:grid-cols-[250px_minmax(0,1fr)]">
        <SectionCard :title="t('docs.categories')" flush class="self-start">
          <template #actions>
            <Button variant="outline" size="icon-sm" :aria-label="t('docs.newCategory')" :title="t('docs.newCategory')" :disabled="busy" @click="editCategory()"><Plus /></Button>
          </template>
          <VueDraggable v-model="categories" handle=".category-drag" :animation="150" :disabled="busy" class="space-y-1 p-2" @end="reorderCategories">
            <div v-for="category in categories" :key="category.id" :class="['flex min-w-0 items-center rounded-lg', selected === category.id ? 'bg-brand/10 text-brand' : 'hover:bg-muted/50']">
              <span class="category-drag cursor-grab px-1.5 py-2 text-muted-foreground" :title="t('docs.ordering')"><GripVertical class="size-4" /></span>
              <button type="button" :aria-pressed="selected === category.id" class="min-w-0 flex-1 break-words py-2 pr-1 text-left text-sm" @click="selected = category.id">{{ category.name }}</button>
              <Button variant="ghost" size="icon-xs" :aria-label="t('docs.edit') + ' ' + category.name" :disabled="busy" @click="editCategory(category)"><Pencil /></Button>
              <Button variant="ghost" size="icon-xs" :aria-label="t('docs.deleteCategory') + ' ' + category.name" :disabled="busy" @click="confirm = { kind: 'category', id: category.id, name: category.name, revision: 0 }"><Trash2 /></Button>
            </div>
          </VueDraggable>
          <template v-if="selectedCategory" #footer>
            <div class="flex justify-end gap-1">
              <Button variant="ghost" size="icon-xs" :disabled="busy || categoryIndex <= 0" :aria-label="t('docs.moveUp')" @click="moveCategory(-1)"><ChevronUp /></Button>
              <Button variant="ghost" size="icon-xs" :disabled="busy || categoryIndex >= categories.length - 1" :aria-label="t('docs.moveDown')" @click="moveCategory(1)"><ChevronDown /></Button>
            </div>
          </template>
        </SectionCard>

        <SectionCard :title="selectedCategory?.name || t('docs.articles')" flush>
          <template #actions>
            <span class="text-xs tabular-nums text-muted-foreground">{{ currentArticles.length }}</span>
          </template>
          <p v-if="loading" class="p-5 text-sm text-muted-foreground" role="status">{{ t('common.loading') }}</p>
          <EmptyState v-else-if="!currentArticles.length" :title="t(selectedCategory ? 'docs.noArticles' : 'docs.selectCategory')" :icon="BookOpen" />
          <Table v-else class="min-w-[28rem]">
            <TableHeader class="border-b border-border bg-muted/40">
              <TableRow class="border-border hover:bg-transparent">
                <TableHead scope="col" class="w-8"><span class="sr-only">{{ t('docs.ordering') }}</span></TableHead>
                <TableHead scope="col">{{ t('docs.articleTitle') }}</TableHead>
                <TableHead scope="col">{{ t('common.status') }}</TableHead>
                <TableHead scope="col" class="text-right">{{ t('common.actions') }}</TableHead>
              </TableRow>
            </TableHeader>
            <VueDraggable v-model="currentArticles" tag="tbody" handle=".article-drag" :animation="150" :disabled="busy" @end="reorderArticles">
              <TableRow v-for="(article, index) in currentArticles" :key="article.id" class="border-border last:border-0">
                <TableCell class="w-8 pl-3">
                  <span class="article-drag inline-flex cursor-grab text-muted-foreground" :title="t('docs.ordering')"><GripVertical class="size-4" /></span>
                </TableCell>
                <TableCell class="max-w-[23rem] whitespace-normal">
                  <button type="button" class="block w-full text-left" :disabled="busy" @click="editArticle(article.id)">
                    <span class="block break-words font-medium hover:underline">{{ article.title }}</span>
                    <span class="mt-0.5 block break-all font-mono text-xs text-muted-foreground">/docs/{{ article.slug }}</span>
                  </button>
                </TableCell>
                <TableCell>
                  <div class="flex items-center gap-1.5">
                    <StatusBadge :label="t('docs.' + article.status)" :tone="article.status === 'published' ? 'success' : 'neutral'" />
                    <LockKeyhole v-if="article.visibility === 'authenticated'" class="size-3.5 text-muted-foreground" :aria-label="t('docs.authenticated')" />
                  </div>
                  <p v-if="article.published_revision && article.published_revision !== article.revision" class="mt-1 text-xs text-warning">{{ t('docs.unpublishedChanges') }}</p>
                </TableCell>
                <TableCell class="pr-3">
                  <div class="flex justify-end gap-0.5">
                    <Button variant="ghost" size="icon-xs" :disabled="busy || index === 0" :aria-label="t('docs.moveUp') + ' ' + article.title" @click="moveArticle(index, -1)"><ChevronUp /></Button>
                    <Button variant="ghost" size="icon-xs" :disabled="busy || index === currentArticles.length - 1" :aria-label="t('docs.moveDown') + ' ' + article.title" @click="moveArticle(index, 1)"><ChevronDown /></Button>
                    <Button v-if="article.published_slug" variant="ghost" size="icon-sm" :disabled="busy" :aria-label="t('docs.takeOffline') + ' ' + article.title" :title="t('docs.takeOffline')" @click="setConfirm('offline', article)"><EyeOff /></Button>
                    <Button variant="ghost" size="icon-sm" :disabled="busy" :aria-label="t('docs.deleteArticle') + ' ' + article.title" :title="t('docs.deleteArticle')" @click="setConfirm('article', article)"><Trash2 /></Button>
                  </div>
                </TableCell>
              </TableRow>
            </VueDraggable>
          </Table>
        </SectionCard>
      </div>
    </div>

    <DocsEditor v-if="editorOpen" :key="editorKey" :article="editingArticle" :category="selected" :categories="categories" @close="editorOpen = false" @changed="load" />

    <BaseDialog :show="categoryOpen" :title="t(categoryForm.id ? 'docs.edit' : 'docs.newCategory')" width="narrow" presentation="sheet" :close-on-escape="!busy" @close="!busy && (categoryOpen = false)">
      <form id="docs-category-form" @submit.prevent="saveCategory">
        <label class="block space-y-1 text-sm"><span>{{ t('docs.categoryName') }}</span><input v-model.trim="categoryForm.name" class="input w-full" maxlength="100" required /></label>
        <p v-if="dialogError" role="alert" class="mt-3 text-sm text-destructive">{{ dialogError }}</p>
      </form>
      <template #footer>
        <Button variant="outline" :disabled="busy" @click="categoryOpen = false">{{ t('common.cancel') }}</Button>
        <Button type="submit" form="docs-category-form" :disabled="busy || !categoryForm.name">{{ t('common.save') }}</Button>
      </template>
    </BaseDialog>

    <BaseDialog :show="!!confirm" :title="t(confirm?.kind === 'category' ? 'docs.deleteCategory' : confirm?.kind === 'offline' ? 'docs.takeOffline' : 'docs.deleteArticle')" width="narrow" :close-on-escape="!busy" @close="!busy && (confirm = null)">
      <p v-if="confirm" class="text-sm">{{ t(confirm.kind === 'category' ? 'docs.confirmCategory' : confirm.kind === 'offline' ? 'docs.confirmOffline' : 'docs.confirmArticle', { name: confirm.name }) }}</p>
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
import { ArrowUpRight, BookOpen, ChevronDown, ChevronUp, EyeOff, GripVertical, LockKeyhole, Pencil, Plus, Trash2 } from '@lucide/vue'
import { VueDraggable } from 'vue-draggable-plus'
import { docsAPI, type DocAdminSummary, type DocArticle, type DocCategory } from '@/api/docs'
import { Button } from '@/components/ui/button'
import { Table, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { EmptyState, PageHeader, SectionCard, StatusBadge } from '@/components/page'
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
