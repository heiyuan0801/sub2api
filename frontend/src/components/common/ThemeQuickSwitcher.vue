<template>
  <div class="flex items-center justify-between gap-3 px-2.5 py-2">
    <span class="text-xs font-medium text-muted-foreground">
      {{ t('themeConfig.theme') }}
    </span>
    <div
      class="inline-flex items-center gap-0.5 rounded-full border border-border bg-muted/60 p-0.5"
      role="radiogroup"
      :aria-label="t('themeConfig.theme')"
    >
      <button
        v-for="option in options"
        :key="option.value"
        type="button"
        role="radio"
        :aria-checked="themeStore.mode === option.value"
        :aria-label="option.label"
        :title="option.label"
        class="inline-flex size-6 items-center justify-center rounded-full transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        :class="
          themeStore.mode === option.value
            ? 'bg-background text-foreground shadow-sm ring-1 ring-border'
            : 'text-muted-foreground hover:text-foreground'
        "
        @click.stop="themeStore.setMode(option.value)"
      >
        <component :is="option.icon" class="size-3.5" aria-hidden="true" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { Monitor, Moon, Sun } from '@lucide/vue'
import { useThemeStore, type ThemeMode } from '@/stores/theme'

const { t } = useI18n()
const themeStore = useThemeStore()

const options = computed<Array<{ value: ThemeMode; label: string; icon: Component }>>(() => [
  { value: 'system', label: t('themeConfig.modes.system'), icon: Monitor },
  { value: 'light', label: t('themeConfig.modes.light'), icon: Sun },
  { value: 'dark', label: t('themeConfig.modes.dark'), icon: Moon }
])
</script>
