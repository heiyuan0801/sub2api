<template>
  <div class="space-y-3" role="group" :aria-label="t('keys.aggregate.groups')">
    <p class="text-xs leading-4 text-gray-500 dark:text-gray-400">{{ t('keys.aggregate.hint') }}</p>
    <ol v-if="modelValue.length" class="space-y-2" :aria-label="t('keys.aggregate.groups')">
      <li v-for="(id, index) in modelValue" :key="id" class="flex min-h-[44px] items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 text-sm dark:border-dark-600">
        <span class="shrink-0 text-xs text-gray-400">{{ t('keys.aggregate.route', { index: index + 1 }) }}</span>
        <div class="min-w-0 flex-1">
          <GroupBadge
            v-if="groupById(id)"
            class="max-w-full"
            :name="groupById(id)!.name"
            :platform="groupById(id)!.platform"
            :rate-multiplier="groupById(id)!.rate_multiplier"
            :user-rate-multiplier="userRates?.[id]"
            :subscription-type="groupById(id)!.subscription_type"
            :peak-rate-enabled="groupById(id)!.peak_rate_enabled"
            :peak-start="groupById(id)!.peak_start"
            :peak-end="groupById(id)!.peak_end"
            :peak-rate-multiplier="groupById(id)!.peak_rate_multiplier"
          />
          <span v-else class="text-xs text-gray-500">{{ t('keys.aggregate.unavailable') }} #{{ id }}</span>
        </div>
        <div class="flex shrink-0 items-center gap-0.5 text-gray-400">
          <button type="button" :aria-label="`${t('keys.aggregate.up')} ${groupById(id)?.name || id}`" :disabled="index === 0" class="rounded p-1 hover:bg-gray-100 hover:text-gray-700 disabled:opacity-30 dark:hover:bg-dark-700" @click="move(index, -1)"><Icon name="arrowUp" size="sm" /></button>
          <button type="button" :aria-label="`${t('keys.aggregate.down')} ${groupById(id)?.name || id}`" :disabled="index === modelValue.length - 1" class="rounded p-1 hover:bg-gray-100 hover:text-gray-700 disabled:opacity-30 dark:hover:bg-dark-700" @click="move(index, 1)"><Icon name="arrowDown" size="sm" /></button>
          <button type="button" :aria-label="`${t('keys.aggregate.remove')} ${groupById(id)?.name || id}`" class="rounded p-1 hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-700" @click="remove(id)"><Icon name="x" size="sm" /></button>
        </div>
      </li>
    </ol>
    <Select
      id="aggregate-add-group"
      :model-value="null"
      :options="availableOptions"
      :disabled="modelValue.length >= 20"
      :placeholder="modelValue.length >= 20 ? t('keys.aggregate.limit') : t('keys.aggregate.addGroup')"
      :aria-label="t('keys.aggregate.addGroup')"
      :searchable="true"
      :search-placeholder="t('keys.searchGroup')"
      :empty-text="t('keys.aggregate.empty')"
      @update:model-value="add"
    >
      <template #option="{ option }">
        <GroupBadge :name="option.label" :platform="groupById(Number(option.value))?.platform" :rate-multiplier="groupById(Number(option.value))?.rate_multiplier" :user-rate-multiplier="userRates?.[Number(option.value)]" />
      </template>
    </Select>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import GroupBadge from '@/components/common/GroupBadge.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Group } from '@/types'
import { getKeyGroupProvider, type KeyGroupProvider } from '@/utils/keyGroupProviders'
type GroupChoice = Pick<Group, 'id' | 'name' | 'platform' | 'rate_multiplier'> & Partial<Pick<Group, 'subscription_type' | 'peak_rate_enabled' | 'peak_start' | 'peak_end' | 'peak_rate_multiplier'>>
const props = defineProps<{ modelValue: number[]; groups: GroupChoice[]; userRates?: Record<number, number>; provider?: KeyGroupProvider }>()
const emit = defineEmits<{ 'update:modelValue': [ids: number[]] }>()
const { t } = useI18n()
const groupById = (id: number) => props.groups.find(group => group.id === id)
const availableOptions = computed(() => props.groups
  .filter(group => !props.modelValue.includes(group.id) && (!props.provider || getKeyGroupProvider(group.platform) === props.provider))
  .map(group => ({ value: group.id, label: group.name })))
function add(value: string | number | boolean | null) {
  const id = Number(value)
  if (props.modelValue.length >= 20 || !availableOptions.value.some(option => option.value === id)) return
  emit('update:modelValue', [...props.modelValue, id])
}
function remove(id: number) { emit('update:modelValue', props.modelValue.filter(value => value !== id)) }
function move(index: number, direction: number) {
  const next = index + direction
  if (next < 0 || next >= props.modelValue.length) return
  const ids = [...props.modelValue]
  ;[ids[index], ids[next]] = [ids[next]!, ids[index]!]
  emit('update:modelValue', ids)
}
</script>
