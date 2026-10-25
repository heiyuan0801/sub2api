<script setup lang="ts">
import { useI18n } from 'vue-i18n'
defineProps<{ code: string; message: string }>()
const { t } = useI18n()
</script>

<template>
  <details class="mt-3 text-xs text-muted-foreground">
    <summary class="w-fit cursor-pointer rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">{{ t('cluster.log.original') }}</summary>
    <pre class="mt-2 max-h-60 overflow-auto whitespace-pre-wrap break-all rounded-lg bg-muted/50 p-3 font-mono">{{ code }}
{{ message }}</pre>
  </details>
</template>
