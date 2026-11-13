import { apiClient } from './client'

export type RewardActivityKind = 'check_in' | 'wheel'

interface RewardPrizeSummary {
  name: string
  reward_cents: number
}

export interface RewardPrize extends RewardPrizeSummary {
  weight: number
}

export interface RewardActivityConfig {
  check_in: {
    enabled: boolean
    threshold_cents: number
    min_reward_cents: number
    max_reward_cents: number
  }
  wheel: {
    enabled: boolean
    threshold_cents: number
    daily_limit: number
    prizes: RewardPrize[]
  }
}

export interface RewardActivityClaim {
  id: number
  user_id: number
  day: string
  kind: RewardActivityKind
  reward_cents: number
  prize_index: number
  prize_name: string
  created_at: string
}

export interface RewardActivityStatus {
  config: {
    check_in: RewardActivityConfig['check_in']
    wheel: Omit<RewardActivityConfig['wheel'], 'prizes'> & { prizes: RewardPrizeSummary[] }
  }
  day: string
  spent_cents: number
  checked_in: boolean
  draws_used: number
  draw_spend_used_cents: number
  draws_available: number
  history: RewardActivityClaim[]
}

export async function getRewardActivityStatus(): Promise<RewardActivityStatus> {
  const { data } = await apiClient.get<RewardActivityStatus>('/reward-activities')
  return data
}

export async function claimReward(kind: RewardActivityKind, requestId: string): Promise<RewardActivityClaim> {
  const action = kind === 'check_in' ? 'check-in' : 'draw'
  const { data } = await apiClient.post<RewardActivityClaim>(`/reward-activities/${action}`, { request_id: requestId })
  return data
}

export async function getRewardActivityConfig(): Promise<RewardActivityConfig> {
  const { data } = await apiClient.get<RewardActivityConfig>('/admin/reward-activities')
  return data
}

export async function saveRewardActivityConfig(config: RewardActivityConfig): Promise<RewardActivityConfig> {
  const { data } = await apiClient.put<RewardActivityConfig>('/admin/reward-activities', config)
  return data
}

export async function listRewardActivityClaims(): Promise<RewardActivityClaim[]> {
  const { data } = await apiClient.get<RewardActivityClaim[]>('/admin/reward-activities/claims')
  return data
}
