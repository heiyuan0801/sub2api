<template>
  <a
    v-if="live"
    :href="statusPageURL"
    target="_blank"
    rel="noopener noreferrer"
    :aria-label="accessibleLabel"
    :title="accessibleLabel"
    class="inline-flex h-8 shrink-0 items-center gap-1 rounded-full border px-1.5 text-xs font-medium focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 sm:gap-1.5 sm:px-2.5"
    :class="tone"
  >
    <span class="size-2 shrink-0 rounded-full bg-current" aria-hidden="true" />
    <span class="whitespace-nowrap" aria-live="polite">{{ label }}</span>
  </a>
  <a
    v-else
    :href="statusPageURL"
    target="_blank"
    rel="noopener noreferrer"
    class="btn btn-secondary btn-sm shrink-0 gap-1.5"
  >
    <Icon name="globe" size="sm" aria-hidden="true" />
    {{ t('channelStatus.websiteStatus') }}
    <Icon name="externalLink" size="xs" aria-hidden="true" />
    <span class="sr-only">{{ t('channelStatus.opensInNewTab') }}</span>
  </a>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { buildApiUrl } from '@/api/url'

const props = defineProps<{ live?: boolean }>()
const { t } = useI18n()
const statusPageURL = 'https://status.weilai.chat/status/weilai'

const statusStyles = {
  up: 'border-emerald-200 bg-emerald-50 text-emerald-800 hover:bg-emerald-100 dark:border-emerald-700/60 dark:bg-emerald-950 dark:text-emerald-300',
  down: 'border-red-200 bg-red-50 text-red-800 hover:bg-red-100 dark:border-red-700/60 dark:bg-red-950 dark:text-red-300',
  maintenance: 'border-blue-200 bg-blue-50 text-blue-800 hover:bg-blue-100 dark:border-blue-700/60 dark:bg-blue-950 dark:text-blue-300',
  unknown: 'border-border bg-muted text-muted-foreground hover:bg-accent hover:text-accent-foreground',
}
type WebsiteStatus = keyof typeof statusStyles

const status = ref<WebsiteStatus>('unknown')
const tone = computed(() => statusStyles[status.value])
const label = computed(() => t(`channelStatus.uptime.${status.value}`))
const accessibleLabel = computed(() => `${t('channelStatus.websiteStatus')} · ${label.value} ${t('channelStatus.opensInNewTab')}`)
let poll: number | undefined
let request: AbortController | undefined

async function refresh() {
  if (request || document.visibilityState === 'hidden') return
  const controller = new AbortController()
  request = controller
  const timeout = window.setTimeout(() => controller.abort(), 6000)
  try {
    const response = await fetch(buildApiUrl('/settings/website-status'), {
      credentials: 'omit',
      signal: controller.signal,
    })
    if (!response.ok) throw new Error('Could not load website status')
    const payload = await response.json()
    const value = payload?.data?.status
    if (request === controller) {
      status.value = payload?.code === 0 && typeof value === 'string' && Object.prototype.hasOwnProperty.call(statusStyles, value)
        ? value as WebsiteStatus
        : 'unknown'
    }
  } catch {
    if (request === controller) status.value = 'unknown'
  } finally {
    window.clearTimeout(timeout)
    if (request === controller) request = undefined
  }
}

function stop() {
  window.clearInterval(poll)
  document.removeEventListener('visibilitychange', refresh)
  request?.abort()
  request = undefined
}

function start() {
  void refresh()
  poll = window.setInterval(refresh, 30000)
  document.addEventListener('visibilitychange', refresh)
}

onMounted(() => watch(() => props.live, live => {
  if (live) start()
  else stop()
}, { immediate: true }))
onBeforeUnmount(stop)
</script>
