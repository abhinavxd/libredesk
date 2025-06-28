import { useConversationStore } from './stores/conversation'
import { WS_EVENT } from './constants/websocket'

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
        [WS_EVENT.CONVERSATION_PROP_UPDATE]: () => this.convStore.updateConversationProp(data.data)
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
      console.warn('WebSocket is not open. Message not sent:', message)
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

let wsClient

export function initWS () {
  if (!wsClient) {
    wsClient = new WebSocketClient()
    wsClient.init()
  }
  return wsClient
}

export const sendMessage = message => wsClient?.send(message)
export const closeWebSocket = () => wsClient?.close()