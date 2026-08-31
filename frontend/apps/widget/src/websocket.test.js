// @vitest-environment jsdom
import { describe, test, expect, beforeEach, vi } from 'vitest'

const chatStore = { addMessageToConversation: vi.fn(), currentConversation: null }
const widgetStore = {
  setConnectionFailed: vi.fn(),
  setConnecting: vi.fn(),
  setConnected: vi.fn(),
  isOpen: false,
  isInChatView: false
}

vi.mock('./store/chat.js', () => ({ useChatStore: () => chatStore }))
vi.mock('./store/widget.js', () => ({ useWidgetStore: () => widgetStore }))
vi.mock('@shared-ui/composables/useNotificationSound.js', () => ({ playNotificationSound: vi.fn() }))

const sockets = []

class FakeSocket {
  constructor() {
    this.readyState = FakeSocket.CONNECTING
    this.listeners = {}
    this.closed = false
    this.sent = []
    sockets.push(this)
  }

  addEventListener(type, handler) {
    this.listeners[type] = handler
  }

  send(payload) {
    this.sent.push(payload)
  }

  close() {
    this.closed = true
  }

  emit(type, data) {
    this.listeners[type]({ target: this, data })
  }
}
FakeSocket.CONNECTING = 0
FakeSocket.OPEN = 1

const newMessageFrame = (body) =>
  JSON.stringify({
    type: 'new_message',
    data: { conversation_uuid: 'convo-1', content: body, author: { type: 'agent' } }
  })

let WidgetWebSocketClient

describe('Widget websocket client', () => {
  beforeEach(async () => {
    sockets.length = 0
    vi.clearAllMocks()
    vi.stubGlobal('WebSocket', FakeSocket)
    vi.useFakeTimers()
    WidgetWebSocketClient = (await import('./websocket.js')).WidgetWebSocketClient
  })

  const connected = () => {
    const client = new WidgetWebSocketClient()
    client.init('token', 'inbox-uuid')
    return client
  }

  test('closes the previous socket before opening a new one', () => {
    const client = connected()
    const first = sockets[0]

    client.reconnect()
    vi.advanceTimersByTime(2000)

    expect(sockets).toHaveLength(2)
    expect(first.closed).toBe(true)
  })

  test('a superseded socket cannot trigger another reconnect', () => {
    const client = connected()
    const first = sockets[0]

    client.reconnect()
    vi.advanceTimersByTime(2000)
    expect(sockets).toHaveLength(2)

    first.emit('close')
    vi.advanceTimersByTime(10000)

    expect(sockets).toHaveLength(2)
  })

  test('a superseded socket cannot deliver messages into the store', () => {
    const client = connected()
    const first = sockets[0]

    client.reconnect()
    vi.advanceTimersByTime(2000)

    first.emit('message', newMessageFrame('from the dead socket'))

    expect(chatStore.addMessageToConversation).not.toHaveBeenCalled()
  })

  test('the current socket still delivers messages', () => {
    connected()

    sockets[0].emit('message', newMessageFrame('hello'))

    expect(chatStore.addMessageToConversation).toHaveBeenCalledWith(
      'convo-1',
      expect.objectContaining({ content: 'hello' })
    )
  })

  test('a superseded socket cannot mark the connection healthy', () => {
    const client = connected()
    const first = sockets[0]

    client.reconnect()
    vi.advanceTimersByTime(2000)
    vi.clearAllMocks()

    first.emit('open')

    expect(widgetStore.setConnectionFailed).not.toHaveBeenCalled()
  })
})
