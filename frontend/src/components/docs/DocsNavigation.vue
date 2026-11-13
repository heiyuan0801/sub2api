<template>
  <nav class="space-y-6">
    <section v-for="category in groups" :key="category.id">
      <h2 v-if="category.articles.length !== 1 || category.articles[0].title !== category.name" class="mb-2 px-2.5 text-xs font-semibold text-muted-foreground">{{ category.name }}</h2>
      <RouterLink v-for="article in category.articles" :key="article.id" :to="{ name: 'DocsArticle', params: { slug: article.slug } }"
        :aria-current="article.slug === active ? 'page' : undefined" @click="$emit('navigate')"
        :class="['flex items-center justify-between gap-2 rounded-lg px-2.5 py-2 text-sm transition-colors', article.slug === active ? 'bg-muted font-semibold text-foreground' : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground']">
        <span class="break-words">{{ article.title }}</span>
        <LockKeyhole v-if="article.visibility === 'authenticated'" class="size-3 shrink-0" />
      </RouterLink>
    </section>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { LockKeyhole } from '@lucide/vue'
import type { DocNavigation } from '@/api/docs'

const props = defineProps<{ navigation: DocNavigation; active?: string }>()
defineEmits<{ navigate: [] }>()
const groups = computed(() => props.navigation.categories.map(category => ({
  ...category, articles: props.navigation.articles.filter(article => article.category_id === category.id),
})))
</script>
