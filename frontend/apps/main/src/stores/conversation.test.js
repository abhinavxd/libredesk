// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { ref } from 'vue'
import { useConversationStore } from './conversation'
import { WebSocketClient } from '@main/websocket'

const { api } = vi.hoisted(() => ({ api: { getConversationMessage: vi.fn(), getConversationMessages: vi.fn(), updateAssigneeLastSeen: vi.fn(), markConversationAsUnread: vi.fn() } }))
vi.mock('@main/api', () => ({ default: api }))
vi.mock('vue-router', () => ({ useRouter: () => ({ currentRoute: ref({ params: { uuid: 'conversation-a' } }) }) }))
vi.mock('@main/stores/user', () => ({ useUserStore: () => ({ userID: 7, user: {}, can: () => true }) }))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit: vi.fn() }) }))
vi.mock('@main/websocket', async importOriginal => ({
  ...await importOriginal(),
  subscribeToConversation: vi.fn(),
  sendTypingIndicator: vi.fn(),
  subscribeListReplace: vi.fn()
}))
vi.mock('@main/i18n', () => ({ getI18n: () => ({ global: { t: key => key } }) }))
vi.mock('@shared-ui/utils/http.js', () => ({ handleHTTPError: error => error }))

const oldMessage = { id: 1, uuid: 'old-message', created_at: '2026-01-01T00:00:00Z', type: 'incoming' }
const reply = { id: 2, uuid: 'new-reply', created_at: '2026-01-01T00:00:01Z', type: 'incoming', conversation_uuid: 'conversation-a' }
const activity = { id: 3, uuid: 'activity', created_at: '2026-01-01T00:00:02Z', type: 'activity', conversation_uuid: 'conversation-a' }

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

const setup = () => {
  const store = useConversationStore()
  store.conversation.data = { uuid: 'conversation-a' }
  store.messages.data.addMessages('conversation-a', [oldMessage], 1, 1)
  store.conversations.data = [{ uuid: 'conversation-a', unread_message_count: 2, last_message_at: reply.created_at }]
  return store
}

describe('conversation read state', () => {
  it('blocks acknowledgements while an earlier reply body is still loading', async () => {
    const store = setup()
    let finish
    api.getConversationMessage.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
    const incoming = store.updateConversationMessage(reply)
    await store.updateConversationMessage(activity)
    expect(store.conversationMessages.at(-1).uuid).toBe('activity')
    expect(store.hasPendingMessages).toBe(true)
    finish({ data: { data: reply } })
    await incoming
    expect(store.hasPendingMessages).toBe(false)
  })

  it('keeps failed message loads unread until refreshed successfully', async () => {
    const store = setup()
    api.getConversationMessage.mockRejectedValueOnce(new Error('offline'))
    api.getConversationMessages.mockRejectedValueOnce(new Error('offline'))
    await store.updateConversationMessage(reply)
    await store.updateConversationMessage(activity)
    expect(store.hasPendingMessages).toBe(true)
    api.getConversationMessages.mockResolvedValueOnce({ data: { data: { results: [oldMessage, reply, activity] } } })
    await store.fetchMessages('conversation-a')
    expect(store.hasPendingMessages).toBe(false)
  })

  it('recovers a missing single-message response from the latest page', async () => {
    const store = setup()
    api.getConversationMessage.mockResolvedValueOnce({ data: { data: null } })
    api.getConversationMessages.mockResolvedValueOnce({ data: { data: { results: [oldMessage, reply] } } })
    await store.updateConversationMessage(reply)
    expect(store.hasPendingMessages).toBe(false)
    expect(store.conversationMessages.at(-1).uuid).toBe(reply.uuid)
  })

  it('stops blocking acknowledgements when a refresh shows the message no longer exists', async () => {
    const store = setup()
    api.getConversationMessage.mockRejectedValueOnce(new Error('not found'))
    api.getConversationMessages.mockResolvedValueOnce({ data: { data: { results: [oldMessage] } } })
    await store.updateConversationMessage(reply)
    expect(store.hasPendingMessages).toBe(false)
  })

  it('clears the list badge only when the read reaches the last message', () => {
    const store = setup()
    store.applyConversationRead({ conversation_uuid: 'conversation-a', last_seen_at: oldMessage.created_at, read_version: 1 })
    expect(store.conversations.data[0].unread_message_count).toBe(2)
    store.applyConversationRead({ conversation_uuid: 'conversation-a', last_seen_at: reply.created_at, read_version: 2 })
    expect(store.conversations.data[0].unread_message_count).toBe(0)
  })

  it('keeps an acknowledged cached reply read when its broadcasts arrive late', async () => {
    const store = setup()
    store.messages.data.addMessage('conversation-a', reply)
    api.updateAssigneeLastSeen.mockResolvedValueOnce({ data: { data: {
      conversation_uuid: 'conversation-a', last_seen_at: reply.created_at, read_version: 1
    } } })
    await store.updateAssigneeLastSeen('conversation-a', reply.uuid)
    const client = new WebSocketClient()
    client.socket = {}

    for (const message of [oldMessage, reply, reply]) {
      client.handleMessage({ target: client.socket, data: JSON.stringify({
        type: 'new_message', data: { ...message, conversation_uuid: 'conversation-a' }
      }) })
    }

    expect(store.conversations.data[0].unread_message_count).toBe(0)
    expect(api.getConversationMessage).not.toHaveBeenCalled()
  })

  it('keeps the latest cutoff when read events arrive out of order', () => {
    const store = setup()
    store.applyConversationRead({ conversation_uuid: 'conversation-a', last_seen_at: reply.created_at, read_version: 2 })
    store.applyConversationRead({ conversation_uuid: 'conversation-a', last_seen_at: oldMessage.created_at, read_version: 1 })
    store.incrementUnread('conversation-a', reply.created_at)
    expect(store.conversations.data[0].unread_message_count).toBe(0)
    store.incrementUnread('conversation-a', activity.created_at)
    expect(store.conversations.data[0].unread_message_count).toBe(1)
  })

  it('preserves a read event received before the conversation enters the list', () => {
    const store = setup()
    const [row] = store.conversations.data
    store.conversations.data = []
    store.applyConversationRead({ conversation_uuid: 'conversation-a', last_seen_at: reply.created_at, read_version: 1 })
    store.conversations.data = [{ ...row, unread_message_count: 0 }]
    store.incrementUnread('conversation-a', reply.created_at)
    expect(store.conversations.data[0].unread_message_count).toBe(0)
  })

  it('increments unread messages in conversations without a saved cutoff', () => {
    const store = setup()
    store.incrementUnread('conversation-a', reply.created_at)
    expect(store.conversations.data[0].unread_message_count).toBe(3)
    store.incrementUnread('missing-conversation', reply.created_at)
    expect(store.conversations.data[0].unread_message_count).toBe(3)
  })

  it('keeps mark unread after an older read event and accepts a fresh reread', async () => {
    const store = setup()
    const read = { conversation_uuid: 'conversation-a', last_seen_at: reply.created_at, read_version: 10 }
    const unread = { ...read, last_seen_at: oldMessage.created_at, read_version: 11, is_unread: true }
    store.applyConversationRead(read)
    api.markConversationAsUnread.mockResolvedValueOnce({ data: { data: unread } })

    await store.markAsUnread('conversation-a')
    expect(store.conversations.data[0].unread_message_count).toBe(1)
    store.applyConversationRead(read)
    store.applyConversationRead({ ...read, last_seen_at: oldMessage.created_at, read_version: 9 })
    expect(store.conversations.data[0].unread_message_count).toBe(1)

    store.applyConversationRead({ ...read, read_version: 12 })
    expect(store.conversations.data[0].unread_message_count).toBe(0)
    store.applyConversationRead(unread)
    expect(store.conversations.data[0].unread_message_count).toBe(0)
  })

  it('ignores a delayed mark unread response after a newer read event', async () => {
    const store = setup()
    let finishUnread
    api.markConversationAsUnread.mockImplementationOnce(() => new Promise(resolve => { finishUnread = resolve }))
    const pending = store.markAsUnread('conversation-a')
    store.applyConversationRead({ conversation_uuid: 'conversation-a', last_seen_at: reply.created_at, read_version: 3 })
    finishUnread({ data: { data: {
      conversation_uuid: 'conversation-a', last_seen_at: oldMessage.created_at, read_version: 2, is_unread: true
    } } })
    await pending
    expect(store.conversations.data[0].unread_message_count).toBe(0)
  })

  it('applies mark unread from another tab and resets the cutoff', () => {
    const store = setup()
    store.applyConversationRead({ conversation_uuid: 'conversation-a', last_seen_at: reply.created_at, read_version: 1 })
    const client = new WebSocketClient()
    client.socket = {}
    client.handleMessage({ target: client.socket, data: JSON.stringify({ type: 'conversation_read', data: {
      conversation_uuid: 'conversation-a', last_seen_at: oldMessage.created_at, read_version: 2, is_unread: true
    } }) })
    expect(store.conversations.data[0].unread_message_count).toBe(1)
    store.incrementUnread('conversation-a', activity.created_at)
    expect(store.conversations.data[0].unread_message_count).toBe(2)
  })

  it.each([false, true])('retains an arriving reply while the first page loads (page includes reply: %s)', async includesReply => {
    const store = setup()
    store.messages.data.purgeConversation('conversation-a')
    let finishPage
    api.getConversationMessages.mockImplementationOnce(() => new Promise(resolve => { finishPage = resolve }))
    api.getConversationMessage.mockResolvedValueOnce({ data: { data: reply } })
    const firstPage = store.fetchMessages('conversation-a')

    await store.updateConversationMessage(reply)
    expect(store.conversationMessages.map(message => message.uuid)).toEqual([reply.uuid])
    expect(store.hasPendingMessages).toBe(false)
    expect(store.messages.fetching).toBe(true)
    expect(store.messages.data.getLastFetchedPage('conversation-a')).toBe(0)

    finishPage({ data: { data: { results: includesReply ? [oldMessage, reply] : [oldMessage], page: 1, total_pages: 2 } } })
    await firstPage
    expect(store.conversationMessages.map(message => message.uuid)).toEqual([oldMessage.uuid, reply.uuid])
    expect(store.messages.data.getLastFetchedPage('conversation-a')).toBe(1)
    expect(store.currentConversationHasMoreMessages).toBe(true)
  })

  it('loads history when only a live reply has been cached', async () => {
    const store = setup()
    store.messages.data.purgeConversation('conversation-a')
    api.getConversationMessage.mockResolvedValueOnce({ data: { data: reply } })
    await store.updateConversationMessage(reply)
    api.getConversationMessages.mockResolvedValueOnce({ data: { data: { results: [oldMessage], page: 1, total_pages: 1 } } })
    await store.fetchMessages('conversation-a')
    expect(api.getConversationMessages).toHaveBeenCalledWith('conversation-a', { page: 1, page_size: 30 })
    expect(store.conversationMessages.map(message => message.uuid)).toEqual([oldMessage.uuid, reply.uuid])
  })
})
