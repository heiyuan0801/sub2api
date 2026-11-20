<template>
  <DialogRoot v-model:open="isOpen">
    <button
      type="button"
      class="inline-flex size-8 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      :aria-label="t('themeConfig.openSettings')"
      :title="t('themeConfig.openSettings')"
      @click="isOpen = true"
    >
      <Palette class="size-4" aria-hidden="true" />
    </button>

    <DialogPortal>
      <DialogOverlay
        class="fixed inset-0 z-50 bg-black/45 backdrop-blur-[1px] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0"
      />
      <DialogContent
        class="fixed inset-y-0 right-0 z-50 flex w-full max-w-sm flex-col border-l border-border bg-background text-foreground shadow-2xl outline-none duration-200 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:slide-out-to-right data-[state=open]:slide-in-from-right"
      >
        <!-- Drawer Header -->
        <div class="flex items-start justify-between gap-3 border-b border-border px-5 py-4">
          <div class="min-w-0">
            <DialogTitle class="text-sm font-semibold tracking-tight text-foreground">
              {{ t('themeConfig.title') }}
            </DialogTitle>
            <DialogDescription class="mt-0.5 text-xs text-muted-foreground">
              {{ t('themeConfig.description') }}
            </DialogDescription>
          </div>
          <DialogClose
            class="inline-flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            :aria-label="t('common.close')"
          >
            <X class="size-4" aria-hidden="true" />
          </DialogClose>
        </div>

        <!-- Scrollable Body -->
        <div class="flex-1 space-y-6 overflow-y-auto px-5 py-4">
          <!-- Appearance / Theme Mode -->
          <section class="space-y-2.5">
            <h3 class="text-xs font-medium text-muted-foreground">
              {{ t('themeConfig.appearance') }}
            </h3>
            <div class="grid grid-cols-3 gap-2.5" role="radiogroup" :aria-label="t('themeConfig.appearance')">
              <button
                v-for="item in modeOptions"
                :key="item.value"
                type="button"
                role="radio"
                :aria-checked="themeStore.mode === item.value"
                class="group flex flex-col items-center gap-1.5 rounded-lg p-1 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                @click="themeStore.setMode(item.value)"
              >
                <div
                  class="relative w-full overflow-hidden rounded-lg border transition-all"
                  :class="
                    themeStore.mode === item.value
                      ? 'border-primary ring-2 ring-primary'
                      : 'border-border group-hover:border-muted-foreground/50'
                  "
                >
                  <!-- System Split Wireframe -->
                  <svg
                    v-if="item.value === 'system'"
                    viewBox="0 0 96 64"
                    class="block h-auto w-full"
                    aria-hidden="true"
                  >
                    <defs>
                      <clipPath id="theme-preview-left-half">
                        <rect x="0" y="0" width="48" height="64" />
                      </clipPath>
                      <clipPath id="theme-preview-right-half">
                        <rect x="48" y="0" width="48" height="64" />
                      </clipPath>
                    </defs>
                    <!-- Light left half -->
                    <g clip-path="url(#theme-preview-left-half)">
                      <rect x="0" y="0" width="96" height="64" fill="#f8fafc" />
                      <rect x="0" y="0" width="24" height="64" fill="#f1f5f9" />
                      <line x1="24" y1="0" x2="24" y2="64" stroke="#e2e8f0" stroke-width="1" />
                      <rect x="5" y="9" width="14" height="3.5" rx="1.5" fill="#94a3b8" />
                      <rect x="5" y="17" width="14" height="3" rx="1.5" fill="#cbd5e1" />
                      <rect x="5" y="23" width="11" height="3" rx="1.5" fill="#cbd5e1" />
                      <rect x="30" y="9" width="24" height="5" rx="2" fill="#e2e8f0" />
                      <rect x="30" y="19" width="58" height="16" rx="3" fill="#ffffff" stroke="#e2e8f0" stroke-width="1" />
                      <rect x="30" y="40" width="58" height="16" rx="3" fill="#ffffff" stroke="#e2e8f0" stroke-width="1" />
                    </g>
                    <!-- Dark right half -->
                    <g clip-path="url(#theme-preview-right-half)">
                      <rect x="0" y="0" width="96" height="64" fill="#18181b" />
                      <rect x="30" y="9" width="24" height="5" rx="2" fill="#3f3f46" />
                      <rect x="30" y="19" width="58" height="16" rx="3" fill="#27272a" stroke="#3f3f46" stroke-width="1" />
                      <rect x="30" y="40" width="58" height="16" rx="3" fill="#27272a" stroke="#3f3f46" stroke-width="1" />
                      <rect x="56" y="25" width="22" height="3" rx="1.5" fill="#52525b" />
                      <rect x="56" y="46" width="16" height="3" rx="1.5" fill="#52525b" />
                    </g>
                    <line x1="48" y1="0" x2="48" y2="64" stroke="#94a3b8" stroke-opacity="0.5" stroke-width="1" />
                  </svg>

                  <!-- Light Wireframe -->
                  <svg
                    v-else-if="item.value === 'light'"
                    viewBox="0 0 96 64"
                    class="block h-auto w-full"
                    aria-hidden="true"
                  >
                    <rect x="0" y="0" width="96" height="64" fill="#f8fafc" />
                    <rect x="0" y="0" width="24" height="64" fill="#f1f5f9" />
                    <line x1="24" y1="0" x2="24" y2="64" stroke="#e2e8f0" stroke-width="1" />
                    <rect x="5" y="9" width="14" height="3.5" rx="1.5" fill="#0ea5e9" />
                    <rect x="5" y="17" width="14" height="3" rx="1.5" fill="#cbd5e1" />
                    <rect x="5" y="23" width="11" height="3" rx="1.5" fill="#cbd5e1" />
                    <rect x="5" y="29" width="13" height="3" rx="1.5" fill="#cbd5e1" />
                    <rect x="30" y="9" width="26" height="4.5" rx="2" fill="#cbd5e1" />
                    <rect x="30" y="19" width="58" height="16" rx="3" fill="#ffffff" stroke="#e2e8f0" stroke-width="1" />
                    <rect x="35" y="24" width="20" height="3" rx="1.5" fill="#94a3b8" />
                    <rect x="35" y="29" width="32" height="2.5" rx="1.25" fill="#e2e8f0" />
                    <rect x="30" y="40" width="27" height="16" rx="3" fill="#ffffff" stroke="#e2e8f0" stroke-width="1" />
                    <rect x="61" y="40" width="27" height="16" rx="3" fill="#ffffff" stroke="#e2e8f0" stroke-width="1" />
                  </svg>

                  <!-- Dark Wireframe -->
                  <svg
                    v-else
                    viewBox="0 0 96 64"
                    class="block h-auto w-full"
                    aria-hidden="true"
                  >
                    <rect x="0" y="0" width="96" height="64" fill="#18181b" />
                    <rect x="0" y="0" width="24" height="64" fill="#141416" />
                    <line x1="24" y1="0" x2="24" y2="64" stroke="#27272a" stroke-width="1" />
                    <rect x="5" y="9" width="14" height="3.5" rx="1.5" fill="#38bdf8" />
                    <rect x="5" y="17" width="14" height="3" rx="1.5" fill="#3f3f46" />
                    <rect x="5" y="23" width="11" height="3" rx="1.5" fill="#3f3f46" />
                    <rect x="5" y="29" width="13" height="3" rx="1.5" fill="#3f3f46" />
                    <rect x="30" y="9" width="26" height="4.5" rx="2" fill="#3f3f46" />
                    <rect x="30" y="19" width="58" height="16" rx="3" fill="#232327" stroke="#323238" stroke-width="1" />
                    <rect x="35" y="24" width="20" height="3" rx="1.5" fill="#71717a" />
                    <rect x="35" y="29" width="32" height="2.5" rx="1.25" fill="#3f3f46" />
                    <rect x="30" y="40" width="27" height="16" rx="3" fill="#232327" stroke="#323238" stroke-width="1" />
                    <rect x="61" y="40" width="27" height="16" rx="3" fill="#232327" stroke="#323238" stroke-width="1" />
                  </svg>

                  <span
                    v-if="themeStore.mode === item.value"
                    class="absolute right-1.5 top-1.5 inline-flex size-4 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-sm"
                  >
                    <Check class="size-2.5" aria-hidden="true" />
                  </span>
                </div>
                <span
                  class="text-xs transition-colors"
                  :class="
                    themeStore.mode === item.value
                      ? 'font-medium text-foreground'
                      : 'text-muted-foreground group-hover:text-foreground'
                  "
                >
                  {{ item.label }}
                </span>
              </button>
            </div>
          </section>

          <!-- Color Preset -->
          <section class="space-y-2.5">
            <h3 class="text-xs font-medium text-muted-foreground">
              {{ t('themeConfig.colorPreset') }}
            </h3>
            <div class="grid grid-cols-3 gap-2" role="radiogroup" :aria-label="t('themeConfig.colorPreset')">
              <button
                v-for="item in presetOptions"
                :key="item.value"
                type="button"
                role="radio"
                :aria-checked="themeStore.preset === item.value"
                class="flex items-center gap-2 rounded-lg border px-2.5 py-2 text-left transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                :class="
                  themeStore.preset === item.value
                    ? 'border-primary bg-accent/60 text-foreground ring-1 ring-primary/30'
                    : 'border-border bg-background text-muted-foreground hover:border-muted-foreground/40 hover:bg-accent/30 hover:text-foreground'
                "
                @click="themeStore.setPreset(item.value)"
              >
                <span
                  class="relative inline-flex size-4 shrink-0 items-center justify-center rounded-full border border-black/10 shadow-sm dark:border-white/15"
                  :style="{ background: item.swatch }"
                  aria-hidden="true"
                >
                  <Check
                    v-if="themeStore.preset === item.value"
                    class="size-2.5"
                    :class="item.checkColorClass"
                  />
                </span>
                <span class="min-w-0 truncate text-xs font-medium">
                  {{ item.label }}
                </span>
              </button>
            </div>
          </section>

          <!-- Border Radius -->
          <section class="space-y-2.5">
            <h3 class="text-xs font-medium text-muted-foreground">
              {{ t('themeConfig.radius') }}
            </h3>
            <div class="grid grid-cols-4 gap-2" role="radiogroup" :aria-label="t('themeConfig.radius')">
              <button
                v-for="item in radiusOptions"
                :key="item.value"
                type="button"
                role="radio"
                :aria-checked="themeStore.radius === item.value"
                class="flex flex-col items-center justify-center gap-1 rounded-lg border px-2 py-2 text-xs transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                :class="
                  themeStore.radius === item.value
                    ? 'border-primary bg-accent/60 font-medium text-foreground ring-1 ring-primary/30'
                    : 'border-border bg-background text-muted-foreground hover:border-muted-foreground/40 hover:bg-accent/30 hover:text-foreground'
                "
                @click="themeStore.setRadius(item.value)"
              >
                <span
                  class="block size-4 border-l-2 border-t-2 border-current"
                  :style="{ borderTopLeftRadius: item.previewRadius }"
                  aria-hidden="true"
                />
                <span>{{ item.label }}</span>
              </button>
            </div>
          </section>

          <!-- Font -->
          <section class="space-y-2.5">
            <h3 class="text-xs font-medium text-muted-foreground">
              {{ t('themeConfig.font') }}
            </h3>
            <div class="grid grid-cols-3 gap-2" role="radiogroup" :aria-label="t('themeConfig.font')">
              <button
                v-for="item in fontOptions"
                :key="item.value"
                type="button"
                role="radio"
                :aria-checked="themeStore.font === item.value"
                class="flex flex-col items-center justify-center gap-1 rounded-lg border px-2.5 py-2 transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                :class="
                  themeStore.font === item.value
                    ? 'border-primary bg-accent/60 font-medium text-foreground ring-1 ring-primary/30'
                    : 'border-border bg-background text-muted-foreground hover:border-muted-foreground/40 hover:bg-accent/30 hover:text-foreground'
                "
                @click="themeStore.setFont(item.value)"
              >
                <span
                  class="text-base leading-none font-semibold text-foreground"
                  :style="{ fontFamily: item.fontFamily }"
                  aria-hidden="true"
                >
                  Aa
                </span>
                <span class="text-xs">{{ item.label }}</span>
              </button>
            </div>
          </section>

          <!-- Sidebar Style -->
          <section v-if="showLayoutOptions" class="space-y-2.5">
            <h3 class="text-xs font-medium text-muted-foreground">
              {{ t('themeConfig.sidebarStyle') }}
            </h3>
            <div class="grid grid-cols-3 gap-2.5" role="radiogroup" :aria-label="t('themeConfig.sidebarStyle')">
              <button
                v-for="item in sidebarOptions"
                :key="item.value"
                type="button"
                role="radio"
                :aria-checked="themeStore.sidebarVariant === item.value"
                class="group flex flex-col items-center gap-1.5 rounded-lg p-1 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                @click="themeStore.setSidebarVariant(item.value)"
              >
                <div
                  class="relative w-full overflow-hidden rounded-lg border bg-muted/40 transition-all"
                  :class="
                    themeStore.sidebarVariant === item.value
                      ? 'border-primary ring-2 ring-primary'
                      : 'border-border group-hover:border-muted-foreground/50'
                  "
                >
                  <!-- Inset Layout Preview -->
                  <svg
                    v-if="item.value === 'inset'"
                    viewBox="0 0 96 64"
                    class="block h-auto w-full text-muted-foreground"
                    aria-hidden="true"
                  >
                    <rect x="0" y="0" width="96" height="64" fill="hsl(var(--sidebar-background))" />
                    <rect x="6" y="10" width="14" height="3.5" rx="1.5" fill="hsl(var(--primary))" />
                    <rect x="6" y="18" width="13" height="2.5" rx="1.25" fill="currentColor" fill-opacity="0.35" />
                    <rect x="6" y="24" width="10" height="2.5" rx="1.25" fill="currentColor" fill-opacity="0.25" />
                    <rect x="6" y="30" width="12" height="2.5" rx="1.25" fill="currentColor" fill-opacity="0.25" />
                    <rect
                      x="25"
                      y="5"
                      width="66"
                      height="54"
                      rx="6"
                      fill="hsl(var(--background))"
                      stroke="hsl(var(--border))"
                      stroke-width="1.2"
                    />
                    <line x1="25" y1="15" x2="91" y2="15" stroke="hsl(var(--border))" stroke-width="1" />
                    <rect x="31" y="21" width="24" height="12" rx="2.5" fill="hsl(var(--muted))" />
                    <rect x="59" y="21" width="26" height="12" rx="2.5" fill="hsl(var(--muted))" />
                    <rect x="31" y="37" width="54" height="16" rx="2.5" fill="hsl(var(--muted))" />
                  </svg>

                  <!-- Floating Layout Preview -->
                  <svg
                    v-else-if="item.value === 'floating'"
                    viewBox="0 0 96 64"
                    class="block h-auto w-full text-muted-foreground"
                    aria-hidden="true"
                  >
                    <rect x="0" y="0" width="96" height="64" fill="hsl(var(--background))" />
                    <rect
                      x="5"
                      y="5"
                      width="20"
                      height="54"
                      rx="5"
                      fill="hsl(var(--sidebar-background))"
                      stroke="hsl(var(--border))"
                      stroke-width="1.2"
                    />
                    <rect x="9" y="11" width="12" height="3.5" rx="1.5" fill="hsl(var(--primary))" />
                    <rect x="9" y="19" width="11" height="2.5" rx="1.25" fill="currentColor" fill-opacity="0.35" />
                    <rect x="9" y="25" width="9" height="2.5" rx="1.25" fill="currentColor" fill-opacity="0.25" />
                    <rect x="9" y="31" width="10" height="2.5" rx="1.25" fill="currentColor" fill-opacity="0.25" />
                    <line x1="30" y1="14" x2="92" y2="14" stroke="hsl(var(--border))" stroke-width="1" />
                    <rect x="31" y="20" width="27" height="14" rx="2.5" fill="hsl(var(--muted))" />
                    <rect x="62" y="20" width="28" height="14" rx="2.5" fill="hsl(var(--muted))" />
                    <rect x="31" y="38" width="59" height="18" rx="2.5" fill="hsl(var(--muted))" />
                  </svg>

                  <!-- Classic Sidebar Preview -->
                  <svg
                    v-else
                    viewBox="0 0 96 64"
                    class="block h-auto w-full text-muted-foreground"
                    aria-hidden="true"
                  >
                    <rect x="0" y="0" width="96" height="64" fill="hsl(var(--background))" />
                    <rect x="0" y="0" width="24" height="64" fill="hsl(var(--sidebar-background))" />
                    <line x1="24" y1="0" x2="24" y2="64" stroke="hsl(var(--border))" stroke-width="1.2" />
                    <rect x="5" y="10" width="14" height="3.5" rx="1.5" fill="hsl(var(--primary))" />
                    <rect x="5" y="18" width="13" height="2.5" rx="1.25" fill="currentColor" fill-opacity="0.35" />
                    <rect x="5" y="24" width="10" height="2.5" rx="1.25" fill="currentColor" fill-opacity="0.25" />
                    <rect x="5" y="30" width="12" height="2.5" rx="1.25" fill="currentColor" fill-opacity="0.25" />
                    <line x1="24" y1="14" x2="96" y2="14" stroke="hsl(var(--border))" stroke-width="1" />
                    <rect x="30" y="20" width="28" height="14" rx="2.5" fill="hsl(var(--muted))" />
                    <rect x="62" y="20" width="28" height="14" rx="2.5" fill="hsl(var(--muted))" />
                    <rect x="30" y="38" width="60" height="18" rx="2.5" fill="hsl(var(--muted))" />
                  </svg>

                  <span
                    v-if="themeStore.sidebarVariant === item.value"
                    class="absolute right-1.5 top-1.5 inline-flex size-4 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-sm"
                  >
                    <Check class="size-2.5" aria-hidden="true" />
                  </span>
                </div>
                <span
                  class="text-xs transition-colors"
                  :class="
                    themeStore.sidebarVariant === item.value
                      ? 'font-medium text-foreground'
                      : 'text-muted-foreground group-hover:text-foreground'
                  "
                >
                  {{ item.label }}
                </span>
              </button>
            </div>
          </section>
        </div>

        <!-- Drawer Footer -->
        <div class="border-t border-border px-5 py-3.5">
          <button
            type="button"
            class="inline-flex h-9 w-full items-center justify-center gap-2 rounded-md border border-border bg-background px-3 text-xs font-medium text-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            @click="themeStore.resetAll()"
          >
            <RotateCcw class="size-3.5" aria-hidden="true" />
            <span>{{ t('themeConfig.reset') }}</span>
          </button>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle
} from 'reka-ui'
import { Check, Palette, RotateCcw, X } from '@lucide/vue'
import {
  useThemeStore,
  type SidebarVariant,
  type ThemeFont,
  type ThemeMode,
  type ThemePreset,
  type ThemeRadius
} from '@/stores/theme'

const props = withDefaults(
  defineProps<{
    showLayoutOptions?: boolean
  }>(),
  {
    showLayoutOptions: true
  }
)

const { t } = useI18n()
const themeStore = useThemeStore()

// Console drawer syncs with store so CommandMenu ("Open Theme Settings") can open it.
// Public header drawer uses isolated local state so multiple instances never clash.
const localOpen = ref(false)
const isOpen = computed({
  get: () => (props.showLayoutOptions ? themeStore.configDrawerOpen : localOpen.value),
  set: (value: boolean) => {
    if (props.showLayoutOptions) {
      themeStore.setConfigDrawerOpen(value)
    } else {
      localOpen.value = value
    }
  }
})

const modeOptions = computed<Array<{ value: ThemeMode; label: string }>>(() => [
  { value: 'system', label: t('themeConfig.modes.system') },
  { value: 'light', label: t('themeConfig.modes.light') },
  { value: 'dark', label: t('themeConfig.modes.dark') }
])

const presetOptions = computed<
  Array<{ value: ThemePreset; label: string; swatch: string; checkColorClass: string }>
>(() => [
  {
    value: 'default',
    label: t('themeConfig.presets.default'),
    swatch: 'hsl(205 84% 50%)',
    checkColorClass: 'text-white'
  },
  {
    value: 'anthropic',
    label: t('themeConfig.presets.anthropic'),
    swatch: 'hsl(15 63% 54%)',
    checkColorClass: 'text-white'
  },
  {
    value: 'rose-garden',
    label: t('themeConfig.presets.roseGarden'),
    swatch: 'hsl(346 77% 52%)',
    checkColorClass: 'text-white'
  },
  {
    value: 'lake-view',
    label: t('themeConfig.presets.lakeView'),
    swatch: 'hsl(162 72% 40%)',
    checkColorClass: 'text-white'
  },
  {
    value: 'sunset-glow',
    label: t('themeConfig.presets.sunsetGlow'),
    swatch: 'hsl(18 82% 52%)',
    checkColorClass: 'text-white'
  },
  {
    value: 'forest-whisper',
    label: t('themeConfig.presets.forestWhisper'),
    swatch: 'hsl(145 55% 38%)',
    checkColorClass: 'text-white'
  },
  {
    value: 'ocean-breeze',
    label: t('themeConfig.presets.oceanBreeze'),
    swatch: 'hsl(221 83% 53%)',
    checkColorClass: 'text-white'
  },
  {
    value: 'lavender-dream',
    label: t('themeConfig.presets.lavenderDream'),
    swatch: 'hsl(263 70% 54%)',
    checkColorClass: 'text-white'
  },
  {
    value: 'neutral',
    label: t('themeConfig.presets.neutral'),
    swatch: themeStore.isDark ? 'hsl(0 0% 92%)' : 'hsl(0 0% 15%)',
    checkColorClass: themeStore.isDark ? 'text-zinc-900' : 'text-white'
  }
])

const radiusOptions = computed<
  Array<{ value: ThemeRadius; label: string; previewRadius: string }>
>(() => [
  { value: 'none', label: t('themeConfig.radiusOptions.none'), previewRadius: '0px' },
  { value: 'sm', label: t('themeConfig.radiusOptions.sm'), previewRadius: '4px' },
  { value: 'md', label: t('themeConfig.radiusOptions.md'), previewRadius: '7px' },
  { value: 'lg', label: t('themeConfig.radiusOptions.lg'), previewRadius: '11px' }
])

const fontOptions = computed<Array<{ value: ThemeFont; label: string; fontFamily: string }>>(() => [
  {
    value: 'system',
    label: t('themeConfig.fontOptions.system'),
    fontFamily: "system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"
  },
  {
    value: 'sans',
    label: t('themeConfig.fontOptions.sans'),
    fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, 'PingFang SC', 'Segoe UI', sans-serif"
  },
  {
    value: 'serif',
    label: t('themeConfig.fontOptions.serif'),
    fontFamily: "Georgia, 'Songti SC', 'Noto Serif SC', 'STSong', serif"
  }
])

const sidebarOptions = computed<Array<{ value: SidebarVariant; label: string }>>(() => [
  { value: 'inset', label: t('themeConfig.sidebarVariants.inset') },
  { value: 'floating', label: t('themeConfig.sidebarVariants.floating') },
  { value: 'sidebar', label: t('themeConfig.sidebarVariants.sidebar') }
])
</script>
