import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import api from '@main/api'

export const useNotificationStore = defineStore('notification', () => {
  const notifications = ref([])
  const unreadCount = ref(0)
  const totalCount = ref(0)
  const isLoading = ref(false)
  const hasMore = ref(true)
  const emitter = useEmitter()
  let statsRequest = 0

  const unreadNotifications = computed(() =>
    notifications.value.filter(n => !n.is_read)
  )

  const readNotifications = computed(() =>
    notifications.value.filter(n => n.is_read)
  )

  // Fetch notifications with pagination
  const fetchNotifications = async (limit = 30, offset = 0, append = false) => {
    isLoading.value = true
    try {
      const data = []
      while (data.length < limit) {
        const pageLimit = Math.min(100, limit - data.length)
        const response = await api.getNotifications({ limit: pageLimit, offset: offset + data.length })
        const page = response?.data?.data || []
        data.push(...page)
        if (page.length < pageLimit) break
      }

      if (append) {
        notifications.value = [...notifications.value, ...data]
      } else {
        notifications.value = data
      }

      hasMore.value = data.length === limit
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    } finally {
      isLoading.value = false
    }
  }

  // Fetch notification stats (unread count)
  const fetchStats = async () => {
    const request = ++statsRequest
    try {
      const response = await api.getNotificationStats()
      const stats = response?.data?.data
      if (stats && request === statsRequest) {
        unreadCount.value = stats.unread_count || 0
        totalCount.value = stats.total_count || 0
      }
    } catch {
      // pass
    }
  }

  // Mark single notification as read
  const markAsRead = async (id) => {
    try {
      await api.markNotificationAsRead(id)
      const notification = notifications.value.find(n => n.id === id)
      if (notification && !notification.is_read) {
        notification.is_read = true
        unreadCount.value = Math.max(0, unreadCount.value - 1)
      }
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  const markAssignmentAsReadForConversation = (conversationUUID) => {
    if (!conversationUUID) return

    let markedCount = 0
    notifications.value.forEach(n => {
      if (
        !n.is_read &&
        n.notification_type === 'assignment' &&
        n.conversation_uuid === conversationUUID
      ) {
        n.is_read = true
        markedCount++
      }
    })

    if (markedCount > 0) {
      unreadCount.value = Math.max(0, unreadCount.value - markedCount)
    }
  }

  const refreshConversationRead = (uuid) => {
    fetchStats()
    if (notifications.value.some(n => !n.is_read && n.conversation_uuid === uuid)) {
      fetchNotifications(Math.max(30, notifications.value.length))
    }
  }

  // Mark all notifications as read
  const markAllAsRead = async () => {
    try {
      await api.markAllNotificationsAsRead()
      notifications.value.forEach(n => {
        n.is_read = true
      })
      unreadCount.value = 0
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  // Delete single notification
  const deleteNotification = async (id) => {
    try {
      await api.deleteNotification(id)
      const index = notifications.value.findIndex(n => n.id === id)
      if (index !== -1) {
        const notification = notifications.value[index]
        if (!notification.is_read) {
          unreadCount.value = Math.max(0, unreadCount.value - 1)
        }
        totalCount.value = Math.max(0, totalCount.value - 1)
        notifications.value.splice(index, 1)
      }
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  // Delete all notifications
  const deleteAll = async () => {
    try {
      await api.deleteAllNotifications()
      notifications.value = []
      unreadCount.value = 0
      totalCount.value = 0
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  // Add a new notification (from WebSocket)
  const addNotification = (notification) => {
    // Add to the beginning of the list
    notifications.value.unshift(notification)
    if (!notification.is_read) unreadCount.value += 1
    totalCount.value += 1
  }

  // Load more notifications (for infinite scroll / load more button)
  const loadMore = async () => {
    if (!hasMore.value || isLoading.value) return
    await fetchNotifications(30, notifications.value.length, true)
  }

  return {
    notifications,
    unreadCount,
    totalCount,
    isLoading,
    hasMore,
    unreadNotifications,
    readNotifications,
    fetchNotifications,
    fetchStats,
    markAsRead,
    refreshConversationRead,
    markAssignmentAsReadForConversation,
    markAllAsRead,
    deleteNotification,
    deleteAll,
    addNotification,
    loadMore
  }
})
