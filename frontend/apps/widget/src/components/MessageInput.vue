<template>
  <div class="border-t focus:ring-0 focus:outline-none">
    <div class="p-2">
      <!-- Unified Input Container -->
      <div class="border border-input rounded-lg bg-background focus-within:border-primary">
        <!-- Textarea Container -->
        <div class="p-2">
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
        <div class="flex justify-between items-center px-3 pb-2">
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
</template>

<script setup>
import { ref, computed, nextTick, watch } from 'vue'
import { ArrowUp } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { useWidgetStore } from '../store/widget.js'
import { useChatStore } from '../store/chat.js'
import { useUserStore } from '@widget/store/user.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { sendWidgetTyping } from '../websocket.js'
import { convertTextToHtml } from '@shared-ui/utils/string.js'
import { useTypingIndicator } from '@shared-ui/composables/useTypingIndicator.js'
import MessageInputActions from './MessageInputActions.vue'
import api from '@widget/api/index.js'

// Define emits
const emit = defineEmits(['error'])

const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const userStore = useUserStore()
const messageInput = ref(null)
const newMessage = ref('')
const isUploading = ref(false)
const config = computed(() => widgetStore.config)

// Setup typing indicator
const { startTyping, stopTyping } = useTypingIndicator((isTyping) => {
  if (chatStore.currentConversation?.uuid) {
    sendWidgetTyping(isTyping, chatStore.currentConversation.uuid)
  }
})

const initChatConversation = async (messageText) => {
  const resp = await api.initChatConversation({
    message: messageText
  })
  const { conversation, jwt, messages } = resp.data.data

  // Set user session token if not already set.
  if (!userStore.userSessionToken) {
    userStore.setSessionToken(jwt)
  }

  // Update chat store with new conversation and messages.
  chatStore.setCurrentConversation(conversation)
  chatStore.replaceMessages(messages)
}

const sendMessageToConversation = async (messageText) => {
  // Add pending message immediately for existing conversation
  const tempMessageID = chatStore.addPendingMessage(
    chatStore.currentConversation.uuid,
    messageText,
    userStore.isVisitor ? 'visitor' : 'contact',
    userStore.userID
  )

  // Send message in existing conversation.
  const messageResp = await api.sendChatMessage(chatStore.currentConversation.uuid, {
    message: messageText
  })

  // Update the pending message with the actual message.
  if (tempMessageID && messageResp.data.data) {
    chatStore.replaceMessage(
      chatStore.currentConversation.uuid,
      tempMessageID,
      messageResp.data.data
    )
  }
  if (messageResp.data.data) {
    chatStore.updateConversationListLastMessage(
      chatStore.currentConversation.uuid,
      messageResp.data.data
    )
  }

  return tempMessageID
}

const sendMessage = async () => {
  // Empty?
  if (!newMessage.value.trim()) return

  // Stop typing when sending message
  stopTyping()

  // Convert text to HTML.
  const messageText = convertTextToHtml(newMessage.value.trim())

  // Clear input field immediately
  newMessage.value = ''

  // Temporary message ID for pending messages.
  let tempMessageID = null
  try {
    // No current conversation ID? Start a new conversation.
    if (!chatStore.currentConversation.uuid) {
      await initChatConversation(messageText)
    } else {
      tempMessageID = await sendMessageToConversation(messageText)
    }
    emit('error', '')
  } catch (error) {
    // Remove failed message if we have a temp ID.
    if (tempMessageID) {
      chatStore.removeMessage(chatStore.currentConversation.uuid, tempMessageID)
    }

    // Unauthorized?
    if (error.response && error.response.status === 401) {
      userStore.clearSessionToken()
      chatStore.setCurrentConversation(null)
      widgetStore.closeWidget()
    }
    emit('error', handleHTTPError(error).message)
  } finally {
    await chatStore.updateCurrentConversationLastSeen()
  }
}

// Handle typing events
const handleTyping = () => {
  startTyping()
}

// Handle Enter vs Shift+Enter for new lines
const handleKeydown = (event) => {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    sendMessage()
  }
}

// File upload handler
const handleFileUpload = async (files) => {
  if (!chatStore.currentConversation.uuid || files.length === 0) return

  isUploading.value = true
  emit('error', '')
  try {
    // Upload files using the widget API
    await api.uploadMedia(chatStore.currentConversation.uuid, files)

    // Refresh conversation to get updated messages with attachments
    const resp = await api.getChatConversation(chatStore.currentConversation.uuid)
    const msgs = resp.data.data.messages
    chatStore.replaceMessages(msgs)
  } catch (error) {
    emit('error', handleHTTPError(error).message)
  } finally {
    isUploading.value = false
  }
}

// Handle emoji selection.
const handleEmojiSelect = (emoji) => {
  const textarea = messageInput.value?.$el?.querySelector?.('textarea') || messageInput.value?.$el
  if (textarea && textarea.selectionStart !== undefined) {
    // Insert emoji at cursor position
    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const before = newMessage.value.substring(0, start)
    const after = newMessage.value.substring(end)

    newMessage.value = before + emoji + after

    // Restore cursor position after emoji
    nextTick(() => {
      const newPos = start + emoji.length
      textarea.setSelectionRange(newPos, newPos)
      textarea.focus()
    })
  } else {
    // Fallback: append emoji
    newMessage.value += emoji
  }
}

// Auto-resize textarea on input.
watch(newMessage, () => {
  nextTick(() => {
    if (messageInput.value?.$el) {
      const textarea = messageInput.value.$el
      textarea.style.height = 'auto'
      textarea.style.height = Math.min(textarea.scrollHeight, 128) + 'px'
    }
  })
})
</script>
