import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { announcementsAPI } from '@/api'
import type { UserAnnouncement } from '@/types'

const THROTTLE_MS = 45 * 1000

export const useAnnouncementStore = defineStore('announcements', () => {
  // State
  const announcements = ref<UserAnnouncement[]>([])
  const loading = ref(false)
  const lastFetchTime = ref(0)
  const popupQueue = ref<UserAnnouncement[]>([])
  const currentPopup = ref<UserAnnouncement | null>(null)

  // Session-scoped dedup set — not reactive, used as plain lookup only
  let shownPopupIds = new Set<number>()
  let fetchGeneration = 0
  let sessionGeneration = 0
  const unreadTotal = ref<number | null>(null)

  // Getters
  const unreadCount = computed(() =>
    unreadTotal.value ?? announcements.value.filter((a) => !a.read_at).length
  )

  // Actions
  async function fetchAnnouncements(force = false) {
    const now = Date.now()
    if (!force && lastFetchTime.value > 0 && now - lastFetchTime.value < THROTTLE_MS) {
      return
    }

    // Set immediately to prevent concurrent duplicate requests
    lastFetchTime.value = now
    const generation = ++fetchGeneration

    try {
      loading.value = true
      const history = await announcementsAPI.history()
      if (generation !== fetchGeneration) return
      announcements.value = history.items
      unreadTotal.value = history.unread_count
      enqueueNewPopups()
    } catch (err: any) {
      if (generation !== fetchGeneration) return
      // Revert throttle timestamp on failure so retry is allowed
      lastFetchTime.value = 0
      console.error('Failed to fetch announcements:', err)
    } finally {
      if (generation === fetchGeneration) loading.value = false
    }
  }

  function enqueueNewPopups() {
    const newPopups = announcements.value.filter(
      (a) => a.notify_mode === 'popup' && !a.read_at && !shownPopupIds.has(a.id)
    )
    if (newPopups.length === 0) return

    for (const p of newPopups) {
      if (!popupQueue.value.some((q) => q.id === p.id)) {
        popupQueue.value.push(p)
      }
    }

    if (!currentPopup.value) {
      showNextPopup()
    }
  }

  function showNextPopup() {
    if (popupQueue.value.length === 0) {
      currentPopup.value = null
      return
    }
    currentPopup.value = popupQueue.value.shift()!
    shownPopupIds.add(currentPopup.value.id)
  }

  async function dismissPopup() {
    if (!currentPopup.value) return
    const id = currentPopup.value.id
    currentPopup.value = null

    // Mark as read (fire-and-forget, UI already updated)
    markAsRead(id)

    // Show next popup after a short delay
    if (popupQueue.value.length > 0) {
      setTimeout(() => showNextPopup(), 300)
    }
  }

  async function markAsRead(id: number) {
    const generation = sessionGeneration
    try {
      await announcementsAPI.markRead(id)
      if (generation !== sessionGeneration) return false
      const ann = announcements.value.find((a) => a.id === id)
      if (ann && !ann.read_at) {
        ann.read_at = new Date().toISOString()
        if (unreadTotal.value !== null) unreadTotal.value = Math.max(0, unreadTotal.value - 1)
      } else if (!ann) {
        await fetchAnnouncements(true)
      }
      popupQueue.value = popupQueue.value.filter(a => a.id !== id)
      return true
    } catch (err: unknown) {
      console.error('Failed to mark announcement as read:', err)
      return false
    }
  }

  async function markAllAsRead() {
    if (unreadCount.value === 0) return
    const generation = sessionGeneration
    const unread = announcements.value.filter(a => !a.read_at).map(a => a.id)

    try {
      loading.value = true
      const result = await announcementsAPI.markAllRead()
      if (generation !== sessionGeneration) return
      if (unreadTotal.value !== null) unreadTotal.value = Math.max(0, unreadTotal.value - result.updated)
      const now = new Date().toISOString()
      for (const item of announcements.value) {
        if (unread.includes(item.id) && !item.read_at) item.read_at = now
      }
      currentPopup.value = null
      popupQueue.value = []
      await fetchAnnouncements(true)
    } finally {
      if (generation === sessionGeneration) loading.value = false
    }
  }

  function reset() {
    sessionGeneration++
    fetchGeneration++
    unreadTotal.value = null
    announcements.value = []
    lastFetchTime.value = 0
    shownPopupIds = new Set()
    popupQueue.value = []
    currentPopup.value = null
    loading.value = false
  }

  return {
    // State
    announcements,
    loading,
    currentPopup,
    // Getters
    unreadCount,
    // Actions
    fetchAnnouncements,
    dismissPopup,
    markAsRead,
    markAllAsRead,
    reset,
  }
})
