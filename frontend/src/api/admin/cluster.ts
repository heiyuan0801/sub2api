import { apiClient } from '../client'
export interface Control { maintenance: boolean; max_concurrency: number; weight: number; revision: number; reason: string }
export interface ClusterResources {
 sampled_at: string; hostname: string; os: string; arch: string; cpu_cores: number
 cpu_percent: number | null; cpu_window_seconds: number | null
 load_1: number | null; load_5: number | null; load_15: number | null
 memory_total_bytes: number; memory_used_bytes: number; memory_available_bytes: number; memory_percent: number | null
 process_rss_bytes: number | null; uptime_seconds: number | null
}
export interface ClusterPerformance {
 completed_samples?: number; duration_mean_ms?: number | null; auth_mean_ms?: number | null; routing_mean_ms?: number | null; upstream_mean_ms?: number | null; phase_samples?: number[]
 sampled_at: string; window_seconds: number; requests_per_second: number | null
 first_token_p95_ms: number | null; first_token_samples: number; first_token_overflow: boolean; cache_read_percent: number | null; usage_samples: number
 db_in_use: number; db_open: number; db_limit: number; db_waits: number; db_wait_mean_ms: number | null
 redis_connections: number; redis_idle: number; redis_pool_timeouts: number
}
export interface ClusterIngressTraffic { sampled_at: string; queued: number; active: number; bytes_in: number; bytes_out: number; cross_node_bytes: number | null }
export interface ClusterVerification {
 storage?: { checked_at: string; status: 'pass' | 'fail'; scope: 'local_lab' | 'cross_node_probe' }
 checked_at: string; scope: 'local_lab' | 'external_probe'; artifacts_match: boolean | null
 endpoints: { id: string; status: 'pass' | 'fail'; latency_ms: number | null; reason?: string; certificate_expires_at: string | null }[]
 restore_checked_at?: string | null; restore_scope?: string
}
export interface ClusterNode {
 id: string; instance_id: string; role: string; version: string; online: boolean; heartbeat_at: string; started_at: string; control: Control
 stats: { runtime_config_ok?: boolean; archive?: { sampled_at: string; known: boolean; enabled: boolean; pending?: number; oldest_pending?: string; last_archived?: string }; performance?: ClusterPerformance; ingress_traffic?: ClusterIngressTraffic; verification?: ClusterVerification; active_requests: number; requests: number; errors: number; rejected: number; heap_bytes: number; disk_total_bytes: number; disk_free_bytes: number; database_ok: boolean; redis_ok: boolean; lease_ok: boolean; control_revision: number; ingress_revision: number; replication_peers: number; replication_known: boolean; sync_replicas: number; replication_lag_bytes: number; redis_role: string; redis_replicas: number; settings_version: string; goroutines?: number; redis_link_ok?: boolean; resources?: ClusterResources; database_latency_ms?: number; redis_latency_ms?: number; ingress_ok?: boolean }
}
export interface ClusterAlert { key: string; node_id: string; severity: string; message: string; raised_at: string }
export interface ClusterEvent { id: number; node_id: string; kind: string; message: string; created_at: string }
export interface ClusterSnapshot { enabled: boolean; ha_enabled: boolean; nodes: ClusterNode[]; alerts: ClusterAlert[]; events: ClusterEvent[]; review_count: number; now: string }
export interface ClusterRequest { id: string; request_id: string; node_id: string; user_id: number; api_key_id: number; route: string; state: string; http_status: number; started_at: string; billing_state: string; expected_cost: string; recorded_cost: string; review_note: string; reviewed_at?: string }
export interface ClusterPreflight { checks: {code: string; status: string; detail: string}[]; checked_at: string; production_verified: boolean }
export interface ClusterBalance { user_id: number; database_balance: string; cache_balance: string | null; cache_status: string; inflight: number }
export interface ClusterHistoryPoint { at: string; cpu?: number; memory?: number; active: number; queued?: number; performance?: ClusterPerformance }
export interface ClusterHistory { node: string; range: '24h' | '7d'; points: ClusterHistoryPoint[] }
const base = '/admin/cluster'
export const clusterApi = {
 history: async (node: string, range: '24h' | '7d', signal?: AbortSignal) => (await apiClient.get<ClusterHistory>(`${base}/history`, { params: { node, range }, signal })).data,
 snapshot: async (signal?: AbortSignal) => (await apiClient.get<ClusterSnapshot>(base, { signal })).data,
 preflight: async () => (await apiClient.get<ClusterPreflight>(`${base}/preflight`)).data,
 update: async (id: string, control: Control) => apiClient.put(`${base}/nodes/${encodeURIComponent(id)}/control`, control),
 requests: async (review: boolean, before?: string) => (await apiClient.get<ClusterRequest[]>(`${base}/requests`, { params: { review, before } })).data,
 review: async (id: string, note: string) => apiClient.post(`${base}/requests/${encodeURIComponent(id)}/review`, { note }),
 balance: async (id: number) => (await apiClient.get<ClusterBalance>(`${base}/users/${id}/balance`)).data,
 invalidate: async (id: number) => apiClient.post(`${base}/users/${id}/invalidate-balance`)
}
