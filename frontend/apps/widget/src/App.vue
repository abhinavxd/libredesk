<template>
  <div
    class="libredesk-widget-app text-foreground bg-background"
    :class="{ dark: widgetStore.config.dark_mode }"
  >
    <div class="widget-container">
      <MainLayout />
    </div>
  </div>
</template>

<script setup>
import { onMounted, watch, getCurrentInstance } from 'vue'
import { useWidgetStore } from './store/widget.js'
import { useChatStore } from '@widget/store/chat.js'
import { useUserStore } from './store/user.js'
import { initWidgetWS, closeWidgetWebSocket } from './websocket.js'
import { useUnreadCount } from './composables/useUnreadCount.js'
import { applyCSSColor } from '@shared-ui/utils/color.js'
import MainLayout from '@widget/layouts/MainLayout.vue'

const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const userStore = useUserStore()

// Initialize unread count tracking and sending to parent window.
useUnreadCount()

onMounted(async () => {
  // Use pre-fetched widget config from main.js
  const widgetConfig = getCurrentInstance().appContext.config.globalProperties.$widgetConfig
  if (widgetConfig) {
    widgetStore.updateConfig(widgetConfig)
    // Apply custom primary color from config
    applyCSSColor('--primary', widgetConfig.colors?.primary)
  }

  initializeWebSocket()

  widgetStore.setOpen(true)

  setupParentMessageListeners()

  await chatStore.fetchConversations()

  // Notify parent window that Vue app is ready
  window.parent.postMessage(
    {
      type: 'VUE_APP_READY'
    },
    '*'
  )
})

// Listen for messages from parent window (widget.js)
const setupParentMessageListeners = () => {
  window.addEventListener('message', (event) => {
    if (event.data.type == 'WIDGET_CLOSED') {
      widgetStore.setOpen(false)
    } else if (event.data.type === 'WIDGET_OPENED') {
      widgetStore.setOpen(true)
    } else if (event.data.type === 'SET_MOBILE_STATE') {
      widgetStore.setMobileFullScreen(event.data.isMobile)
    } else if (event.data.type === 'WIDGET_EXPANDED') {
      widgetStore.setExpanded(event.data.isExpanded)
    } else if (event.data.type === 'SET_JWT_TOKEN') {
      // Set the JWT token in user store when received from parent
      if (event.data.jwt) {
        userStore.setSessionToken(event.data.jwt)
      }
    }
  })
}

// Initialize WebSocket only when JWT token exists
const initializeWebSocket = () => {
  const jwt = userStore.userSessionToken
  if (jwt) {
    const urlParams = new URLSearchParams(window.location.search)
    const inboxId = urlParams.get('inbox_id')
    if (inboxId) {
      initWidgetWS(jwt, inboxId)
    } else {
      console.error('Cannot initialize WebSocket: missing `inbox_id`')
    }
  } else {
    closeWidgetWebSocket()
  }
}

// Re-initialize WebSocket when user gets authenticated
const handleUserAuthentication = () => {
  initializeWebSocket()
}

// Watch for changes in user session token to initialize WebSocket
watch(
  () => userStore.userSessionToken,
  (newToken) => {
    if (newToken) {
      handleUserAuthentication()
    } else {
      closeWidgetWebSocket()
    }
  }
)
</script>

<style scoped>
.libredesk-widget-app {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
}

.widget-container {
  width: 100%;
  height: 100%;
}
</style>
