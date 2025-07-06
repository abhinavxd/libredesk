<template>
  <div class="flex flex-col h-full">
    <!-- Chat Header -->
    <div class="flex items-center p-4 border-b border-border bg-background gap-3 relative">
      <Button @click="goBack" variant="ghost" size="sm" class="absolute left-2">
        <ArrowLeft />
      </Button>

      <!-- Title -->
      <div class="flex items-center justify-center gap-3 ml-10">
        <Avatar class="size-10">
          <AvatarImage :src="chatTitle.avatarUrl" />
          <AvatarFallback>
            {{ chatTitle.avatarFallback }}
          </AvatarFallback>
        </Avatar>
        <div class="flex flex-col">
          <h3 class="text-base font-bold text-foreground">
            {{ chatTitle.name }}
          </h3>
          <p v-if="chatTitle.showStatus" class="text-xs text-muted-foreground">
            <span v-if="chatTitle.isOnline">
              <span class="inline-block w-2 h-2 bg-green-500 rounded-full mr-1"></span>
              {{ chatTitle.statusText }}
            </span>
            <span v-else> Active {{ chatTitle.lastActiveText }} </span>
          </p>
        </div>
      </div>
    </div>

    <!-- Messages Container -->
    <div
      class="flex-1 min-h-0 overflow-y-auto p-4 flex flex-col gap-4 scrollbar-thin scrollbar-track-transparent scrollbar-thumb-muted-foreground/30 hover:scrollbar-thumb-muted-foreground/50"
      ref="messagesContainer"
    >
      <!-- Chat Intro -->
      <ChatIntro :introText="config.chat_introduction" />

      <!-- Notice -->
      <NoticeBanner
        v-if="config.notice_banner.enabled === true"
        :noticeText="config.notice_banner.text"
      />

      <!-- Messages -->
      <div
        v-for="message in messages"
        :key="message.uuid"
        :class="[
          'flex flex-col animate-slide-in',
          message.sender_type === 'contact' ? 'items-end' : 'items-start'
        ]"
      >
        <div
          :class="[
            'max-w-[85%] px-4 py-3 rounded-2xl text-sm leading-5 break-words',
            message.sender_type === 'contact'
              ? 'bg-primary text-primary-foreground rounded-br-sm'
              : 'bg-background text-foreground rounded-bl-sm border border-border'
          ]"
        >
          <Letter
            :html="message.content"
            :allowedSchemas="['cid', 'https', 'http', 'mailto']"
            class="mb-1 native-html"
          />
          <!-- Show attachments if available -->
          <MessageAttachment
            v-if="message.attachments && message.attachments.length > 0"
            :attachments="message.attachments"
          />
        </div>
        <div class="text-xs text-muted-foreground mt-1">
          {{ getMessageTime(message.created_at) }}
        </div>
      </div>

      <!-- Typing Indicator -->
      <div v-if="isTyping" class="flex flex-col items-start">
        <div
          class="flex items-center px-4 py-3 bg-background text-foreground rounded-2xl rounded-bl-sm border border-border"
        >
          <div class="flex gap-0.5">
            <span class="w-1.5 h-1.5 rounded-full bg-muted-foreground animate-dot-flashing"></span>
            <span
              class="w-1.5 h-1.5 rounded-full bg-muted-foreground animate-dot-flashing"
              style="animation-delay: 0.2s"
            ></span>
            <span
              class="w-1.5 h-1.5 rounded-full bg-muted-foreground animate-dot-flashing"
              style="animation-delay: 0.4s"
            ></span>
          </div>
        </div>
      </div>
    </div>

    <!-- Error Display -->
    <WidgetError :errorMessage="errorMessage" />

    <!-- Message Input -->
    <div class="border-t flex-shrink-0 focus:ring-0 focus:outline-none">
      <div class="p-3">
        <!-- Unified Input Container -->
        <div class="border border-input rounded-lg bg-background focus-within:border-primary">
          <!-- Textarea Container -->
          <div class="p-3 pb-2">
            <Textarea
              v-model="newMessage"
              @keydown="handleKeydown"
              @input="handleTyping"
              placeholder="Type your message..."
              class="w-full min-h-6 max-h-32 resize-none border-0 bg-transparent focus:ring-0 focus:outline-none focus-visible:ring-0 p-0 shadow-none"
              ref="messageInput"
            ></Textarea>
          </div>

          <!-- Actions and Send Button -->
          <div class="flex justify-between items-center px-3 py-2">
            <!-- Message Input Actions (file upload + emoji) -->
            <MessageInputActions
              :fileUploadEnabled="config.features?.file_upload || false"
              :emojiEnabled="config.features?.emoji || false"
              :uploading="isUploading"
              @fileUpload="handleFileUpload"
              @emojiSelect="handleEmojiSelect"
            />

            <!-- Send Button -->
            <Button
              @click="sendMessage"
              size="sm"
              class="h-8 px-2 rounded-full disabled:opacity-50 disabled:cursor-not-allowed border-0"
              :disabled="!newMessage.trim() || isUploading"
            >
              <ArrowUp class="w-4 h-4" />
            </Button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, watch, onUnmounted } from 'vue'
import { ArrowLeft, ArrowUp } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { useWidgetStore } from '../store/widget.js'
import { useChatStore } from '../store/chat.js'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { sendWidgetTyping } from '../websocket.js'
import { debounce } from '@shared-ui/utils/debounce.js'
import { Letter } from 'vue-letter'
import { convertTextToHtml } from '@shared-ui/utils/string.js'
import ChatIntro from '@widget/components/ChatIntro.vue'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import NoticeBanner from '@widget/components/NoticeBanner.vue'
import WidgetError from '@widget/components/WidgetError.vue'
import MessageAttachment from '@widget/components/MessageAttachment.vue'
import MessageInputActions from '@widget/components/MessageInputActions.vue'
import api from '@widget/api/index.js'

const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const messagesContainer = ref(null)
const messageInput = ref(null)
const newMessage = ref('')
const errorMessage = ref('')
const timeUpdateTrigger = ref(0) // Force reactivity for time updates
const isUploading = ref(false)


const config = computed(() => widgetStore.config)
const isTyping = computed(() => chatStore.isTyping)
const messages = computed(() => chatStore.getCurrentConversationMessages())
const chatTitle = computed(() => {
  const assignee = chatStore.currentConversation?.assignee
  const hasAssignee = assignee?.id > 0

  if (hasAssignee) {
    return {
      name: assignee.first_name,
      avatarUrl: assignee.avatar_url || '',
      avatarFallback: assignee.first_name?.charAt(0).toUpperCase() || 'A',
      showStatus: !!assignee.active_at,
      isOnline: assignee.availability_status === 'online',
      statusText:
        assignee.availability_status?.charAt(0).toUpperCase() +
        assignee.availability_status?.slice(1),
      lastActiveText: assignee.active_at ? getRelativeTime(assignee.active_at).toLowerCase() : ''
    }
  } else {
    return {
      name: config.value.brand_name,
      avatarUrl: '',
      avatarFallback: config.value.brand_name?.charAt(0).toUpperCase() || 'B',
      showStatus: false,
      isOnline: false,
      statusText: '',
      lastActiveText: ''
    }
  }
})

const getMessageTime = computed(() => {
  timeUpdateTrigger.value
  return (timestamp) => getRelativeTime(timestamp)
})

// Methods
const goBack = () => {
  widgetStore.navigateToMessages()
}

const sendMessage = async () => {
  if (!newMessage.value.trim()) return

  // Convert text to HTML.
  const messageText = convertTextToHtml(newMessage.value.trim())

  newMessage.value = ''

  try {
    let resp = {}
    // No current conversation ID? Start a new conversation.
    if (!chatStore.currentConversation.uuid) {
      resp = await api.initChatConversation({
        message: messageText
      })
      let data = resp.data.data

      const conversation = data.conversation
      const conversationUUID = data.conversation.uuid

      if (!localStorage.getItem('libredesk_session')) {
        localStorage.setItem('libredesk_session', data.jwt)
      }

      scrollToBottom()

      // Fetch entire conversation and replace messages
      if (conversationUUID) {
        // Use the openConversation method which handles both setting ID and WebSocket joining
        chatStore.openConversation(conversation)

        let resp = await api.getChatConversation(conversationUUID)
        let msgs = resp.data.data.messages

        // Replace all messages with the fetched conversation
        chatStore.replaceMessages(msgs)
      }
    } else {
      // Send message in existing conversation
      await api.sendChatMessage(chatStore.currentConversation.uuid, {
        message: messageText
      })
      await chatStore.fetchAndReplaceConversationAndMessages()
    }
    errorMessage.value = ''
  } catch (error) {
    if (error.response && error.response.status === 401) {
      localStorage.removeItem('libredesk_session')
      chatStore.setCurrentConversation(null)
    }
    errorMessage.value = handleHTTPError(error).message
  } finally {
    await chatStore.updateCurrentConversationLastSeen()
  }
}

const scrollToBottom = () => {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
}

let timeUpdateTimer = null

// Debounced typing functions
const sendTypingStart = debounce(() => {
  if (chatStore.currentConversation.uuid) {
    sendWidgetTyping(true)
  }
}, 300)

const sendTypingStop = debounce(() => {
  if (chatStore.currentConversation.uuid) {
    sendWidgetTyping(false)
  }
}, 3000)

const handleTyping = () => {
  sendTypingStart()
  sendTypingStop()
}

// Handle Enter vs Shift+Enter
const handleKeydown = (event) => {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    sendMessage()
  }
  // Allow Shift+Enter for new lines (default behavior)
}

// File upload handler
const handleFileUpload = async (files) => {
  if (!chatStore.currentConversation.uuid || files.length === 0) return

  isUploading.value = true
  errorMessage.value = ''

  try {
    // Upload files using the widget API
    await api.uploadMedia(chatStore.currentConversation.uuid, files)

    // Refresh conversation to get updated messages with attachments
    const resp = await api.getChatConversation(chatStore.currentConversation.uuid)
    const msgs = resp.data.data.messages
    chatStore.replaceMessages(msgs)

    // Scroll to bottom to show new messages
    scrollToBottom()
  } catch (error) {
    console.error('Error uploading files:', error)
    errorMessage.value = handleHTTPError(error).message
  } finally {
    isUploading.value = false
  }
}

// Emoji selection handler
const handleEmojiSelect = (emoji) => {
  if (messageInput.value) {
    const textarea = messageInput.value.$el.querySelector('textarea') || messageInput.value.$el
    if (textarea) {
      const cursorPos = textarea.selectionStart || 0
      const textBefore = newMessage.value.substring(0, cursorPos)
      const textAfter = newMessage.value.substring(cursorPos)
      newMessage.value = textBefore + emoji + textAfter

      // Set cursor position after the emoji
      nextTick(() => {
        const newPos = cursorPos + emoji.length
        textarea.setSelectionRange(newPos, newPos)
        textarea.focus()
      })
    } else {
      // Fallback: just append emoji
      newMessage.value += emoji
    }
  }
}

onMounted(async () => {
  await chatStore.fetchCurrentConversation()

  // Start timer to update relative times every 1 minute
  timeUpdateTimer = setInterval(() => {
    timeUpdateTrigger.value++
  }, 60000)

  // Scroll to bottom on mount
  scrollToBottom()

  // Update last seen timestamp for the contact
  await chatStore.updateCurrentConversationLastSeen()
})

// Watch for new messages and scroll to bottom
watch(
  messages,
  () => {
    scrollToBottom()
  },
  { deep: true }
)

// Auto-resize textarea
watch(newMessage, () => {
  nextTick(() => {
    if (messageInput.value?.$el) {
      const textarea = messageInput.value.$el
      textarea.style.height = 'auto'
      textarea.style.height = Math.min(textarea.scrollHeight, 128) + 'px'
    }
  })
})

// Cleanup on unmount
onUnmounted(() => {
  if (timeUpdateTimer) {
    clearInterval(timeUpdateTimer)
  }
})
</script>
