/**
 * Admin Announcements API endpoints
 */

import type { NotificationDelivery, NotificationStats } from '@/types/operations'
import { apiClient } from '../client'
import type {
  Announcement,
  AnnouncementUserReadStatus,
  BasePaginationResponse,
  CreateAnnouncementRequest,
  UpdateAnnouncementRequest
} from '@/types'

export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    status?: string
    search?: string
    sort_by?: string
    sort_order?: 'asc' | 'desc'
  },
  options?: {
    signal?: AbortSignal
  }
): Promise<BasePaginationResponse<Announcement>> {
  const { data } = await apiClient.get<BasePaginationResponse<Announcement>>('/admin/announcements', {
    params: { page, page_size: pageSize, ...filters },
    signal: options?.signal
  })
  return data
}

export async function getById(id: number): Promise<Announcement> {
  const { data } = await apiClient.get<Announcement>(`/admin/announcements/${id}`)
  return data
}

export async function create(request: CreateAnnouncementRequest): Promise<Announcement> {
  const { data } = await apiClient.post<Announcement>('/admin/announcements', request)
  return data
}

export async function update(id: number, request: UpdateAnnouncementRequest): Promise<Announcement> {
  const { data } = await apiClient.put<Announcement>(`/admin/announcements/${id}`, request)
  return data
}

export async function deleteAnnouncement(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/announcements/${id}`)
  return data
}

export async function getReadStatus(
  id: number,
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    search?: string
    sort_by?: string
    sort_order?: 'asc' | 'desc'
  },
  options?: {
    signal?: AbortSignal
  }
): Promise<BasePaginationResponse<AnnouncementUserReadStatus>> {
  const { data } = await apiClient.get<BasePaginationResponse<AnnouncementUserReadStatus>>(
    `/admin/announcements/${id}/read-status`,
    {
      params: { page, page_size: pageSize, ...filters },
      signal: options?.signal
    }
  )
  return data
}

export async function publish(id: number, startsAt?: number): Promise<Announcement> {
  const { data } = await apiClient.post<Announcement>(`/admin/announcements/${id}/publish`, { starts_at: startsAt })
  return data
}

async function withdraw(id: number): Promise<void> {
  await apiClient.post(`/admin/announcements/${id}/withdraw`)
}

async function retry(id: number): Promise<void> {
  await apiClient.post(`/admin/announcements/${id}/retry`)
}

async function stats(id: number, signal?: AbortSignal): Promise<NotificationStats> {
  const { data } = await apiClient.get<NotificationStats>(`/admin/announcements/${id}/stats`, { signal })
  return data
}

async function deliveries(id: number, page: number, pageSize: number, signal?: AbortSignal) {
  const { data } = await apiClient.get<BasePaginationResponse<NotificationDelivery>>(`/admin/announcements/${id}/deliveries`, {
    params: { page, page_size: pageSize }, signal,
  })
  return data
}

const announcementsAPI = {
  list,
  getById,
  create,
  update,
  delete: deleteAnnouncement,
  getReadStatus,
  publish,
  withdraw,
  retry,
  stats,
  deliveries
}

export default announcementsAPI
