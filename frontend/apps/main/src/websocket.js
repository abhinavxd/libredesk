import { useConversationStore } from './stores/conversation'
import { WS_EVENT, WS_EPHEMERAL_TYPES } from './constants/websocket'

export class WebSocketClient {
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
    this.convStore = useConversationStore()
    this.messageQueue = []
    this.maxQueueSize = 50
    // 30 sec.
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
    this.reconnectInterval = 1000
    this.reconnectAttempts = 0
    this.isReconnecting = false
    this.lastPong = Date.now()
    this.setupPing()
    // Send any queued messages after connection is established.
    this.flushMessageQueue()
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
        // On new message, update the message in the conversation list and in the currently opened conversation.
        [WS_EVENT.NEW_MESSAGE]: () => {
          this.convStore.updateConversationList(data.data)
          this.convStore.updateConversationMessage(data.data)
        },
        [WS_EVENT.MESSAGE_PROP_UPDATE]: () => this.convStore.updateMessageProp(data.data),
        [WS_EVENT.CONVERSATION_PROP_UPDATE]: () => this.convStore.updateConversationProp(data.data),
        [WS_EVENT.CONVERSATION_SUBSCRIBED]: () => {
          console.log('Successfully subscribed to conversation:', data.data.conversation_uuid)
        },
        [WS_EVENT.TYPING]: () => {
          this.convStore.updateTypingStatus(data.data)
        }
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
          this.socket.send('ping')
          if (Date.now() - this.lastPong > 60000) {
            console.warn('No pong received in 60 seconds, closing connection')
            this.socket.close()
          }
        } catch (e) {
          console.error('Ping error:', e)
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
export const sendTypingIndicator = (conversationUUID, isTyping, isPrivateMessage) => wsClient?.sendTypingIndicator(conversationUUID, isTyping, isPrivateMessage)
export const closeWebSocket = () => wsClient?.close()