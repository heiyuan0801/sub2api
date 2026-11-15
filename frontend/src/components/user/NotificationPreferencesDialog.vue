<template>
  <BaseDialog :show="show" :title="t('operations.notificationPreferences')" @close="emit('close')">
    <p class="mb-4 text-sm text-muted-foreground">{{ t('operations.notificationPreferencesHint') }}</p>
    <p v-if="partiallyLoaded && !loading && !failed" class="mb-3 text-sm text-muted-foreground" role="status">
      {{ t('operations.groupPreferencesPartial') }}
      <button type="button" class="ml-2 font-medium underline" @click="load">{{ t('common.refresh') }}</button>
    </p>
    <div v-if="loading" class="grid min-h-24 place-items-center"><LoadingSpinner /></div>
    <p v-else-if="failed" class="text-sm text-muted-foreground">
      {{ t('common.error') }}
      <button type="button" class="ml-2 font-medium underline" @click="load">{{ t('common.refresh') }}</button>
    </p>
    <p v-else-if="!groups.length" class="rounded-lg border border-border p-4 text-sm text-muted-foreground">{{ t('operations.noAvailableGroups') }}</p>
    <ul v-else class="max-h-80 divide-y divide-border overflow-y-auto rounded-lg border border-border">
      <li v-for="group in groups" :key="group.id" class="flex flex-wrap items-center justify-between gap-3 p-3.5">
        <div class="min-w-0">
          <p class="break-words text-sm font-medium">{{ group.name }}</p>
          <div class="mt-1 flex flex-wrap items-center gap-2">
            <span v-if="group.type" class="badge gap-1" :class="group.type === 'standard' ? 'badge-success' : 'badge-primary'">
              <Icon :name="group.type === 'standard' ? 'creditCard' : 'gift'" size="xs" aria-hidden="true" />
              {{ t(group.type === 'standard' ? 'operations.standardBalanceGroup' : 'operations.subscriptionGroup') }}
            </span>
            <span class="text-xs text-muted-foreground">{{ t(group.subscribed ? 'operations.subscribed' : 'operations.unsubscribed') }}</span>
          </div>
        </div>
        <button
          type="button"
          role="switch"
          :aria-label="group.name"
          :aria-checked="group.subscribed"
          :disabled="updating.includes(group.id)"
          class="relative h-6 w-11 shrink-0 rounded-full border-2 border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 disabled:opacity-50"
          :class="group.subscribed ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'"
          @click="toggle(group)"
        >
          <span class="pointer-events-none block h-5 w-5 rounded-full bg-white shadow transition-transform" :class="{ 'translate-x-5': group.subscribed }" />
        </button>
      </li>
    </ul>
    <template #footer>
      <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.close') }}</button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import announcementsAPI from '@/api/announcements'
import userGroupsAPI from '@/api/groups'
import { paymentAPI } from '@/api/payment'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { useAppStore } from '@/stores/app'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t, locale } = useI18n()
const app = useAppStore()
const subscriptions = useSubscriptionStore()
const loading = ref(false)
const failed = ref(false)
const partiallyLoaded = ref(false)
const availableGroups = ref(new Map<number, { name: string; type: 'standard' | 'subscription' }>())
const muted = ref<number[]>([])
const updating = ref<number[]>([])
let generation = 0

const groups = computed(() => [...new Set([...availableGroups.value.keys(), ...muted.value])].map(id => ({
  id,
  name: availableGroups.value.get(id)?.name || `${t('operations.notificationGroup')} #${id}`,
  type: availableGroups.value.get(id)?.type,
  subscribed: !muted.value.includes(id),
})).sort((first, second) => first.name.localeCompare(second.name, locale.value)))

async function load() {
  const current = ++generation
  loading.value = true
  failed.value = false
  partiallyLoaded.value = false
  try {
    const [prefs, standard, plans, active] = await Promise.allSettled([
      announcementsAPI.preferences(),
      userGroupsAPI.getAvailable(),
      isFeatureFlagEnabled(FeatureFlags.payment) ? paymentAPI.getPlans() : Promise.resolve({ data: [] }),
      isFeatureFlagEnabled(FeatureFlags.subscription) ? subscriptions.fetchActiveSubscriptions() : Promise.resolve([]),
    ])
    if (current !== generation) return
    if (prefs.status === 'rejected') throw prefs.reason
    partiallyLoaded.value = [standard, plans, active].some(result => result.status === 'rejected')
    const available = new Map<number, { name: string; type: 'standard' | 'subscription' }>()
    if (standard.status === 'fulfilled') {
      for (const group of standard.value) {
        if (group.subscription_type === 'standard') available.set(group.id, { name: group.name, type: 'standard' })
      }
    }
    if (plans.status === 'fulfilled') {
      for (const plan of plans.value.data) {
        available.set(plan.group_id, { name: plan.group_name || `${t('operations.subscriptionGroup')} #${plan.group_id}`, type: 'subscription' })
      }
    }
    if (active.status === 'fulfilled') {
      for (const item of active.value) {
        const name = item.group?.name || available.get(item.group_id)?.name || `${t('operations.subscriptionGroup')} #${item.group_id}`
        available.set(item.group_id, { name, type: 'subscription' })
      }
    }
    availableGroups.value = available
    muted.value = prefs.value.muted_group_ids ?? []
  } catch (error) {
    if (current !== generation) return
    failed.value = true
    app.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    if (current === generation) loading.value = false
  }
}

async function toggle(group: { id: number; subscribed: boolean }) {
  if (updating.value.includes(group.id)) return
  updating.value.push(group.id)
  try {
    const result = await announcementsAPI.setGroupSubscription(group.id, !group.subscribed)
    muted.value = result.subscribed
      ? muted.value.filter(id => id !== group.id)
      : [...new Set([...muted.value, group.id])]
  } catch (error) {
    app.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    updating.value = updating.value.filter(id => id !== group.id)
  }
}
watch(() => props.show, show => {
  if (show) void load()
  else generation++
}, { immediate: true })
</script>
