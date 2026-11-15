<script setup lang="ts">
import type { HTMLAttributes } from "vue"
import { cn } from "@/lib/utils"

const props = defineProps<{ class?: HTMLAttributes["class"] }>()
</script>

<template>
  <div
    data-slot="sheet-header"
    :class="cn('flex shrink-0 flex-col gap-1 border-b border-border px-4 py-3.5 pr-14', props.class)"
  >
    <slot />
  </div>
</template>
