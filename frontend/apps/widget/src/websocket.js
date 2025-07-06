import { useChatStore } from './store/chat'

// Widget WebSocket message types (matching backend constants)
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
    this.chatStore = useChatStore()
    this.jwt = null
    this.isJoined = false
  }

  init (jwt) {
    this.jwt = jwt
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
    console.log('Widget WebSocket connected')
    this.reconnectInterval = 1000
    this.reconnectAttempts = 0
    this.isReconnecting = false
    this.lastPong = Date.now()
    this.setupPing()

    // Auto-join conversation after connection if a conversation uuid is set.
    if (this.chatStore.currentConversation.uuid && this.jwt && !this.isJoined) {
      this.joinConversation()
    }
  }

  handleMessage (event) {
    try {
      if (!event.data) return
      const data = JSON.parse(event.data)
      const handlers = {
        [WS_EVENT.JOINED]: () => {
          this.isJoined = true
        },
        [WS_EVENT.PONG]: () => {
          this.lastPong = Date.now()
        },
        [WS_EVENT.NEW_MESSAGE]: () => {
          // Add new message to chat store
          if (data.data) {
            this.chatStore.addMessageToConversation(data.data.conversation_uuid, data.data)
          }
        },
        [WS_EVENT.ERROR]: () => {
          console.error('Widget WebSocket error:', data.data)
        },
        [WS_EVENT.TYPING]: () => {
          // TODO: check conversation uuid and then set typing as true.
          // if (data.data && data.data.is_typing !== undefined) {
          //   this.chatStore.setTypingStatus(data.data.is_typing)
          // }
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
    this.isJoined = false
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

  joinConversation () {
    const currentConversationUuid = this.chatStore.currentConversation.uuid
    if (!currentConversationUuid || !this.jwt) {
      console.error('Cannot join conversation: missing conversationUuid or JWT')
      return
    }

    const joinMessage = {
      type: WS_EVENT.JOIN,
      jwt: this.jwt,
      data: {
        conversation_uuid: currentConversationUuid
      }
    }

    this.send(joinMessage)
  }

  sendTyping (isTyping = true) {
    if (!this.isJoined) {
      console.warn('Cannot send typing indicator: not joined to conversation')
      return
    }

    const currentConversationUUID = this.chatStore.currentConversation.uuid
    const typingMessage = {
      type: WS_EVENT.TYPING,
      jwt: this.jwt,
      data: {
        conversation_uuid: currentConversationUUID,
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

  // Method to join a new conversation without reinitializing the connection
  joinNewConversation () {
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.isJoined = false
      this.joinConversation()
    } else {
      console.warn('WebSocket not connected, cannot join new conversation')
    }
  }

  close () {
    this.manualClose = true
    this.isJoined = false
    this.clearPing()
    if (this.socket) {
      this.socket.close()
    }
  }
}

let widgetWSClient

export function initWidgetWS (jwt) {
  if (!widgetWSClient) {
    widgetWSClient = new WidgetWebSocketClient()
    widgetWSClient.init(jwt)
  } else {
    // Update JWT and rejoin if connection exists
    widgetWSClient.jwt = jwt
    if (widgetWSClient.socket?.readyState === WebSocket.OPEN) {
      // Reset joined status and join the new conversation
      widgetWSClient.isJoined = false
      widgetWSClient.joinConversation()
    } else {
      // If connection is not open, reconnect
      widgetWSClient.init(jwt)
    }
  }
  return widgetWSClient
}

export const sendWidgetMessage = message => widgetWSClient?.send(message)
export const sendWidgetTyping = (isTyping = true) => widgetWSClient?.sendTyping(isTyping)
export const closeWidgetWebSocket = () => widgetWSClient?.close()
export const joinNewConversation = () => widgetWSClient?.joinNewConversation()
