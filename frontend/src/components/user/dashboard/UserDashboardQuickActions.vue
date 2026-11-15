<template>
  <SectionCard :title="t('dashboard.quickActions')">
    <div class="space-y-2">
      <button @click="router.push('/keys')" class="group flex w-full items-center gap-3 rounded-md border border-border px-3 py-2 text-left transition-colors hover:bg-muted">
        <div class="flex size-9 shrink-0 items-center justify-center rounded-md bg-muted">
          <Icon name="key" size="lg" class="text-gray-700 dark:text-gray-200" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-foreground">{{ t('dashboard.createApiKey') }}</p>
          <p class="text-xs text-muted-foreground">{{ t('dashboard.generateNewKey') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="md"
          class="text-gray-400 transition-colors group-hover:text-foreground dark:text-dark-500"
        />
      </button>

      <button @click="router.push('/usage')" class="group flex w-full items-center gap-3 rounded-md border border-border px-3 py-2 text-left transition-colors hover:bg-muted">
        <div class="flex size-9 shrink-0 items-center justify-center rounded-md bg-muted">
          <Icon name="chart" size="lg" class="text-gray-700 dark:text-gray-200" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-foreground">{{ t('dashboard.viewUsage') }}</p>
          <p class="text-xs text-muted-foreground">{{ t('dashboard.checkDetailedLogs') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="md"
          class="text-gray-400 transition-colors group-hover:text-foreground dark:text-dark-500"
        />
      </button>

      <button v-if="canUseBatchImage" @click="router.push('/batch-image')" class="group flex w-full items-center gap-3 rounded-md border border-border px-3 py-2 text-left transition-colors hover:bg-muted">
        <div class="flex size-9 shrink-0 items-center justify-center rounded-md bg-muted">
          <Icon name="sparkles" size="lg" class="text-gray-700 dark:text-gray-200" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-foreground">{{ t('dashboard.batchImageAgent') }}</p>
          <p class="text-xs text-muted-foreground">{{ t('dashboard.batchImageAgentDesc') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="md"
          class="text-gray-400 transition-colors group-hover:text-foreground dark:text-dark-500"
        />
      </button>

      <button @click="router.push('/redeem')" class="group flex w-full items-center gap-3 rounded-md border border-border px-3 py-2 text-left transition-colors hover:bg-muted">
        <div class="flex size-9 shrink-0 items-center justify-center rounded-md bg-muted">
          <Icon name="gift" size="lg" class="text-gray-700 dark:text-gray-200" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-foreground">{{ t('dashboard.redeemCode') }}</p>
          <p class="text-xs text-muted-foreground">{{ t('dashboard.addBalanceWithCode') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="md"
          class="text-gray-400 transition-colors group-hover:text-foreground dark:text-dark-500"
        />
      </button>
    </div>
  </SectionCard>
</template>

<script setup lang="ts">
import { SectionCard } from '@/components/page'
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
const router = useRouter()
const { t } = useI18n()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()

onMounted(() => {
  void refreshBatchImageAccess()
})
</script>
