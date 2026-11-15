<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Flame, Gem, Shield, Sparkles, Star, Tag, Zap } from '@lucide/vue'
import { groupDisplayTags, type CustomGroupDisplayTag, type GroupDisplayTag } from '@/constants/groupDisplayTags'

const props = defineProps<{
  tags?: readonly GroupDisplayTag[] | null
  customTags?: readonly CustomGroupDisplayTag[] | null
}>()
const { t } = useI18n()

const colors = {
  blue: 'bg-blue-100 text-blue-800 dark:bg-blue-500/20 dark:text-blue-200',
  violet: 'bg-violet-100 text-violet-800 dark:bg-violet-500/20 dark:text-violet-200',
  emerald: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-500/20 dark:text-emerald-200',
  orange: 'bg-orange-100 text-orange-800 dark:bg-orange-500/20 dark:text-orange-200',
  rose: 'bg-rose-100 text-rose-800 dark:bg-rose-500/20 dark:text-rose-200',
  slate: 'bg-slate-100 text-slate-800 dark:bg-slate-500/20 dark:text-slate-200',
}

const icons = { tag: Tag, star: Star, flame: Flame, sparkles: Sparkles, bolt: Zap, shield: Shield, gem: Gem }
const presets = {
  recommended: { icon: Star, color: colors.violet },
  popular: { icon: Flame, color: colors.orange },
  new: { icon: Sparkles, color: colors.emerald },
}

const visibleTags = computed(() => [
  ...groupDisplayTags.filter(tag => props.tags?.includes(tag)).map(tag => ({
    key: tag,
    label: t('planOps.tags.' + tag),
    ...presets[tag],
  })),
  ...(props.customTags ?? []).filter(tag => tag.label.trim()).map(tag => ({
    key: `custom-${tag.label}`,
    label: tag.label,
    icon: icons[tag.icon] ?? Tag,
    color: colors[tag.color] ?? colors.slate,
  })),
])
</script>

<template>
  <span v-if="visibleTags.length" class="inline-flex max-w-full flex-wrap items-center gap-1 align-middle">
    <span
      v-for="tag in visibleTags"
      :key="tag.key"
      class="inline-flex min-w-0 max-w-full items-center gap-1 rounded-md px-1.5 py-0.5 text-[11px] font-semibold leading-4"
      :class="tag.color"
    >
      <component :is="tag.icon" class="h-3 w-3 shrink-0" aria-hidden="true" />
      <span class="[overflow-wrap:anywhere]">{{ tag.label }}</span>
    </span>
  </span>
</template>
