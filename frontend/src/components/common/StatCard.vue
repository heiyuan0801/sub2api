<template>
  <Card class="p-5">
    <div class="flex items-center justify-between gap-3">
      <p class="truncate text-xs font-medium text-muted-foreground">{{ title }}</p>
      <div
        v-if="icon"
        :class="['flex size-7 shrink-0 items-center justify-center rounded-md', iconClass]"
      >
        <component :is="icon" class="size-4" aria-hidden="true" />
      </div>
    </div>
    <div class="mt-2.5 flex items-baseline justify-between gap-2">
      <p
        class="truncate text-2xl font-semibold tracking-tight text-foreground tabular-nums"
        :title="String(formattedValue)"
      >
        {{ formattedValue }}
      </p>
      <span
        v-if="change !== undefined"
        :class="[
          'inline-flex shrink-0 items-center gap-0.5 rounded-md px-1.5 py-0.5 text-xs font-medium tabular-nums',
          trendClass
        ]"
      >
        <Icon
          v-if="changeType !== 'neutral'"
          name="arrowUp"
          size="xs"
          :class="changeType === 'down' && 'rotate-180'"
        />
        {{ formattedChange }}
      </span>
    </div>
  </Card>
</template>

<script setup lang="ts">
import { Card } from '@/components/ui/card'
import { computed } from 'vue'
import type { Component } from 'vue'
import Icon from '@/components/icons/Icon.vue'

type ChangeType = 'up' | 'down' | 'neutral'
type IconVariant = 'primary' | 'success' | 'warning' | 'danger'

interface Props {
  title: string
  value: number | string
  icon?: Component
  iconVariant?: IconVariant
  change?: number
  changeType?: ChangeType
  formatValue?: (value: number | string) => string
}

const props = withDefaults(defineProps<Props>(), {
  changeType: 'neutral',
  iconVariant: 'primary'
})

const formattedValue = computed(() => {
  if (props.formatValue) {
    return props.formatValue(props.value)
  }
  if (typeof props.value === 'number') {
    return props.value.toLocaleString()
  }
  return props.value
})

const formattedChange = computed(() => {
  if (props.change === undefined) return ''
  const absChange = Math.abs(props.change)
  return `${absChange}%`
})

const iconClass = computed(() => {
  const classes: Record<IconVariant, string> = {
    primary: 'bg-muted/70 text-muted-foreground',
    success: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
    warning: 'bg-amber-500/10 text-amber-600 dark:text-amber-400',
    danger: 'bg-red-500/10 text-red-600 dark:text-red-400'
  }
  return classes[props.iconVariant]
})

const trendClass = computed(() => {
  const classes: Record<ChangeType, string> = {
    up: 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400',
    down: 'bg-red-500/10 text-red-700 dark:text-red-400',
    neutral: 'bg-muted text-muted-foreground'
  }
  return classes[props.changeType]
})
</script>
