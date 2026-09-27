<template>
  <div
    class="libredesk-widget-app text-foreground bg-background"
    :class="{ dark: widgetStore.isDark, mobile: widgetStore.isMobileFullScreen }"
    :style="customColorStyle"
    @click.once="initAudioContext"
    @touchstart.once="initAudioContext"
  >
    <div class="widget-container">
      <MainLayout />
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, watch, getCurrentInstance } from 'vue'
import { useWidgetStore } from './store/widget.js'
import { useChatStore } from '@widget/store/chat.js'
import { useUserStore } from './store/user.js'
import { initWidgetWS, closeWidgetWebSocket, sendPageVisit, skipInitialWsSync } from './websocket.js'
import api, { setApiSessionToken, initVisitorToken, saveSession, registerStores } from '@widget/api/index.js'
import { useUnreadCount } from './composables/useUnreadCount.js'
import { initAudioContext } from '@shared-ui/composables/useNotificationSound.js'
import { hexToHSL, getContrastingHSL } from '@shared-ui/utils/color.js'
import MainLayout from '@widget/layouts/MainLayout.vue'
import { useHelpStore } from '@widget/store/help.js'
import { useI18n } from 'vue-i18n'
import { useProactiveStore } from '@widget/store/proactive.js'
import { useReplyPreviews } from '@widget/composables/useReplyPreviews.js'

const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const userStore = useUserStore()
const help = useHelpStore()
const proactive = useProactiveStore()
const { locale } = useI18n()
useReplyPreviews()

// Register stores for the global 401 response interceptor.
registerStores({ userStore, chatStore, widgetStore })

// Initialize unread count tracking and sending to parent window.
useUnreadCount()

const widgetConfig = getCurrentInstance().appContext.config.globalProperties.$widgetConfig
if (widgetConfig) {
  widgetStore.updateConfig(widgetConfig)
}

const customColorStyle = computed(() => {
  const style = {}
  const colors = widgetStore.branding?.colors
  if (colors?.primary) {
    style['--primary'] = hexToHSL(colors.primary)
    style['--primary-foreground'] = getContrastingHSL(colors.primary)
  }
  return style
})

// Dropdowns and dialogs portal to document.body, outside the app wrapper.
watch(
  () => widgetStore.isDark,
  (dark) => document.documentElement.classList.toggle('dark', dark),
  { immediate: true }
)

onMounted(() => {
  setupParentMessageListeners()
  window.parent.postMessage({ type: 'VUE_APP_READY' }, '*')
})

const signalWidgetLoaded = async () => {
  if (widgetStore.config.help?.help_center_id) await help.load(locale.value)
  window.parent.postMessage(
    {
      type: 'WIDGET_LOADED',
      campaigns: widgetStore.config.has_campaigns,
      campaignDelay: widgetStore.config.campaign_delay_seconds || 0
    },
    '*'
  )
}

const fetchInitialConversations = async () => {
  const success = await chatStore.fetchConversations()
  if (success && chatStore.hasConversations) {
    try {
      await chatStore.loadConversation(chatStore.getConversations[0].uuid)
    } catch { /* non-blocking */ }
  }
  const audience = userStore.isVisitor ? widgetStore.config?.visitors : widgetStore.config?.users
  const directToConversation =
    audience?.direct_to_conversation ?? widgetStore.config?.direct_to_conversation
  if (directToConversation && success) {
    widgetStore.navigateToChat()
  }
}

// Listen for messages from parent window (widget.js)
const setupParentMessageListeners = () => {
  window.addEventListener('message', async (event) => {
    if (event.source !== window.parent || !event.data || typeof event.data !== 'object') return
    const parentOrigin = new URLSearchParams(window.location.search).get('parent_origin')
    if (parentOrigin && event.origin !== parentOrigin) return
    if (event.data.campaignSessionKey) proactive.setSessionKey(event.data.campaignSessionKey)
    if (event.data.campaignBrowserKey) proactive.setBrowserKey(event.data.campaignBrowserKey)
    if (event.data.type == 'WIDGET_CLOSED') {
      widgetStore.setOpen(false)
    } else if (event.data.type === 'WIDGET_OPENED') {
      widgetStore.setOpen(true)
    } else if (event.data.type === 'SET_MOBILE_STATE') {
      widgetStore.setMobileFullScreen(event.data.isMobile)
    } else if (event.data.type === 'WIDGET_EXPANDED') {
      widgetStore.setExpanded(event.data.isExpanded)
    } else if (event.data.type === 'SESSION_DATA') {
      if (event.data.visitorToken) {
        initVisitorToken(event.data.visitorToken)
      }
      const sessionToken = event.data.sessionToken
      try {
        if (sessionToken) {
          userStore.setSessionToken(sessionToken)
          setApiSessionToken(sessionToken)
          // Session exists, fetchInitialConversations will load data. Skip WS sync.
          skipInitialWsSync()
          // Fetch user metadata for returning visitors.
          // Guard against stale response if SET_JWT_TOKEN exchange replaced the token.
          try {
            const meResp = await api.getAuthMe()
            if (userStore.userSessionToken === sessionToken) {
              userStore.setUserMeta(meResp.data.data)
            }
          } catch {
            // 401 is handled by the global response interceptor.
          }
        }
        await fetchInitialConversations()
      } finally {
        signalWidgetLoaded()
      }
    } else if (event.data.type === 'SET_JWT_TOKEN') {
      if (event.data.visitorToken) {
        initVisitorToken(event.data.visitorToken)
      }
      if (event.data.jwt) {
        proactive.reset()
        help.reset()
        if (widgetStore.currentView === 'help') widgetStore.navigateToHome()
        chatStore.drafts = {}
        chatStore.preChatDraft = {}
        chatStore.handoffDraft = {}
        chatStore.conversations = null
        chatStore.setCurrentConversation(null)
        try {
          const resp = await api.exchangeJWTForSession(event.data.jwt)
          const { session_token, user } = resp.data.data
          saveSession(session_token, user, userStore)
          // Session exists, fetchInitialConversations will load data. Skip WS sync.
          skipInitialWsSync()
          chatStore.conversations = null
          await fetchInitialConversations()
        } catch (err) {
          console.error('Failed to exchange JWT for session:', err)
        } finally {
          signalWidgetLoaded()
        }
      }
    } else if (event.data.type === 'CLEAR_SESSION') {
      proactive.reset()
      chatStore.drafts = {}
      chatStore.preChatDraft = {}
      chatStore.handoffDraft = {}
      userStore.clearSessionToken()
      chatStore.conversations = null
      chatStore.setCurrentConversation(null)
      help.reset()
      widgetStore.navigateToHome()
      signalWidgetLoaded()
    } else if (event.data.type === 'CAMPAIGN_CONTEXT') {
      await proactive.next(event.data.context)
    } else if (event.data.type === 'CAMPAIGN_EVENT') {
      await proactive.event(event.data.event, event.data.id)
    } else if (event.data.type === 'PAGE_VISIT') {
      sendPageVisit(event.data.url, event.data.title)
    } else if (event.data.type === 'OPEN_CONVERSATION') {
      if (!chatStore.getConversations.some(conversation => conversation.uuid === event.data.uuid)) return
      widgetStore.navigateToMessages()
      if (await chatStore.loadConversation(event.data.uuid, true)) {
        const seen = new Date(chatStore.currentConversation.contact_last_seen_at)
        chatStore.previewUnreadUUID = chatStore.getCurrentConversationMessages.find(message => ['agent', 'ai_assistant'].includes(message.author?.type) && new Date(message.created_at) > seen)?.uuid || null
        widgetStore.navigateToChat()
      }
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
}

.widget-container {
  width: 100%;
  height: 100%;
}

/* iOS Safari auto-zooms on focus when font-size < 16px. Force 16px on mobile to prevent it. */
.mobile :deep(input),
.mobile :deep(textarea),
.mobile :deep(select) {
  font-size: 16px;
}
</style>
