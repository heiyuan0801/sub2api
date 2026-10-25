import type { ClusterEvent } from '@/api/admin/cluster'

type Translate = (key: string, params?: Record<string, string | number>) => string

// Keep persisted event codes and messages unchanged. These exact source messages
// also translate historical rows; unfamiliar messages remain available as raw text.
const messages: Record<string, string> = {
  'node heartbeat expired': 'offline',
  'database writer unavailable': 'database',
  'Redis unavailable or replication link down': 'redis',
  'cluster lease proof unavailable': 'quorum',
  'no synchronous database replica; reduced redundancy': 'single_database_copy',
  'database replication lag exceeds 64 MiB': 'replication_lag',
  'application filesystem has less than 10 GiB or 15 percent free': 'disk_space',
  'node request capacity reached': 'capacity',
  'ingress has not acknowledged the latest weight and capacity': 'ingress_pending',
  'application versions differ between nodes': 'versions',
  'runtime settings digests differ between nodes': 'settings',
  'cluster mode is disabled': 'clusterDisabled',
  'at least two healthy application nodes': 'nodesPass',
  'fewer than two healthy application nodes': 'nodesFail',
  'registered node versions match': 'versionsPass',
  'node versions differ': 'versions',
  'current supervisor has confirmed quorum; this does not prove physical independence': 'quorumPass',
  'HA lease proof is unavailable': 'quorumFail',
  'WAL archiving is configured; verify successful delivery and restore separately': 'archiveConfigured',
  'continuous WAL archiving is not configured': 'archiveUnconfigured',
  'database role cannot inspect archiving': 'archiveUnknown',
  'most recent WAL archive attempt failed': 'archiveFailed',
  'database reports a successful WAL archive': 'archiveSucceeded',
  'no successful WAL archive has been observed': 'archiveUnobserved',
  'verify configured image/object storage and batch provider files from both nodes; configuration alone does not prove availability': 'sharedStorage',
  'verify Linux watchdog and physical/network failure fencing on both hosts': 'fencing',
  'verify the third quorum member is in an independent fault domain': 'witness',
  'verify the public hostname automatically switches origins and never replays generation POSTs': 'ingress',
  'restore a backup into an isolated database and verify schema, ledger and object references': 'restore'
}
const values: Record<string, readonly string[]> = {
  roles: ['primary', 'worker', 'standalone'],
  redisRoles: ['master', 'slave', 'replica', 'sentinel'],
  kinds: ['node_started', 'control_changed', 'alert_raised', 'alert_resolved', 'billing_review', 'balance_cache_invalidated'],
  alerts: ['offline', 'database', 'redis', 'quorum', 'single_database_copy', 'replication_lag', 'disk_space', 'capacity', 'ingress_pending', 'primary', 'versions', 'settings'],
  checks: ['cluster', 'nodes', 'versions', 'quorum', 'wal_archive', 'archive_delivery', 'shared_storage', 'fencing', 'independent_witness', 'external_ingress', 'restore_drill'],
  statuses: ['pass', 'fail', 'warning', 'unknown', 'manual', 'disabled'],
  requests: ['running', 'finished', 'needs_review'],
  severities: ['info', 'warning', 'critical', 'error'],
  billing: ['no_evidence', 'unknown', 'settlement_pending', 'usage_missing', 'settled', 'cost_mismatch']
}

export function createClusterText(t: Translate) {
  function label(group: string, value: string) {
    if (!Object.prototype.hasOwnProperty.call(values, group) || !values[group]?.includes(value)) return t('cluster.log.unknownValue', { value })
    return t(group === 'billing' ? `cluster.${value}` : `cluster.log.${group}.${value}`)
  }
  function scope(nodeID: string, alertKey?: string) {
    if (nodeID) return nodeID
    const separator = alertKey?.lastIndexOf(':') ?? -1
    const prefix = separator > 0 ? alertKey!.slice(0, separator) : ''
    return prefix && prefix !== 'cluster' ? prefix : t('cluster.log.clusterScope')
  }
  function alertName(key: string) {
    return label('alerts', key.slice(key.lastIndexOf(':') + 1))
  }
  function message(raw: string) {
    if (Object.prototype.hasOwnProperty.call(messages, raw)) return t(`cluster.log.messages.${messages[raw]}`)
    const primary = /^expected one application primary; observed (\d+)$/.exec(raw)
    if (primary) return t('cluster.log.messages.primaryCount', { count: primary[1]! })
    return t('cluster.log.untranslatedMessage')
  }
  function event(event: Pick<ClusterEvent, 'kind' | 'message'>) {
    const raw = event.message
    if (event.kind === 'node_started') {
      const match = /^node started with role ([a-z_]+)$/.exec(raw)
      if (match) return t('cluster.log.messages.nodeStarted', { role: label('roles', match[1]!) })
    }
    if (event.kind === 'control_changed') {
      const match = /^maintenance=(true|false) capacity=(\d+) weight=(\d+): ([\s\S]*)$/.exec(raw)
      if (match) return t('cluster.log.messages.controlChanged', {
        maintenance: t(match[1] === 'true' ? 'cluster.log.enabled' : 'cluster.log.disabled'),
        capacity: match[2]!, weight: match[3]!, reason: match[4]!
      })
    }
    if (event.kind === 'alert_resolved') {
      const separator = raw.lastIndexOf(':')
      if (separator > 0) return t('cluster.log.messages.alertResolved', { node: raw.slice(0, separator) === 'cluster' ? t('cluster.log.clusterScope') : raw.slice(0, separator), alert: alertName(raw) })
    }
    if (event.kind === 'billing_review') {
      const match = /^([0-9a-f-]+): review recorded$/i.exec(raw)
      if (match) return t('cluster.log.messages.billingReview', { id: match[1]! })
    }
    if (event.kind === 'balance_cache_invalidated') {
      const match = /^user (\d+); rebuild from database on next use$/.exec(raw)
      if (match) return t('cluster.log.messages.balanceInvalidated', { id: match[1]! })
    }
    return message(raw)
  }
  return { label, scope, alertName, message, event }
}
