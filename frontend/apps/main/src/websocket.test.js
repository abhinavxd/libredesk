// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { conversations, notifyNewMessage, reviews } = vi.hoisted(() => ({
  reviews: { handleLiveEvent: vi.fn(), fetchCounts: vi.fn() },
  conversations: {
    conversation: { data: null },
    conversations: { data: [] },
    sidebarCounts: { unread: 0, addresses: {} },
    handleConvPush: vi.fn(),
    mergeConversationUpdate: vi.fn(),
    incrementUnread: vi.fn(),
    incrementAddressUnread: vi.fn(),
    isConversationInList: vi.fn(() => true),
    updateConversationMessage: vi.fn(),
    resyncMail: vi.fn(),
    updateTypingStatus: vi.fn(),
    refreshSidebarCounts: vi.fn()
  },
  notifyNewMessage: vi.fn()
}))
vi.mock('./stores/conversation', () => ({ useConversationStore: () => conversations }))
vi.mock('./stores/address', () => ({
  useAddressStore: () => ({ fetchAddresses: vi.fn() })
}))
vi.mock('./stores/browserNotifications', () => ({
  useBrowserNotificationsStore: () => ({ notifyNewMessage })
}))
vi.mock('./stores/users', () => ({ useUsersStore: () => ({}) }))
vi.mock('./stores/user', () => ({ useUserStore: () => ({ userID: 1 }) }))
vi.mock('./stores/review', () => ({ useReviewStore: () => reviews }))
vi.mock('./stores/connection', () => ({ useConnectionStore: () => ({ setConnecting: vi.fn(), setConnectionFailed: vi.fn() }) }))
vi.mock('./composables/useEmitter', () => ({ useEmitter: () => ({}) }))
import { WebSocketClient } from './websocket'

const mail = {
  uuid: 'mail-1',
  conversation_uuid: 'thread-1',
  type: 'incoming',
  sender_type: 'contact',
  preview: 'Hello'
}
let client
beforeEach(() => {
  vi.clearAllMocks()
  conversations.conversation.data = null
  conversations.isViewingConversation = (uuid) => conversations.conversation.data?.uuid === uuid
  conversations.sidebarCounts.unread = 0
  Object.defineProperty(document, 'hidden', { configurable: true, value: false })
  client = new WebSocketClient()
  client.socket = {}
})
const receive = (type, data) =>
  client.handleMessage({ target: client.socket, data: JSON.stringify({ type, data }) })

describe('live mail delivery', () => {
  it('resynchronizes authoritative state when its socket reconnects or another tab marks an address read', () => {
    client.setupPing = vi.fn()
    client.subscribeListReplace = vi.fn()
    client.subscribeToConversation = vi.fn()
    client.reconnectAttempts = 1
    client.handleOpen({ target: client.socket })
    expect(conversations.resyncMail).toHaveBeenCalledTimes(1)
    receive('address_read', { address_id: 6, marked_at: '2026-10-01T10:00:00Z' })
    expect(conversations.resyncMail).toHaveBeenCalledTimes(2)
  })
  it('ignores its own typing while showing another agent typing', () => {
    receive('typing', { user_id: 1, conversation_uuid: 'thread-1', is_typing: true })
    expect(conversations.updateTypingStatus).not.toHaveBeenCalled()
    receive('typing', { user_id: 2, conversation_uuid: 'thread-1', is_typing: true })
    expect(conversations.updateTypingStatus).toHaveBeenCalledTimes(1)
  })
  it('updates the inbox and sends each new-message event to browser notifications', () => {
    receive('new_message', mail)
    expect(conversations.incrementUnread).toHaveBeenCalledWith('thread-1')
    expect(conversations.updateConversationMessage).toHaveBeenCalledWith(mail)
    expect(notifyNewMessage).toHaveBeenCalledWith(mail)
  })
  it('keeps a visible open thread read, but marks arrivals unread in a background tab', () => {
    conversations.conversation.data = { uuid: 'thread-1' }
    receive('new_message', mail)
    expect(conversations.incrementUnread).not.toHaveBeenCalled()
    Object.defineProperty(document, 'hidden', { configurable: true, value: true })
    receive('new_message', mail)
    expect(conversations.incrementUnread).toHaveBeenCalledWith('thread-1')
  })
  it('hands review queue events to the review store', () => {
    const event = { uuid: 'review-1', status: 'pending', author_id: 2 }
    receive('review_created', event)
    receive('review_updated', { ...event, status: 'approved' })
    expect(reviews.handleLiveEvent).toHaveBeenNthCalledWith(1, 'review_created', event)
    expect(reviews.handleLiveEvent).toHaveBeenNthCalledWith(2, 'review_updated', { ...event, status: 'approved' })
  })
  it('does not duplicate alerts for new-conversation broadcasts or stale sockets', () => {
    receive('new_conversation', { uuid: 'thread-1' })
    client.handleMessage({ target: {}, data: JSON.stringify({ type: 'new_message', data: mail }) })
    expect(notifyNewMessage).not.toHaveBeenCalled()
  })
})
