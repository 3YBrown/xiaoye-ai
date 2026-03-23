import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import axios from 'axios'

const api = axios.create({ baseURL: '/api/user' })

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

const normalizeTimestamp = (value) => {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string') {
    const ts = new Date(value).getTime()
    if (Number.isFinite(ts)) return ts
  }
  return Date.now()
}

const mapNotification = (row) => ({
  id: String(row?.id ?? ''),
  title: row?.title || '',
  summary: row?.summary || '',
  content: row?.content || '',
  createdAt: normalizeTimestamp(row?.created_at),
  read: Boolean(row?.is_read)
})

export const useNotificationStore = defineStore('notifications', () => {
  const notifications = ref([])
  const unreadCount = ref(0)
  const total = ref(0)
  const loading = ref(false)
  const loaded = ref(false)

  const sortedNotifications = computed(() =>
    [...notifications.value].sort((a, b) => b.createdAt - a.createdAt)
  )

  async function loadNotifications(force = false) {
    const token = localStorage.getItem('token')
    if (!token) {
      reset()
      return
    }
    if (loaded.value && !force) return

    loading.value = true
    try {
      const { data } = await api.get('/notifications', {
        params: { limit: 50, offset: 0 }
      })
      notifications.value = (data?.items || []).map(mapNotification)
      unreadCount.value = Number(data?.unread_count || 0)
      total.value = Number(data?.total || notifications.value.length)
      loaded.value = true
    } catch (error) {
      console.error('Failed to load notifications:', error)
      notifications.value = []
      unreadCount.value = 0
      total.value = 0
      loaded.value = false
    } finally {
      loading.value = false
    }
  }

  async function markAsRead(id) {
    const target = notifications.value.find((item) => item.id === String(id))
    if (!target || target.read) return

    target.read = true
    unreadCount.value = Math.max(0, unreadCount.value - 1)

    try {
      await api.post(`/notifications/${id}/read`)
    } catch (error) {
      console.error('Failed to mark notification as read:', error)
      await loadNotifications(true)
    }
  }

  async function markAllAsRead() {
    const hasUnread = notifications.value.some((item) => !item.read)
    if (!hasUnread) return

    notifications.value = notifications.value.map((item) => ({ ...item, read: true }))
    unreadCount.value = 0

    try {
      await api.post('/notifications/read-all')
    } catch (error) {
      console.error('Failed to mark all notifications as read:', error)
      await loadNotifications(true)
    }
  }

  function reset() {
    notifications.value = []
    unreadCount.value = 0
    total.value = 0
    loading.value = false
    loaded.value = false
  }

  return {
    notifications,
    sortedNotifications,
    unreadCount,
    total,
    loading,
    loaded,
    loadNotifications,
    markAsRead,
    markAllAsRead,
    reset
  }
})
