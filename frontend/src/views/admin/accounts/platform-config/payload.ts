/**
 * The common subset of the existing account payload. Provider patches,
 * confirmation checks and request orchestration stay with the account form.
 */
export function buildCreateApiKeyCredentials(
  baseUrl: string,
  apiKey: string,
  defaultBaseUrl: string
): Record<string, unknown> {
  return {
    base_url: baseUrl.trim() || defaultBaseUrl,
    api_key: apiKey.trim()
  }
}

export function buildEditApiKeyCredentials(
  current: Record<string, unknown>,
  baseUrl: string,
  apiKey: string,
  defaultBaseUrl: string,
  hasExistingApiKey: boolean
): Record<string, unknown> | null {
  const trimmedApiKey = apiKey.trim()
  if (!trimmedApiKey && !hasExistingApiKey) return null
  return {
    ...current,
    base_url: baseUrl.trim() || defaultBaseUrl,
    ...(trimmedApiKey ? { api_key: trimmedApiKey } : {})
  }
}

export function buildBulkBaseUrlCredentials(enabled: boolean, baseUrl: string): { base_url: string } | undefined {
  if (!enabled) return undefined
  const trimmedBaseUrl = baseUrl.trim()
  return trimmedBaseUrl ? { base_url: trimmedBaseUrl } : undefined
}

export function isValidBulkBaseUrl(baseUrl: string): boolean {
  const trimmedBaseUrl = baseUrl.trim()
  return !trimmedBaseUrl || /^https?:\/\//i.test(trimmedBaseUrl)
}

export function normalizeAccountUpdateLoadFactor(loadFactor: number | null): number {
  return loadFactor != null && !Number.isNaN(loadFactor) && loadFactor > 0 ? loadFactor : 0
}
