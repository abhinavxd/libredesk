<template>
  <div class="border-t focus:ring-0 focus:outline-none">
    <!-- Message Input -->
    <div class="p-2">
      <!-- Unified Input Container -->
      <div class="border border-input rounded-lg bg-background focus-within:border-secondary">
        <!-- Textarea Container -->
        <div class="p-2">
          <Textarea
            v-model="newMessage"
            @keydown="handleKeydown"
            @input="handleTyping"
            :placeholder="$t('globals.terms.typeMessage')"
            :disabled="isSending"
            maxlength="10000"
            class="w-full max-h-32 resize-none border-0 bg-transparent focus:ring-0 focus:outline-none focus-visible:ring-0 p-0 shadow-none" style="min-height:20px;height:20px"
            ref="messageInput"
          ></Textarea>
        </div>

        <!-- Actions and Send Button -->
        <div class="flex justify-between items-center px-2 pb-2 rtl:flex-row-reverse">
          <!-- Message Input Actions (file upload + emoji) -->
          <MessageInputActions
            :fileUploadEnabled="config.features?.file_upload || false"
            :emojiEnabled="config.features?.emoji || false"
            :uploading="isUploading"
            :canUploadFiles="!!chatStore.currentConversation?.uuid"
            :disabled="isSending"
            @fileUpload="handleFileUpload"
            @emojiSelect="handleEmojiSelect"
          />

          <!-- Send Button -->
          <Button
            @click="sendMessage"
            :aria-label="$t('globals.messages.send')"
            size="sm"
            class="h-9 w-9 p-0 rounded-full disabled:opacity-50 disabled:cursor-not-allowed border-0"
            :disabled="!newMessage.trim() || isUploading || isSending"
          >
            <ArrowUp class="w-4 h-4" />
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, watch, onMounted } from 'vue'
import { ArrowUp } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { useWidgetStore } from '../store/widget.js'
import { useChatStore } from '../store/chat.js'
import { useUserStore } from '@widget/store/user.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { sendWidgetTyping } from '../websocket.js'
import { useTypingIndicator } from '@shared-ui/composables/useTypingIndicator.js'
import MessageInputActions from './MessageInputActions.vue'
import api, { saveSession } from '@widget/api/index.js'

const emit = defineEmits(['error'])
const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const userStore = useUserStore()
const messageInput = ref(null)
const newMessage = ref('')
const isUploading = ref(false)
const isSending = ref(false)
const config = computed(() => widgetStore.config)

const getTextareaEl = () => messageInput.value?.$el?.querySelector?.('textarea') || messageInput.value?.$el

const focusTextarea = () => {
  nextTick(() => getTextareaEl()?.focus())
}

onMounted(focusTextarea)
watch(() => widgetStore.isOpen, (open) => {
  if (open) focusTextarea()
})

// Setup typing indicator
const { startTyping, stopTyping } = useTypingIndicator((isTyping) => {
  if (chatStore.currentConversation?.uuid) {
    sendWidgetTyping(isTyping, chatStore.currentConversation.uuid)
  }
})

const initChatConversation = async (messageText) => {
  const resp = await api.initChatConversation({ message: messageText })
  const { conversation, session_token, user, messages, business_hours_id, working_hours_utc_offset } = resp.data.data
  conversation.business_hours_id = business_hours_id
  conversation.working_hours_utc_offset = working_hours_utc_offset

  if (!userStore.userSessionToken && session_token) {
    saveSession(session_token, user, userStore, true)
  }

  // Add the new conversation to the list
  chatStore.addConversationToList(conversation)

  // Update chat store with new conversation and messages.
  chatStore.setCurrentConversation(conversation)
  chatStore.replaceMessages(messages)
}

const sendMessageToConversation = async (messageText, tempMessageID) => {
  const messageResp = await api.sendChatMessage(chatStore.currentConversation.uuid, {
    message: messageText
  })

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
}

const sendMessage = async () => {
  // Empty or already sending?
  if (!newMessage.value.trim() || isSending.value) return

  // Stop typing when sending message
  stopTyping()

  // Convert text to HTML.
  const messageText = newMessage.value.trim()

  // Clear input field immediately
  newMessage.value = ''

  // Add pending message before API call so we can remove it on failure.
  let tempMessageID = null
  if (chatStore.currentConversation?.uuid) {
    tempMessageID = chatStore.addPendingMessage(
      chatStore.currentConversation.uuid,
      messageText,
      userStore.isVisitor ? 'visitor' : 'contact',
      userStore.userID
    )
  }
  try {
    isSending.value = true
    if (!chatStore.currentConversation.uuid) {
      await initChatConversation(messageText)
    } else {
      await sendMessageToConversation(messageText, tempMessageID)
    }
    emit('error', '')
  } catch (error) {
    // Remove failed message if we have a temp ID.
    if (tempMessageID) {
      chatStore.removeMessage(chatStore.currentConversation.uuid, tempMessageID)
    }

    emit('error', handleHTTPError(error).message)
  } finally {
    isSending.value = false
    focusTextarea()
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

  // Create pending file message immediately
  const fileNames = Array.from(files)
    .map((f) => f.name)
    .join(', ')

  const trimmedFileNames =
    fileNames.length > 40 ? fileNames.slice(0, 40).trimEnd() + '...' : fileNames
  const pendingMessage = `${trimmedFileNames}`
  const tempMessageID = chatStore.addPendingMessage(
    chatStore.currentConversation.uuid,
    pendingMessage,
    userStore.isVisitor ? 'visitor' : 'contact',
    userStore.userID,
    Array.from(files)
  )

  try {
    const resp = await api.uploadMedia(chatStore.currentConversation.uuid, files)

    if (tempMessageID && resp.data.data) {
      chatStore.replaceMessage(chatStore.currentConversation.uuid, tempMessageID, resp.data.data)
    }
    if (resp.data.data) {
      chatStore.updateConversationListLastMessage(chatStore.currentConversation.uuid, resp.data.data)
    }
  } catch (error) {
    // Remove failed upload message
    if (tempMessageID) {
      chatStore.removeMessage(chatStore.currentConversation.uuid, tempMessageID)
    }
    emit('error', handleHTTPError(error).message)
  } finally {
    isUploading.value = false
  }
}

// Handle emoji selection.
const handleEmojiSelect = (emoji) => {
  const textarea = getTextareaEl()
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
    const textarea = getTextareaEl()
    if (!textarea) return
    textarea.style.height = '20px'
    textarea.style.height = Math.min(textarea.scrollHeight, 128) + 'px'
  })
})
</script>
