<template>
  <BaseDialog :show="open" :title="t('docs.search')" width="wide" close-on-click-outside @close="$emit('close')">
    <div class="relative">
      <Search class="absolute left-3 top-3 size-4 text-muted-foreground" />
      <input ref="field" v-model="query" type="search" maxlength="120" class="input w-full !pl-9" :placeholder="t('docs.searchPlaceholder')" :aria-label="t('docs.search')" @keydown.down.prevent="move(1)" @keydown.up.prevent="move(-1)" @keydown.enter.prevent="select()" />
    </div>
    <div class="mt-3 min-h-32" aria-live="polite">
      <p v-if="pending" class="px-2 py-3 text-sm text-muted-foreground">{{ t('common.loading') }}</p>
      <p v-else-if="failed" class="px-2 py-3 text-sm text-destructive">{{ t('docs.loadFailed') }}</p>
      <p v-else-if="!results.length" class="px-2 py-3 text-sm text-muted-foreground">{{ t(query.trim() ? 'docs.noResults' : 'docs.searchHint') }}</p>
      <ul v-else class="max-h-[50dvh] space-y-1 overflow-y-auto">
        <li v-for="(result, index) in results" :key="result.id">
          <RouterLink :to="{ name: 'DocsArticle', params: { slug: result.slug } }" @click="$emit('close')" @mouseenter="active = index"
            :class="['block rounded-lg px-3 py-2.5', active === index ? 'bg-muted' : 'hover:bg-muted']">
            <span class="block text-sm font-semibold">{{ result.title }}</span>
            <span class="mt-1 line-clamp-2 break-words text-xs leading-5 text-muted-foreground">{{ result.snippet }}</span>
          </RouterLink>
        </li>
      </ul>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { Search } from '@lucide/vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { docsAPI, type DocSearchResult } from '@/api/docs'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const router = useRouter()
const field = ref<HTMLInputElement>()
const query = ref('')
const pending = ref(false)
const failed = ref(false)
const results = ref<DocSearchResult[]>([])
const active = ref(0)

watch(() => props.open, async open => {
  if (!open) query.value = ''
  else {
    await nextTick()
    field.value?.focus()
  }
})
watch([() => props.open, query], ([open, text], _, cleanup) => {
  results.value = []
  active.value = 0
  failed.value = false
  const q = text.trim()
  pending.value = open && !!q
  if (!pending.value) return
  const controller = new AbortController()
  const timer = setTimeout(async () => {
    try {
      const data = await docsAPI.search(q, controller.signal)
      if (!controller.signal.aborted) results.value = data
    } catch {
      if (!controller.signal.aborted) failed.value = true
    } finally {
      if (!controller.signal.aborted) pending.value = false
    }
  }, 240)
  cleanup(() => { clearTimeout(timer); controller.abort() })
})

function move(delta: number) {
  if (!results.value.length) return
  active.value = (active.value + delta + results.value.length) % results.value.length
  field.value?.closest('[role=dialog]')?.querySelectorAll('li a')[active.value]?.scrollIntoView({ block: 'nearest' })
}

function select() {
  const result = results.value[active.value]
  if (!result) return
  void router.push({ name: 'DocsArticle', params: { slug: result.slug } })
  emit('close')
}
</script>
