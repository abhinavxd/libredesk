<template>
  <div class="flex flex-col h-screen bg-background widget-slide-up">
    <!-- Chat Header -->
    <div class="flex items-center p-4 border-b border-border bg-background gap-3">
      <Button @click="goBack" variant="ghost" size="sm" class="min-w-8 h-8 p-0">
        <ArrowLeft class="w-4 h-4" />
      </Button>

      <div class="flex-1 text-center">
        <h3 class="text-base font-semibold m-0 text-foreground">
          {{ config.brand_name }}
        </h3>
      </div>

      <!-- Remove close button in iframe mode -->
      <div class="min-w-8 h-8"></div>
    </div>

    <!-- Messages Container -->
    <div
      class="flex-1 overflow-y-auto p-4 flex flex-col gap-4 bg-muted/30 scrollbar-thin scrollbar-track-transparent scrollbar-thumb-muted-foreground/30 hover:scrollbar-thumb-muted-foreground/50"
      ref="messagesContainer"
    >
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
          {{ message.content }}
        </div>
        <div class="text-xs text-muted-foreground mt-1 px-2">
          {{ getRelativeTime(message.created_at) }}
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
    <div v-if="errorMessage" class="p-4 bg-destructive/10 border-t border-destructive/20">
      <p class="text-sm text-destructive text-center">{{ errorMessage }}</p>
    </div>

    <!-- Message Input -->
    <div class="border-t border-border bg-background">
      <div class="flex gap-2 p-4 items-end">
        <Input
          v-model="newMessage"
          @keypress.enter="sendMessage"
          placeholder="Type your message..."
          class="flex-1 min-h-10 rounded-3xl"
        />
        <Button
          @click="sendMessage"
          size="sm"
          class="min-w-10 h-10 rounded-full p-0 disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="!newMessage.trim()"
        >
          <Send class="w-4 h-4" />
        </Button>
      </div>

      <div class="px-4 pb-3 text-center">
        <span class="text-xs text-muted-foreground"
          >Powered by <a href="https://libredesk.io" target="_blank">Libredesk</a></span
        >
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, watch } from 'vue'
import { ArrowLeft, Send } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { useWidgetStore } from '../store/widget.js'
import { useChatStore } from '../store/chat.js'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'

import api from '@widget/api/index.js'

const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const messagesContainer = ref(null)
const newMessage = ref('')
const errorMessage = ref('')

// Computed properties
const config = computed(() => widgetStore.config)
const isTyping = computed(() => chatStore.isTyping)
const messages = computed(() => chatStore.getMessages())

// Methods
const goBack = () => {
  widgetStore.navigateToWelcome()
}

const sendMessage = async () => {
  if (!newMessage.value.trim()) return

  const messageText = newMessage.value.trim()
  newMessage.value = ''

  try {
    let resp = {}
    if (!chatStore.currentConversationId) {
      resp = await api.initChatConversation({
        message: messageText
      })
      let data = resp.data.data

      // store data.jwt in localStorage as 'libredesk_session'
      const conversationUUID = data.conversation.uuid

      localStorage.setItem('libredesk_session', data.jwt)
      localStorage.setItem('libredesk_current_conversation', data.conversation.uuid)

      scrollToBottom()

      // Fetch entire conversation and replace messages
      if (conversationUUID) {
        chatStore.setCurrentConversationId(conversationUUID)

        let resp = await api.getChatConversation(conversationUUID)
        let msgs = resp.data.data.messages

        // Replace all messages with the fetched conversation
        chatStore.replaceMessages(msgs)
      }
    } else {
      resp = await api.sendChatMessage(chatStore.currentConversationId, {
        message: messageText
      })
    }
  } catch (error) {
    console.error('Error sending message:', error)
    if (error.response && error.response.status === 403) {
      localStorage.removeItem('libredesk_session')
      localStorage.removeItem('libredesk_current_conversation')
      widgetStore.navigateToWelcome()
    }
    errorMessage.value = handleHTTPError(error).message
  }
}

const scrollToBottom = () => {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
}

onMounted(() => {
  const conversationUUID = localStorage.getItem('libredesk_current_conversation')

  if (conversationUUID) {
    chatStore.setCurrentConversationId(conversationUUID)

    // Fetch entire conversation and replace messages
    api
      .getChatConversation(conversationUUID)
      .then((resp) => {
        let msgs = resp.data.data.messages
        chatStore.replaceMessages(msgs)
      })
      .catch((error) => {
        console.error('Error fetching conversation:', error)
      })
  }

  // Scroll to bottom on mount
  scrollToBottom()
})

// Watch for new messages and scroll to bottom
watch(
  messages,
  () => {
    scrollToBottom()
  },
  { deep: true }
)
</script>
