<script setup lang="ts">
import type { Component } from 'vue'
import { Card } from '@/components/ui/card'
import { toneText, type PageTone } from './tones'

withDefaults(defineProps<{
  label: string
  value?: string | number
  unit?: string
  icon?: Component
  trend?: string
  trendTone?: PageTone
  description?: string
}>(), { trendTone: 'neutral' })
</script>

<template>
  <Card data-slot="page-stat-card" class="min-w-0 rounded-lg border-border px-4 pb-3 pt-3.5 shadow-none">
    <div class="flex items-center gap-1.5 text-xs text-muted-foreground">
      <component :is="icon" v-if="icon" class="size-3.5 shrink-0" aria-hidden="true" />
      {{ label }}
    </div>
    <div class="mt-1.5 flex flex-wrap items-baseline gap-x-1 leading-tight text-foreground">
      <span class="break-words text-2xl font-medium tabular-nums tracking-tight">
        <slot name="value">{{ value }}</slot>
      </span>
      <span v-if="unit" class="text-xs text-muted-foreground">{{ unit }}</span>
    </div>
    <div v-if="trend || description || $slots.footer || $slots.chart" class="mt-2 flex flex-wrap items-center justify-between gap-2 text-xs">
      <div v-if="trend || description || $slots.footer" class="flex min-w-0 flex-wrap items-baseline gap-1">
        <slot name="footer">
          <span v-if="trend" :class="toneText[trendTone]">{{ trend }}</span>
          <span v-if="description" class="text-muted-foreground">{{ description }}</span>
        </slot>
      </div>
      <div v-if="$slots.chart" class="ml-auto h-7 w-24 shrink-0">
        <slot name="chart" />
      </div>
    </div>
  </Card>
</template>
