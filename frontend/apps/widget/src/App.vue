<template>
  <TooltipProvider :delay-duration="150">
    <Toaster class="pointer-events-auto" position="top-center" />
    <div
      class="libredesk-widget-app text-foreground bg-background"
      :class="{
        dark: isDark,
        light: !isDark,
        mobile: widgetStore.isMobileFullScreen
      }"
      :style="customColorStyle"
      @click.once="initAudioContext"
      @touchstart.once="initAudioContext"
    >
      <div class="widget-container">
        <MainLayout />
      </div>
    </div>
  </TooltipProvider>
</template>

<script setup>
import { computed, onMounted, ref, watch, getCurrentInstance } from 'vue'
import { Toaster } from '@shared-ui/components/ui/sonner'
import { TooltipProvider } from '@shared-ui/components/ui/tooltip'
import { useWidgetStore } from './store/widget.js'
import { useChatStore } from '@widget/store/chat.js'
import { useUserStore } from './store/user.js'
import { initWidgetWS, closeWidgetWebSocket, sendPageVisit, skipInitialWsSync } from './websocket.js'
import api, { setApiSessionToken, initVisitorToken, saveSession, registerStores } from '@widget/api/index.js'
import { useUnreadCount } from './composables/useUnreadCount.js'
import { initAudioContext } from '@shared-ui/composables/useNotificationSound.js'
import { hexToHSL, getContrastingHSL } from '@shared-ui/utils/color.js'
import MainLayout from '@widget/layouts/MainLayout.vue'

const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const userStore = useUserStore()
const initialDataStarted = ref(false)

// Register stores for the global 401 response interceptor.
registerStores({ userStore, chatStore, widgetStore })

// Initialize unread count tracking and sending to parent window.
useUnreadCount()

const widgetConfig = getCurrentInstance().appContext.config.globalProperties.$widgetConfig
if (widgetConfig) {
  widgetStore.updateConfig(widgetConfig)
}

const isDark = computed(() => Boolean(widgetStore.config.dark_mode))

const customColorStyle = computed(() => {
  const style = {}
  const colors = widgetStore.config.colors
  if (colors?.primary) {
    style['--primary'] = hexToHSL(colors.primary)
    style['--primary-foreground'] = getContrastingHSL(colors.primary)
  }
  return style
})

const applyDocumentTheme = (dark) => {
  document.documentElement.style.colorScheme = dark ? 'dark' : 'light'
  document.documentElement.classList.toggle('dark', dark)
  document.documentElement.classList.toggle('light', !dark)
  document.body.classList.toggle('dark', dark)
  document.body.classList.toggle('light', !dark)
  const meta = document.querySelector('meta[name="theme-color"]')
  if (meta) meta.setAttribute('content', dark ? '#1a1a1d' : '#ffffff')
}

watch(isDark, (dark) => applyDocumentTheme(dark), { immediate: true })

onMounted(() => {
  setupParentMessageListeners()
  window.parent.postMessage({ type: 'VUE_APP_READY' }, '*')
})

const signalWidgetLoaded = () => {
  window.parent.postMessage({ type: 'WIDGET_LOADED' }, '*')
}

const fetchInitialConversations = async () => {
  const success = await chatStore.fetchConversations()
  if (success && chatStore.hasConversations) {
    try {
      await chatStore.loadConversation(chatStore.getConversations[0].uuid)
    } catch { /* non-blocking */ }
  }
  if (widgetStore.config?.direct_to_conversation && success) {
    widgetStore.navigateToChat()
  }
}

const ensureInitialConversations = async () => {
  if (initialDataStarted.value) return
  initialDataStarted.value = true
  await fetchInitialConversations()
}

const prepareSession = async (sessionToken) => {
  if (!sessionToken) return

  userStore.setSessionToken(sessionToken)
  setApiSessionToken(sessionToken)
  skipInitialWsSync()

  try {
    const meResp = await api.getAuthMe()
    if (userStore.userSessionToken === sessionToken) {
      userStore.setUserMeta(meResp.data.data)
    }
  } catch {
    // 401 is handled by the global response interceptor.
  }
}

// Listen for messages from parent window (widget.js)
const setupParentMessageListeners = () => {
  window.addEventListener('message', async (event) => {
    if (event.data.type == 'WIDGET_CLOSED') {
      widgetStore.setOpen(false)
    } else if (event.data.type === 'WIDGET_OPENED') {
      widgetStore.setOpen(true)
      void ensureInitialConversations()
    } else if (event.data.type === 'SET_MOBILE_STATE') {
      widgetStore.setMobileFullScreen(event.data.isMobile)
    } else if (event.data.type === 'WIDGET_EXPANDED') {
      widgetStore.setExpanded(event.data.isExpanded)
    } else if (event.data.type === 'SESSION_DATA') {
      if (event.data.visitorToken) {
        initVisitorToken(event.data.visitorToken)
      }
      try {
        await prepareSession(event.data.sessionToken)
      } finally {
        signalWidgetLoaded()
      }
      void ensureInitialConversations()
    } else if (event.data.type === 'SET_JWT_TOKEN') {
      if (event.data.visitorToken) {
        initVisitorToken(event.data.visitorToken)
      }
      if (event.data.jwt) {
        try {
          const resp = await api.exchangeJWTForSession(event.data.jwt)
          const { session_token, user } = resp.data.data
          saveSession(session_token, user, userStore)
          skipInitialWsSync()
          chatStore.conversations = null
        } catch (err) {
          console.error('Failed to exchange JWT for session:', err)
        } finally {
          signalWidgetLoaded()
        }
        void ensureInitialConversations()
      }
    } else if (event.data.type === 'CLEAR_SESSION') {
      userStore.clearSessionToken()
    } else if (event.data.type === 'PAGE_VISIT') {
      sendPageVisit(event.data.url, event.data.title)
    }
  })
}

const initializeWebSocket = () => {
  const token = userStore.userSessionToken
  if (token) {
    const urlParams = new URLSearchParams(window.location.search)
    const inboxId = urlParams.get('inbox_id')
    if (inboxId) {
      initWidgetWS(token, inboxId)
    } else {
      console.error('Cannot initialize WebSocket: missing `inbox_id`')
    }
  } else {
    closeWidgetWebSocket()
  }
}

watch(
  () => userStore.userSessionToken,
  (newToken) => {
    if (newToken) {
      initializeWebSocket()
    } else {
      closeWidgetWebSocket()
    }
  }
)
</script>

<style scoped>
.libredesk-widget-app {
  width: 100vw;
  height: 100dvh;
  overflow: hidden;
  font-family: 'Inter', system-ui, -apple-system, sans-serif;
  font-weight: 400;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

.widget-container {
  width: 100%;
  height: 100%;
}

/* 16px inputs prevent iOS zoom-on-focus and improve composer readability. */
.libredesk-widget-app :deep(input),
.libredesk-widget-app :deep(textarea),
.libredesk-widget-app :deep(select) {
  font-size: 16px;
}

.libredesk-widget-app.mobile .widget-container {
  box-sizing: border-box;
  padding-top: env(safe-area-inset-top, 0px);
  padding-left: env(safe-area-inset-left, 0px);
  padding-right: env(safe-area-inset-right, 0px);
}

.libredesk-widget-app.mobile :deep(.message-input) {
  padding-bottom: env(safe-area-inset-bottom, 0px);
}
</style>
