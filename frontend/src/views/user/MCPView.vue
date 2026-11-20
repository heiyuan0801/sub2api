<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-6">
      <PageHeader :title="t('mcp.title')" :description="t('mcp.description')" />

      <!-- MCP Connection Card -->
      <section
        class="rounded-2xl border border-border/80 bg-card p-6 shadow-sm"
        aria-labelledby="mcp-connection-title"
        :aria-busy="connectionLoading"
      >
        <div class="flex items-start gap-3.5">
          <div class="flex size-10 shrink-0 items-center justify-center rounded-full bg-indigo-500 text-white shadow-sm">
            <Icon name="mcp" size="sm" :stroke-width="2" />
          </div>
          <div class="min-w-0">
            <h2 id="mcp-connection-title" class="text-base font-semibold tracking-tight text-foreground">{{ t('mcp.connectionTitle') }}</h2>
            <p class="mt-1 text-sm text-muted-foreground">{{ t('mcp.connectionDescription') }}</p>
          </div>
        </div>

        <div class="mt-5">
          <label for="mcp-endpoint" class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('mcp.endpoint') }}</label>
          <div class="flex flex-col gap-2 sm:flex-row">
            <input
              id="mcp-endpoint"
              :value="endpoint"
              :placeholder="connectionLoading ? t('mcp.connectionLoading') : ''"
              class="h-9 min-w-0 flex-1 rounded-xl border border-border bg-muted/40 px-3.5 font-mono text-sm text-foreground placeholder:font-sans placeholder:text-muted-foreground focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/20"
              readonly
            />
            <button
              data-testid="copy-endpoint"
              class="btn btn-secondary h-9 shrink-0 px-3.5 text-xs"
              type="button"
              :disabled="!endpoint || connectionLoading"
              @click="copyToClipboard(endpoint)"
            >
              <Icon name="copy" size="xs" />
              <span>{{ t('mcp.copyEndpoint') }}</span>
            </button>
          </div>
          <div v-if="connectionError" class="mt-3 flex flex-wrap items-center gap-3 rounded-xl border border-red-500/30 bg-red-500/10 px-3.5 py-2.5">
            <p role="alert" class="text-sm text-red-700 dark:text-red-400">{{ connectionError }}</p>
            <button
              data-testid="retry-connection"
              class="btn btn-secondary h-7 px-2.5 text-xs"
              type="button"
              :disabled="connectionLoading"
              @click="loadConnection"
            >
              {{ t('common.refresh') }}
            </button>
          </div>
          <ol class="mt-5 list-decimal space-y-2 rounded-xl border border-border/70 bg-muted/30 py-3.5 pl-9 pr-4 text-sm text-muted-foreground">
            <li>{{ t('mcp.connectStep1') }}</li>
            <li>{{ t('mcp.connectStep2') }}</li>
            <li>{{ t('mcp.connectStep3') }}</li>
          </ol>
        </div>
        <p class="mt-5 border-t border-border/80 pt-4 text-xs text-muted-foreground">{{ t('mcp.capabilities') }}</p>
      </section>

      <!-- Authorized Applications Card -->
      <section class="overflow-hidden rounded-2xl border border-border/80 bg-card shadow-sm" aria-labelledby="mcp-grants-title">
        <div class="flex flex-wrap items-start justify-between gap-4 border-b border-border/80 bg-muted/30 px-6 py-4">
          <div class="max-w-xl">
            <h2 id="mcp-grants-title" class="text-base font-semibold tracking-tight text-foreground">{{ t('mcp.grantsTitle') }}</h2>
            <p class="mt-0.5 text-xs text-muted-foreground">{{ t('mcp.grantsDescription') }}</p>
          </div>
          <button
            data-testid="refresh-grants"
            type="button"
            class="inline-flex h-8 items-center gap-1.5 rounded-full border border-border bg-card px-3 text-xs font-medium text-foreground transition-colors hover:bg-muted/60 disabled:pointer-events-none disabled:opacity-50"
            :disabled="loading || revoking"
            @click="loadGrants"
          >
            <Icon name="refresh" size="xs" :stroke-width="1.75" :class="loading ? 'animate-spin' : ''" />
            <span>{{ t('common.refresh') }}</span>
          </button>
        </div>

        <div class="p-6" :aria-busy="loading">
          <div v-if="loadError" class="rounded-xl border border-red-500/30 bg-red-500/10 px-4 py-3">
            <p role="alert" class="text-sm text-red-700 dark:text-red-400">{{ loadError }}</p>
          </div>

          <div v-else-if="loading" role="status" class="divide-y divide-border/70">
            <span class="sr-only">{{ t('mcp.loading') }}</span>
            <div v-for="i in 2" :key="`mcp-grant-sk-${i}`" class="flex items-start justify-between gap-4 py-4 first:pt-0 last:pb-0">
              <div class="min-w-0 flex-1 space-y-2.5">
                <div class="flex flex-wrap items-center gap-2">
                  <Skeleton class="h-4 w-36" />
                  <Skeleton class="h-5 w-16 rounded-full" />
                  <Skeleton class="h-5 w-16 rounded-full" />
                </div>
                <Skeleton class="h-3 w-48" />
                <div class="flex gap-4">
                  <Skeleton class="h-3 w-28" />
                  <Skeleton class="h-3 w-28" />
                </div>
              </div>
              <Skeleton class="h-8 w-16 rounded-lg" />
            </div>
          </div>

          <p
            v-else-if="grants.length === 0"
            class="rounded-xl border border-dashed border-border px-4 py-10 text-center text-sm text-muted-foreground"
          >
            {{ t('mcp.empty') }}
          </p>

          <ul v-else class="divide-y divide-border/70">
            <li
              v-for="grant in grants"
              :key="grant.id"
              class="flex items-start justify-between gap-4 py-4 first:pt-0 last:pb-0"
              :data-grant-id="grant.id"
            >
              <div class="min-w-0 space-y-2">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="break-all text-sm font-semibold text-foreground">{{ grant.client_name }}</span>
                  <span class="rounded-full border border-border bg-muted/50 px-2 py-0.5 text-[11px] font-medium text-foreground">
                    {{ grant.scope === 'admin' ? t('mcp.adminScope') : t('mcp.userScope') }}
                  </span>
                  <span class="rounded-full border border-indigo-500/20 bg-indigo-500/10 px-2 py-0.5 text-[11px] font-medium text-indigo-700 dark:text-indigo-300">
                    {{ grant.read_only ? t('mcp.readOnly') : t('mcp.readWrite') }}
                  </span>
                  <span
                    class="inline-flex items-center gap-1.5 text-xs font-medium"
                    :class="isActive(grant) ? 'text-emerald-600 dark:text-emerald-400' : 'text-muted-foreground'"
                  >
                    <span class="size-1.5 rounded-full" :class="isActive(grant) ? 'bg-emerald-500' : 'bg-muted-foreground/50'" />
                    <span>{{ grantStatus(grant) }}</span>
                  </span>
                </div>
                <p class="truncate font-mono text-xs text-muted-foreground" :title="grant.client_id">{{ grant.client_id }}</p>
                <div class="flex flex-wrap gap-x-4 gap-y-1 text-xs tabular-nums text-muted-foreground">
                  <span>{{ t('mcp.createdAt', { date: formatDate(grant.created_at) }) }}</span>
                  <span>{{ t('mcp.expiresAt', { date: formatDate(grant.expires_at) }) }}</span>
                  <span>{{ grant.last_used_at ? t('mcp.lastUsedAt', { date: formatDate(grant.last_used_at) }) : t('mcp.neverUsed') }}</span>
                </div>
              </div>
              <button
                v-if="!grant.revoked_at"
                type="button"
                class="inline-flex h-8 shrink-0 items-center rounded-lg px-2.5 text-xs font-medium text-red-600 transition-colors hover:bg-red-500/10 dark:text-red-400"
                :aria-label="`${t('mcp.revoke')} ${grant.client_name}`"
                @click="revokeTarget = grant"
              >
                {{ t('mcp.revoke') }}
              </button>
            </li>
          </ul>
        </div>
      </section>
    </div>

    <BaseDialog :show="!!revokeTarget" :title="t('mcp.revokeTitle')" width="narrow" :close-on-escape="!revoking" :show-close-button="!revoking" @close="closeRevoke">
      <p class="text-sm text-muted-foreground">{{ t('mcp.revokeConfirm', { name: revokeTarget?.client_name || '' }) }}</p>
      <p v-if="revokeError" role="alert" class="mt-3 text-sm text-red-600 dark:text-red-400">{{ revokeError }}</p>
      <template #footer>
        <div class="flex justify-end gap-2.5">
          <button type="button" class="btn btn-secondary h-9 px-3.5 text-sm" :disabled="revoking" @click="closeRevoke">{{ t('common.cancel') }}</button>
          <button data-testid="confirm-revoke" type="button" class="btn btn-danger h-9 px-3.5 text-sm" :disabled="revoking || loading" @click="revokeGrant">{{ revoking ? t('common.processing') : t('mcp.revoke') }}</button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { mcpAPI, type MCPGrant } from '@/api/mcp'
import BaseDialog from '@/components/common/BaseDialog.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import { Icon } from '@/components/icons'
import AppLayout from '@/components/layout/AppLayout.vue'
import { Skeleton } from '@/components/ui/skeleton'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores/app'
import { formatDate } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()
const endpoint = ref('')
const connectionLoading = ref(false)
const connectionError = ref('')
const grants = ref<MCPGrant[]>([])
const loading = ref(false)
const loadError = ref('')
const revokeTarget = ref<MCPGrant | null>(null)
const revoking = ref(false)
const revokeError = ref('')
let unmounted = false

function errorMessage(error: unknown, fallback: string): string {
  const message = (error as { message?: unknown } | null)?.message
  return typeof message === 'string' && message ? message : fallback
}

async function loadConnection(): Promise<void> {
  if (connectionLoading.value) return
  connectionLoading.value = true
  connectionError.value = ''
  endpoint.value = ''
  try {
    const result = await mcpAPI.getConnectionConfig()
    if (!unmounted) endpoint.value = result.resource
  } catch (error) {
    if (!unmounted) connectionError.value = errorMessage(error, t('mcp.connectionLoadFailed'))
  } finally {
    connectionLoading.value = false
  }
}

function isActive(grant: MCPGrant): boolean {
  return !grant.revoked_at && !grant.invalidated && new Date(grant.expires_at).getTime() > Date.now()
}

function grantStatus(grant: MCPGrant): string {
  if (grant.revoked_at) return t('mcp.revoked')
  if (grant.invalidated) return t('mcp.invalidated')
  return isActive(grant) ? t('mcp.active') : t('mcp.expired')
}

async function loadGrants(): Promise<void> {
  if (loading.value || revoking.value) return
  loading.value = true
  loadError.value = ''
  try {
    const result = await mcpAPI.listGrants()
    if (!unmounted) grants.value = result
  } catch (error) {
    if (!unmounted) loadError.value = errorMessage(error, t('mcp.loadFailed'))
  } finally {
    loading.value = false
  }
}

function closeRevoke(): void {
  if (revoking.value) return
  revokeTarget.value = null
  revokeError.value = ''
}

async function revokeGrant(): Promise<void> {
  if (!revokeTarget.value || revoking.value || loading.value) return
  revoking.value = true
  revokeError.value = ''
  const id = revokeTarget.value.id
  try {
    await mcpAPI.revokeGrant(id)
    if (unmounted) return
    grants.value = grants.value.map(grant => grant.id === id ? { ...grant, revoked_at: new Date().toISOString() } : grant)
    revokeTarget.value = null
    appStore.showSuccess(t('mcp.revokeSuccess'))
  } catch (error) {
    if (!unmounted) revokeError.value = errorMessage(error, t('mcp.revokeFailed'))
  } finally {
    revoking.value = false
  }
}

onMounted(() => {
  void loadConnection()
  void loadGrants()
})
onBeforeUnmount(() => { unmounted = true })
</script>
