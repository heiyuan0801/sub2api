import type { AccountCredentialFieldDefinition } from './types'

export const ANTHROPIC_DEFAULT_BASE_URL = 'https://api.anthropic.com'

export const anthropicApiKeyFields: readonly AccountCredentialFieldDefinition[] = [
  {
    key: 'base_url',
    section: 'credentials',
    type: 'text',
    showIn: ['create', 'edit', 'bulk'],
    labelKey: 'admin.accounts.baseUrl',
    placeholder: ANTHROPIC_DEFAULT_BASE_URL,
    defaultValue: '',
    create: { hintKey: 'admin.accounts.baseUrlHint' },
    edit: { hintKey: 'admin.accounts.baseUrlHint' },
    bulk: { placeholderKey: 'admin.accounts.bulkEdit.baseUrlPlaceholder' }
  },
  {
    key: 'api_key',
    section: 'credentials',
    type: 'password',
    // Bulk operations never write an API key.
    showIn: ['create', 'edit'],
    labelKey: 'admin.accounts.apiKey',
    placeholder: 'sk-ant-...',
    defaultValue: '',
    create: { labelKey: 'admin.accounts.apiKeyRequired', hintKey: 'admin.accounts.apiKeyHint', required: true },
    edit: { hintKey: 'admin.accounts.leaveEmptyToKeep' }
  }
]
