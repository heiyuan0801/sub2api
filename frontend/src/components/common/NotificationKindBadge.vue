<template>
  <span :class="['badge gap-1 whitespace-nowrap', appearance.tone]">
    <Icon :name="appearance.icon" size="xs" aria-hidden="true" />
    {{ t(`operations.kind.${kind || 'system'}`) }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { NotificationKind } from '@/types/operations'

const props = defineProps<{ kind?: NotificationKind }>()
const { t } = useI18n()
const appearances = {
  price_increase: { icon: 'arrowUp', tone: 'badge-danger' },
  price_decrease: { icon: 'arrowDown', tone: 'badge-success' },
  new_plan: { icon: 'sparkles', tone: 'badge-primary' },
  new_group: { icon: 'sparkles', tone: 'badge-primary' },
  plan_unlisted: { icon: 'inbox', tone: 'badge-warning' },
  group_unlisted: { icon: 'inbox', tone: 'badge-warning' },
  activity: { icon: 'gift', tone: 'badge-warning' },
  system: { icon: 'bell', tone: 'badge-gray' },
} as const satisfies Record<NotificationKind, { icon: string; tone: string }>
const appearance = computed(() => appearances[props.kind ?? 'system'] ?? appearances.system)
</script>
