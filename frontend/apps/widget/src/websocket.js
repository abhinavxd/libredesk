// Widget WebSocket message types (matching backend constants)
import { useChatStore } from './store/chat.js'

export const WS_EVENT = {
  JOIN: 'join',
  MESSAGE: 'message',
  TYPING: 'typing',
  ERROR: 'error',
  NEW_MESSAGE: 'new_message',
  STATUS: 'status',
  JOINED: 'joined',
  PONG: 'pong',
}

export class WidgetWebSocketClient {
  constructor() {
    this.socket = null
    this.reconnectInterval = 1000
    this.maxReconnectInterval = 30000
    this.reconnectAttempts = 0
    this.maxReconnectAttempts = 50
    this.isReconnecting = false
    this.manualClose = false
    this.pingInterval = null
    this.lastPong = Date.now()
    this.jwt = null
    this.inboxId = null
  }

  init (jwt, inboxId) {
    this.manualClose = false
    this.jwt = jwt
    this.inboxId = inboxId
    this.connect()
    this.setupNetworkListeners()
  }

  connect () {
    if (this.isReconnecting || this.manualClose) return

    try {
      this.socket = new WebSocket('/widget/ws')
      this.socket.addEventListener('open', this.handleOpen.bind(this))
      this.socket.addEventListener('message', this.handleMessage.bind(this))
      this.socket.addEventListener('error', this.handleError.bind(this))
      this.socket.addEventListener('close', this.handleClose.bind(this))
    } catch (error) {
      console.error('Widget WebSocket connection error:', error)
      this.reconnect()
    }
  }

  handleOpen () {
    this.reconnectInterval = 1000
    const wasReconnecting = this.reconnectAttempts > 0
    this.reconnectAttempts = 0
    this.isReconnecting = false
    this.lastPong = Date.now()
    this.setupPing()

    // Auto-join inbox after connection if inbox_id is set.
    if (this.inboxId && this.jwt) {
      this.joinInbox()
    }

    // If this was a reconnection, sync current conversation messages
    if (wasReconnecting) {
      this.resyncCurrentConversation()
    }
  }

  handleMessage (event) {
    const chatStore = useChatStore()
    try {
      if (!event.data) return
      const data = JSON.parse(event.data)
      const handlers = {
        [WS_EVENT.JOINED]: () => {
          // Joined inbox.
        },
        [WS_EVENT.PONG]: () => {
          this.lastPong = Date.now()
        },
        [WS_EVENT.NEW_MESSAGE]: () => {
          if (data.data) {
            chatStore.addMessageToConversation(data.data.conversation_uuid, data.data)
          }
        },
        [WS_EVENT.ERROR]: () => {
          console.error('Widget WebSocket error:', data.data)
        },
        [WS_EVENT.TYPING]: () => {
          if (data.data && data.data.is_typing !== undefined) {
            chatStore.setTypingStatus(data.data.conversation_uuid, data.data.is_typing)
          }
        }
      }
      const handler = handlers[data.type]
      if (handler) {
        handler()
      } else {
        console.warn(`Unknown widget websocket event: ${data.type}`)
      }
    } catch (error) {
      console.error('Widget message handling error:', error)
    }
  }

  handleError (event) {
    console.error('Widget WebSocket error:', event)
    this.reconnect()
  }

  handleClose () {
    this.clearPing()
    if (!this.manualClose) {
      this.reconnect()
    }
  }

  reconnect () {
    if (this.isReconnecting || this.reconnectAttempts >= this.maxReconnectAttempts) return

    this.isReconnecting = true
    this.reconnectAttempts++

    setTimeout(() => {
      this.isReconnecting = false
      this.connect()
      this.reconnectInterval = Math.min(this.reconnectInterval * 1.5, this.maxReconnectInterval)
    }, this.reconnectInterval)
  }

  setupNetworkListeners () {
    window.addEventListener('online', () => {
      if (this.socket?.readyState !== WebSocket.OPEN) {
        this.reconnectInterval = 1000
        this.reconnect()
      }
    })

    window.addEventListener('focus', () => {
      if (this.socket?.readyState !== WebSocket.OPEN) {
        this.reconnect()
      }
    })
  }

  setupPing () {
    this.clearPing()
    this.pingInterval = setInterval(() => {
      if (this.socket?.readyState === WebSocket.OPEN) {
        try {
          this.socket.send(JSON.stringify({
            type: 'ping',
          }))
          if (Date.now() - this.lastPong > 60000) {
            console.warn('No pong received in 60 seconds, closing widget connection')
            this.socket.close()
          }
        } catch (e) {
          console.error('Widget ping error:', e)
          this.reconnect()
        }
      }
    }, 5000)
  }

  clearPing () {
    if (this.pingInterval) {
      clearInterval(this.pingInterval)
      this.pingInterval = null
    }
  }

  joinInbox () {
    if (!this.inboxId || !this.jwt) {
      console.error('Cannot join inbox: missing inbox_id or JWT')
      return
    }

    const joinMessage = {
      type: WS_EVENT.JOIN,
      jwt: this.jwt,
      data: {
        inbox_id: parseInt(this.inboxId, 10)
      }
    }

    this.send(joinMessage)
  }

  // Resync current conversation after reconnection to catch any missed messages.
  resyncCurrentConversation () {
    const chatStore = useChatStore()
    const currentConversationUUID = chatStore.currentConversation?.uuid
    if (currentConversationUUID) {
      chatStore.loadConversation(currentConversationUUID)
    }
  }

  sendTyping (isTyping = true, conversationUUID = null) {
    const typingMessage = {
      type: WS_EVENT.TYPING,
      jwt: this.jwt,
      data: {
        conversation_uuid: conversationUUID,
        is_typing: isTyping
      }
    }
    this.send(typingMessage)
  }

  send (message) {
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(message))
    } else {
      console.warn('Widget WebSocket is not open. Message not sent:', message)
    }
  }

  close () {
    this.manualClose = true
    this.clearPing()
    if (this.socket) {
      this.socket.close()
    }
  }
}

let widgetWSClient

export function initWidgetWS (jwt, inboxId) {
  if (!widgetWSClient) {
    widgetWSClient = new WidgetWebSocketClient()
    widgetWSClient.init(jwt, inboxId)
  } else {
    // Update JWT and inbox_id and rejoin if connection exists
    widgetWSClient.jwt = jwt
    widgetWSClient.inboxId = inboxId
    if (widgetWSClient.socket?.readyState === WebSocket.OPEN) {
      widgetWSClient.joinInbox()
    } else {
      // If connection is not open, reconnect
      widgetWSClient.init(jwt, inboxId)
    }
  }
  return widgetWSClient
}

export const sendWidgetMessage = message => widgetWSClient?.send(message)
export const sendWidgetTyping = (isTyping = true, conversationUUID = null) => widgetWSClient?.sendTyping(isTyping, conversationUUID)
export const closeWidgetWebSocket = () => widgetWSClient?.close()
export const reOpenWidgetWebSocket = () => widgetWSClient?.reOpen()
