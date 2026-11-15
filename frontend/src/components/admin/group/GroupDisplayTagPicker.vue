<script setup lang="ts">
import { computed, nextTick, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus, Trash2 } from '@lucide/vue'
import GroupDisplayTags from '@/components/common/GroupDisplayTags.vue'
import {
  customGroupDisplayTagColors,
  customGroupDisplayTagIcons,
  customGroupDisplayTagLabelError,
  customGroupDisplayTagLabelLimit,
  customGroupDisplayTagLimit,
  groupDisplayTags,
  type CustomGroupDisplayTag,
  type GroupDisplayTag,
} from '@/constants/groupDisplayTags'

const tags = defineModel<GroupDisplayTag[]>({ required: true })
const customTags = defineModel<CustomGroupDisplayTag[]>('customTags', { required: true })
const { t } = useI18n()
const id = useId()
const labelErrors = computed(() => customTags.value.map((tag, index) =>
  tag.label ? customGroupDisplayTagLabelError(customTags.value, index) : undefined,
))

function updateTag(index: number, updates: Partial<CustomGroupDisplayTag>) {
  customTags.value = customTags.value.map((tag, position) => position === index ? { ...tag, ...updates } : tag)
}

function updateLabel(index: number, event: Event) {
  if (event instanceof InputEvent && event.isComposing) return
  updateTag(index, { label: (event.target as HTMLInputElement).value })
}

async function addTag() {
  const index = customTags.value.length
  if (index >= customGroupDisplayTagLimit) return
  customTags.value = [...customTags.value, { label: '', color: 'blue', icon: 'tag' }]
  await nextTick()
  document.getElementById(`${id}-label-${index}`)?.focus()
}

async function removeTag(index: number) {
  customTags.value = customTags.value.filter((_, position) => position !== index)
  await nextTick()
  const nextIndex = Math.min(index, customTags.value.length - 1)
  const target = nextIndex < 0 ? `${id}-add` : `${id}-label-${nextIndex}`
  document.getElementById(target)?.focus()
}
</script>

<template>
  <fieldset class="space-y-3">
    <legend class="input-label">{{ t('planOps.displayTags') }}</legend>
    <div class="flex flex-wrap gap-2">
      <label
        v-for="tag in groupDisplayTags"
        :key="tag"
        class="inline-flex min-h-10 cursor-pointer items-center gap-2 rounded-lg border border-gray-200 px-2.5 dark:border-dark-600"
      >
        <input v-model="tags" type="checkbox" :value="tag" class="h-4 w-4 rounded" />
        <GroupDisplayTags :tags="[tag]" />
      </label>
    </div>

    <div v-if="customTags.length" class="space-y-2">
      <div
        v-for="(tag, index) in customTags"
        :key="index"
        class="space-y-2 rounded-lg border border-gray-200 p-3 dark:border-dark-600"
      >
        <div class="flex items-center justify-between gap-2">
          <GroupDisplayTags :custom-tags="[{ ...tag, label: tag.label.trim() || t('planOps.customTags.preview') }]" />
          <button
            type="button"
            class="btn btn-ghost h-8 w-8 shrink-0 p-0 text-gray-500 hover:text-red-700 dark:text-gray-400 dark:hover:text-red-400"
            :aria-label="t('planOps.customTags.remove', { label: tag.label.trim() || index + 1 })"
            @click="removeTag(index)"
          >
            <Trash2 class="h-4 w-4" aria-hidden="true" />
          </button>
        </div>

        <div class="grid grid-cols-2 gap-2 sm:grid-cols-[minmax(0,2fr)_repeat(2,minmax(0,1fr))]">
          <div class="col-span-2 min-w-0 sm:col-span-1">
            <label :for="`${id}-label-${index}`" class="input-label">{{ t('planOps.customTags.label') }}</label>
            <input
              :id="`${id}-label-${index}`"
              :value="tag.label"
              :aria-invalid="Boolean(labelErrors[index])"
              :aria-describedby="labelErrors[index] ? `${id}-error-${index}` : undefined"
              :class="{ 'border-red-500 dark:border-red-400': labelErrors[index] }"
              type="text"
              class="input"
              required
              @input="updateLabel(index, $event)"
              @compositionend="updateLabel(index, $event)"
            />
            <p v-if="labelErrors[index]" :id="`${id}-error-${index}`" class="mt-1 text-xs text-red-700 dark:text-red-400">
              {{ t(`planOps.customTags.errors.${labelErrors[index]}`, { limit: customGroupDisplayTagLabelLimit }) }}
            </p>
          </div>
          <div class="min-w-0">
            <label :for="`${id}-color-${index}`" class="input-label">{{ t('planOps.customTags.color') }}</label>
            <select
              :id="`${id}-color-${index}`"
              :value="tag.color"
              class="input"
              @change="updateTag(index, { color: ($event.target as HTMLSelectElement).value as CustomGroupDisplayTag['color'] })"
            >
              <option v-for="color in customGroupDisplayTagColors" :key="color" :value="color">
                {{ t(`planOps.customTags.colors.${color}`) }}
              </option>
            </select>
          </div>
          <div class="min-w-0">
            <label :for="`${id}-icon-${index}`" class="input-label">{{ t('planOps.customTags.icon') }}</label>
            <select
              :id="`${id}-icon-${index}`"
              :value="tag.icon"
              class="input"
              @change="updateTag(index, { icon: ($event.target as HTMLSelectElement).value as CustomGroupDisplayTag['icon'] })"
            >
              <option v-for="icon in customGroupDisplayTagIcons" :key="icon" :value="icon">
                {{ t(`planOps.customTags.icons.${icon}`) }}
              </option>
            </select>
          </div>
        </div>
      </div>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <button
        :id="`${id}-add`"
        type="button"
        class="btn btn-secondary gap-1.5"
        :disabled="customTags.length >= customGroupDisplayTagLimit"
        @click="addTag"
      >
        <Plus class="h-4 w-4" aria-hidden="true" />
        {{ t('planOps.customTags.add') }}
      </button>
      <span v-if="customTags.length" class="text-xs text-gray-500 dark:text-gray-400">
        {{ t('planOps.customTags.count', { count: customTags.length, limit: customGroupDisplayTagLimit }) }}
      </span>
    </div>
  </fieldset>
</template>
