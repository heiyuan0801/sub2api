import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAnnouncementStore } from '../announcements'
import type { UserAnnouncement } from '@/types'

const { markAllRead, history } = vi.hoisted(() => ({ markAllRead: vi.fn(), history: vi.fn() }))
vi.mock('@/api', () => ({ announcementsAPI: { markAllRead, history } }))

const announcement = (id: number): UserAnnouncement => ({
  id, title: `Notice ${id}`, content: 'Content', notify_mode: 'silent',
  created_at: '2026-09-15T00:00:00Z', updated_at: '2026-09-15T00:00:00Z'
})

beforeEach(() => {
  setActivePinia(createPinia())
  vi.resetAllMocks()
})

describe('mark all announcements read', () => {
  it('marks unread notices beyond the loaded page using one server operation', async () => {
    const store = useAnnouncementStore()
    history.mockResolvedValueOnce({ items: [announcement(1)], unread_count: 30 })
    await store.fetchAnnouncements()
    expect(store.unreadCount).toBe(30)
    markAllRead.mockResolvedValue({ updated: 30 })
    history.mockResolvedValueOnce({ items: [{ ...announcement(1), read_at: '2026-09-16T00:00:00Z' }], unread_count: 0 })
    await store.markAllAsRead()
    expect(markAllRead).toHaveBeenCalledOnce()
    expect(store.unreadCount).toBe(0)
    expect(store.announcements[0].read_at).toBeTruthy()
  })

  it('preserves unread state when the batch request fails', async () => {
    const store = useAnnouncementStore()
    store.announcements = [announcement(1)]
    const error = new Error('offline')
    markAllRead.mockRejectedValue(error)
    await expect(store.markAllAsRead()).rejects.toBe(error)
    expect(store.unreadCount).toBe(1)
    expect(store.loading).toBe(false)
    expect(history).not.toHaveBeenCalled()
  })

  it('keeps a previous session response from changing the current session', async () => {
    const store = useAnnouncementStore()
    store.announcements = [announcement(1)]
    let complete!: () => void
    markAllRead.mockImplementation(() => new Promise(resolve => { complete = () => resolve({ updated: 1 }) }))
    const old = store.markAllAsRead()
    expect(store.loading).toBe(true)
    store.reset()
    history.mockResolvedValue({ items: [announcement(2)], unread_count: 1 })
    await store.fetchAnnouncements()
    complete()
    await old
    expect(store.unreadCount).toBe(1)
    expect(store.announcements[0].id).toBe(2)
    expect(store.announcements[0].read_at).toBeFalsy()
  })
})
