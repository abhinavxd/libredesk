// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
const { api, route } = vi.hoisted(() => ({
  route: { value: { params: { uuid: 'A', addressID: '6' } } },
  api: {
    getSidebarCounts: vi.fn(),
    getAddressConversations: vi.fn(),
    getConversation: vi.fn(),
    getConversationMessages: vi.fn(),
    markAddressAsRead: vi.fn()
  }
}))
vi.mock('vue-router', () => ({ useRouter: () => ({ currentRoute: route }) }))
vi.mock('@main/api', () => ({ default: api }))
vi.mock('@main/websocket', () => ({
  subscribeToConversation: vi.fn(),
  sendTypingIndicator: vi.fn(),
  subscribeListReplace: vi.fn()
}))
vi.mock('@main/i18n', () => ({ getI18n: () => ({ global: { t: (key) => key } }) }))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit: vi.fn() }) }))
import { useConversationStore } from './conversation'
const response = (data) => ({ data: { data } })
const list = (results) => response({ results, page: 1, total_pages: 1, total: results.length })
let store
beforeEach(() => {
  vi.useFakeTimers()
  vi.resetAllMocks()
  setActivePinia(createPinia())
  route.value.params = { uuid: 'A', addressID: '6' }
  store = useConversationStore()
  store.conversation.data = { uuid: 'A', address_id: 6, status: 'Open' }
  api.getSidebarCounts.mockResolvedValue(response({ unread: 1, addresses: { 6: 1 } }))
  api.getConversation.mockResolvedValue(response({ uuid: 'A', address_id: 6, status: 'Open' }))
  api.getAddressConversations.mockResolvedValue(
    list([{ uuid: 'A', address_id: 6, status: 'Open', unread_message_count: 1 }])
  )
})
afterEach(() => {
  store.$dispose()
  vi.clearAllTimers()
  vi.useRealTimers()
})

describe('mail synchronization', () => {
  it('keeps other named agents typing when one stops and expires stale actors', async () => {
    store.updateTypingStatus({
      conversation_uuid: 'A',
      user_id: 2,
      user_name: 'Alex',
      is_typing: true
    })
    store.updateTypingStatus({
      conversation_uuid: 'A',
      user_id: 3,
      user_name: 'Sam',
      is_typing: true
    })
    expect(store.typingNames('A')).toBe('Alex, Sam')
    store.updateTypingStatus({ conversation_uuid: 'A', user_id: 2, is_typing: false })
    expect(store.typingNames('A')).toBe('Sam')
    expect(store.conversation.isTyping).toBe(true)
    await vi.advanceTimersByTimeAsync(5000)
    expect(store.typingNames('A')).toBe('')
    expect(store.conversation.isTyping).toBe(false)
  })

  it('refetches missed mail after reconnect, replacing stale pagination and invalidating other cached threads', async () => {
    store.messages.data.addMessages('A', [{ uuid: 'old-A' }], 1, 1)
    store.messages.data.addMessages('B', [{ uuid: 'old-B' }], 1, 1)
    api.getConversationMessages.mockResolvedValue(
      response({ results: [{ uuid: 'new-A' }], page: 1, total_pages: 3 })
    )
    await store.resyncMail()
    expect(store.messages.data.getAllPagesMessages('A').map((m) => m.uuid)).toEqual(['new-A'])
    expect(store.messages.data.hasMore('A')).toBe(true)
    api.getConversationMessages.mockResolvedValue(
      response({ results: [{ uuid: 'new-B' }], page: 1, total_pages: 1 })
    )
    await store.fetchMessages('B')
    expect(store.messages.data.getAllPagesMessages('B').map((m) => m.uuid)).toEqual(['new-B'])
  })

  it('marks only the selected address and preserves mail arriving after the cutoff', async () => {
    route.value.params = { addressID: '7' }
    store.conversations.addressID = 7
    store.conversations.data = [
      {
        uuid: 'old-A',
        address_id: 6,
        unread_message_count: 3,
        last_message_at: '2026-10-01T10:00:00Z'
      },
      {
        uuid: 'new-A',
        address_id: 6,
        unread_message_count: 1,
        last_message_at: '2026-10-01T10:02:00Z'
      },
      { uuid: 'B', address_id: 7, unread_message_count: 2 }
    ]
    api.markAddressAsRead.mockResolvedValue(
      response({ address_id: 6, marked_at: '2026-10-01T10:01:00Z' })
    )
    await store.markAddressAsRead(6)
    expect(api.markAddressAsRead).toHaveBeenCalledWith(6)
    expect(store.conversations.data.map((row) => row.unread_message_count)).toEqual([0, 1, 2])
    expect(api.getAddressConversations).not.toHaveBeenCalled()
    expect(api.getSidebarCounts).toHaveBeenCalled()
  })

  it('keeps the list unread when mark-as-read fails', async () => {
    store.conversations.data = [{ uuid: 'A', address_id: 6, unread_message_count: 3 }]
    api.markAddressAsRead.mockRejectedValue(new Error('offline'))
    await expect(store.markAddressAsRead(6)).rejects.toThrow('offline')
    expect(store.conversations.data[0].unread_message_count).toBe(3)
  })
})
