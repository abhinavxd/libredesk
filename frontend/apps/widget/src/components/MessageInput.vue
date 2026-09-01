<template>
  <div class="message-input border-t border-border/60 bg-background">
    <div class="p-3 pt-2">
      <div
        class="message-composer rounded-xl border border-input/70 bg-background transition-[border-color,box-shadow] focus-within:border-primary/35 focus-within:ring-2 focus-within:ring-primary/10"
        :class="{ 'ring-2 ring-primary/10 border-primary/35': isFocused }"
      >
        <div class="px-3 pt-3 pb-1">
          <Textarea
            v-model="newMessage"
            @keydown="handleKeydown"
            @input="handleInput"
            @focus="isFocused = true"
            @blur="isFocused = false"
            :placeholder="$t('globals.terms.typeMessage')"
            :disabled="isSending"
            maxlength="10000"
            rows="1"
            class="message-composer-field w-full resize-none border-0 bg-transparent p-0 shadow-none text-[15px] leading-relaxed text-foreground placeholder:text-muted-foreground/55 focus:ring-0 focus:outline-none focus-visible:ring-0"
            ref="messageInput"
          />
        </div>

        <div class="flex justify-between items-center gap-2 px-2 pb-2">
          <MessageInputActions
            :fileUploadEnabled="config.features?.file_upload || false"
            :emojiEnabled="config.features?.emoji || false"
            :uploading="isUploading"
            :canUploadFiles="!!chatStore.currentConversation?.uuid"
            :disabled="isSending"
            @fileUpload="handleFileUpload"
            @emojiSelect="handleEmojiSelect"
          />

          <Button
            @click="sendMessage"
            :aria-label="$t('globals.messages.send')"
            size="sm"
            class="h-9 w-9 shrink-0 rounded-full border-0 transition-transform active:scale-95 disabled:opacity-40"
            :disabled="!newMessage.trim() || isUploading || isSending"
          >
            <ArrowUp class="w-4 h-4" />
          </Button>
        </div>
      </div>

      <p
        v-if="isFocused"
        class="mt-2 text-center text-[11px] leading-snug text-muted-foreground/55"
      >
        <kbd class="font-sans">Enter</kbd> to send · <kbd class="font-sans">Shift+Enter</kbd> for a new line
      </p>
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

const MIN_INPUT_HEIGHT = 44
const MAX_INPUT_HEIGHT = 128

const emit = defineEmits(['error'])
const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const userStore = useUserStore()
const messageInput = ref(null)
const newMessage = ref('')
const isUploading = ref(false)
const isSending = ref(false)
const isFocused = ref(false)
const config = computed(() => widgetStore.config)

const getTextareaEl = () => {
  const el = messageInput.value?.$el
  return el instanceof HTMLTextAreaElement ? el : el?.querySelector?.('textarea')
}

const resizeTextarea = () => {
  nextTick(() => {
    const textarea = getTextareaEl()
    if (!textarea) return
    textarea.style.height = 'auto'
    const nextHeight = Math.min(Math.max(textarea.scrollHeight, MIN_INPUT_HEIGHT), MAX_INPUT_HEIGHT)
    textarea.style.height = `${nextHeight}px`
    textarea.style.overflowY = textarea.scrollHeight > MAX_INPUT_HEIGHT ? 'auto' : 'hidden'
  })
}

const focusTextarea = () => {
  nextTick(() => getTextareaEl()?.focus())
}

onMounted(() => {
  // A conversation starter chosen on the home screen pre-fills the composer so
  // the visitor can review and send it (we never auto-send on their behalf).
  if (chatStore.pendingStarterMessage) {
    newMessage.value = chatStore.pendingStarterMessage
    chatStore.pendingStarterMessage = null
  }
  resizeTextarea()
  focusTextarea()
})

watch(() => widgetStore.isOpen, (open) => {
  if (open) {
    resizeTextarea()
    focusTextarea()
  }
})

const { startTyping, stopTyping } = useTypingIndicator((isTyping) => {
  if (chatStore.currentConversation?.uuid) {
    sendWidgetTyping(isTyping, chatStore.currentConversation.uuid)
  }
})

const initChatConversation = async (messageText) => {
  const payload = { message: messageText }
  if (chatStore.pendingFormData) {
    payload.form_data = chatStore.pendingFormData
  }
  const resp = await api.initChatConversation(payload)
  chatStore.pendingFormData = null
  const { conversation, session_token, user, messages, business_hours_id, working_hours_utc_offset } =
    resp.data.data
  conversation.business_hours_id = business_hours_id
  conversation.working_hours_utc_offset = working_hours_utc_offset

  if (!userStore.userSessionToken && session_token) {
    saveSession(session_token, user, userStore, true)
  }

  chatStore.addConversationToList(conversation)
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
  if (!newMessage.value.trim() || isSending.value) return

  stopTyping()
  const messageText = newMessage.value.trim()
  newMessage.value = ''
  resizeTextarea()

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
    if (tempMessageID) {
      chatStore.removeMessage(chatStore.currentConversation.uuid, tempMessageID)
    }
    emit('error', handleHTTPError(error).message)
  } finally {
    isSending.value = false
    focusTextarea()
  }
}

const handleInput = () => {
  resizeTextarea()
  startTyping()
}

const handleKeydown = (event) => {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    sendMessage()
  }
}

const handleFileUpload = async (files) => {
  if (!chatStore.currentConversation.uuid || files.length === 0) return

  isUploading.value = true
  emit('error', '')

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
    if (tempMessageID) {
      chatStore.removeMessage(chatStore.currentConversation.uuid, tempMessageID)
    }
    emit('error', handleHTTPError(error).message)
  } finally {
    isUploading.value = false
  }
}

const handleEmojiSelect = (emoji) => {
  const textarea = getTextareaEl()
  if (textarea && textarea.selectionStart !== undefined) {
    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const before = newMessage.value.substring(0, start)
    const after = newMessage.value.substring(end)
    newMessage.value = before + emoji + after

    nextTick(() => {
      const newPos = start + emoji.length
      textarea.setSelectionRange(newPos, newPos)
      textarea.focus()
      resizeTextarea()
    })
  } else {
    newMessage.value += emoji
    resizeTextarea()
  }
}

watch(newMessage, () => resizeTextarea())
</script>

<style scoped>
.message-composer-field {
  min-height: 44px;
  max-height: 128px;
  field-sizing: content;
}

/* Fallback when field-sizing is unsupported */
@supports not (field-sizing: content) {
  .message-composer-field {
    field-sizing: normal;
  }
}

.message-input kbd {
  @apply rounded px-1 py-0.5 text-[10px] bg-muted/50 text-muted-foreground/80;
}
</style>
