<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <PageHeader
          :title="t('availableChannels.title')"
          :description="t('availableChannels.description')"
        >
          <template #actions>
            <button
              type="button"
              @click="loadChannels"
              :disabled="loading"
              class="inline-flex h-9 items-center gap-1.5 rounded-full border border-border bg-card px-3.5 text-xs font-medium text-foreground transition-colors hover:bg-muted/60 disabled:pointer-events-none disabled:opacity-50"
              :title="t('common.refresh', 'Refresh')"
            >
              <Icon name="refresh" size="xs" :stroke-width="1.75" :class="loading ? 'animate-spin' : ''" />
              <span>{{ t('common.refresh', 'Refresh') }}</span>
            </button>
          </template>
        </PageHeader>
      </template>

      <template #filters>
        <div class="flex flex-wrap items-center gap-2.5 rounded-2xl border border-border/80 bg-card p-3.5 shadow-sm">
          <div class="relative w-full sm:w-80">
            <Icon
              name="search"
              size="sm"
              class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
            />
            <input
              v-model="searchQuery"
              type="text"
              :placeholder="t('availableChannels.searchPlaceholder')"
              class="h-9 w-full rounded-xl border border-border bg-background pl-9 pr-3 text-sm text-foreground placeholder:text-muted-foreground transition-colors focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/20"
            />
          </div>
          <button
            v-if="linkedGroupID"
            type="button"
            class="inline-flex h-9 items-center gap-1.5 rounded-full border border-border bg-muted/50 px-3 text-xs font-medium text-foreground transition-colors hover:bg-muted"
            @click="clearLinkedGroup"
          >
            <span>{{ linkedGroupName || `${t('operations.notificationGroup')} #${linkedGroupID}` }}</span>
            <Icon name="x" size="xs" aria-hidden="true" />
            <span class="sr-only">{{ t('operations.clearGroupFilter') }}</span>
          </button>
        </div>
      </template>

      <template #table>
        <AvailableChannelsTable
          :columns="columnLabels"
          :rows="filteredChannels"
          :loading="loading"
          :user-group-rates="userGroupRates"
          pricing-key-prefix="availableChannels.pricing"
          :no-pricing-label="t('availableChannels.noPricing')"
          :no-models-label="t('availableChannels.noModels')"
          :empty-label="t('availableChannels.empty')"
        />
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import Icon from '@/components/icons/Icon.vue'
import AvailableChannelsTable from '@/components/channels/AvailableChannelsTable.vue'
import userChannelsAPI, { type UserAvailableChannel } from '@/api/channels'
import userGroupsAPI from '@/api/groups'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const route = useRoute()
const router = useRouter()

const channels = ref<UserAvailableChannel[]>([])
const userGroupRates = ref<Record<number, number>>({})
const loading = ref(false)
const searchQuery = ref('')
const linkedGroupID = computed(() => {
  const id = route.query.group_id
  if (typeof id !== 'string' || !/^[1-9][0-9]*$/.test(id)) return 0
  const parsed = Number(id)
  return Number.isSafeInteger(parsed) ? parsed : 0
})
const linkedGroupName = computed(() => linkedGroupID.value
  ? channels.value.flatMap(channel => channel.platforms).flatMap(platform => platform.groups)
    .find(group => group.id === linkedGroupID.value)?.name
  : undefined)

function clearLinkedGroup() {
  const { group_id: _groupID, ...query } = route.query
  void router.replace({ query })
}

const columnLabels = computed(() => ({
  name: t('availableChannels.columns.name'),
  description: t('availableChannels.columns.description'),
  platform: t('availableChannels.columns.platform'),
  groups: t('availableChannels.columns.groups'),
  supportedModels: t('availableChannels.columns.supportedModels'),
}))

/**
 * 搜索过滤：
 * - 命中渠道名/描述 → 整个渠道（所有 platforms）都保留
 * - 否则按 platform/group/model 维度在 sections 里过滤，保留有匹配的 section
 * - 所有 sections 都不匹配时，渠道本身被过滤掉
 */
const filteredChannels = computed(() => {
  const groupID = linkedGroupID.value
  const scopedChannels = !groupID ? channels.value : channels.value.flatMap(channel => {
    const platforms = channel.platforms.flatMap(platform => {
      const groups = platform.groups.filter(group => group.id === groupID)
      return groups.length ? [{ ...platform, groups }] : []
    })
    return platforms.length ? [{ ...channel, platforms }] : []
  })
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return scopedChannels
  return scopedChannels
    .map((ch) => {
      const nameHit = ch.name.toLowerCase().includes(q)
      const descHit = (ch.description || '').toLowerCase().includes(q)
      if (nameHit || descHit) return ch
      const matchingSections = ch.platforms.filter(
        (p) =>
          p.platform.toLowerCase().includes(q) ||
          p.groups.some((g) => g.name.toLowerCase().includes(q)) ||
          p.supported_models.some((m) => m.name.toLowerCase().includes(q)),
      )
      if (matchingSections.length === 0) return null
      return { ...ch, platforms: matchingSections }
    })
    .filter((ch): ch is UserAvailableChannel => ch !== null)
})

async function loadChannels() {
  loading.value = true
  try {
    // 渠道列表和用户专属倍率并发拉取。专属倍率失败不阻塞渠道展示——
    // 失败时只是无法渲染专属倍率角标，降级为仅显示默认倍率。
    const [list, rates] = await Promise.all([
      userChannelsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates().catch((err: unknown) => {
        console.error('Failed to load user group rates:', err)
        return {} as Record<number, number>
      }),
    ])
    channels.value = list
    userGroupRates.value = rates
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(loadChannels)
</script>
