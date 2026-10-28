<template>
  <div class="w-28" data-testid="usage-cache-rate">
    <PopoverRoot v-if="breakdown" v-model:open="open">
      <PopoverTrigger as-child>
        <button
          type="button"
          class="w-full rounded-md px-1 py-1 text-left hover:bg-gray-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:hover:bg-dark-700"
          :aria-label="t('usage.cacheDetails.open', { rate: breakdown.percent.toFixed(1) })"
        >
          <span class="flex items-center justify-between gap-2">
            <strong class="text-sm tabular-nums" :class="breakdown.read > 0 ? 'text-sky-600 dark:text-sky-400' : 'text-gray-500 dark:text-gray-400'">{{ breakdown.percent.toFixed(1) }}%</strong>
            <Icon name="infoCircle" size="sm" class="h-3.5 w-3.5 text-gray-400" aria-hidden="true" />
          </span>
          <span class="mt-1.5 flex h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700" aria-hidden="true">
            <span class="bg-sky-500" :style="{ width: `${breakdown.read / breakdown.total * 100}%` }" />
            <span class="bg-amber-400" :style="{ width: `${breakdown.write / breakdown.total * 100}%` }" />
            <span class="bg-slate-400 dark:bg-slate-500" :style="{ width: `${breakdown.input / breakdown.total * 100}%` }" />
          </span>
        </button>
      </PopoverTrigger>
      <PopoverPortal>
        <PopoverContent
          :side-offset="8" :collision-padding="16" align="start"
          class="z-[80] w-80 max-w-[calc(100vw-2rem)] rounded-xl border border-gray-200 bg-white p-4 text-gray-800 shadow-xl dark:border-dark-600 dark:bg-dark-800 dark:text-gray-100"
          :aria-label="t('usage.cacheDetails.title')"
        >
          <div class="flex items-center justify-between gap-3">
            <h4 class="text-sm font-semibold">{{ t('usage.cacheDetails.title') }}</h4>
            <PopoverClose :aria-label="t('common.close')" class="rounded px-2 py-1 text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-700">×</PopoverClose>
          </div>
          <dl class="mt-3 space-y-2.5 text-xs">
            <div class="flex justify-between gap-4"><dt class="text-sky-600 dark:text-sky-400">{{ t('usage.cacheDetails.read') }}</dt><dd class="tabular-nums">{{ number(breakdown.read) }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="text-amber-600 dark:text-amber-400">{{ t('usage.cacheDetails.write') }}</dt><dd class="tabular-nums">{{ number(breakdown.write) }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="text-gray-500 dark:text-gray-400">{{ t('usage.cacheDetails.uncached') }}</dt><dd class="tabular-nums">{{ number(breakdown.input) }}</dd></div>
            <div class="flex justify-between gap-4 border-t border-gray-100 pt-2 dark:border-dark-700"><dt>{{ t('usage.cacheDetails.totalInput') }}</dt><dd class="tabular-nums">{{ number(breakdown.total) }}</dd></div>
          </dl>
          <p class="mt-3 border-t border-gray-100 pt-3 text-xs leading-relaxed text-gray-500 dark:border-dark-700 dark:text-gray-400">{{ t('usage.cacheDetails.formula') }}</p>
        </PopoverContent>
      </PopoverPortal>
    </PopoverRoot>
    <span v-else class="px-1 text-gray-400" :title="t('usage.cacheDetails.unavailable')">—</span>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { PopoverRoot, PopoverTrigger, PopoverPortal, PopoverContent, PopoverClose } from 'reka-ui'
import Icon from '@/components/icons/Icon.vue'
import { usageCacheBreakdown, type UsageCacheRateRow } from '@/utils/usageCacheRate'

const props = defineProps<{ row: UsageCacheRateRow }>()
const { t, locale } = useI18n()
const open = ref(false)
const breakdown = computed(() => usageCacheBreakdown(props.row))
const number = (value: number) => new Intl.NumberFormat(locale.value).format(value)
watch(() => props.row, () => { open.value = false })
</script>
