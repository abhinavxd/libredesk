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

// sync retries ~5 min (100 × 3s); WS reconnects every 2s up to ~5 min (150 attempts).
const SYNC_MAX_RETRIES = 100
const SYNC_RETRY_DELAY_MS = 3000
const RECONNECT_INTERVAL_MS = 2000

let widgetWSClient
let _syncOnFirstConnect = true

export class WidgetWebSocketClient {
  constructor() {
    this.socket = null
    this.reconnectAttempts = 0
    this.maxReconnectAttempts = 150
    this.isReconnecting = false
    this.reconnectTimer = null
    this.manualClose = false
    this.pingInterval = null
    this.lastSyncAt = 0
    this.lastPong = Date.now()
    this.wsInitiated = false
    this.token = null
    this.inboxId = null
    this.syncRetryTimer = null
    this.networkListenersSetup = false
    this.recovering = false
    this.connectedBannerTimer = null
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
    this.reconnectAttempts = 0
    this.isReconnecting = false
    this.lastPong = Date.now()
    useWidgetStore().setConnectionFailed(false)
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

    this.finishRecovery()
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
      this.beginRecovery()
      this.reconnect()
    }
  }

  reconnect () {
    if (this.isReconnecting) return
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      const widgetStore = useWidgetStore()
      widgetStore.setConnectionFailed(true)
      widgetStore.setConnecting(false)
      widgetStore.setConnected(false)
      this.recovering = false
      return
    }

    this.isReconnecting = true
    this.reconnectAttempts++

    this.reconnectTimer = setTimeout(() => {
      this.isReconnecting = false
      this.reconnectTimer = null
      this.connect()
    }, RECONNECT_INTERVAL_MS)
  }

  beginRecovery () {
    if (!this.wsInitiated || this.recovering) return
    this.recovering = true
    clearTimeout(this.connectedBannerTimer)
    const widgetStore = useWidgetStore()
    widgetStore.setConnected(false)
    widgetStore.setConnecting(true)
  }

  finishRecovery () {
    if (!this.recovering) return
    this.recovering = false
    const widgetStore = useWidgetStore()
    widgetStore.setConnecting(false)
    widgetStore.setConnected(true)
    clearTimeout(this.connectedBannerTimer)
    this.connectedBannerTimer = setTimeout(() => widgetStore.setConnected(false), 2000)
  }

  setupNetworkListeners () {
    if (this.networkListenersSetup) return
    this.networkListenersSetup = true

    window.addEventListener('online', () => {
      if (this.manualClose) return
      if (this.reconnectTimer) {
        clearTimeout(this.reconnectTimer)
        this.reconnectTimer = null
      }
      this.reconnectAttempts = 0
      this.isReconnecting = false
      useWidgetStore().setConnectionFailed(false)
      if (this.socket) {
        this.socket.close()
      }
      this.beginRecovery()
      this.syncMissedMessages()
      this.reconnect()
    })

    // On tab return, if WS is not connected, sync data immediately and reconnect in parallel.
    document.addEventListener('visibilitychange', () => {
      if (this.manualClose) return
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

  async syncMissedMessages (attempt = 0) {
    const now = Date.now()
    if (attempt === 0 && now - this.lastSyncAt < 2000) return
    this.lastSyncAt = now
    clearTimeout(this.syncRetryTimer)

    const chatStore = useChatStore()
    const conversationsOk = await chatStore.fetchConversations(true, true)
    const currentConversationUUID = chatStore.currentConversation?.uuid
    let conversationOk = true
    if (currentConversationUUID) {
      conversationOk = await chatStore.loadConversation(currentConversationUUID, true, true)
    }

    if ((!conversationsOk || !conversationOk) && attempt < SYNC_MAX_RETRIES) {
      this.syncRetryTimer = setTimeout(() => this.syncMissedMessages(attempt + 1), SYNC_RETRY_DELAY_MS)
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
    clearTimeout(this.syncRetryTimer)
    this.syncRetryTimer = null
    clearTimeout(this.connectedBannerTimer)
    this.connectedBannerTimer = null
    clearTimeout(this.reconnectTimer)
    this.reconnectTimer = null
    this.isReconnecting = false
    this.reconnectAttempts = 0
    this.recovering = false
    this.lastSyncAt = 0
    const widgetStore = useWidgetStore()
    widgetStore.setConnectionFailed(false)
    widgetStore.setConnecting(false)
    widgetStore.setConnected(false)
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
