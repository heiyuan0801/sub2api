/**
 * User Announcements API endpoints
 */

import { apiClient } from './client'
import type { BasePaginationResponse, UserAnnouncement } from '@/types'

export async function list(unreadOnly: boolean = false): Promise<UserAnnouncement[]> {
  const { data } = await apiClient.get<UserAnnouncement[]>('/announcements', {
    params: unreadOnly ? { unread_only: 1 } : {}
  })
  return data
}

export async function markRead(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>(`/announcements/${id}/read`)
  return data
}

export async function history(
  page = 1,
  pageSize = 20,
  unreadOnly = false,
  signal?: AbortSignal,
): Promise<BasePaginationResponse<UserAnnouncement> & { unread_count: number }> {
  const { data } = await apiClient.get<BasePaginationResponse<UserAnnouncement> & { unread_count: number }>('/announcements/history', {
    params: { page, page_size: pageSize, unread_only: unreadOnly ? 1 : undefined },
    signal,
  })
  return data
}

export async function detail(id: number, signal?: AbortSignal): Promise<UserAnnouncement> {
  const { data } = await apiClient.get<UserAnnouncement>(`/announcements/${id}`, { signal })
  return data
}

export async function markAllRead(): Promise<{ updated: number }> {
  const { data } = await apiClient.post<{ updated: number }>('/announcements/read-all')
  return data
}

export async function click(id: number): Promise<{ action_url: string }> {
  const { data } = await apiClient.post<{ action_url: string }>(`/announcements/${id}/click`)
  return data
}

export async function preferences(): Promise<{ muted_group_ids: number[] }> {
  const { data } = await apiClient.get<{ muted_group_ids: number[] }>('/announcements/preferences')
  return data
}

export async function setGroupSubscription(groupID: number, subscribed: boolean): Promise<{ group_id: number; subscribed: boolean }> {
  const { data } = await apiClient.put<{ group_id: number; subscribed: boolean }>(
    `/announcements/preferences/${groupID}`,
    { subscribed },
  )
  return data
}

const announcementsAPI = {
  list,
  history,
  detail,
  markAllRead,
  preferences,
  setGroupSubscription,
  click,
  markRead
}

export default announcementsAPI

