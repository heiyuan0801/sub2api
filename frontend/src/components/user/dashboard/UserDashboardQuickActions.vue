<template>
  <Card class="overflow-hidden border-border">
    <div class="border-b border-border px-5 py-3.5">
      <h2 class="text-sm font-semibold text-foreground">{{ t('dashboard.quickActions') }}</h2>
    </div>
    <div class="space-y-2.5 p-4">
      <button
        type="button"
        class="group flex w-full items-center gap-3 rounded-lg border border-border bg-card p-3 text-left transition-colors hover:bg-accent/50"
        @click="router.push('/keys')"
      >
        <div class="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted/50 text-foreground transition-colors group-hover:bg-background">
          <Icon name="key" size="sm" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm font-medium text-foreground">{{ t('dashboard.createApiKey') }}</p>
          <p class="truncate text-xs text-muted-foreground">{{ t('dashboard.generateNewKey') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="sm"
          class="shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5 group-hover:text-foreground"
        />
      </button>

      <button
        type="button"
        class="group flex w-full items-center gap-3 rounded-lg border border-border bg-card p-3 text-left transition-colors hover:bg-accent/50"
        @click="router.push('/usage')"
      >
        <div class="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted/50 text-foreground transition-colors group-hover:bg-background">
          <Icon name="chart" size="sm" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm font-medium text-foreground">{{ t('dashboard.viewUsage') }}</p>
          <p class="truncate text-xs text-muted-foreground">{{ t('dashboard.checkDetailedLogs') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="sm"
          class="shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5 group-hover:text-foreground"
        />
      </button>

      <button
        v-if="canUseBatchImage"
        type="button"
        class="group flex w-full items-center gap-3 rounded-lg border border-border bg-card p-3 text-left transition-colors hover:bg-accent/50"
        @click="router.push('/batch-image')"
      >
        <div class="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted/50 text-foreground transition-colors group-hover:bg-background">
          <Icon name="sparkles" size="sm" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm font-medium text-foreground">{{ t('dashboard.batchImageAgent') }}</p>
          <p class="truncate text-xs text-muted-foreground">{{ t('dashboard.batchImageAgentDesc') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="sm"
          class="shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5 group-hover:text-foreground"
        />
      </button>

      <button
        type="button"
        class="group flex w-full items-center gap-3 rounded-lg border border-border bg-card p-3 text-left transition-colors hover:bg-accent/50"
        @click="router.push('/redeem')"
      >
        <div class="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted/50 text-foreground transition-colors group-hover:bg-background">
          <Icon name="gift" size="sm" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm font-medium text-foreground">{{ t('dashboard.redeemCode') }}</p>
          <p class="truncate text-xs text-muted-foreground">{{ t('dashboard.addBalanceWithCode') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="sm"
          class="shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5 group-hover:text-foreground"
        />
      </button>
    </div>
  </Card>
</template>

<script setup lang="ts">
import { Card } from '@/components/ui/card'
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
