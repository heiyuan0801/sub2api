import { apiClient } from '../client'
export interface Control { maintenance: boolean; max_concurrency: number; weight: number; revision: number; reason: string }
export interface ClusterNode {
 id: string; instance_id: string; role: string; version: string; online: boolean; heartbeat_at: string; started_at: string; control: Control
 stats: { active_requests: number; requests: number; errors: number; rejected: number; heap_bytes: number; disk_total_bytes: number; disk_free_bytes: number; database_ok: boolean; redis_ok: boolean; lease_ok: boolean; control_revision: number; ingress_revision: number; replication_peers: number; replication_known: boolean; sync_replicas: number; replication_lag_bytes: number; redis_role: string; redis_replicas: number; settings_version: string }
}
export interface ClusterAlert { key: string; node_id: string; severity: string; message: string; raised_at: string }
export interface ClusterEvent { id: number; node_id: string; kind: string; message: string; created_at: string }
export interface ClusterSnapshot { enabled: boolean; ha_enabled: boolean; nodes: ClusterNode[]; alerts: ClusterAlert[]; events: ClusterEvent[]; review_count: number; now: string }
export interface ClusterRequest { id: string; request_id: string; node_id: string; user_id: number; api_key_id: number; route: string; state: string; http_status: number; started_at: string; billing_state: string; expected_cost: string; recorded_cost: string; review_note: string; reviewed_at?: string }
export interface ClusterPreflight { checks: {code: string; status: string; detail: string}[]; checked_at: string; production_verified: boolean }
export interface ClusterBalance { user_id: number; database_balance: string; cache_balance: string | null; cache_status: string; inflight: number }
const base = '/admin/cluster'
export const clusterApi = {
 snapshot: async (signal?: AbortSignal) => (await apiClient.get<ClusterSnapshot>(base, { signal })).data,
 preflight: async () => (await apiClient.get<ClusterPreflight>(`${base}/preflight`)).data,
 update: async (id: string, control: Control) => apiClient.put(`${base}/nodes/${encodeURIComponent(id)}/control`, control),
 requests: async (review: boolean, before?: string) => (await apiClient.get<ClusterRequest[]>(`${base}/requests`, { params: { review, before } })).data,
 review: async (id: string, note: string) => apiClient.post(`${base}/requests/${encodeURIComponent(id)}/review`, { note }),
 balance: async (id: number) => (await apiClient.get<ClusterBalance>(`${base}/users/${id}/balance`)).data,
 invalidate: async (id: number) => apiClient.post(`${base}/users/${id}/invalidate-balance`)
}
