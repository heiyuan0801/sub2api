<script setup lang="ts">
import { ChevronRight, Menu, PanelLeft } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { getShellShortcutAriaKey, getShellShortcutHint } from './shortcuts'

defineProps<{
  title: string
  description?: string
  section?: string
  collapsed: boolean
  mobileOpen: boolean
}>()
const emit = defineEmits<{ 'toggle-desktop': []; 'toggle-mobile': [] }>()
const { t } = useI18n()
</script>

<template>
  <div class="flex min-w-0 items-center gap-1.5 sm:gap-2.5">
    <Button
      data-console-mobile-trigger
      type="button" variant="ghost" size="icon-sm" class="lg:hidden"
      aria-haspopup="dialog" :aria-expanded="mobileOpen" :aria-label="t('common.toggleMenu')"
      @click="emit('toggle-mobile')"
    >
      <Menu aria-hidden="true" />
    </Button>
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          type="button" variant="ghost" size="icon-sm" class="hidden lg:inline-flex"
          :aria-expanded="!collapsed" aria-controls="console-desktop-navigation" :aria-label="collapsed ? t('nav.expand') : t('nav.collapse')"
          :aria-keyshortcuts="getShellShortcutAriaKey('sidebar')" @click="emit('toggle-desktop')"
        >
          <PanelLeft aria-hidden="true" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>
        {{ collapsed ? t('nav.expand') : t('nav.collapse') }} · <kbd>{{ getShellShortcutHint('sidebar') }}</kbd>
      </TooltipContent>
    </Tooltip>
    <span class="hidden h-4 w-px shrink-0 bg-border lg:block" aria-hidden="true" />
    <nav :aria-label="t('design.shell.breadcrumb')" class="flex min-w-0 items-center gap-1.5">
      <span v-if="section" class="hidden shrink-0 text-xs text-muted-foreground xl:inline">{{ section }}</span>
      <ChevronRight v-if="section" class="hidden size-3.5 shrink-0 text-muted-foreground xl:block" aria-hidden="true" />
      <h1 :title="title" aria-current="page" class="min-w-0 truncate text-sm font-medium text-foreground max-md:sr-only">{{ title }}</h1>
      <p v-if="description" class="sr-only">{{ description }}</p>
    </nav>
  </div>
</template>
