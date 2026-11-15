import { apiClient } from '../client'
import type { BasePaginationResponse } from '@/types'
import type { RegistrationSourceQuery, RegistrationSourceSummary, RegistrationSourceUser } from '@/types/operations'

const endpoint = '/admin/registration-sources'

export async function getRegistrationSources(params: RegistrationSourceQuery, signal?: AbortSignal) {
  const { data } = await apiClient.get<RegistrationSourceSummary>(`${endpoint}/summary`, { params, signal })
  return data
}

export async function listRegistrationSourceUsers(params: RegistrationSourceQuery, signal?: AbortSignal) {
  const { data } = await apiClient.get<BasePaginationResponse<RegistrationSourceUser>>(`${endpoint}/users`, { params, signal })
  return data
}

export async function exportRegistrationSources(params: RegistrationSourceQuery) {
  const { data } = await apiClient.get<Blob>(`${endpoint}/export`, { params, responseType: 'blob' })
  return data
}
