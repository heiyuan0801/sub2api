import type { AccountFormMode, AccountSchedulingFieldDefinition, AccountSchedulingKey } from './types'

type SharedSchedulingField = Omit<AccountSchedulingFieldDefinition, 'key' | 'section' | 'type' | 'defaultValue'>

const sharedSchedulingFields = {
  concurrency: {
    labelKey: 'admin.accounts.concurrency',
    min: 1
  },
  load_factor: {
    labelKey: 'admin.accounts.loadFactor',
    hintKey: 'admin.accounts.loadFactorHint',
    min: 1
  },
  priority: {
    labelKey: 'admin.accounts.priority',
    hintKey: 'admin.accounts.priorityHint',
    min: 1
  },
  rate_multiplier: {
    labelKey: 'admin.accounts.billingRateMultiplier',
    hintKey: 'admin.accounts.billingRateMultiplierHint',
    min: 0,
    step: 0.001
  }
} satisfies Record<AccountSchedulingKey, SharedSchedulingField>

export function getAccountSchedulingField(key: AccountSchedulingKey, mode: AccountFormMode): AccountSchedulingFieldDefinition {
  return {
    key,
    section: 'scheduling',
    type: 'number',
    ...sharedSchedulingFields[key],
    ...(mode === 'bulk' && key === 'rate_multiplier' ? { step: 0.01 } : {}),
    defaultValue: key === 'load_factor' ? null : key === 'concurrency' && mode === 'create' ? 10 : 1
  }
}
