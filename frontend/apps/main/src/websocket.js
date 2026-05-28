import { useConversationStore } from './stores/conversation'
import { useNotificationStore } from './stores/notification'
import { useUsersStore } from './stores/users'
import { WS_EVENT, WS_EPHEMERAL_TYPES } from './constants/websocket'
import { playNotificationSound } from '@shared-ui/composables/useNotificationSound'

export class WebSocketClient {
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
    this.lastPong = Date.now()
    this.convStore = useConversationStore()
    this.notificationStore = useNotificationStore()
    this.usersStore = useUsersStore()
    this.messageQueue = []
    this.maxQueueSize = 50
    this.queueTimeoutMs = 30000
  }

  init () {
    this.connect()
    this.setupNetworkListeners()
  }

  connect () {
    if (this.isReconnecting || this.manualClose) return

    try {
      this.socket = new WebSocket('/ws')
      this.socket.addEventListener('open', this.handleOpen.bind(this))
      this.socket.addEventListener('message', this.handleMessage.bind(this))
      this.socket.addEventListener('error', this.handleError.bind(this))
      this.socket.addEventListener('close', this.handleClose.bind(this))
    } catch (error) {
      console.error('WebSocket connection error:', error)
      this.reconnect()
    }
  }

  handleOpen () {
    console.log('WebSocket connected')
    const wasReconnect = this.reconnectAttempts > 0
    this.reconnectInterval = 1000
    this.reconnectAttempts = 0
    this.isReconnecting = false
    this.lastPong = Date.now()
    this.setupPing()
    this.flushMessageQueue()
    if (wasReconnect) {
      // RESUB!
      const uuids = this.convStore.conversations.data?.map(c => c.uuid) || []
      this.subscribeListReplace(uuids)
      const openUUID = this.convStore.conversation.data?.uuid
      if (openUUID) this.subscribeToConversation(openUUID)
    }
  }

  handleMessage (event) {
    try {
      if (!event.data) return

      if (event.data === 'pong') {
        this.lastPong = Date.now()
        return
      }

      const data = JSON.parse(event.data)
      const handlers = {
        [WS_EVENT.NEW_MESSAGE]: () => {
          const uuid = data.data.conversation_uuid
          const isOpen = this.convStore.conversation.data?.uuid === uuid
          const isFromContact = data.data.sender_type === 'contact'
          const convPayload = data.data.conversation

          if (convPayload) {
            this.convStore.handleConvPush(convPayload)
          } else {
            this.convStore.mergeConversationUpdate({
              uuid,
              last_message: data.data.preview,
              last_message_at: data.data.created_at,
              last_message_sender: data.data.sender_type,
            })
          }

          if (isFromContact && document.hidden) {
            if (isOpen || this.convStore.isConversationInList(uuid)) {
              playNotificationSound()
            } else {
              this.convStore.addPendingNotification(uuid)
            }
          }

          if (!isOpen && this.convStore.isConversationInList(uuid)) {
            this.convStore.incrementUnread(uuid)
          }

          this.convStore.updateConversationMessage(data.data)
        },
        [WS_EVENT.NEW_CONVERSATION]: () => {
          if (data.data && data.data.uuid) {
            this.convStore.handleConvPush(data.data)
          } else {
            this.convStore.refreshConversationList()
          }
        },
        // Property updates for conversation and message.
        [WS_EVENT.MESSAGE_UPDATE]: () => this.convStore.mergeMessageUpdate(data.data),
        [WS_EVENT.CONVERSATION_UPDATE]: () => this.convStore.mergeConversationUpdate(data.data),
        [WS_EVENT.CONTACT_UPDATE]: () => this.convStore.mergeContactUpdate(data.data),
        [WS_EVENT.TYPING]: () => {
          this.convStore.updateTypingStatus(data.data)
        },
        // New notification.
        [WS_EVENT.NEW_NOTIFICATION]: () => this.notificationStore.addNotification(data.data),
        [WS_EVENT.AGENT_AVAILABILITY_UPDATE]: () =>
          this.usersStore.setAvailability(data.data.agent_id, data.data.availability_status),
      }

      const handler = handlers[data.type]
      if (handler) {
        handler()
      } else {
        console.warn(`Unknown websocket event: ${data.type}`)
      }
    } catch (error) {
      console.error('Message handling error:', error)
    }
  }

  handleError (event) {
    console.error('WebSocket error:', event)
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
      // Clear any pending reconnect attempts.
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
          this.socket.send('ping')
          if (Date.now() - this.lastPong > 90000) {
            console.warn('No pong received in 90 seconds, closing connection')
            this.socket.close()
          }
        } catch (e) {
          console.error('Ping error:', e)
          this.reconnect()
        }
      }
    }, 30000)
  }

  clearPing () {
    if (this.pingInterval) {
      clearInterval(this.pingInterval)
      this.pingInterval = null
    }
  }

  send (message) {
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(message))
    } else {
      console.warn('WebSocket is not open. Queueing message:', message)
      this.queueMessage(message)
    }
  }

  queueMessage (message) {
    // Don't queue ephemeral message types.
    if (WS_EPHEMERAL_TYPES.includes(message.type)) {
      console.log('Skipping queue for ephemeral message type:', message.type)
      return
    }

    // Remove expired messages from queue.
    const now = Date.now()
    this.messageQueue = this.messageQueue.filter(item =>
      now - item.timestamp < this.queueTimeoutMs
    )

    // Remove all existing conversation subscriptions since only one is allowed.
    if (message.type === WS_EVENT.CONVERSATION_SUBSCRIBE) {
      this.messageQueue = this.messageQueue.filter(item =>
        item.type !== WS_EVENT.CONVERSATION_SUBSCRIBE
      )
    }

    // Evict oldest message if queue is full.
    if (this.messageQueue.length >= this.maxQueueSize) {
      console.warn('Message queue is full, removing oldest message')
      this.messageQueue.shift()
    }

    // Push.
    this.messageQueue.push({
      ...message,
      timestamp: now
    })
  }

  flushMessageQueue () {
    if (this.messageQueue.length === 0) return

    // Remove expired messages before sending
    const now = Date.now()
    this.messageQueue = this.messageQueue.filter(item =>
      now - item.timestamp < this.queueTimeoutMs
    )

    if (this.messageQueue.length === 0) return

    console.log(`Sending ${this.messageQueue.length} queued messages`)
    while (this.messageQueue.length > 0 && this.socket?.readyState === WebSocket.OPEN) {
      const queuedItem = this.messageQueue.shift()
      // Remove timestamp before sending
      delete queuedItem.timestamp
      this.socket.send(JSON.stringify(queuedItem))
    }
  }

  subscribeToConversation (conversationUUID) {
    if (!conversationUUID) return

    const subscribeMessage = {
      type: WS_EVENT.CONVERSATION_SUBSCRIBE,
      data: {
        conversation_uuid: conversationUUID
      }
    }

    this.send(subscribeMessage)
  }

  subscribeListReplace (uuids) {
    this.send({ type: WS_EVENT.LIST_SUBSCRIBE_REPLACE, data: { uuids: uuids || [] } })
  }

  sendTypingIndicator (conversationUUID, isTyping, isPrivateMessage) {
    if (!conversationUUID) return

    const typingMessage = {
      type: WS_EVENT.TYPING,
      data: {
        conversation_uuid: conversationUUID,
        is_typing: isTyping,
        is_private_message: isPrivateMessage,
      }
    }

    this.send(typingMessage)
  }

  close () {
    this.manualClose = true
    this.clearPing()
    if (this.socket) {
      this.socket.close()
    }
  }
}

let wsClient

export function initWS () {
  if (!wsClient) {
    wsClient = new WebSocketClient()
    wsClient.init()
  }
  return wsClient
}

export const sendMessage = message => wsClient?.send(message)
export const subscribeToConversation = conversationUUID => wsClient?.subscribeToConversation(conversationUUID)
export const subscribeListReplace = uuids => wsClient?.subscribeListReplace(uuids)
export const sendTypingIndicator = (conversationUUID, isTyping, isPrivateMessage) => wsClient?.sendTypingIndicator(conversationUUID, isTyping, isPrivateMessage)
export const closeWebSocket = () => wsClient?.close()