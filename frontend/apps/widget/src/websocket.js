// Widget WebSocket message types (matching backend constants)
import { useChatStore } from './store/chat.js'
import { useWidgetStore } from './store/widget.js'
import { playNotificationSound } from '@shared-ui/composables/useNotificationSound.js'

export const WS_EVENT = {
  JOIN: 'join',
  MESSAGE: 'message',
  TYPING: 'typing',
  ERROR: 'error',
  NEW_MESSAGE: 'new_message',
  STATUS: 'status',
  JOINED: 'joined',
  PONG: 'pong',
  CONVERSATION_UPDATE: 'conversation_update',
}

let widgetWSClient
let _syncOnFirstConnect = true

export class WidgetWebSocketClient {
  constructor() {
    this.socket = null
    this.reconnectInterval = 1000
    this.maxReconnectInterval = 30000
    this.reconnectAttempts = 0
    this.maxReconnectAttempts = 50
    this.isReconnecting = false
    this.reconnectTimer = null
    this.manualClose = false
    this.pingInterval = null
    this.lastSyncAt = 0
    this.lastPong = Date.now()
    this.wsInitiated = false
    this.token = null
    this.inboxId = null
  }

  init (token, inboxId) {
    this.manualClose = false
    this.token = token
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
    this.reconnectAttempts = 0
    this.isReconnecting = false
    this.lastPong = Date.now()
    this.setupPing()

    // Auto-join inbox after connection if inbox_id is set.
    if (this.inboxId && this.token) {
      this.joinInbox()
    }

    // Reconnect: always sync to catch missed messages.
    // First connect: sync only for new visitors (no pre-existing session).
    // Returning visitors skip - fetchInitialConversations handles initial data.
    if (this.wsInitiated || _syncOnFirstConnect) {
      this.syncMissedMessages()
    }
    this.wsInitiated = true
  }

  handleMessage (event) {
    const chatStore = useChatStore()
    try {
      if (!event.data) return
      const data = JSON.parse(event.data)
      const handlers = {
        [WS_EVENT.JOINED]: () => {
          if (window.parent && window.parent !== window) {
            window.parent.postMessage({ type: 'REQUEST_PAGE_INFO' }, '*')
          }
        },
        [WS_EVENT.PONG]: () => {
          this.lastPong = Date.now()
        },
        [WS_EVENT.NEW_MESSAGE]: () => {
          if (!data.data) return

          const message = data.data
          chatStore.addMessageToConversation(message.conversation_uuid, message)

          // Play notification sound if message is from agent and widget is not focused on this conversation.
          const widgetStore = useWidgetStore()
          const isFromAgent = message.author?.type === 'agent'
          const isViewingConversation = widgetStore.isOpen &&
            widgetStore.isInChatView &&
            chatStore.currentConversation?.uuid === message.conversation_uuid

          if (isFromAgent && (!isViewingConversation || document.hidden)) {
            playNotificationSound()
          }
        },
        [WS_EVENT.ERROR]: () => {
          console.error('Widget WebSocket error:', data.data)
        },
        [WS_EVENT.TYPING]: () => {
          if (data.data && data.data.is_typing !== undefined) {
            chatStore.setTypingStatus(data.data.conversation_uuid, data.data.is_typing)
          }
        },
        [WS_EVENT.CONVERSATION_UPDATE]: () => {
          if (data.data) {
            chatStore.updateCurrentConversation(data.data)
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

    this.reconnectTimer = setTimeout(() => {
      this.isReconnecting = false
      this.reconnectTimer = null
      this.connect()
      this.reconnectInterval = Math.min(this.reconnectInterval * 1.5, this.maxReconnectInterval)
    }, this.reconnectInterval)
  }

  setupNetworkListeners () {
    window.addEventListener('online', () => {
      // Cancel any pending backoff timer and reconnect immediately.
      if (this.reconnectTimer) {
        clearTimeout(this.reconnectTimer)
        this.reconnectTimer = null
      }
      this.reconnectAttempts = 0
      this.reconnectInterval = 1000
      this.isReconnecting = false
      if (this.socket) {
        this.socket.close()
      }
      this.reconnect()
    })

    // On tab return, if WS is not connected, sync data immediately and reconnect in parallel.
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'visible' && this.socket?.readyState !== WebSocket.OPEN) {
        this.syncMissedMessages()
        this.reconnect()
      }
    })
  }

  setupPing () {
    this.clearPing()
    // Backend read deadline is 20s; ping every 10s so a single missed ping doesn't trip it.
    this.pingInterval = setInterval(() => {
      if (this.socket?.readyState === WebSocket.OPEN) {
        try {
          this.socket.send(JSON.stringify({
            type: 'ping',
          }))
          if (Date.now() - this.lastPong > 30000) {
            console.warn('No pong received in 30 seconds, closing widget connection')
            this.socket.close()
          }
        } catch (e) {
          console.error('Widget ping error:', e)
          this.reconnect()
        }
      }
    }, 10000)
  }

  clearPing () {
    if (this.pingInterval) {
      clearInterval(this.pingInterval)
      this.pingInterval = null
    }
  }

  joinInbox () {
    if (!this.inboxId || !this.token) {
      console.error('Cannot join inbox: missing inbox_id or token')
      return
    }

    const joinMessage = {
      type: WS_EVENT.JOIN,
      token: this.token,
      data: {
        inbox_id: this.inboxId
      }
    }

    this.send(joinMessage)
  }

  // Silently refresh conversation list and current conversation to catch messages missed while WS was disconnected.
  syncMissedMessages () {
    const now = Date.now()
    if (now - this.lastSyncAt < 2000) return
    this.lastSyncAt = now

    const chatStore = useChatStore()
    chatStore.fetchConversations(true, true)
    const currentConversationUUID = chatStore.currentConversation?.uuid
    if (currentConversationUUID) {
      chatStore.loadConversation(currentConversationUUID, true, true)
    }
  }

  sendTyping (isTyping = true, conversationUUID = null) {
    const typingMessage = {
      type: WS_EVENT.TYPING,
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

export function initWidgetWS (token, inboxId) {
  if (!widgetWSClient) {
    widgetWSClient = new WidgetWebSocketClient()
    widgetWSClient.init(token, inboxId)
  } else {
    widgetWSClient.token = token
    widgetWSClient.inboxId = inboxId
    if (widgetWSClient.socket?.readyState === WebSocket.OPEN) {
      widgetWSClient.joinInbox()
    } else {
      // If connection is not open, reconnect
      widgetWSClient.init(token, inboxId)
    }
  }
  return widgetWSClient
}

export const sendWidgetTyping = (isTyping = true, conversationUUID = null) => widgetWSClient?.sendTyping(isTyping, conversationUUID)
export const closeWidgetWebSocket = () => widgetWSClient?.close()
export const skipInitialWsSync = () => { _syncOnFirstConnect = false }

export function sendPageVisit (url, title) {
  if (!widgetWSClient) return
  widgetWSClient.send({
    type: 'page_visit',
    data: { url, title }
  })
}
