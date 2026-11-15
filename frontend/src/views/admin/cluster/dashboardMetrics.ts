import type { ClusterNode, ClusterPerformance } from '@/api/admin/cluster'

export function hasRecentSample(value: string | undefined, now: number): boolean {
  const elapsed = value ? now - Date.parse(value) : Number.NaN
  return Number.isFinite(elapsed) && elapsed >= -5000 && elapsed < 20000
}

export function isCurrentNode(node: ClusterNode, now: number): boolean {
  return node.online && hasRecentSample(node.heartbeat_at, now)
}

export function validMetric(value: number | null | undefined): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0
}

export function nodePerformance(node: ClusterNode, now: number): ClusterPerformance | undefined {
  const performance = node.stats.performance
  return isCurrentNode(node, now) && hasRecentSample(performance?.sampled_at, now) ? performance : undefined
}

export function formatMetric(value: number | null | undefined, digits = 0): string {
  return validMetric(value) ? value.toLocaleString(undefined, { maximumFractionDigits: digits }) : '—'
}

export function nodeRates(nodes: readonly ClusterNode[], now: number) {
  return nodes.flatMap(node => {
    const rps = nodePerformance(node, now)?.requests_per_second
    return validMetric(rps) ? [{ id: node.id, rps, maintenance: node.control.maintenance }] : []
  })
}

export function clusterMetrics(nodes: readonly ClusterNode[], now: number) {
  const online = nodes.filter(node => isCurrentNode(node, now))
  const rates = nodeRates(online, now)
  const percentiles = online.flatMap(node => {
    const sample = nodePerformance(node, now)
    return sample && sample.first_token_samples > 0 ? [sample] : []
  })
  const knownP95 = percentiles.map(sample => sample.first_token_p95_ms).filter(validMetric)
  const counters = online.filter(({ stats }) => validMetric(stats.requests) && validMetric(stats.errors))
  // These counters are cumulative for each process, and only cover admitted
  // requests. Active work includes background tasks and is not a comparable denominator.
  const counts = counters.reduce(({ admitted, errors }, { stats }) => ({
    admitted: admitted + stats.requests,
    errors: errors + stats.errors
  }), { admitted: 0, errors: 0 })

  return {
    onlineCount: online.length,
    drainingCount: online.filter(node => node.control.maintenance).length,
    totalRps: rates.length ? rates.reduce((sum, node) => sum + node.rps, 0) : null,
    reportingRps: rates.length,
    slowestP95: knownP95.length ? Math.max(...knownP95) : null,
    p95Overflow: percentiles.some(sample => sample.first_token_overflow),
    errorRate: counts.admitted > 0 && counts.errors <= counts.admitted ? 100 * counts.errors / counts.admitted : null
  }
}
