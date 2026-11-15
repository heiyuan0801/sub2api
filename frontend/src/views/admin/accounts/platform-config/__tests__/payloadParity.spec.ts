import { describe, expect, it } from 'vitest'
import { ANTHROPIC_DEFAULT_BASE_URL } from '../anthropic'
import { OPENAI_DEFAULT_BASE_URL } from '../openai'
import {
  buildBulkBaseUrlCredentials,
  buildCreateApiKeyCredentials,
  buildEditApiKeyCredentials,
  isValidBulkBaseUrl,
  normalizeAccountUpdateLoadFactor
} from '../payload'
import { getAnthropicApiKeyField, getApiKeyField, usesConfiguredApiKeyFields } from '../registry'
import { getAccountSchedulingField } from '../scheduling'

// Golden fragments transcribed from the original modal submit paths. Use
// placeholders and reserved example domains; the remaining provider-specific
// payload branches still execute in their original form controllers.
describe('API-key configuration and legacy payload parity', () => {
  it('creates an account with a trimmed replacement key and the legacy default or custom URL', () => {
    expect(buildCreateApiKeyCredentials(' ', ' <new-value> ', ANTHROPIC_DEFAULT_BASE_URL)).toEqual({
      base_url: 'https://api.anthropic.com',
      api_key: '<new-value>'
    })
    expect(buildCreateApiKeyCredentials(' https://relay.example/api ', ' <new-value> ', ANTHROPIC_DEFAULT_BASE_URL)).toEqual({
      base_url: 'https://relay.example/api',
      api_key: '<new-value>'
    })
  })

  it('does not send a new key value when editing a redacted account with a stored key', () => {
    const existing = {
      base_url: 'https://relay.example',
      model_mapping: { model: 'alias' },
      pool_mode: true
    }
    expect(buildEditApiKeyCredentials(existing, ' ', ' ', ANTHROPIC_DEFAULT_BASE_URL, true)).toEqual({
      base_url: 'https://api.anthropic.com',
      model_mapping: { model: 'alias' },
      pool_mode: true
    })
    expect(existing.base_url).toBe('https://relay.example')
  })

  it('preserves older response fields, allows replacement, and rejects a missing key', () => {
    const existing = { api_key: '<old-value>', model_mapping: { model: 'alias' } }
    const expected = { ...existing, base_url: 'https://api.anthropic.com' }
    expect(buildEditApiKeyCredentials(existing, '', ' ', ANTHROPIC_DEFAULT_BASE_URL, true)).toEqual(expected)
    expect(buildEditApiKeyCredentials(existing, '', ' <replacement> ', ANTHROPIC_DEFAULT_BASE_URL, false)).toEqual({
      ...expected,
      api_key: '<replacement>'
    })
    expect(buildEditApiKeyCredentials({}, '', ' ', ANTHROPIC_DEFAULT_BASE_URL, false)).toBeNull()
  })

  it('only emits a bulk credentials patch when the nonempty URL field is selected', () => {
    expect(buildBulkBaseUrlCredentials(false, 'https://relay.example')).toBeUndefined()
    expect(buildBulkBaseUrlCredentials(true, '  ')).toBeUndefined()
    expect(buildBulkBaseUrlCredentials(true, ' https://relay.example ')).toEqual({
      base_url: 'https://relay.example'
    })
    expect(isValidBulkBaseUrl('  ')).toBe(true)
    expect(isValidBulkBaseUrl(' HTTP://relay.example ')).toBe(true)
    expect(isValidBulkBaseUrl('relay.example')).toBe(false)
  })

  it('keeps provider fallback, required-on-create behavior and the bulk credential boundary', () => {
    expect(usesConfiguredApiKeyFields('anthropic', 'apikey')).toBe(true)
    expect(usesConfiguredApiKeyFields('anthropic', 'oauth')).toBe(false)
    expect(usesConfiguredApiKeyFields('anthropic', 'bedrock')).toBe(false)
    expect(usesConfiguredApiKeyFields('openai', 'apikey')).toBe(true)
    expect(usesConfiguredApiKeyFields('openai', 'oauth')).toBe(false)
    expect(usesConfiguredApiKeyFields('gemini', 'apikey')).toBe(false)
    expect(getAnthropicApiKeyField('api_key', 'create')?.create?.required).toBe(true)
    expect(getAnthropicApiKeyField('api_key', 'edit')).toBeDefined()
    expect(getAnthropicApiKeyField('api_key', 'bulk')).toBeUndefined()
  })

  it('keeps OpenAI defaults and leaves provider-specific credential values in place', () => {
    expect(getApiKeyField('openai', 'base_url', 'create')?.placeholder).toBe('https://api.openai.com')
    expect(getApiKeyField('openai', 'api_key', 'edit')?.placeholder).toBe('sk-proj-...')
    expect(getApiKeyField('openai', 'api_key', 'bulk')).toBeUndefined()
    expect(getApiKeyField('gemini', 'base_url', 'create')).toBeUndefined()
    expect(buildCreateApiKeyCredentials(' ', ' <new-value> ', OPENAI_DEFAULT_BASE_URL)).toEqual({
      base_url: 'https://api.openai.com',
      api_key: '<new-value>'
    })
    expect(buildEditApiKeyCredentials({
      openai_capabilities: ['chat_completions'],
      compact_model_mapping: { compact: 'upstream' }
    }, '', '', OPENAI_DEFAULT_BASE_URL, true)).toEqual({
      base_url: 'https://api.openai.com',
      openai_capabilities: ['chat_completions'],
      compact_model_mapping: { compact: 'upstream' }
    })
  })

  it('keeps the create default, per-operation rate step and update clearing sentinel', () => {
    expect(getAccountSchedulingField('concurrency', 'create').defaultValue).toBe(10)
    expect(getAccountSchedulingField('concurrency', 'bulk').defaultValue).toBe(1)
    expect(getAccountSchedulingField('rate_multiplier', 'edit').step).toBe(0.001)
    expect(getAccountSchedulingField('rate_multiplier', 'bulk').step).toBe(0.01)
    expect([null, Number.NaN, 0, -1, 1].map(normalizeAccountUpdateLoadFactor)).toEqual([0, 0, 0, 0, 1])
  })
})
