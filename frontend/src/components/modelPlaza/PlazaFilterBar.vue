<template>
  <div class="space-y-3 rounded-lg border border-border bg-card p-3">
    <div class="flex items-start gap-2">
      <span class="w-14 shrink-0 pt-2 text-xs font-medium text-muted-foreground">
        {{ t('modelPlaza.filters.platformLabel') }}
      </span>
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-for="p in ['all', ...platforms]"
          :key="`platform-${p}`"
          type="button"
          class="inline-flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-xs font-medium transition disabled:cursor-not-allowed disabled:opacity-40 disabled:grayscale"
          :class="p === 'all' ? chipClass(platform === 'all') : platform === p ? 'chip-tinted-active' : 'chip-tinted'"
          :style="p === 'all' ? undefined : { '--chip-accent': platformAccentColor(p) }"
          :aria-pressed="platform === p"
          @click="$emit('update:platform', p)"
        >
          <PlatformIcon v-if="p !== 'all'" :platform="p as GroupPlatform" size="xs" />
          {{ p === 'all' ? t('modelPlaza.filters.all') : platformLabel(p) }}
        </button>
      </div>
    </div>

    <div class="flex items-start gap-2">
      <span class="w-14 shrink-0 pt-2 text-xs font-medium text-muted-foreground">
        {{ t('modelPlaza.filters.groupLabel') }}
      </span>
      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          class="rounded-md px-2.5 py-1.5 text-xs font-medium transition"
          :class="chipClass(groupId === 'all')"
          :aria-pressed="groupId === 'all'"
          @click="$emit('update:groupId', 'all')"
        >
          {{ t('modelPlaza.filters.all') }}
        </button>
        <button
          v-for="g in groups"
          :key="`group-${g.id}`"
          type="button"
          class="rounded-md px-2.5 py-1.5 text-xs font-medium transition disabled:cursor-not-allowed disabled:opacity-40 disabled:grayscale"
          :class="groupId === g.id ? 'chip-tinted-active' : 'chip-tinted'"
          :style="{ '--chip-accent': platformAccentColor(g.platform) }"
          :aria-pressed="groupId === g.id"
          @click="$emit('update:groupId', g.id)"
        >
          {{ g.name }}
        </button>
      </div>
    </div>

    <div class="flex flex-wrap items-start gap-2">
      <span class="w-14 shrink-0 pt-2 text-xs font-medium text-muted-foreground">
        {{ t('modelPlaza.filters.modelLabel') }}
      </span>
      <div class="relative w-full sm:w-72">
        <Icon
          name="search"
          size="sm"
          class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
        />
        <input
          :value="search"
          type="text"
          :placeholder="t('modelPlaza.filters.searchPlaceholder')"
          class="input rounded-md py-1.5 pl-9 pr-9"
          @input="$emit('update:search', ($event.target as HTMLInputElement).value)"
        />
        <button
          v-if="search"
          type="button"
          class="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
          @click="$emit('update:search', '')"
        >
          <Icon name="x" size="xs" class="h-3.5 w-3.5" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { platformAccentColor, platformLabel } from '@/utils/platformColors'
import type { GroupPlatform } from '@/types'

defineProps<{
  /** 数据中出现的平台(去重排序后)。 */
  platforms: string[]
  groups: Array<{ id: number; name: string; platform: string }>
  platform: string
  groupId: number | 'all'
  /** 模型名搜索词(纯前端过滤)。 */
  search: string
}>()

defineEmits<{
  'update:platform': [value: string]
  'update:groupId': [value: number | 'all']
  'update:search': [value: string]
}>()

const { t } = useI18n()

function chipClass(active: boolean): string {
  return active
    ? 'bg-brand/10 text-brand ring-1 ring-inset ring-brand/35'
    : 'bg-card text-muted-foreground ring-1 ring-inset ring-border enabled:hover:bg-muted enabled:hover:text-foreground'
}
</script>

<style scoped>
/* Platform color remains a small marker; the active state uses the same tokens across filters. */
.chip-tinted, .chip-tinted-active {
  box-shadow: inset 2px 0 var(--chip-accent);
}
.chip-tinted {
  color: hsl(var(--foreground));
  background: hsl(var(--muted) / .6);
}
.chip-tinted:not(:disabled):hover {
  background: hsl(var(--muted));
}
.chip-tinted-active {
  color: hsl(var(--brand));
  background: hsl(var(--brand) / .13);
}
</style>
