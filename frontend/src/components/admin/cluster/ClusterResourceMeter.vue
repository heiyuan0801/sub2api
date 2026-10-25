<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
const props = defineProps<{ label: string; value?: number | null; detail: string }>()
const { t } = useI18n()
const percent = computed(() => typeof props.value === 'number' && Number.isFinite(props.value) ? Math.max(0, Math.min(100, props.value)) : null)
const bar = computed(() => percent.value !== null && percent.value >= 90 ? 'bg-red-500' : percent.value !== null && percent.value >= 75 ? 'bg-amber-500' : 'bg-foreground/80')
</script>

<template>
  <div class="min-w-0 rounded-lg border border-border bg-muted/20 p-3">
    <p class="text-xs text-muted-foreground">{{ label }}</p>
    <p class="my-2 text-xl font-semibold tabular-nums">{{ percent === null ? '—' : `${percent.toFixed(1)}%` }}</p>
    <div v-if="percent !== null" role="progressbar" :aria-label="label" :aria-valuenow="Number(percent.toFixed(1))" :aria-valuemin="0" :aria-valuemax="100" class="h-1.5 overflow-hidden rounded-full bg-muted">
      <div class="h-full rounded-full transition-[width] duration-300" :class="bar" :style="{ width: `${percent}%` }" />
    </div>
    <div v-else class="h-1.5 rounded-full bg-muted" />
    <p class="mt-2 break-words text-xs leading-5 text-muted-foreground">{{ detail || t('cluster.noData') }}</p>
  </div>
</template>
