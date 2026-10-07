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
})
