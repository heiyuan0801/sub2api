import type { MonitorCoverage, MonitorMatrixRow } from '@/api/channelMonitorV2'

export type MonitorSort = 'health' | 'success' | 'latency' | 'name'
export const monitorRowKey = (row: Pick<MonitorMatrixRow, 'platform' | 'model' | 'group_id'>) =>
  JSON.stringify([row.platform, row.model || '', row.group_id ?? null])

export interface MonitorPreferences {
  range?: string
  platform?: string[]
  group?: number[]
  model?: string[]
  q?: string
  status?: string
  model_type?: string
  group_by?: string
  health_mode?: string
  trend_view?: string
  tab?: string
  sort?: MonitorSort
  favorites_only?: boolean
  favorites?: string[]
}

export function preferenceKey(userId: number | undefined): string | null {
  return Number.isSafeInteger(userId) && Number(userId) > 0 ? `channel-monitor-v2:v1:${userId}` : null
}

export function readPreferences(key: string | null): MonitorPreferences {
  if (!key) return {}
  try {
    const value: unknown = JSON.parse(localStorage.getItem(key) || '{}')
    if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
    const raw = value as Record<string, unknown>
    const result: MonitorPreferences = {}
    for (const field of ['range', 'q', 'status', 'model_type', 'group_by', 'health_mode', 'trend_view', 'tab'] as const) {
      if (typeof raw[field] === 'string' && raw[field].length <= 256) result[field] = raw[field]
    }
    for (const field of ['platform', 'model'] as const) {
      if (Array.isArray(raw[field])) result[field] = raw[field].filter((item): item is string => typeof item === 'string' && item.length <= 256).slice(0, 100)
    }
    if (Array.isArray(raw.group)) result.group = raw.group.filter((id): id is number => Number.isSafeInteger(id) && Number(id) > 0).slice(0, 100)
    if (['health', 'success', 'latency', 'name'].includes(String(raw.sort))) result.sort = raw.sort as MonitorSort
    if (typeof raw.favorites_only === 'boolean') result.favorites_only = raw.favorites_only
    if (Array.isArray(raw.favorites)) result.favorites = [...new Set(raw.favorites.filter((item): item is string => {
      if (typeof item !== 'string' || item.length > 1024) return false
      try {
        const parts: unknown = JSON.parse(item)
        return Array.isArray(parts) && parts.length === 3 && typeof parts[0] === 'string' && typeof parts[1] === 'string' && (parts[2] === null || (Number.isSafeInteger(parts[2]) && parts[2] > 0))
      } catch { return false }
    }))].slice(0, 500)
    return result
  } catch { return {} }
}

export function savePreferences(key: string | null, value: MonitorPreferences) {
  if (!key) return
  try { localStorage.setItem(key, JSON.stringify(value)) } catch { /* Storage may be disabled or full. */ }
}

/** A successful HTTP response cannot make an old aggregation watermark fresh. */
export function monitorFreshness(coverage: MonitorCoverage | null | undefined, refreshSeconds: number, now: number) {
  const threshold = Math.max(180, refreshSeconds * 3)
  if (!coverage) return { stale: false, ageSeconds: 0, threshold }
  const computed = Date.parse(coverage.computed_at)
  const through = Date.parse(coverage.data_through)
  const valid = Number.isFinite(computed) && computed > 0 && Number.isFinite(through) && through > 0
  const ageSeconds = valid ? Math.max(0, (now - Math.min(computed, through)) / 1000, coverage.aggregation_lag_seconds || 0) : 0
  return { stale: !valid || computed > now + 300_000 || through > now + 300_000 || ageSeconds > threshold, ageSeconds, threshold }
}

export function sortMonitorRows(rows: MonitorMatrixRow[], sort: MonitorSort, favorites: string[]) {
  const saved = new Set(favorites)
  const priority: Record<string, number> = { critical: 0, warning: 1, unknown: 2, healthy: 3 }
  const latency = (row: MonitorMatrixRow) => row.is_image_model
    ? row.health.generation_latency_ms ?? row.metrics.duration.p95_ms ?? row.metrics.duration.p90_ms ?? row.metrics.duration.p50_ms
    : row.metrics.ttft.p50_ms ?? row.metrics.duration.avg_ms
  const success = (row: MonitorMatrixRow) => row.metrics.has_data === false ? null : row.metrics.success_rate ?? 1 - row.metrics.error_rate
  const compareNumber = (a: number | null | undefined, b: number | null | undefined, descending = false) =>
    a == null ? b == null ? 0 : 1 : b == null ? -1 : descending ? b - a : a - b
  return [...rows].sort((a, b) => {
    const favoriteDifference = Number(saved.has(monitorRowKey(b))) - Number(saved.has(monitorRowKey(a)))
    if (favoriteDifference) return favoriteDifference
    let difference = 0
    if (sort === 'health') difference = (a.metrics.has_data === false ? 4 : priority[a.health.overall] ?? 2) - (b.metrics.has_data === false ? 4 : priority[b.health.overall] ?? 2)
    if (sort === 'success') difference = compareNumber(success(a), success(b))
    if (sort === 'latency') difference = compareNumber(latency(a), latency(b), true)
    return difference || `${a.model || ''}\0${a.platform}\0${a.group_name || ''}`.localeCompare(`${b.model || ''}\0${b.platform}\0${b.group_name || ''}`)
  })
}
