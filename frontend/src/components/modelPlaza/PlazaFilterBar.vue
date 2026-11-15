<template>
  <div class="plaza-filter">
    <div class="plaza-filter-search">
      <Icon name="search" size="sm" />
      <input
        :value="search"
        type="search"
        :aria-label="t('publicSite.models.searchLabel')"
        :placeholder="t('modelPlaza.filters.searchPlaceholder')"
        autocomplete="off"
        @input="$emit('update:search', ($event.target as HTMLInputElement).value)"
      />
    </div>

    <div class="plaza-filter-row">
      <span class="plaza-filter-label">{{ t('modelPlaza.filters.platformLabel') }}</span>
      <div class="plaza-filter-chips" role="group" :aria-label="t('modelPlaza.filters.platformLabel')">
        <button
          v-for="item in ['all', ...platforms]"
          :key="item"
          type="button"
          :aria-pressed="platform === item"
          @click="$emit('update:platform', item)"
        >
          <PlatformIcon v-if="item !== 'all'" :platform="item as GroupPlatform" size="sm" />
          {{ item === 'all' ? t('modelPlaza.filters.all') : platformLabel(item) }}
        </button>
      </div>
    </div>

    <div class="plaza-filter-row">
      <span class="plaza-filter-label">{{ t('modelPlaza.filters.groupLabel') }}</span>
      <div class="plaza-filter-chips" role="group" :aria-label="t('modelPlaza.filters.groupLabel')">
        <button type="button" :aria-pressed="groupId === 'all'" @click="$emit('update:groupId', 'all')">
          {{ t('modelPlaza.filters.all') }}
        </button>
        <button
          v-for="group in groups"
          :key="group.id"
          type="button"
          :aria-pressed="groupId === group.id"
          @click="$emit('update:groupId', group.id)"
        >
          {{ group.name }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { platformLabel } from '@/utils/platformColors'
import type { GroupPlatform } from '@/types'

defineProps<{
  platforms: string[]
  groups: Array<{ id: number; name: string; platform: string }>
  platform: string
  groupId: number | 'all'
  search: string
}>()

defineEmits<{
  'update:platform': [value: string]
  'update:groupId': [value: number | 'all']
  'update:search': [value: string]
}>()

const { t } = useI18n()
</script>

<style scoped>
.plaza-filter {
  display: grid;
  gap: 15px;
  padding: clamp(16px, 2.2vw, 24px);
  border-radius: 24px;
  background: var(--market-soft);
}
.plaza-filter-search {
  display: flex;
  align-items: center;
  gap: 10px;
  width: min(100%, 480px);
  min-height: 48px;
  padding-inline: 16px;
  border: 1px solid var(--market-line);
  border-radius: 999px;
  color: var(--market-muted);
  background: var(--market-surface);
  transition: box-shadow 220ms;
}
.plaza-filter-search:focus-within {
  box-shadow: 0 0 0 2px var(--market-ink);
}
.plaza-filter-search :deep(svg) { flex: none; }
.plaza-filter-search input {
  min-width: 0;
  width: 100%;
  border: 0;
  padding: 0;
  background: transparent;
  color: var(--market-ink);
  box-shadow: none;
  outline: none;
  font-size: .88rem;
}
.plaza-filter-search input::placeholder { color: var(--market-muted); }
.plaza-filter-row {
  display: flex;
  gap: 12px;
  align-items: baseline;
  min-width: 0;
}
.plaza-filter-label {
  min-width: 3.2rem;
  color: var(--market-muted);
  font-size: .75rem;
}
.plaza-filter-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
  min-width: 0;
}
.plaza-filter-chips button {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 37px;
  max-width: 100%;
  padding: 6px 13px;
  border: 1px solid var(--market-line);
  border-radius: 999px;
  background: var(--market-surface);
  color: var(--market-ink);
  font-size: .8rem;
  text-align: left;
  overflow-wrap: anywhere;
  transition: background-color 200ms, color 200ms, transform 240ms cubic-bezier(.34, 1.38, .64, 1);
}
.plaza-filter-chips button :deep(svg) { flex: none; }
.plaza-filter-chips button:hover { transform: translateY(-2px); }
.plaza-filter-chips button:active { transform: scale(.96); }
.plaza-filter-chips button[aria-pressed='true'] {
  border-color: var(--market-ink);
  color: var(--market-surface);
  background: var(--market-ink);
}
.plaza-filter-chips button:focus-visible { outline: 2px solid var(--market-ink); outline-offset: 2px; }
@media (max-width: 600px) {
  .plaza-filter-row { flex-direction: column; gap: 8px; }
  .plaza-filter-search input { font-size: 1rem; }
  .plaza-filter-chips button { min-height: 44px; }
}
@media (prefers-reduced-motion: reduce) {
  .plaza-filter-chips button, .plaza-filter-search { transition: none; }
  .plaza-filter-chips button:is(:hover, :active) { transform: none; }
}
</style>
