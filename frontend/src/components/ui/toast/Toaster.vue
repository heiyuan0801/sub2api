<script setup lang="ts">
import type { CSSProperties } from 'vue'
import { useI18n } from 'vue-i18n'
import { Toaster as Sonner, type ToasterProps } from 'vue-sonner'
import 'vue-sonner/style.css'

const props = withDefaults(defineProps<ToasterProps>(), {
  position: 'bottom-right',
  closeButton: true,
  richColors: true
})
const { t } = useI18n()

// Using CSS variables directly follows the app's theme, including manual
// light/dark overrides that do not match the system setting.
const themeStyle: CSSProperties = {
  '--normal-bg': 'hsl(var(--popover))',
  '--normal-text': 'hsl(var(--popover-foreground))',
  '--normal-border': 'hsl(var(--border))',
  '--normal-bg-hover': 'hsl(var(--accent))',
  '--normal-border-hover': 'hsl(var(--border))',
  '--success-bg': 'hsl(var(--card))',
  '--success-text': 'hsl(var(--success))',
  '--success-border': 'hsl(var(--success) / 0.3)',
  '--info-bg': 'hsl(var(--card))',
  '--info-text': 'hsl(var(--info))',
  '--info-border': 'hsl(var(--info) / 0.3)',
  '--warning-bg': 'hsl(var(--card))',
  '--warning-text': 'hsl(var(--warning))',
  '--warning-border': 'hsl(var(--warning) / 0.3)',
  '--error-bg': 'hsl(var(--card))',
  '--error-text': 'hsl(var(--danger))',
  '--error-border': 'hsl(var(--danger) / 0.3)',
  '--border-radius': 'var(--radius)'
}
</script>

<template>
  <Sonner
    v-bind="props"
    :style="{ ...themeStyle, ...props.style }"
    :container-aria-label="props.containerAriaLabel ?? t('design.toast.notifications')"
    :toast-options="{ closeButtonAriaLabel: t('common.close'), ...props.toastOptions }"
  />
</template>
