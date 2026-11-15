<template>
  <AuthFlowFrame>
    <SectionCard class="[&>div]:p-5 sm:[&>div]:p-6" aria-labelledby="mcp-consent-title" :aria-busy="loading || submitting">
      <div class="mb-6 flex items-start gap-3">
        <div class="inline-flex size-10 shrink-0 items-center justify-center rounded-lg border border-brand/20 bg-brand/10 text-brand"><Icon name="mcp" size="lg" /></div>
        <div>
          <h1 id="mcp-consent-title" class="text-xl font-semibold text-foreground">{{ t('mcp.consentTitle') }}</h1>
          <p class="mt-1 text-sm text-muted-foreground">{{ t('mcp.consentDescription') }}</p>
        </div>
      </div>
      <p v-if="loading" role="status" class="py-8 text-center text-sm text-muted-foreground">{{ t('mcp.consentLoading') }}</p>
      <div v-else-if="previewError" class="space-y-4">
        <p role="alert" class="text-sm text-danger">{{ previewError }}</p>
        <p class="text-sm text-muted-foreground">{{ t('mcp.consentRetryHint') }}</p>
        <RouterLink to="/settings/mcp" class="btn btn-secondary">{{ t('mcp.backToConnections') }}</RouterLink>
      </div>
      <form v-else-if="consent" class="space-y-6" @submit.prevent="decide(true)">
        <div class="rounded-lg border border-border bg-muted/50 p-4">
          <dl class="grid gap-4 text-sm">
            <div>
              <dt class="text-muted-foreground">{{ t('mcp.applicationName') }}</dt>
              <dd data-testid="client-name" class="mt-1 break-all text-lg font-semibold text-foreground">{{ consent.client_name }}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">{{ t('mcp.clientId') }}</dt>
              <dd class="mt-1 truncate font-mono text-xs text-foreground" :title="consent.client_id">{{ consent.client_id }}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">{{ t('mcp.returnAddress') }}</dt>
              <dd data-testid="redirect-origin" class="mt-1 break-all font-semibold text-foreground">{{ redirectOrigin }}</dd>
              <dd data-testid="redirect-uri" class="mt-1 break-all font-mono text-xs text-muted-foreground">{{ consent.redirect_uri }}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">{{ t('mcp.requestedAccess') }}</dt>
              <dd class="mt-1 text-foreground">{{ t('mcp.requestedAccessDescription') }}</dd>
              <dd class="mt-1 break-all font-mono text-xs text-muted-foreground">{{ consent.resource }}</dd>
            </div>
          </dl>
          <p class="mt-4 border-t border-border pt-4 text-sm text-foreground">{{ t('mcp.untrustedApplicationHint') }}</p>
        </div>
        <p v-if="authStore.user?.email" class="break-all text-sm text-muted-foreground">{{ t('mcp.signedInAs', { email: authStore.user.email }) }}</p>
        <div>
          <label for="mcp-consent-scope" class="input-label">{{ t('mcp.scope') }}</label>
          <select id="mcp-consent-scope" v-model="scope" class="input" :disabled="submitting" aria-describedby="mcp-scope-hint">
            <option value="user">{{ t('mcp.userScope') }}</option>
            <option v-if="authStore.isAdmin" value="admin">{{ t('mcp.adminScope') }}</option>
          </select>
          <p id="mcp-scope-hint" class="mt-2 text-xs text-muted-foreground">{{ scope === 'admin' ? t('mcp.adminScopeHint') : t('mcp.userScopeHint') }}</p>
        </div>
        <div>
          <label for="mcp-consent-permissions" class="input-label">{{ t('mcp.permissions') }}</label>
          <select id="mcp-consent-permissions" v-model="readOnly" class="input" :disabled="submitting" aria-describedby="mcp-permissions-hint">
            <option :value="true">{{ t('mcp.readOnly') }}</option>
            <option :value="false">{{ t('mcp.readWrite') }}</option>
          </select>
          <p id="mcp-permissions-hint" class="mt-2 text-xs" :class="readOnly ? 'text-muted-foreground' : 'font-medium text-foreground'">{{ readOnly ? t('mcp.readOnlyHint') : t('mcp.readWriteHint') }}</p>
          <p v-if="scope === 'admin' && !readOnly" data-testid="admin-write-hint" class="mt-1 text-xs font-medium text-foreground">{{ t('mcp.adminWriteHint') }}</p>
        </div>
        <p v-if="submitError" role="alert" class="text-sm text-danger">{{ submitError }}</p>
        <div class="flex flex-wrap justify-end gap-2 border-t border-border pt-4">
          <button data-testid="deny" type="button" class="btn btn-secondary" :disabled="submitting" @click="decide(false)">{{ t('mcp.deny') }}</button>
          <button data-testid="approve" type="submit" class="btn btn-primary" :disabled="submitting">{{ submitting ? t('common.processing') : t('mcp.approve') }}</button>
        </div>
      </form>
    </SectionCard>
    <TotpStepUpDialog :controller="stepUp" />
  </AuthFlowFrame>
</template>

<script setup lang="ts">
import AuthFlowFrame from '@/components/auth/AuthFlowFrame.vue'
import SectionCard from '@/components/page/SectionCard.vue'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { mcpAPI, type MCPAccessScope, type MCPAuthorization, type MCPConsent } from '@/api/mcp'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'
import { Icon } from '@/components/icons'
import { isStepUpBlocked, isStepUpCancelled, stepUpBlockReason, useStepUp } from '@/composables/useStepUp'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const route = useRoute()
const authStore = useAuthStore()
const stepUp = useStepUp()
const loading = ref(true)
const submitting = ref(false)
const previewError = ref('')
const submitError = ref('')
const consent = ref<MCPConsent | null>(null)
const scope = ref<MCPAccessScope>('user')
const readOnly = ref(true)
let authorization: MCPAuthorization | null = null
let generation = 0
const redirectOrigin = computed(() => consent.value ? new URL(consent.value.redirect_uri).origin : '')

function errorMessage(error: unknown, fallback: string): string {
  const message = (error as { message?: unknown } | null)?.message
  return typeof message === 'string' && message ? message : fallback
}

function parseAuthorization(): MCPAuthorization {
  const required = ['response_type', 'client_id', 'redirect_uri', 'code_challenge', 'code_challenge_method', 'resource'] as const
  const allowed = [...required, 'scope', 'state']
  const values: Record<string, string> = {}
  for (const [key, value] of Object.entries(route.query)) {
    if (!allowed.includes(key) || typeof value !== 'string' || value.length > 4096) {
      throw new Error(t('mcp.invalidRequest'))
    }
    values[key] = value
  }
  if (required.some(key => !values[key]) || values.response_type !== 'code' ||
    values.code_challenge_method !== 'S256' || !/^[A-Za-z0-9_-]{43}$/.test(values.code_challenge) ||
    (values.scope !== undefined && values.scope !== '' && values.scope !== 'mcp')) {
    throw new Error(t('mcp.invalidRequest'))
  }
  return values as unknown as MCPAuthorization
}

function validatePreview(result: MCPConsent, request: MCPAuthorization): void {
  if (result.client_id !== request.client_id || result.redirect_uri !== request.redirect_uri ||
    result.resource !== request.resource || result.scope !== 'mcp' || typeof result.client_name !== 'string') {
    throw new Error(t('mcp.invalidRequest'))
  }
  const redirect = new URL(result.redirect_uri)
  if (!['https:', 'http:'].includes(redirect.protocol) || redirect.username || redirect.password || redirect.hash) {
    throw new Error(t('mcp.invalidRequest'))
  }
}

async function loadConsent(): Promise<void> {
  const current = ++generation
  stepUp.onCancel()
  loading.value = true
  submitting.value = false
  previewError.value = ''
  submitError.value = ''
  consent.value = null
  authorization = null
  scope.value = 'user'
  readOnly.value = true
  try {
    const request = parseAuthorization()
    const result = await mcpAPI.previewAuthorization({ ...request })
    if (current !== generation) return
    validatePreview(result, request)
    authorization = Object.freeze({ ...request })
    consent.value = result
  } catch (error) {
    if (current === generation) previewError.value = errorMessage(error, t('mcp.consentLoadFailed'))
  } finally {
    if (current === generation) loading.value = false
  }
}

async function decide(approved: boolean): Promise<void> {
  if (loading.value || submitting.value || !consent.value || !authorization) return
  const current = generation
  const expectedRedirect = new URL(consent.value.redirect_uri)
  const input = {
    authorization: { ...authorization },
    approved,
    scope: approved && authStore.isAdmin ? scope.value : 'user' as const,
    read_only: approved ? readOnly.value : true
  }
  submitting.value = true
  submitError.value = ''
  try {
    const action = async () => {
      // A changed route or unmounted page must not retry a previous consent.
      if (current !== generation) throw new Error(t('mcp.invalidRequest'))
      try {
        return await mcpAPI.authorize(input)
      } catch (error) {
        // Do not open a step-up prompt for a request that is no longer shown.
        if (current !== generation) throw new Error(t('mcp.invalidRequest'))
        throw error
      }
    }
    const result = approved && input.scope === 'admin' ? await stepUp.run(action) : await action()
    if (current !== generation) return
    const destination = new URL(result.redirect_url)
    if (destination.origin !== expectedRedirect.origin || destination.pathname !== expectedRedirect.pathname ||
      destination.username || destination.password || destination.hash) {
      throw new Error(t('mcp.invalidRedirect'))
    }
    window.location.assign(result.redirect_url)
  } catch (error) {
    if (current !== generation) return
    if (!isStepUpCancelled(error)) {
      submitError.value = isStepUpBlocked(error)
        ? stepUpBlockReason(error) === 'STEP_UP_ADMIN_API_KEY_FORBIDDEN' ? t('stepUp.adminApiKeyForbidden') : t('stepUp.notEnabled')
        : errorMessage(error, t('mcp.consentSubmitFailed'))
    }
    submitting.value = false
  }
}

watch(() => route.fullPath, loadConsent, { immediate: true })
onBeforeUnmount(() => {
  generation++
  authorization = null
  stepUp.onCancel()
})
</script>
