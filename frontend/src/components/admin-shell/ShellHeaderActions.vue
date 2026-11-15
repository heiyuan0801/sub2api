<script setup lang="ts">
import { Moon, Search, Sun } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { getShellShortcutAriaKey, getShellShortcutHint } from './shortcuts'

defineProps<{ dark: boolean; open: boolean }>()
const emit = defineEmits<{ 'open-command': []; 'toggle-theme': [] }>()
const { t } = useI18n()
</script>

<template>
  <Tooltip>
    <TooltipTrigger as-child>
      <Button
        data-console-command-trigger type="button" variant="ghost" size="sm"
        class="size-8 shrink-0 px-0 text-muted-foreground xl:w-44 xl:justify-start xl:border xl:border-border xl:bg-card xl:px-2.5"
        :aria-label="t('design.command.label')" :aria-expanded="open"
        aria-haspopup="dialog" :aria-keyshortcuts="getShellShortcutAriaKey('command')"
        @click="emit('open-command')"
      >
        <Search aria-hidden="true" />
        <span class="hidden min-w-0 flex-1 truncate text-left text-xs xl:inline">{{ t('design.command.placeholder') }}</span>
        <kbd class="hidden rounded border border-border bg-muted px-1 py-0.5 font-mono text-[10px] leading-none xl:inline" aria-hidden="true">
          {{ getShellShortcutHint('command') }}
        </kbd>
      </Button>
    </TooltipTrigger>
    <TooltipContent>{{ t('design.command.label') }} · <kbd>{{ getShellShortcutHint('command') }}</kbd></TooltipContent>
  </Tooltip>
  <Tooltip>
    <TooltipTrigger as-child>
      <Button
        type="button" variant="ghost" size="icon-sm" class="shrink-0 text-muted-foreground"
        :aria-label="dark ? t('nav.lightMode') : t('nav.darkMode')"
        :aria-keyshortcuts="getShellShortcutAriaKey('theme')" @click="emit('toggle-theme')"
      >
        <Sun v-if="dark" aria-hidden="true" />
        <Moon v-else aria-hidden="true" />
      </Button>
    </TooltipTrigger>
    <TooltipContent>
      {{ dark ? t('nav.lightMode') : t('nav.darkMode') }} · <kbd>{{ getShellShortcutHint('theme') }}</kbd>
    </TooltipContent>
  </Tooltip>
</template>
