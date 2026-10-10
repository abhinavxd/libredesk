// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useNotificationStore } from './notification'

const { getNotificationStats, getNotifications } = vi.hoisted(() => ({
  getNotificationStats: vi.fn(),
  getNotifications: vi.fn()
}))
vi.mock('@main/api', () => ({ default: { getNotificationStats, getNotifications } }))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit: vi.fn() }) }))
vi.mock('@shared-ui/utils/http.js', () => ({ handleHTTPError: error => error }))

const notification = (id, conversationUUID, isRead = false) => ({ id, conversation_uuid: conversationUUID, is_read: isRead })

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  getNotificationStats.mockResolvedValue({ data: { data: { unread_count: 0, total_count: 0 } } })
  getNotifications.mockResolvedValue({ data: { data: [] } })
})

describe('conversation read refresh', () => {
  it('reloads the list only when it shows an unread alert for the conversation', () => {
    const store = useNotificationStore()
    store.notifications = [notification(1, 'conversation-a', true), notification(2, 'conversation-b')]
    store.refreshConversationRead('conversation-a')
    expect(getNotificationStats).toHaveBeenCalledTimes(1)
    expect(getNotifications).not.toHaveBeenCalled()
    store.refreshConversationRead('conversation-b')
    expect(getNotifications).toHaveBeenCalledWith({ limit: 30, offset: 0 })
  })

  it('ignores an older unread-count response after a newer one', async () => {
    const store = useNotificationStore()
    let finishOld
    getNotificationStats.mockImplementationOnce(() => new Promise(resolve => { finishOld = resolve }))
    const oldRequest = store.fetchStats()
    store.refreshConversationRead('conversation-a')
    await Promise.resolve()
    finishOld({ data: { data: { unread_count: 10, total_count: 10 } } })
    await oldRequest
    expect(store.unreadCount).toBe(0)
  })

  it.each([30, 60, 120, 135])('retains %i loaded alerts after a read refresh', async loadedCount => {
    const store = useNotificationStore()
    const alerts = Array.from({ length: loadedCount + 5 }, (_, index) =>
      notification(index + 1, index === loadedCount - 1 ? 'conversation-a' : 'conversation-b')
    )
    store.notifications = alerts.slice(0, loadedCount)
    const refreshed = alerts.map(alert => ({ ...alert, is_read: alert.conversation_uuid === 'conversation-a' }))
    getNotifications.mockImplementation(({ limit, offset }) => Promise.resolve({ data: { data:
      refreshed.slice(offset, offset + (limit > 100 ? 20 : limit))
    } }))

    store.refreshConversationRead('conversation-a')
    await vi.waitFor(() => expect(store.isLoading).toBe(false))

    expect(store.notifications).toEqual(refreshed.slice(0, loadedCount))
    expect(store.notifications[loadedCount - 1].is_read).toBe(true)
    expect(store.notifications[0].is_read).toBe(false)
    expect(store.hasMore).toBe(true)
    for (const [params] of getNotifications.mock.calls) {
      expect(params.limit).toBeLessThanOrEqual(100)
    }

    await store.loadMore()
    expect(getNotifications).toHaveBeenLastCalledWith({ limit: 30, offset: loadedCount })
    expect(store.notifications).toEqual(refreshed)
    expect(store.hasMore).toBe(false)
  })

  it('stops loading when the refreshed range ends before the requested count', async () => {
    const store = useNotificationStore()
    const alerts = Array.from({ length: 60 }, (_, index) => notification(index + 1, 'conversation-a'))
    store.notifications = alerts
    getNotifications.mockResolvedValueOnce({ data: { data: alerts.slice(0, 45) } })

    store.refreshConversationRead('conversation-a')
    await vi.waitFor(() => expect(store.isLoading).toBe(false))

    expect(store.notifications).toHaveLength(45)
    expect(store.hasMore).toBe(false)
    expect(getNotifications).toHaveBeenCalledTimes(1)
  })

  it('keeps loaded pages when fetching a later refresh page fails', async () => {
    const store = useNotificationStore()
    const alerts = Array.from({ length: 120 }, (_, index) => notification(index + 1, 'conversation-a'))
    store.notifications = alerts
    getNotifications.mockResolvedValueOnce({ data: { data: alerts.slice(0, 100) } })
    getNotifications.mockRejectedValueOnce(new Error('offline'))

    store.refreshConversationRead('conversation-a')
    await vi.waitFor(() => expect(store.isLoading).toBe(false))

    expect(store.notifications).toEqual(alerts)
    expect(getNotifications).toHaveBeenLastCalledWith({ limit: 20, offset: 100 })
  })

  it('discards an older list response after a newer read refresh finishes', async () => {
    const store = useNotificationStore()
    const alerts = [notification(1, 'conversation-a'), notification(2, 'conversation-b')]
    const readAlerts = alerts.map(alert => ({ ...alert, is_read: true }))
    store.notifications = alerts
    let finishOld
    getNotifications.mockImplementationOnce(() => new Promise(resolve => { finishOld = resolve }))
    getNotifications.mockResolvedValueOnce({ data: { data: readAlerts } })

    const oldRequest = store.fetchNotifications()
    store.refreshConversationRead('conversation-b')
    await vi.waitFor(() => expect(store.isLoading).toBe(false))
    expect(store.notifications).toEqual(readAlerts)

    finishOld({ data: { data: alerts } })
    await oldRequest
    expect(store.notifications).toEqual(readAlerts)
  })

  it('keeps loading until the newer request finishes when an older one returns first', async () => {
    const store = useNotificationStore()
    let finishOld, finishNew
    getNotifications.mockImplementationOnce(() => new Promise(resolve => { finishOld = resolve }))
    getNotifications.mockImplementationOnce(() => new Promise(resolve => { finishNew = resolve }))
    const oldRequest = store.fetchNotifications()
    const newRequest = store.fetchNotifications()

    finishOld({ data: { data: [notification(1, 'conversation-a')] } })
    await oldRequest
    expect(store.isLoading).toBe(true)
    expect(store.notifications).toEqual([])

    finishNew({ data: { data: [notification(1, 'conversation-a', true)] } })
    await newRequest
    expect(store.isLoading).toBe(false)
    expect(store.notifications[0].is_read).toBe(true)
  })

  it('discards an older refresh spanning multiple pages', async () => {
    const store = useNotificationStore()
    const alerts = Array.from({ length: 120 }, (_, index) => notification(index + 1, 'conversation-a'))
    const readAlerts = alerts.map(alert => ({ ...alert, is_read: true }))
    let finishOldPage
    getNotifications.mockResolvedValueOnce({ data: { data: alerts.slice(0, 100) } })
    getNotifications.mockImplementationOnce(() => new Promise(resolve => { finishOldPage = resolve }))
    getNotifications.mockResolvedValueOnce({ data: { data: readAlerts.slice(0, 100) } })
    getNotifications.mockResolvedValueOnce({ data: { data: readAlerts.slice(100) } })
    const oldRequest = store.fetchNotifications(120)
    await vi.waitFor(() => expect(getNotifications).toHaveBeenCalledTimes(2))

    await store.fetchNotifications(120)
    finishOldPage({ data: { data: alerts.slice(100) } })
    await oldRequest
    expect(store.notifications).toEqual(readAlerts)
  })
})
