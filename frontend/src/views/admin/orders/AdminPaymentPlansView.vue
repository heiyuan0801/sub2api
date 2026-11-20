<template>
  <AppLayout>
    <div class="space-y-4">
      <!-- Page Header & Actions -->
      <PageHeader
        :title="t('payment.admin.plansPageTitle')"
        :description="t('payment.admin.plansPageDesc')"
      >
        <template #actions>
          <RouterLink to="/admin/orders/plans/prices" class="btn btn-secondary h-9">
            {{ t('planOps.title') }}
          </RouterLink>
          <button
            type="button"
            class="btn btn-secondary h-9"
            :disabled="plansLoading || plans.length < 2"
            @click="openSort"
          >
            {{ t('planOps.sort') }}
          </button>
          <button
            type="button"
            class="btn btn-secondary btn-icon h-9 w-9"
            :disabled="plansLoading"
            :title="t('common.refresh')"
            @click="loadPlans"
          >
            <Icon name="refresh" size="sm" :class="plansLoading ? 'animate-spin' : ''" />
          </button>
          <button type="button" class="btn btn-primary h-9" @click="openPlanEdit(null)">
            <Icon name="plus" size="sm" />
            <span>{{ t('payment.admin.createPlan') }}</span>
          </button>
        </template>
      </PageHeader>

      <!-- Plans Table -->
      <div class="lg:overflow-hidden lg:rounded-2xl lg:border lg:border-border/80 lg:bg-card lg:shadow-sm">
        <DataTable :columns="planColumns" :data="plans" :loading="plansLoading">
          <template #cell-id="{ value }">
            <span class="font-mono text-xs tabular-nums text-muted-foreground">#{{ value }}</span>
          </template>
          <template #cell-name="{ value, row }">
            <span class="text-sm font-medium" :class="getPlanNameClass(row.group_id)">{{ value }}</span>
            <div v-if="row.display_tags?.length" class="mt-1 flex flex-wrap gap-1">
              <span
                v-for="tag in row.display_tags"
                :key="tag"
                class="badge border border-blue-500/20 bg-blue-500/10 text-blue-700 dark:text-blue-400"
              >
                {{ t('planOps.tags.' + tag) }}
              </span>
            </div>
          </template>
          <template #cell-visibility_scope="{ row }">
            <span class="text-sm text-foreground">{{ t('planOps.visibilityModes.' + (row.visibility_scope || 'all')) }}</span>
          </template>
          <template #cell-group_id="{ value }">
            <span v-if="isGroupMissing(value)" class="inline-flex items-center gap-1.5 text-sm">
              <span class="font-mono text-xs tabular-nums text-muted-foreground">#{{ value }}</span>
              <span class="badge badge-danger">{{ t('payment.admin.groupMissing') }}</span>
            </span>
            <GroupBadge
              v-else-if="getGroup(value)"
              :name="getGroup(value)!.name"
              :platform="getGroup(value)!.platform"
              :rate-multiplier="getGroup(value)!.rate_multiplier"
            />
            <span v-else class="text-sm text-muted-foreground/60">-</span>
          </template>
          <template #cell-price="{ value, row }">
            <div class="text-sm tabular-nums">
              <span class="font-semibold text-foreground">{{ planCurrencySymbol(row.currency) }}{{ (value ?? 0).toFixed(2) }}</span><span v-if="row.currency" class="ml-1 text-xs font-medium text-muted-foreground">{{ row.currency }}</span>
              <span v-if="row.original_price" class="ml-1.5 text-xs text-muted-foreground/70 line-through">{{ planCurrencySymbol(row.currency) }}{{ row.original_price.toFixed(2) }}</span>
            </div>
          </template>
          <template #cell-validity_days="{ value, row }">
            <span class="text-sm tabular-nums text-foreground">{{ value }} {{ t('payment.admin.' + (row.validity_unit || 'days')) }}</span>
          </template>
          <template #cell-for_sale="{ value, row }">
            <button
              type="button"
              role="switch"
              :aria-checked="value"
              :aria-label="t('payment.admin.forSale')"
              :disabled="planChanging === row.id"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background disabled:opacity-50',
                value ? 'bg-foreground' : 'bg-input'
              ]"
              @click="toggleForSale(row)"
            >
              <span :class="[
                'pointer-events-none inline-block h-4 w-4 transform rounded-full shadow-sm ring-0 transition duration-200 ease-in-out',
                value ? 'translate-x-4 bg-background' : 'translate-x-0 bg-background dark:bg-foreground/90'
              ]" />
            </button>
          </template>
          <template #cell-sort_order="{ value }">
            <span class="font-mono text-xs tabular-nums text-muted-foreground">{{ value }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1.5">
              <RouterLink :to="{ path: '/admin/orders/plans/prices', query: { plan_id: row.id, adjust: '1' } }" class="btn btn-secondary btn-sm">
                {{ t('planOps.adjust') }}
              </RouterLink>
              <button
                type="button"
                class="inline-flex h-8 items-center gap-1 rounded-md px-2 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                @click="openPlanEdit(row)"
              >
                <Icon name="edit" size="sm" />
                <span>{{ t('common.edit') }}</span>
              </button>
              <button
                type="button"
                class="inline-flex h-8 items-center gap-1 rounded-md px-2 text-xs font-medium text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
                @click="confirmDeletePlan(row)"
              >
                <Icon name="trash" size="sm" />
                <span>{{ t('common.delete') }}</span>
              </button>
            </div>
          </template>
        </DataTable>
      </div>
    </div>

    <!-- Plan Edit Dialog -->
    <PlanEditDialog :show="showPlanDialog" :plan="editingPlan" :groups="groups" :payment-config="paymentConfig" @close="showPlanDialog = false" @saved="loadPlans" />

    <ConfirmDialog :show="showDeletePlanDialog" :title="t('payment.admin.deletePlan')" :message="t('payment.admin.deletePlanConfirm')" :confirm-text="t('common.delete')" danger @confirm="handleDeletePlan" @cancel="showDeletePlanDialog = false" />
    <BaseDialog :show="Boolean(unlistingPlan)" :title="t('planOps.unsale')" @close="closeUnsale">
      <p class="text-sm text-foreground">
        <span class="font-medium">{{ unlistingPlan?.name }}</span>
        <span class="text-muted-foreground"> · {{ t('planOps.unsaleHint') }}</span>
      </p>
      <label class="mt-3.5 flex cursor-pointer items-center gap-2 rounded-lg border border-border/80 bg-muted/30 px-3 py-2.5 text-sm text-foreground">
        <input v-model="notifyUnlisting" type="checkbox" class="h-4 w-4 rounded border-input" />
        {{ t('planOps.notifyDraft') }}
      </label>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary" :disabled="planChanging != null" @click="closeUnsale">{{ t('common.cancel') }}</button>
          <button type="button" class="btn btn-primary" :disabled="planChanging != null" @click="confirmUnsale">{{ t('common.confirm') }}</button>
        </div>
      </template>
    </BaseDialog>
    <BaseDialog :show="showSort" :title="t('planOps.sort')" @close="closeSort">
      <p class="mb-3 text-sm text-muted-foreground">{{ t('planOps.sortHint') }}</p>
      <VueDraggable v-model="orderDraft" handle=".plan-drag" :animation="150" :disabled="sortSaving" class="space-y-2">
        <div
          v-for="(item, index) in orderDraft"
          :key="item.id"
          class="flex items-center gap-2 rounded-xl border border-border/80 bg-muted/20 px-2.5 py-2 transition-colors hover:bg-muted/40"
        >
          <span class="plan-drag cursor-grab px-1.5 py-1 text-muted-foreground hover:text-foreground" aria-hidden="true">⠿</span>
          <span class="min-w-0 flex-1 break-words text-sm font-medium text-foreground">{{ item.name }}</span>
          <button type="button" class="btn btn-secondary btn-sm" :aria-label="t('planOps.moveUp')" :disabled="index === 0 || sortSaving" @click="moveRow(index, -1)">
            <Icon name="chevronUp" size="xs" />
          </button>
          <button type="button" class="btn btn-secondary btn-sm" :aria-label="t('planOps.moveDown')" :disabled="index === orderDraft.length - 1 || sortSaving" @click="moveRow(index, 1)">
            <Icon name="chevronDown" size="xs" />
          </button>
        </div>
      </VueDraggable>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary" :disabled="sortSaving" @click="closeSort">{{ t('common.cancel') }}</button>
          <button type="button" class="btn btn-primary" :disabled="sortSaving" @click="saveSort">{{ t('common.save') }}</button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink, useRouter } from 'vue-router'
import { VueDraggable } from 'vue-draggable-plus'
import { useAppStore } from '@/stores/app'
import { adminPaymentAPI } from '@/api/admin/payment'
import type { AdminPaymentConfig } from '@/api/admin/payment'
import { extractI18nErrorMessage } from '@/utils/apiError'
import adminAPI from '@/api/admin'
import type { SubscriptionPlan } from '@/types/payment'
import type { AdminGroup } from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import DataTable from '@/components/common/DataTable.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import PlanEditDialog from './PlanEditDialog.vue'
import { currencySymbol } from '@/components/payment/currency'
import { platformTextClass } from '@/utils/platformColors'

const { t } = useI18n()
const appStore = useAppStore()
const router = useRouter()

function planCurrencySymbol(currency?: string): string {
  return currencySymbol(currency || 'USD')
}

// ==================== Groups ====================

const groups = ref<AdminGroup[]>([])
const paymentConfig = ref<AdminPaymentConfig | null>(null)

async function loadGroups() {
  try {
    groups.value = await adminAPI.groups.getAll()
  } catch { /* ignore */ }
}

async function loadPaymentConfig() {
  try {
    const res = await adminPaymentAPI.getConfig()
    paymentConfig.value = res.data
  } catch { /* preview only */ }
}

function getGroup(id: number): AdminGroup | undefined {
  return groups.value.find(g => g.id === id)
}

function isGroupMissing(id: number): boolean {
  return id > 0 && !groups.value.find(g => g.id === id)
}

function getPlanNameClass(groupId: number): string {
  const group = getGroup(groupId)
  return group ? platformTextClass(group.platform) : 'text-foreground'
}


// ==================== Plans ====================

const plansLoading = ref(false)
const plans = ref<SubscriptionPlan[]>([])
const showPlanDialog = ref(false)
const showDeletePlanDialog = ref(false)
const editingPlan = ref<SubscriptionPlan | null>(null)
const deletingPlanId = ref<number | null>(null)
const planChanging = ref<number | null>(null)
const unlistingPlan = ref<SubscriptionPlan | null>(null)
const notifyUnlisting = ref(false)
const showSort = ref(false)
const orderDraft = ref<SubscriptionPlan[]>([])
const sortSaving = ref(false)

const planColumns = computed((): Column[] => [
  { key: 'id', label: 'ID' },
  { key: 'name', label: t('payment.admin.planName') },
  { key: 'group_id', label: t('payment.admin.group') },
  { key: 'price', label: t('payment.admin.price') },
  { key: 'validity_days', label: t('payment.admin.validity') },
  { key: 'for_sale', label: t('payment.admin.forSale') },
  { key: 'visibility_scope', label: t('planOps.visibility') },
  { key: 'sort_order', label: t('payment.admin.sortOrder') },
  { key: 'actions', label: t('common.actions') },
])

async function loadPlans() {
  plansLoading.value = true
  try {
    const res = await adminPaymentAPI.getPlans()
    // Backend returns features as newline-separated string; parse to array
    plans.value = (res.data || []).map((p: Omit<SubscriptionPlan, 'features'> & { features: string | string[] }) => ({
      ...p,
      features: typeof p.features === 'string'
        ? p.features.split('\n').map((f: string) => f.trim()).filter(Boolean)
        : (p.features || []),
    }))
  }
  catch (err: unknown) { appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error'))) }
  finally { plansLoading.value = false }
}

function openPlanEdit(plan: SubscriptionPlan | null) {
  editingPlan.value = plan
  showPlanDialog.value = true
}


function toggleForSale(plan: SubscriptionPlan) {
  if (planChanging.value != null) return
  if (plan.for_sale) {
    notifyUnlisting.value = false
    unlistingPlan.value = plan
  } else {
    void setForSale(plan, true)
  }
}

async function setForSale(plan: SubscriptionPlan, value: boolean) {
  planChanging.value = plan.id
  try {
    await adminPaymentAPI.updatePlan(plan.id, { for_sale: value })
    plan.for_sale = value
    return true
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
    return false
  } finally {
    planChanging.value = null
  }
}

function closeUnsale() {
  if (planChanging.value == null) unlistingPlan.value = null
}

async function confirmUnsale() {
  const plan = unlistingPlan.value
  if (!plan || planChanging.value != null || !await setForSale(plan, false)) return
  closeUnsale()
  if (notifyUnlisting.value) {
    await router.push({ path: '/admin/announcements', query: { compose: '1', kind: 'plan_unlisted', plan_id: plan.id } })
  }
}

function openSort() {
  orderDraft.value = [...plans.value]
  showSort.value = true
}

function closeSort() {
  if (!sortSaving.value) showSort.value = false
}

function moveRow(index: number, offset: number) {
  const [item] = orderDraft.value.splice(index, 1)
  orderDraft.value.splice(index + offset, 0, item)
}

async function saveSort() {
  if (sortSaving.value) return
  sortSaving.value = true
  try {
    await adminPaymentAPI.reorderPlans(orderDraft.value.map(item => item.id))
    showSort.value = false
    await loadPlans()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    sortSaving.value = false
  }
}

function confirmDeletePlan(plan: SubscriptionPlan) { deletingPlanId.value = plan.id; showDeletePlanDialog.value = true }
async function handleDeletePlan() {
  if (!deletingPlanId.value) return
  try { await adminPaymentAPI.deletePlan(deletingPlanId.value); appStore.showSuccess(t('common.deleted')); showDeletePlanDialog.value = false; loadPlans() }
  catch (err: unknown) { appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error'))) }
}

// ==================== Lifecycle ====================

onMounted(() => {
  loadGroups()
  loadPaymentConfig()
  loadPlans()
})
</script>
