<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { PaginationEllipsisProps } from "reka-ui"
import type { HTMLAttributes } from "vue"
import { MoreHorizontal } from "@lucide/vue"
import { reactiveOmit } from "@vueuse/core"
import { cn } from "@/lib/utils"
import { PaginationEllipsis } from "reka-ui"

const props = defineProps<PaginationEllipsisProps & { class?: HTMLAttributes["class"] }>()

const delegatedProps = reactiveOmit(props, "class")
const { t } = useI18n()
</script>

<template>
  <PaginationEllipsis
    data-slot="pagination-ellipsis"
    v-bind="delegatedProps"
    :class="cn('flex size-9 items-center justify-center', props.class)"
  >
    <slot>
      <MoreHorizontal class="size-4" />
      <span class="sr-only">{{ t('design.pagination.ellipsis') }}</span>
    </slot>
  </PaginationEllipsis>
</template>
