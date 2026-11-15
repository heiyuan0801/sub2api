import type { AccountPlatform, AccountType } from '@/types'
import { anthropicApiKeyFields } from './anthropic'
import { openaiApiKeyFields } from './openai'
import type { AccountCredentialKey, AccountFormMode } from './types'

const apiKeyFields = {
  anthropic: anthropicApiKeyFields,
  openai: openaiApiKeyFields
} as const

function isConfiguredPlatform(platform: AccountPlatform): platform is keyof typeof apiKeyFields {
  return platform === 'anthropic' || platform === 'openai'
}

export function usesConfiguredApiKeyFields(platform: AccountPlatform, accountType: AccountType): boolean {
  return accountType === 'apikey' && isConfiguredPlatform(platform)
}

export function getApiKeyField(platform: AccountPlatform, key: AccountCredentialKey, mode: AccountFormMode) {
  if (!isConfiguredPlatform(platform)) return undefined
  return apiKeyFields[platform].find(field => field.key === key && field.showIn.includes(mode))
}

export function getAnthropicApiKeyField(key: AccountCredentialKey, mode: AccountFormMode) {
  return getApiKeyField('anthropic', key, mode)
}
