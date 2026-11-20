<template>
  <BaseDialog :show="show" :title="plan ? t('payment.admin.editPlan') : t('payment.admin.createPlan')" width="wide" @close="emit('close')">
    <form id="plan-form" @submit.prevent="handleSavePlan" class="space-y-5">
      <!-- Basic Info & Group -->
      <div class="space-y-4">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">{{ t('payment.admin.planName') }} <span class="text-destructive">*</span></label>
            <input v-model="planForm.name" type="text" class="input h-9" required />
          </div>
          <div>
            <label class="input-label">{{ t('payment.admin.group') }} <span class="text-destructive">*</span></label>
            <Select v-model="planForm.group_id" :options="groupOptions" :placeholder="t('payment.admin.selectGroup')" class="w-full">
              <template #selected="{ option }">
                <span v-if="option?.platform" :class="platformTextClass(String(option.platform))">{{ option.label }}</span>
                <span v-else>{{ option?.label || t('payment.admin.selectGroup') }}</span>
              </template>
              <template #option="{ option, selected }">
                <span class="flex-1 truncate text-left" :class="option.platform ? platformTextClass(String(option.platform)) : ''">{{ option.label }}</span>
                <Icon v-if="selected" name="check" size="sm" class="text-primary" :stroke-width="2" />
              </template>
            </Select>
          </div>
        </div>

        <!-- Group Info Preview -->
        <div v-if="selectedGroupInfo" class="rounded-xl border border-border/80 bg-muted/30 p-3.5">
          <div class="mb-2.5 flex items-center gap-2">
            <GroupBadge :name="selectedGroupInfo.name" :platform="selectedGroupInfo.platform" :rate-multiplier="selectedGroupInfo.rate_multiplier" />
          </div>
          <div class="grid grid-cols-2 gap-2 text-xs sm:grid-cols-3">
            <div>
              <span class="text-muted-foreground">{{ t('payment.admin.dailyLimit') }}:</span>
              <span class="ml-1 font-medium tabular-nums text-foreground">{{ selectedGroupInfo.daily_limit_usd != null ? '$' + selectedGroupInfo.daily_limit_usd : t('payment.admin.unlimited') }}</span>
            </div>
            <div>
              <span class="text-muted-foreground">{{ t('payment.admin.weeklyLimit') }}:</span>
              <span class="ml-1 font-medium tabular-nums text-foreground">{{ selectedGroupInfo.weekly_limit_usd != null ? '$' + selectedGroupInfo.weekly_limit_usd : t('payment.admin.unlimited') }}</span>
            </div>
            <div>
              <span class="text-muted-foreground">{{ t('payment.admin.monthlyLimit') }}:</span>
              <span class="ml-1 font-medium tabular-nums text-foreground">{{ selectedGroupInfo.monthly_limit_usd != null ? '$' + selectedGroupInfo.monthly_limit_usd : t('payment.admin.unlimited') }}</span>
            </div>
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('payment.admin.planDescription') }} <span class="text-destructive">*</span></label>
          <textarea v-model="planForm.description" rows="2" class="input h-auto min-h-[4.5rem] py-2" required></textarea>
        </div>
      </div>

      <!-- Pricing & Validity -->
      <div class="space-y-4 border-t border-border pt-4">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">{{ t('payment.admin.price') }} <span class="text-destructive">*</span></label>
            <input v-model.number="planForm.price" type="number" step="0.01" min="0.01" class="input h-9 tabular-nums read-only:bg-muted/50 read-only:text-muted-foreground" :readonly="Boolean(plan)" required />
            <p v-if="plan" class="mt-1.5 text-xs text-muted-foreground">{{ t('planOps.priceEditHint') }}</p>
            <p v-if="subscriptionCnyPreview" class="mt-1.5 rounded-md border border-border/80 bg-muted/40 px-2.5 py-1 text-xs font-medium tabular-nums text-foreground">
              {{ t('payment.admin.subscriptionCnyPayPreview', { amount: subscriptionCnyPreview.amount }) }}
              <span v-if="subscriptionCnyPreview.feeRate > 0" class="text-muted-foreground">
                {{ t('payment.admin.subscriptionCnyPayPreviewWithFee', { feeRate: subscriptionCnyPreview.feeRate, total: subscriptionCnyPreview.total }) }}
              </span>
            </p>
          </div>
          <div>
            <label class="input-label">{{ t('payment.admin.originalPrice') }}</label>
            <input v-model.number="planForm.original_price" type="number" step="0.01" min="0" class="input h-9 tabular-nums" />
          </div>
        </div>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">{{ t('payment.admin.validity') }} <span class="text-destructive">*</span></label>
            <input v-model.number="planForm.validity_days" type="number" min="1" class="input h-9 tabular-nums" required />
          </div>
          <div>
            <label class="input-label">{{ t('payment.admin.validityUnit') }} <span class="text-destructive">*</span></label>
            <Select v-model="planForm.validity_unit" :options="validityUnitOptions" />
          </div>
        </div>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">{{ t('payment.admin.sortOrder') }}</label>
            <input v-model.number="planForm.sort_order" type="number" min="0" class="input h-9 tabular-nums" />
          </div>
          <div>
            <label class="input-label">{{ t('payment.admin.currency') }}</label>
            <input v-model="planForm.currency" type="text" maxlength="3" class="input h-9 uppercase" :placeholder="t('payment.admin.currencyPlaceholder')" />
            <p class="mt-1 text-xs text-muted-foreground">{{ t('payment.admin.currencyHint') }}</p>
          </div>
        </div>
      </div>

      <!-- Features & Display Tags -->
      <div class="space-y-4 border-t border-border pt-4">
        <div>
          <label class="input-label">{{ t('payment.admin.features') }}</label>
          <textarea v-model="planFeaturesText" rows="3" class="input h-auto min-h-[5.5rem] py-2" :placeholder="t('payment.admin.featuresPlaceholder')"></textarea>
          <p class="mt-1 text-xs text-muted-foreground">{{ t('payment.admin.featuresHint') }}</p>
        </div>
        <fieldset>
          <legend class="input-label">{{ t('planOps.displayTags') }}</legend>
          <div class="flex flex-wrap gap-2">
            <label
              v-for="tag in subscriptionPlanTags"
              :key="tag"
              class="inline-flex cursor-pointer items-center gap-2 rounded-lg border border-border/80 bg-muted/20 px-3 py-1.5 text-sm text-foreground transition-colors hover:bg-muted/50"
            >
              <input v-model="tags" type="checkbox" :value="tag" class="h-4 w-4 rounded border-input" />
              {{ t('planOps.tags.' + tag) }}
            </label>
          </div>
        </fieldset>
      </div>

      <!-- Visibility & Publishing -->
      <div class="space-y-4 border-t border-border pt-4">
        <div class="space-y-2.5">
          <label class="block">
            <span class="input-label">{{ t('planOps.visibility') }}</span>
            <select v-model="visibility" class="input h-9">
              <option v-for="mode in visibilityModes" :key="mode" :value="mode">{{ t('planOps.visibilityModes.' + mode) }}</option>
            </select>
          </label>
          <label v-if="visibility === 'users'" class="block">
            <span class="input-label">{{ t('planOps.userIDs') }}</span>
            <input v-model="userIds" type="text" class="input h-9 font-mono" placeholder="123, 456" />
          </label>
          <div v-if="visibility === 'groups'">
            <p class="mb-2 text-xs text-muted-foreground">{{ t('planOps.groupHint') }}</p>
            <div class="grid max-h-36 gap-1 overflow-y-auto rounded-xl border border-border/80 bg-muted/20 p-2.5 sm:grid-cols-2">
              <label
                v-for="group in groups"
                :key="group.id"
                class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1 text-sm text-foreground transition-colors hover:bg-muted/60"
              >
                <input v-model="visibilityGroupIds" type="checkbox" :value="group.id" class="h-4 w-4 rounded border-input" />
                <span class="truncate">{{ group.name }}</span>
                <span class="font-mono text-xs tabular-nums text-muted-foreground">(#{{ group.id }})</span>
              </label>
            </div>
          </div>
        </div>

        <div class="flex items-center justify-between rounded-xl border border-border/80 bg-muted/30 px-3.5 py-2.5">
          <label class="text-sm font-medium text-foreground">{{ t('payment.admin.forSale') }}</label>
          <button
            type="button"
            role="switch"
            :aria-checked="planForm.for_sale"
            :aria-label="t('payment.admin.forSale')"
            :class="[
              'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background',
              planForm.for_sale ? 'bg-foreground' : 'bg-input'
            ]"
            @click="planForm.for_sale = !planForm.for_sale"
          >
            <span :class="[
              'pointer-events-none inline-block h-5 w-5 transform rounded-full shadow-sm ring-0 transition duration-200 ease-in-out',
              planForm.for_sale ? 'translate-x-5 bg-background' : 'translate-x-0 bg-background dark:bg-foreground/90'
            ]" />
          </button>
        </div>

        <label v-if="!plan || (plan.for_sale && !planForm.for_sale)" class="flex cursor-pointer items-center gap-2 text-sm text-foreground">
          <input v-model="notify" type="checkbox" class="h-4 w-4 rounded border-input" />
          {{ t('planOps.notifyDraft') }}
        </label>
      </div>
    </form>
    <template #footer>
      <div class="flex justify-end gap-2.5">
        <button type="button" @click="emit('close')" class="btn btn-secondary">{{ t('common.cancel') }}</button>
        <button type="submit" form="plan-form" :disabled="saving" class="btn btn-primary">{{ saving ? t('common.saving') : t('common.save') }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { adminPaymentAPI } from '@/api/admin/payment'
import type { AdminPaymentConfig } from '@/api/admin/payment'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatPaymentAmount } from '@/components/payment/currency'
import { subscriptionPlanTags, type SubscriptionPlan, type SubscriptionPlanTag, type PlanVisibilityScope } from '@/types/payment'
import type { AdminGroup } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import { platformTextClass } from '@/utils/platformColors'

const props = defineProps<{
  show: boolean
  plan: SubscriptionPlan | null
  groups: AdminGroup[]
  paymentConfig?: AdminPaymentConfig | null
}>()

const emit = defineEmits<{
  close: []
  saved: []
}>()

const { t } = useI18n()
const appStore = useAppStore()
const router = useRouter()

const saving = ref(false)
const planForm = reactive({ name: '', group_id: null as number | null, description: '', price: 0, original_price: 0, currency: '', validity_days: 30, validity_unit: 'days', sort_order: 0, for_sale: true })
const planFeaturesText = ref('')
const tags = ref<SubscriptionPlanTag[]>([])
const visibilityModes = ['all', 'users', 'groups'] as const
const visibility = ref<PlanVisibilityScope>('all')
const userIds = ref('')
const visibilityGroupIds = ref<number[]>([])
const notify = ref(false)

const validityUnitOptions = computed(() => [
  { value: 'days', label: t('payment.admin.days') },
  { value: 'weeks', label: t('payment.admin.weeks') },
  { value: 'months', label: t('payment.admin.months') },
])

const groupOptions = computed(() =>
  props.groups
    .filter(g => g.subscription_type === 'subscription')
    .map(g => ({
      value: g.id,
      label: `${g.name} — ${g.platform} (${g.rate_multiplier}x)`,
      platform: g.platform,
    })),
)

const selectedGroupInfo = computed(() => {
  if (!planForm.group_id) return null
  return props.groups.find(g => g.id === planForm.group_id) || null
})

function roundCnyAmount(value: number): number {
  return Math.round(value * 100) / 100
}

function ceilCnyAmount(value: number): number {
  return Math.ceil(value * 100) / 100
}

const subscriptionCnyPreview = computed(() => {
  const price = Number(planForm.price) || 0
  const rate = Number(props.paymentConfig?.subscription_usd_to_cny_rate) || 0
  if (price <= 0 || rate <= 0) return null

  const amount = roundCnyAmount(price * rate)
  const feeRate = Number(props.paymentConfig?.recharge_fee_rate) || 0
  const fee = feeRate > 0 ? ceilCnyAmount((amount * feeRate) / 100) : 0
  const total = feeRate > 0 ? roundCnyAmount(amount + fee) : amount

  return {
    amount: formatPaymentAmount(amount, 'CNY'),
    feeRate,
    total: formatPaymentAmount(total, 'CNY'),
  }
})

// Reset form when dialog opens
watch(() => props.show, (visible) => {
  if (!visible) return
  tags.value = [...(props.plan?.display_tags ?? [])]
  visibility.value = props.plan?.visibility_scope ?? 'all'
  userIds.value = props.plan?.visibility_user_ids?.join(', ') ?? ''
  visibilityGroupIds.value = [...(props.plan?.visibility_group_ids ?? [])]
  notify.value = false
  if (props.plan) {
    Object.assign(planForm, { name: props.plan.name, group_id: props.plan.group_id, description: props.plan.description, price: props.plan.price, original_price: props.plan.original_price || 0, currency: props.plan.currency || '', validity_days: props.plan.validity_days, validity_unit: props.plan.validity_unit || 'days', sort_order: props.plan.sort_order || 0, for_sale: props.plan.for_sale })
    planFeaturesText.value = (props.plan.features || []).join('\n')
  } else {
    Object.assign(planForm, { name: '', group_id: null, description: '', price: 0, original_price: 0, currency: '', validity_days: 30, validity_unit: 'days', sort_order: 0, for_sale: true })
    planFeaturesText.value = ''
  }
})

/** Build request payload with snake_case keys matching backend JSON tags */
function buildPlanPayload() {
  const features = planFeaturesText.value.split('\n').map(f => f.trim()).filter(Boolean).join('\n')
  return {
    name: planForm.name,
    group_id: planForm.group_id,
    description: planForm.description,
    ...(props.plan ? {} : { price: planForm.price }),
    original_price: planForm.original_price || 0,
    currency: planForm.currency.trim().toUpperCase(),
    validity_days: planForm.validity_days,
    validity_unit: planForm.validity_unit,
    sort_order: planForm.sort_order,
    for_sale: planForm.for_sale,
    features,
    display_tags: tags.value,
    visibility_scope: visibility.value,
    visibility_user_ids: visibility.value === 'users' ? parseUserIds() : [],
    visibility_group_ids: visibility.value === 'groups' ? visibilityGroupIds.value : [],
  }
}

function parseUserIds() {
  return [...new Set(userIds.value.split(/[,，\s]+/).filter(Boolean).map(Number))]
}

async function handleSavePlan() {
  if (!planForm.group_id) {
    appStore.showError(t('payment.admin.groupRequired'))
    return
  }
  if (!planForm.price || planForm.price <= 0) {
    appStore.showError(t('payment.admin.priceRequired'))
    return
  }
  if (!planForm.validity_days || planForm.validity_days < 1) {
    appStore.showError(t('payment.admin.validityRequired'))
    return
  }
  const audienceIds = visibility.value === 'users' ? parseUserIds() : visibilityGroupIds.value
  if (visibility.value !== 'all' && (!audienceIds.length || audienceIds.some(id => !Number.isSafeInteger(id) || id <= 0))) {
    appStore.showError(t('planOps.audienceRequired'))
    return
  }
  saving.value = true
  try {
    const data = buildPlanPayload()
    const existing = props.plan
    const result = existing ? await adminPaymentAPI.updatePlan(existing.id, data) : await adminPaymentAPI.createPlan(data)
    appStore.showSuccess(t('common.saved'))
    emit('close')
    emit('saved')
    if (notify.value && (!existing || (existing.for_sale && !planForm.for_sale))) {
      await router.push({
        path: '/admin/announcements',
        query: { compose: '1', kind: existing ? 'plan_unlisted' : 'new_plan', plan_id: result.data.id },
      })
    }
  } catch (err: unknown) { appStore.showError(extractApiErrorMessage(err, t('common.error'))) }
  finally { saving.value = false }
}
</script>
