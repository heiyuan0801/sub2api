<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Moon, Sun } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { useDocumentTheme } from '@/composables/useDocumentTheme'

const props = defineProps<{ lightLabel?: string; darkLabel?: string }>()
const { t } = useI18n()
const { isDark, toggleTheme } = useDocumentTheme()
const label = computed(() => isDark.value
  ? props.lightLabel ?? t('publicSite.nav.lightMode')
  : props.darkLabel ?? t('publicSite.nav.darkMode'))
</script>

<template>
  <Button type="button" variant="ghost" size="icon-sm" class="text-muted-foreground" :aria-label="label" :title="label" @click="toggleTheme">
    <Sun v-if="isDark" aria-hidden="true" />
    <Moon v-else aria-hidden="true" />
  </Button>
</template>
