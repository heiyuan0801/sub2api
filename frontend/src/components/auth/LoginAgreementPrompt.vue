<template>
  <div
    v-if="mode === 'checkbox' && documents.length > 0"
    class="px-0.5"
  >
    <div class="flex items-start gap-2">
      <input
        id="login-agreement-consent"
        type="checkbox"
        :checked="accepted"
        class="mt-[2px] h-4 w-4 flex-shrink-0 rounded border-gray-300 text-brand focus:ring-brand dark:border-dark-600 dark:bg-dark-900"
        @change="handleCheckboxChange"
      />
      <div class="min-w-0 flex-1">
        <p class="text-[13px] leading-5 text-gray-600 dark:text-dark-300">
          <label
            for="login-agreement-consent"
            class="cursor-pointer text-gray-700 dark:text-dark-200"
          >
            {{ t('legal.loginAgreementPrompt.checkboxPrefix') }}
          </label>
          <template v-for="(doc, index) in documents" :key="doc.id || doc.title">
            <RouterLink
              :to="documentRoute(doc)"
              target="_blank"
              rel="noopener noreferrer"
              class="font-medium text-primary-600 underline-offset-4 transition hover:text-primary-700 hover:underline dark:text-primary-300 dark:hover:text-primary-200"
            >
              {{ doc.title }}
            </RouterLink>
            <span v-if="index < documents.length - 1">{{ t('legal.loginAgreementPrompt.documentSeparator') }}</span>
          </template>
        </p>
      </div>
    </div>
  </div>

  <div
    v-else-if="!accepted && documents.length > 0"
    class="rounded-lg border border-primary-100 bg-primary-50/70 p-3 text-sm text-primary-900 dark:border-primary-500/20 dark:bg-primary-500/10 dark:text-primary-100"
  >
    <div class="flex items-start gap-3">
      <Icon name="shield" size="sm" class="mt-0.5 flex-shrink-0 text-primary-600 dark:text-primary-300" />
      <div class="min-w-0 flex-1">
        <p class="font-medium">{{ t('legal.loginAgreementPrompt.noticeTitle') }}</p>
        <p class="mt-1 text-primary-700 dark:text-primary-200/80">
          {{ t('legal.loginAgreementPrompt.noticeDescription') }}
        </p>
      </div>
      <button
        type="button"
        class="flex-shrink-0 rounded-md bg-brand px-3 py-1.5 text-xs font-medium text-brand-foreground transition hover:bg-brand/90"
        @click="emit('open')"
      >
        {{ t('legal.loginAgreementPrompt.viewTerms') }}
      </button>
    </div>
  </div>

  <Teleport to="body">
    <Transition name="agreement-fade">
      <div
        v-if="dialogVisible"
        class="fixed inset-0 z-[140] flex items-center justify-center overflow-y-auto bg-gray-950/60 p-4 backdrop-blur-sm"
      >
        <div class="w-full max-w-[600px] overflow-hidden rounded-2xl bg-background shadow-2xl ring-1 ring-black/10 dark:ring-white/10">
          <div class="border-b border-gray-100 bg-background px-6 py-6 dark:border-dark-800">
            <div class="flex items-start gap-4">
              <span class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl bg-brand/10 text-brand ring-1 ring-primary-100 dark:ring-primary-500/20">
                <Icon name="shield" size="md" />
              </span>
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-2">
                  <h2 class="text-xl font-bold tracking-normal text-gray-950 dark:text-white">
                    {{ t('legal.loginAgreementPrompt.dialogTitle') }}
                  </h2>
                  <span
                    v-if="updatedAt"
                    class="rounded-full bg-muted px-2.5 py-1 text-xs font-medium text-gray-600 dark:text-dark-300"
                  >
                    {{ updatedAt }}
                  </span>
                </div>
                <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
                  {{
                    t('legal.loginAgreementPrompt.dialogDescription', {
                      date: updatedAt || t('legal.loginAgreementPrompt.recently'),
                    })
                  }}
                </p>
              </div>
            </div>
          </div>

          <div class="max-h-[58vh] overflow-y-auto px-6 py-5">
            <div class="mb-3 flex items-center justify-between gap-3">
              <p class="text-sm font-semibold text-foreground">{{ t('legal.loginAgreementPrompt.relatedDocuments') }}</p>
            </div>
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <RouterLink
                v-for="(doc, index) in documents"
                :key="doc.id || doc.title"
                :to="documentRoute(doc)"
                target="_blank"
                rel="noopener noreferrer"
                class="group flex min-h-[72px] w-full items-center gap-3 rounded-xl border border-border bg-gray-50/70 px-4 py-3 text-left transition hover:-translate-y-0.5 hover:border-primary-200 hover:bg-card hover:shadow-sm dark:bg-dark-800/70 dark:hover:border-primary-500/30"
              >
                <span class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-background text-gray-700 ring-1 ring-gray-200 transition group-hover:bg-brand/10 group-hover:text-primary-700 group-hover:ring-primary-100 dark:text-dark-200 dark:ring-dark-700 dark:group-hover:text-primary-200 dark:group-hover:ring-primary-500/20">
                  <Icon :name="documentIcon(index, doc.title)" size="sm" />
                </span>
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-sm font-semibold text-gray-950 dark:text-white">{{ doc.title }}</span>
                </span>
                <span class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full text-gray-400 transition group-hover:bg-brand/10 group-hover:text-primary-600 dark:group-hover:text-primary-300">
                  <Icon name="externalLink" size="sm" />
                </span>
              </RouterLink>
            </div>
          </div>

          <div class="border-t border-gray-100 bg-gray-50/80 px-6 py-4 dark:border-dark-800 dark:bg-dark-950/60">
            <div class="grid grid-cols-2 gap-3">
              <button
                type="button"
                class="rounded-xl border border-border bg-card px-4 py-3 text-sm font-semibold text-gray-700 transition hover:bg-gray-100 dark:text-dark-200 dark:hover:bg-dark-700"
                @click="emit('reject')"
              >
                {{ t('legal.loginAgreementPrompt.reject') }}
              </button>
              <button
                type="button"
                class="rounded-xl bg-brand px-4 py-3 text-sm font-semibold text-brand-foreground shadow-sm shadow-brand/20 transition hover:bg-brand/90"
                @click="emit('accept')"
              >
                {{ t('legal.loginAgreementPrompt.accept') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { LoginAgreementDocument } from '@/types'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  accepted: boolean
  documents: LoginAgreementDocument[]
  mode: 'modal' | 'checkbox' | string
  updatedAt?: string
  visible: boolean
}>(), {
  updatedAt: ''
})

const emit = defineEmits<{
  accept: []
  reject: []
  open: []
}>()

const dialogVisible = computed(() => props.visible && documents.value.length > 0)
const documents = computed(() => props.documents.filter((doc) => doc.title.trim()))
const updatedAt = computed(() => props.updatedAt || '')
const accepted = computed(() => props.accepted)
const mode = computed(() => props.mode === 'checkbox' ? 'checkbox' : 'modal')

function documentRoute(doc: LoginAgreementDocument) {
  return {
    name: 'LegalDocument',
    params: {
      documentId: doc.id || doc.title,
    },
  }
}

function handleCheckboxChange(event: Event): void {
  const checked = (event.target as HTMLInputElement).checked
  if (checked) {
    emit('accept')
  } else {
    emit('reject')
  }
}

function documentIcon(index: number, title: string): 'document' | 'shield' | 'globe' | 'cog' {
  const normalizedTitle = title.toLowerCase()
  if (
    normalizedTitle.includes('policy') ||
    normalizedTitle.includes('privacy') ||
    title.includes('政策') ||
    title.includes('隐私')
  ) {
    return 'shield'
  }
  if (
    normalizedTitle.includes('country') ||
    normalizedTitle.includes('region') ||
    title.includes('国家') ||
    title.includes('地区')
  ) {
    return 'globe'
  }
  if (index === 3) {
    return 'cog'
  }
  return 'document'
}
</script>

<style scoped>
.agreement-fade-enter-active,
.agreement-fade-leave-active {
  transition: opacity 0.18s ease;
}

.agreement-fade-enter-from,
.agreement-fade-leave-to {
  opacity: 0;
}

.agreement-fade-enter-active > div,
.agreement-fade-leave-active > div {
  transition: transform 0.18s ease, opacity 0.18s ease;
}

.agreement-fade-enter-from > div,
.agreement-fade-leave-to > div {
  opacity: 0;
  transform: translateY(8px) scale(0.98);
}
</style>
