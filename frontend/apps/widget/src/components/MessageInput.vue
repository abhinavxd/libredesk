<template>
  <div class="focus:ring-0 focus:outline-none">
    <ReplyButtons
      v-if="quickReplies.length"
      :replies="quickReplies"
      :disabled="isSending"
      align="end"
      class="px-2 pb-2"
      @select="sendQuickReply"
    />
    <!-- Message Input -->
    <div class="p-2 border-t">
      <!-- Unified Input Container -->
      <div class="border border-input rounded-md bg-background focus-within:border-secondary">
        <MessageInputAttachmentPreview
          v-if="mediaFiles.length || currentUploadingFiles.length"
          :attachments="mediaFiles"
          :uploadingFiles="currentUploadingFiles"
          @delete="handleFileDelete"
        />
        <!-- Textarea Container -->
        <div class="p-2">
          <Textarea
            v-model="newMessage"
            @keydown="handleKeydown"
            @input="handleTyping"
            :aria-label="$t('globals.terms.typeMessage')"
            :placeholder="$t('globals.terms.typeMessage')"
            :disabled="isSending"
            maxlength="10000"
            class="w-full max-h-32 resize-none border-0 bg-transparent focus:ring-0 focus:outline-none focus-visible:ring-0 p-0 shadow-none"
            style="min-height: 20px; height: 20px"
            ref="messageInput"
          ></Textarea>
        </div>

        <!-- Actions and Send Button -->
        <div class="flex justify-between items-center px-2 pb-2">
          <!-- Message Input Actions (file upload + emoji) -->
          <MessageInputActions
            :fileUploadEnabled="config.features?.file_upload || false"
            :fileUploadDisabled="isAttachmentLimitReached"
            :emojiEnabled="config.features?.emoji || false"
            :canUploadFiles="!!chatStore.currentConversation?.uuid"
            :disabled="isSending"
            @fileUpload="handleFileUpload"
            @emojiSelect="handleEmojiSelect"
          />

          <!-- Send Button -->
          <Button
            type="button"
            @click="sendMessage"
            :aria-label="$t('globals.messages.send')"
            size="sm"
            class="h-9 w-9 p-0 rounded-full disabled:opacity-50 disabled:cursor-not-allowed border-0"
            :disabled="(!newMessage.trim() && !mediaFiles.length) || isUploading || isSending"
          >
            <ArrowUp class="w-4 h-4" aria-hidden="true" />
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
import ReplyButtons from './ReplyButtons.vue'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { useWidgetStore } from '@widget/store/widget.js'
import { useChatStore } from '@widget/store/chat.js'
import { useUserStore } from '@widget/store/user.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { sendWidgetTyping } from '@widget/websocket.js'
import { useTypingIndicator } from '@shared-ui/composables/useTypingIndicator.js'
import { useI18n } from 'vue-i18n'
import MessageInputActions from './MessageInputActions.vue'
import MessageInputAttachmentPreview from './MessageInputAttachmentPreview.vue'
import api, { saveSession } from '@widget/api/index.js'

import { useProactiveStore } from '@widget/store/proactive.js'
const proactive = useProactiveStore()
const { t } = useI18n()
const emit = defineEmits(['error'])
const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const userStore = useUserStore()
const messageInput = ref(null)
const draftKey = computed(
  () => chatStore.currentConversation?.uuid || proactive.pending?.id || 'new'
)
const newMessage = computed({
  get: () => chatStore.drafts[draftKey.value] || '',
  set: (value) => {
    chatStore.drafts[draftKey.value] = value
  }
})
let uploadSequence = 0
const isSending = ref(false)
const config = computed(() => widgetStore.config)
const quickReplies = computed(() => {
  if (chatStore.currentConversation?.uuid) return []
  const audience = userStore.isVisitor ? config.value.visitors : config.value.users
  return audience?.quick_replies ?? config.value.quick_replies ?? []
})
const mediaFiles = computed(() => chatStore.attachmentDrafts[draftKey.value] || [])
const currentUploadingFiles = computed(() =>
  chatStore.uploadingFiles.filter((item) => item.conversationUUID === draftKey.value)
)
const isUploading = computed(() => currentUploadingFiles.value.length > 0)
const MAX_STAGED_ATTACHMENTS = 5
const stagedAttachmentCount = computed(
  () => mediaFiles.value.length + currentUploadingFiles.value.length
)
const isAttachmentLimitReached = computed(
  () => stagedAttachmentCount.value >= MAX_STAGED_ATTACHMENTS
)

const getTextareaEl = () =>
  messageInput.value?.$el?.querySelector?.('textarea') || messageInput.value?.$el

const focusTextarea = () => {
  nextTick(() => getTextareaEl()?.focus())
}

onMounted(focusTextarea)
watch(
  () => widgetStore.isOpen,
  (open) => {
    if (open) focusTextarea()
  }
)

// Setup typing indicator
const { startTyping, stopTyping } = useTypingIndicator((isTyping) => {
  if (chatStore.currentConversation?.uuid) {
    sendWidgetTyping(isTyping, chatStore.currentConversation.uuid)
  }
})

const initChatConversation = async (messageText) => {
  const resp = await api.initChatConversation({ message: messageText, ...proactive.replyPayload() })
  const {
    conversation,
    session_token,
    user,
    messages,
    business_hours_id,
    working_hours_utc_offset
  } = resp.data.data
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
  proactive.replied()
}

const sendMessageToConversation = async (
  conversationUUID,
  messageText,
  attachments,
  tempMessageID
) => {
  const messageResp = await api.sendChatMessage(conversationUUID, {
    message: messageText,
    attachments: attachments.map((attachment) => attachment.id)
  })

  if (tempMessageID && messageResp.data.data) {
    chatStore.replaceMessage(
      conversationUUID,
      tempMessageID,
      messageResp.data.data
    )
  }
  if (messageResp.data.data) {
    chatStore.updateConversationListLastMessage(
      conversationUUID,
      messageResp.data.data
    )
  }
}

const sendMessage = async () => {
  if ((!newMessage.value.trim() && !mediaFiles.value.length) || isUploading.value || isSending.value) {
    return
  }

  // Stop typing when sending message
  stopTyping()

  // Convert text to HTML.
  const messageText = newMessage.value.trim()
  const attachments = [...mediaFiles.value]
  const activeDraftKey = draftKey.value
  const conversationUUID = chatStore.currentConversation?.uuid

  // Clear input field immediately
  newMessage.value = ''

  // Add pending message before API call so we can remove it on failure.
  let tempMessageID = null
  if (conversationUUID) {
    tempMessageID = chatStore.addPendingMessage(
      conversationUUID,
      messageText,
      userStore.isVisitor ? 'visitor' : 'contact',
      userStore.userID,
      attachments
    )
  }
  try {
    isSending.value = true
    if (!conversationUUID) {
      await initChatConversation(messageText)
    } else {
      await sendMessageToConversation(conversationUUID, messageText, attachments, tempMessageID)
    }
    chatStore.attachmentDrafts = { ...chatStore.attachmentDrafts, [activeDraftKey]: [] }
    emit('error', '')
  } catch (error) {
    // Remove failed message if we have a temp ID.
    if (tempMessageID) {
      chatStore.removeMessage(conversationUUID, tempMessageID)
    }

    chatStore.drafts[activeDraftKey] = messageText
    emit('error', handleHTTPError(error).message)
  } finally {
    isSending.value = false
    focusTextarea()
  }
}

const sendQuickReply = (reply) => {
  newMessage.value = reply
  sendMessage()
}

defineExpose({ sendQuickReply })

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

const handleFileUpload = async (files) => {
  if (!chatStore.currentConversation.uuid || files.length === 0) return

  const conversationUUID = chatStore.currentConversation.uuid
  const remainingSlots = Math.max(0, MAX_STAGED_ATTACHMENTS - stagedAttachmentCount.value)
  const acceptedFiles = Array.from(files).slice(0, remainingSlots)
  const limitExceeded = files.length > acceptedFiles.length
  if (limitExceeded) {
    emit('error', t('widget.attachmentLimitReached', { max: MAX_STAGED_ATTACHMENTS }))
  }
  if (acceptedFiles.length === 0) return

  const selectedFiles = acceptedFiles.map((file) => ({
    file,
    name: file.name,
    size: file.size,
    conversationUUID,
    tempId: `${conversationUUID}-${Date.now()}-${uploadSequence++}`
  }))
  chatStore.uploadingFiles.push(...selectedFiles)
  if (!limitExceeded) emit('error', '')

  await Promise.all(
    selectedFiles.map(async (selectedFile) => {
      try {
        const resp = await api.uploadMedia(conversationUUID, selectedFile.file)
        if (resp.data.data) {
          chatStore.attachmentDrafts = {
            ...chatStore.attachmentDrafts,
            [conversationUUID]: [
              ...(chatStore.attachmentDrafts[conversationUUID] || []),
              resp.data.data
            ]
          }
        }
      } catch (error) {
        emit('error', handleHTTPError(error).message)
      } finally {
        chatStore.uploadingFiles = chatStore.uploadingFiles.filter(
          (item) => item.tempId !== selectedFile.tempId
        )
      }
    })
  )
}

const handleFileDelete = (uuid) => {
  chatStore.attachmentDrafts = {
    ...chatStore.attachmentDrafts,
    [draftKey.value]: mediaFiles.value.filter((attachment) => attachment.uuid !== uuid)
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
