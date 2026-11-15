<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { PaginationNextProps } from "reka-ui"
import type { HTMLAttributes } from "vue"
import type { ButtonVariants } from "@/components/ui/button"
import { ChevronRightIcon } from "@lucide/vue"
import { reactiveOmit } from "@vueuse/core"
import { cn } from "@/lib/utils"
import { PaginationNext, useForwardProps } from "reka-ui"
import { buttonVariants } from "@/components/ui/button"

const props = withDefaults(defineProps<PaginationNextProps & {
  size?: ButtonVariants["size"]
  class?: HTMLAttributes["class"]
}>(), {
  size: "default",
})

const delegatedProps = reactiveOmit(props, "class", "size")
const forwarded = useForwardProps(delegatedProps)
const { t } = useI18n()
</script>

<template>
  <PaginationNext
    data-slot="pagination-next"
    :aria-label="t('design.pagination.next')"
    :class="cn(buttonVariants({ variant: 'ghost', size }), 'gap-1 px-2.5 sm:pr-2.5', props.class)"
    v-bind="forwarded"
  >
    <slot>
      <span class="hidden sm:block">{{ t('design.pagination.next') }}</span>
      <ChevronRightIcon />
    </slot>
  </PaginationNext>
</template>
