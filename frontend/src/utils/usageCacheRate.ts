import { BILLING_MODE_TOKEN, getDisplayBillingMode } from './billingMode'

export interface UsageCacheRateRow {
  input_tokens?: number | null
  cache_read_tokens?: number | null
  cache_creation_tokens?: number | null
  image_count?: number | null
  billing_mode?: string | null
}

// Usage logs store mutually exclusive uncached, cache-read and cache-write inputs.
// Output tokens and the 5m/1h write subtotals must not be counted again.
export function usageCacheBreakdown(row: UsageCacheRateRow) {
  const values = [row.input_tokens, row.cache_read_tokens, row.cache_creation_tokens]
  const mode = getDisplayBillingMode({ ...row, image_count: row.image_count ?? 0 })
  if ((mode && mode !== BILLING_MODE_TOKEN) || values.some(value =>
    value != null && (!Number.isFinite(value) || value < 0)
  )) return null

  const [input, read, write] = values.map(value => value ?? 0)
  const total = input + read + write
  if (total <= 0 || !Number.isFinite(total)) return null
  return { input, read, write, total, percent: read / total * 100 }
}

export function formatUsageCacheRatePercent(row: UsageCacheRateRow): string {
  const breakdown = usageCacheBreakdown(row)
  return breakdown ? `${breakdown.percent.toFixed(1)}%` : ''
}
