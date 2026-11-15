<script setup lang="ts">
import TablePageLayout from '@/components/layout/TablePageLayout.vue'

defineOptions({ name: 'TablePageLayout' })
</script>

<template>
  <TablePageLayout class="admin-list-layout">
    <template v-if="$slots.actions" #actions><slot name="actions" /></template>
    <template v-if="$slots.filters" #filters><slot name="filters" /></template>
    <template #table><slot name="table" /></template>
    <template v-if="$slots.pagination" #pagination><slot name="pagination" /></template>
  </TablePageLayout>
</template>

<style scoped>
.admin-list-layout { @apply gap-3; }
.admin-list-layout :deep(.table-scroll-container) { @apply rounded-lg shadow-none; }
.admin-list-layout :deep(.table-scroll-container:has(.admin-data-grid)) { @apply overflow-visible border-0 bg-transparent; }
.admin-list-layout :deep(.table-scroll-container th) { @apply bg-muted px-3 py-1.5 text-xs font-medium text-muted-foreground; }
.admin-list-layout :deep(.table-scroll-container td:not([colspan])) { @apply border-border px-3 py-1.5 text-foreground; }
.admin-list-layout :deep(.table-scroll-container > [data-admin-data-table]) { @apply min-h-0 flex-1; }

@media (min-width: 1024px) {
  .admin-list-layout { height: max(390px, calc(100dvh - 112px)); }
}
</style>
