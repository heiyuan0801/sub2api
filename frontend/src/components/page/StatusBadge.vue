<script setup lang="ts">
import type { Component } from 'vue'
import { toneDot, toneText, type PageTone } from './tones'

withDefaults(defineProps<{
  label: string
  tone?: PageTone
  icon?: Component
  soft?: boolean
}>(), { tone: 'neutral' })
</script>

<template>
  <span
    data-slot="page-status-badge"
    class="inline-flex max-w-full items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium"
    :class="soft ? 'border-transparent bg-muted' : 'border-border bg-card'"
  >
    <component :is="icon" v-if="icon" class="size-3 shrink-0" :class="toneText[tone]" aria-hidden="true" />
    <span v-else class="size-1.5 shrink-0 rounded-full ring-2" :class="toneDot[tone]" aria-hidden="true" />
    <span class="min-w-0 break-words">{{ label }}</span>
  </span>
</template>
