<script setup lang="ts">
import { ref, type HTMLAttributes } from "vue"
import { cn } from "@/lib/utils"

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  class?: HTMLAttributes["class"]
  scrollClass?: HTMLAttributes["class"]
}>()

const scrollElement = ref<HTMLElement | null>(null)
defineExpose({ scrollElement })
</script>

<template>
  <div ref="scrollElement" data-slot="table-container" :class="cn('relative w-full overflow-auto', scrollClass)">
    <table v-bind="$attrs" data-slot="table" :class="cn('w-full caption-bottom text-sm', props.class)">
      <slot />
    </table>
  </div>
</template>
