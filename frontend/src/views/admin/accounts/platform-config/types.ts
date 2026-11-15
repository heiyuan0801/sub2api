export type AccountFormMode = 'create' | 'edit' | 'bulk'
export type AccountCredentialKey = 'base_url' | 'api_key'
export type AccountSchedulingKey = 'concurrency' | 'load_factor' | 'priority' | 'rate_multiplier'

export interface AccountCredentialFieldDefinition {
  key: AccountCredentialKey
  section: 'credentials'
  type: 'text' | 'password'
  showIn: readonly AccountFormMode[]
  labelKey: string
  placeholder: string
  defaultValue: string
  create?: { labelKey?: string; hintKey?: string; required?: boolean }
  edit?: { hintKey?: string }
  bulk?: { placeholderKey?: string; hintKey?: string }
}

export interface AccountSchedulingFieldDefinition {
  key: AccountSchedulingKey
  section: 'scheduling'
  type: 'number'
  labelKey: string
  hintKey?: string
  min: number
  step?: number
  defaultValue: number | null
}
