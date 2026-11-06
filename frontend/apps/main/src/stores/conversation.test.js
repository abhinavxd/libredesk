// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { ref } from 'vue'
import { useConversationStore } from './conversation'
import { useUsersStore } from '@main/stores/users'
import { useNotificationStore } from '@main/stores/notification'
import { WebSocketClient } from '@main/websocket'

const { api } = vi.hoisted(() => ({ api: {
  getConversationMessage: vi.fn(), getConversationMessages: vi.fn(), updateAssigneeLastSeen: vi.fn(),
  markConversationAsUnread: vi.fn(), getConversation: vi.fn(), getAllConversations: vi.fn(), getUsersCompact: vi.fn(),
  updateConversationStatus: vi.fn(), updateAssignee: vi.fn(), removeAssignee: vi.fn(), getSidebarCounts: vi.fn()
} }))
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
  api.getSidebarCounts.mockResolvedValue({ data: { data: {} } })
})

const setup = () => {
  const store = useConversationStore()
  store.conversation.data = { uuid: 'conversation-a' }
  store.messages.data.addMessages('conversation-a', [oldMessage], 1, 1)
  store.conversations.data = [{ uuid: 'conversation-a', unread_message_count: 2, last_message_at: reply.created_at }]
  return store
}

describe('sidebar count refreshes', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-01-01T00:00:00Z'))
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
  })

  it('reuses counts across navigation for two minutes', async () => {
    const store = setup()
    await store.fetchSidebarCounts()
    vi.advanceTimersByTime(119_999)
    await store.fetchSidebarCounts()
    expect(api.getSidebarCounts).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(1)
    await store.fetchSidebarCounts()
    expect(api.getSidebarCounts).toHaveBeenCalledTimes(2)
  })

  it('coalesces navigation requests while a count request is running', async () => {
    const store = setup()
    let finish
    api.getSidebarCounts.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
    const first = store.fetchSidebarCounts()
    const second = store.fetchSidebarCounts()
    expect(api.getSidebarCounts).toHaveBeenCalledTimes(1)
    finish({ data: { data: { assigned: 3 } } })
    await Promise.all([first, second])
    expect(store.sidebarCounts.assigned).toBe(3)
    expect(api.getSidebarCounts).toHaveBeenCalledTimes(1)
  })

  it('refreshes after a mutation even when an earlier count request is running', async () => {
    const store = setup()
    store.conversation.data.status = 'Open'
    let finish
    api.getSidebarCounts.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
      .mockResolvedValueOnce({ data: { data: { assigned: 2 } } })
    api.updateConversationStatus.mockResolvedValueOnce()
    const pending = store.fetchSidebarCounts()
    await store.updateStatus('conversation-a', 'Resolved')
    expect(api.getSidebarCounts).toHaveBeenCalledTimes(1)
    finish({ data: { data: { assigned: 3 } } })
    await pending
    await Promise.resolve()
    expect(api.getSidebarCounts).toHaveBeenCalledTimes(2)
    expect(store.sidebarCounts.assigned).toBe(2)
  })

  it('does not fetch counts for unrelated websocket bursts', () => {
    const store = setup()
    const client = new WebSocketClient()
    client.socket = {}
    vi.spyOn(store, 'handleConvPush').mockImplementation(() => {})
    vi.spyOn(store, 'mergeConversationUpdate').mockImplementation(() => {})
    vi.spyOn(useNotificationStore(), 'addNotification').mockImplementation(() => {})
    for (let n = 0; n < 100; n++) {
      for (const type of ['new_conversation', 'conversation_update', 'new_notification']) {
        client.handleMessage({ target: client.socket, data: JSON.stringify({
          type, data: { uuid: 'conversation-a', status: 'Open', notification_type: 'new_reply' }
        }) })
      }
    }
    expect(api.getSidebarCounts).not.toHaveBeenCalled()
  })

  it.each(['assignment', 'mention'])('refreshes real counts immediately for %s notifications', async notificationType => {
    const store = setup()
    store.sidebarCounts.assigned = 8
    store.sidebarCounts.mentioned = 5
    api.getSidebarCounts.mockResolvedValueOnce({ data: { data: { assigned: 2, mentioned: 1 } } })
    const client = new WebSocketClient()
    client.socket = {}
    client.handleMessage({ target: client.socket, data: JSON.stringify({
      type: 'new_notification', data: { notification_type: notificationType, is_read: false }
    }) })

    expect(api.getSidebarCounts).toHaveBeenCalledTimes(1)
    expect(useNotificationStore().unreadCount).toBe(1)
    await vi.advanceTimersByTimeAsync(0)
    expect(store.sidebarCounts.assigned).toBe(2)
    expect(store.sidebarCounts.mentioned).toBe(1)
    await vi.advanceTimersByTimeAsync(20_000)
    expect(api.getSidebarCounts).toHaveBeenCalledTimes(1)
  })

  it('combines assignment and mention bursts and fetches final counts after ten seconds', async () => {
    const store = setup()
    api.getSidebarCounts.mockResolvedValueOnce({ data: { data: { assigned: 1, mentioned: 0 } } })
      .mockResolvedValueOnce({ data: { data: { assigned: 77, mentioned: 12 } } })
    const client = new WebSocketClient()
    client.socket = {}
    for (let n = 0; n < 100; n++) {
      client.handleMessage({ target: client.socket, data: JSON.stringify({
        type: 'new_notification', data: { notification_type: n % 2 ? 'mention' : 'assignment', is_read: false }
      }) })
    }

    expect(useNotificationStore().unreadCount).toBe(100)
    expect(api.getSidebarCounts).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(9_999)
    expect(api.getSidebarCounts).toHaveBeenCalledTimes(1)
    expect(store.sidebarCounts.assigned).toBe(1)
    await vi.advanceTimersByTimeAsync(1)
    expect(api.getSidebarCounts).toHaveBeenCalledTimes(2)
    expect(store.sidebarCounts.assigned).toBe(77)
    expect(store.sidebarCounts.mentioned).toBe(12)
    await vi.advanceTimersByTimeAsync(20_000)
    expect(api.getSidebarCounts).toHaveBeenCalledTimes(2)
  })

  it('limits continuous assignment events to one count refresh every ten seconds', async () => {
    setup()
    const starts = []
    const initialTime = Date.now()
    api.getSidebarCounts.mockImplementation(() => {
      starts.push(Date.now() - initialTime)
      return Promise.resolve({ data: { data: {} } })
    })
    const client = new WebSocketClient()
    client.socket = {}
    for (let n = 0; n < 35; n++) {
      client.handleMessage({ target: client.socket, data: JSON.stringify({
        type: 'new_notification', data: { notification_type: 'assignment', is_read: false }
      }) })
      await vi.advanceTimersByTimeAsync(1_000)
    }

    expect(starts).toEqual([0, 10_000, 20_000, 30_000])
    await vi.advanceTimersByTimeAsync(5_000)
    expect(starts).toEqual([0, 10_000, 20_000, 30_000, 40_000])
    await vi.advanceTimersByTimeAsync(20_000)
    expect(api.getSidebarCounts).toHaveBeenCalledTimes(5)
  })
})

describe('conversation assignees', () => {
  const agent = { id: 2194, first_name: 'Agent', last_name: 'One', avatar_url: '/avatar.png', type: 'agent', availability_status: 'online' }

  it('loads list assignees and preserves selected agent labels without an id lookup', async () => {
    const store = useConversationStore()
    const users = useUsersStore()
    api.getAllConversations.mockResolvedValueOnce({ data: { data: {
      results: [{ uuid: 'conversation-a', assigned_user_id: agent.id, assignee: agent }],
      page: 1, total_pages: 1, total: 1
    } } })

    await store.fetchConversationsList(false, 'all')
    await users.ensureUserIDs([agent.id])

    expect(store.conversations.data).toHaveLength(1)
    expect(users.options).toContainEqual(expect.objectContaining({ value: '2194', label: 'Agent One', avatar_url: '/avatar.png' }))
    expect(api.getUsersCompact).not.toHaveBeenCalled()
  })

  it('loads the assignee when opening a conversation directly', async () => {
    const store = useConversationStore()
    const users = useUsersStore()
    api.getConversation.mockResolvedValueOnce({ data: { data: { uuid: 'conversation-a', assigned_user_id: agent.id, assignee: agent } } })

    await store.fetchConversation('conversation-a')
    await users.ensureUserIDs([agent.id])

    expect(users.users).toEqual([agent])
    expect(api.getUsersCompact).not.toHaveBeenCalled()
  })

  it.each(['new_conversation', 'new_message'])('uses assignee details from %s events without an id lookup', async type => {
    const store = setup()
    const users = useUsersStore()
    const client = new WebSocketClient()
    client.socket = {}
    vi.spyOn(store, 'updateConversationMessage').mockResolvedValue()
    const payload = { uuid: 'conversation-a', assigned_user_id: agent.id, assignee: agent }

    client.handleMessage({ target: client.socket, data: JSON.stringify({
      type, data: type === 'new_message' ? { ...reply, conversation: payload } : payload
    }) })
    await users.ensureUserIDs([agent.id])

    expect(users.users).toEqual([agent])
    expect(store.conversations.data[0].assigned_user_id).toBe(agent.id)
    expect(api.getUsersCompact).not.toHaveBeenCalled()
  })

  it('handles unassignment without adding an empty agent or changing another agent', () => {
    const store = setup()
    const users = useUsersStore()
    store.handleConvPush({ uuid: 'conversation-a', assigned_user_id: agent.id, assignee: { ...agent } })
    store.handleConvPush({ uuid: 'conversation-a', assigned_user_id: null, assignee: null })

    expect(store.conversations.data[0].assigned_user_id).toBeNull()
    expect(users.users).toEqual([agent])
    expect(api.getUsersCompact).not.toHaveBeenCalled()
  })
})

describe('conversation changes during navigation', () => {
  it('keeps a successful status change on its original conversation', async () => {
    const store = setup()
    const notifications = useNotificationStore()
    const markRead = vi.spyOn(notifications, 'markAssignmentAsReadForConversation')
    store.conversation.data.status = 'Open'
    let finish
    api.updateConversationStatus.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))

    const pending = store.updateStatus('conversation-a', 'Resolved')
    expect(store.conversation.data.status).toBe('Resolved')
    store.conversation.data = { uuid: 'conversation-b', status: 'Open' }
    finish()
    await pending

    expect(api.updateConversationStatus).toHaveBeenCalledExactlyOnceWith('conversation-a', { status: 'Resolved' })
    expect(store.conversation.data).toEqual({ uuid: 'conversation-b', status: 'Open' })
    expect(markRead).toHaveBeenCalledExactlyOnceWith('conversation-a')
  })

  it('keeps the next conversation unchanged when an earlier status request fails', async () => {
    const store = setup()
    const markRead = vi.spyOn(useNotificationStore(), 'markAssignmentAsReadForConversation')
    store.conversation.data.status = 'Open'
    let fail
    api.updateConversationStatus.mockImplementationOnce(() => new Promise((resolve, reject) => { fail = reject }))

    const pending = store.updateStatus('conversation-a', 'Resolved')
    store.conversation.data = { uuid: 'conversation-b', status: 'Snoozed' }
    fail(new Error('offline'))
    await pending

    expect(store.conversation.data).toEqual({ uuid: 'conversation-b', status: 'Snoozed' })
    expect(markRead).not.toHaveBeenCalled()
  })

  it('restores the previous status after a failure while the conversation stays open', async () => {
    const store = setup()
    store.conversation.data.status = 'Open'
    api.updateConversationStatus.mockRejectedValueOnce(new Error('offline'))

    await store.updateStatus('conversation-a', 'Resolved')

    expect(store.conversation.data.status).toBe('Open')
  })

  it('uses the requested conversation when another conversation is already open', async () => {
    const store = setup()
    store.conversation.data = { uuid: 'conversation-b', status: 'Open' }
    api.updateConversationStatus.mockResolvedValueOnce()

    await store.updateStatus('conversation-a', 'Resolved')

    expect(api.updateConversationStatus).toHaveBeenCalledExactlyOnceWith('conversation-a', { status: 'Resolved' })
    expect(store.conversation.data).toEqual({ uuid: 'conversation-b', status: 'Open' })
  })

  it.each(['user', 'team'])('keeps a pending %s assignment on its original conversation', async type => {
    const store = setup()
    store.conversation.data = { uuid: 'conversation-a', assigned_user_id: 7, assigned_team_id: 3 }
    const original = store.conversation.data
    let finish
    api.updateAssignee.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))

    const pending = store.updateAssignee(type, { assignee_id: 12 })
    store.conversation.data = { uuid: 'conversation-b', assigned_user_id: 8, assigned_team_id: 4 }
    finish()
    await pending

    expect(api.updateAssignee).toHaveBeenCalledExactlyOnceWith('conversation-a', type, { assignee_id: 12 })
    expect(original[`assigned_${type}_id`]).toBe(12)
    if (type === 'team') expect(original.assigned_user_id).toBeNull()
    expect(store.conversation.data).toEqual({ uuid: 'conversation-b', assigned_user_id: 8, assigned_team_id: 4 })
  })

  it.each(['user', 'team'])('keeps a pending %s removal on its original conversation', async type => {
    const store = setup()
    store.conversation.data = { uuid: 'conversation-a', assigned_user_id: 7, assigned_team_id: 3 }
    const original = store.conversation.data
    let finish
    api.removeAssignee.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))

    const pending = store.removeAssignee(type)
    store.conversation.data = { uuid: 'conversation-b', assigned_user_id: 8, assigned_team_id: 4 }
    finish()
    await pending

    expect(api.removeAssignee).toHaveBeenCalledExactlyOnceWith('conversation-a', type)
    expect(original[`assigned_${type}_id`]).toBeNull()
    expect(store.conversation.data).toEqual({ uuid: 'conversation-b', assigned_user_id: 8, assigned_team_id: 4 })
  })
})

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

  it.each([7, '7'])('keeps the sender unread count stable for replies and private notes (sender: %s)', senderID => {
    const store = setup()
    const client = new WebSocketClient()
    client.socket = {}
    vi.spyOn(store, 'updateConversationMessage').mockResolvedValue()
    for (const privateNote of [false, true]) {
      client.handleMessage({ target: client.socket, data: JSON.stringify({
        type: 'new_message', data: {
          ...reply, type: 'outgoing', sender_type: 'agent', sender_id: senderID,
          private: privateNote, conversation_uuid: 'conversation-a'
        }
      }) })
      expect(store.conversations.data[0].unread_message_count).toBe(2)
    }
    expect(store.updateConversationMessage).toHaveBeenCalledTimes(2)
  })

  it.each([
    { sender_type: 'agent', sender_id: 8 },
    { sender_type: 'contact', sender_id: 7 },
    { sender_type: 'agent', sender_id: 0 },
    { sender_type: 'agent' }
  ])('keeps unread updates for messages that cannot be identified as the sender own message (%j)', sender => {
    const store = setup()
    const client = new WebSocketClient()
    client.socket = {}
    vi.spyOn(store, 'updateConversationMessage').mockResolvedValue()
    client.handleMessage({ target: client.socket, data: JSON.stringify({
      type: 'new_message', data: { ...reply, ...sender, conversation_uuid: 'conversation-a' }
    }) })
    expect(store.conversations.data[0].unread_message_count).toBe(3)
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
