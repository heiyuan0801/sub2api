import type { AccountCredentialFieldDefinition } from './types'

export const OPENAI_DEFAULT_BASE_URL = 'https://api.openai.com'

export const openaiApiKeyFields: readonly AccountCredentialFieldDefinition[] = [
  {
    key: 'base_url',
    section: 'credentials',
    type: 'text',
    showIn: ['create', 'edit', 'bulk'],
    labelKey: 'admin.accounts.baseUrl',
    placeholder: OPENAI_DEFAULT_BASE_URL,
    defaultValue: '',
    create: { hintKey: 'admin.accounts.openai.baseUrlHint' },
    edit: { hintKey: 'admin.accounts.openai.baseUrlHint' },
    bulk: { placeholderKey: 'admin.accounts.bulkEdit.baseUrlPlaceholder' }
  },
  {
    key: 'api_key',
    section: 'credentials',
    type: 'password',
    showIn: ['create', 'edit'],
    labelKey: 'admin.accounts.apiKey',
    placeholder: 'sk-proj-...',
    defaultValue: '',
    create: { labelKey: 'admin.accounts.apiKeyRequired', hintKey: 'admin.accounts.openai.apiKeyHint', required: true },
    edit: { hintKey: 'admin.accounts.leaveEmptyToKeep' }
  }
]
