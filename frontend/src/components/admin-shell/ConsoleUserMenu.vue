<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ChevronDown, CircleHelp, KeyRound, LogOut, MessageCircle, UserRound } from '@lucide/vue'
import { useAppStore, useAuthStore } from '@/stores'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem,
  DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'

defineProps<{
  availableBalance: string
  frozenBalance?: string
}>()
const emit = defineEmits<{ logout: []; replay: [] }>()

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const user = computed(() => authStore.user)
const avatarUrl = computed(() => user.value?.avatar_url?.trim() || '')
const displayName = computed(() => user.value?.username || user.value?.email?.split('@')[0] || '')
const initials = computed(() => displayName.value.slice(0, 2).toUpperCase())
const showReplay = computed(() => !authStore.isSimpleMode && user.value?.role === 'admin')
</script>

<template>
  <DropdownMenu v-if="user">
    <DropdownMenuTrigger as-child>
      <Button
        type="button"
        variant="ghost"
        :aria-label="t('common.userMenu')"
        class="h-8 min-w-0 gap-1.5 px-0.5 sm:px-1.5"
      >
        <span aria-hidden="true" class="grid size-7 shrink-0 place-items-center overflow-hidden rounded-full border border-border bg-muted text-xs font-semibold text-foreground">
          <img v-if="avatarUrl" :src="avatarUrl" alt="" class="size-full object-cover">
          <span v-else>{{ initials }}</span>
        </span>
        <span class="hidden max-w-24 truncate text-xs font-medium text-foreground xl:block">{{ displayName }}</span>
        <ChevronDown aria-hidden="true" class="hidden size-3 shrink-0 text-muted-foreground xl:block" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" :side-offset="8" class="w-64 max-w-[calc(100vw-1rem)]">
      <DropdownMenuLabel class="px-2.5 py-2">
        <span class="block truncate leading-tight text-foreground">{{ displayName }}</span>
        <span class="mt-0.5 block break-all text-xs font-normal text-muted-foreground">{{ user.email }}</span>
        <span class="mt-1 block text-xs font-normal text-muted-foreground">{{ t('admin.users.roles.' + user.role) }}</span>
      </DropdownMenuLabel>
      <div class="md:hidden">
        <DropdownMenuSeparator />
        <DropdownMenuLabel class="space-y-1 px-2.5 text-xs">
          <span class="flex flex-wrap items-center justify-between gap-x-2">
            <span class="font-normal text-muted-foreground">{{ t('common.balance') }}</span>
            <span class="tabular-nums">{{ availableBalance }}</span>
          </span>
          <span v-if="frozenBalance" class="flex flex-wrap items-center justify-between gap-x-2 text-warning">
            <span class="font-normal">{{ t('common.frozenBalance') }}</span>
            <span class="tabular-nums">{{ frozenBalance }}</span>
          </span>
        </DropdownMenuLabel>
      </div>
      <DropdownMenuSeparator />
      <DropdownMenuGroup>
        <DropdownMenuItem as-child class="cursor-pointer">
          <RouterLink to="/profile">
            <UserRound aria-hidden="true" />
            {{ t('nav.profile') }}
          </RouterLink>
        </DropdownMenuItem>
        <DropdownMenuItem as-child class="cursor-pointer">
          <RouterLink to="/keys">
            <KeyRound aria-hidden="true" />
            {{ t('nav.apiKeys') }}
          </RouterLink>
        </DropdownMenuItem>
        <DropdownMenuItem v-if="authStore.isAdmin" as-child class="cursor-pointer">
          <a href="https://github.com/Wei-Shaw/sub2api" target="_blank" rel="noopener noreferrer">
            <svg aria-hidden="true" class="size-4" fill="currentColor" viewBox="0 0 24 24">
              <path
                fill-rule="evenodd"
                clip-rule="evenodd"
                d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.17 6.839 9.49.5.092.682-.217.682-.482 0-.237-.008-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.03-2.683-.104-.253-.446-1.27.098-2.647 0 0 .839-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.746-1.025 2.746-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.167 22 16.418 22 12c0-5.523-4.477-10-10-10z"
              />
            </svg>
            {{ t('nav.github') }}
          </a>
        </DropdownMenuItem>
      </DropdownMenuGroup>
      <template v-if="appStore.contactInfo">
        <DropdownMenuSeparator />
        <DropdownMenuLabel class="flex items-start gap-2 px-2.5 font-normal text-muted-foreground">
          <MessageCircle aria-hidden="true" class="mt-0.5 size-3.5 shrink-0" />
          <span class="min-w-0 break-words text-xs">
            {{ t('common.contactSupport') }}: <span class="font-medium text-foreground">{{ appStore.contactInfo }}</span>
          </span>
        </DropdownMenuLabel>
      </template>
      <DropdownMenuSeparator />
      <DropdownMenuGroup>
        <DropdownMenuItem v-if="showReplay" class="cursor-pointer" @select="emit('replay')">
          <CircleHelp aria-hidden="true" />
          {{ t('onboarding.restartTour') }}
        </DropdownMenuItem>
        <DropdownMenuItem variant="destructive" class="cursor-pointer" @select="emit('logout')">
          <LogOut aria-hidden="true" />
          {{ t('nav.logout') }}
        </DropdownMenuItem>
      </DropdownMenuGroup>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
