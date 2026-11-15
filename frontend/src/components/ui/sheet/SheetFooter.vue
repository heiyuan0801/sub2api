<script setup lang="ts">
import type { HTMLAttributes } from "vue"
import { cn } from "@/lib/utils"

const props = defineProps<{ class?: HTMLAttributes["class"] }>()
</script>

<template>
  <div
    data-slot="sheet-footer"
    :class="cn('mt-auto flex shrink-0 flex-wrap items-center justify-end gap-2 border-t border-border px-4 py-3', props.class)
    "
  >
    <slot />
  </div>
</template>
