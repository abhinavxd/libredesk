<template>
  <div class="flex flex-col h-full">
    <!-- Chat header -->
    <ChatHeader @goBack="goBack" />

    <!-- Pre-chat form -->
    <PreChatForm
      v-if="showPreChatForm"
      @submit="handlePreChatFormSubmit"
      :exclude-default-fields="!!userStore.userSessionToken"
      :is-submitting="isInitializing"
      class="flex-1 min-h-0"
    />

    <!-- Messages container (when no pre-chat form) -->
    <ChatMessages
      v-else
      ref="chatMessages"
      :showPreChatForm="showPreChatForm"
      @quick-reply="handleQuickReply"
    />

    <!-- Error display -->
    <WidgetError :errorMessage="errorMessage" />

    <!-- Message input (only when pre-chat form is not shown) -->
    <MessageInput
      v-if="!showPreChatForm && !isConversationClosed"
      ref="messageInput"
      @error="handleError"
    />

    <!-- Closed conversation notice -->
    <div v-if="isConversationClosed" class="border-t p-4 text-center text-sm text-muted-foreground">
      {{ $t('widget.conversationClosed') }}
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useWidgetStore } from '../store/widget.js'
import { useUserStore } from '../store/user.js'
import { useChatStore } from '../store/chat.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api, { saveSession } from '@widget/api/index.js'
import WidgetError from '@widget/components/WidgetError.vue'
import ChatHeader from '@widget/components/ChatHeader.vue'
import ChatMessages from '@widget/components/ChatMessages.vue'
import MessageInput from '@widget/components/MessageInput.vue'
import PreChatForm from '@widget/components/PreChatForm.vue'

const widgetStore = useWidgetStore()
const userStore = useUserStore()
const chatStore = useChatStore()
const errorMessage = ref('')
const preChatFormSubmitted = ref(false)
const isInitializing = ref(false)
const config = computed(() => widgetStore.config)
const messageInput = ref(null)

// Forwards a guided-form choice button click to the same send path as typed messages.
const handleQuickReply = (text) => {
  messageInput.value?.sendQuickReply(text)
}

// Determine if pre-chat form should be shown
const showPreChatForm = computed(() => {
  const preChatForm = config.value?.prechat_form

  // Must be enabled and not submitted
  if (!preChatForm?.enabled || preChatFormSubmitted.value) {
    return false
  }

  // Atleast one field must be enabled
  const hasEnabledFields = preChatForm.fields?.some((field) => field.enabled)
  if (!hasEnabledFields) {
    return false
  }

  const isAnonymous = !userStore.userSessionToken
  const isNewConversation = !!userStore.userSessionToken && !chatStore.currentConversation?.uuid
  return isAnonymous || isNewConversation
})

// Check if conversation is closed and replies are not allowed
const isConversationClosed = computed(() => {
  const status = chatStore.currentConversation?.status
  if (status !== 'Closed') return false

  const settingsKey = userStore.isVisitor ? 'visitors' : 'users'
  return config.value?.[settingsKey]?.prevent_reply_to_closed_conversation ?? false
})

const goBack = () => {
  widgetStore.navigateToMessages()
}

const handleError = (message) => {
  errorMessage.value = message
  if (message) {
    setTimeout(() => {
      errorMessage.value = ''
    }, 5000)
  }
}

// Handle pre-chat form submission - init chat with form data and message
const handlePreChatFormSubmit = async ({ formData, message }) => {
  // Auto-submit with no message (e.g., all fields excluded) - just skip to chat
  if (!message) {
    preChatFormSubmitted.value = true
    return
  }

  isInitializing.value = true
  errorMessage.value = ''

  try {
    const payload = {
      message: message
    }

    if (Object.keys(formData).length > 0) {
      payload.form_data = formData
    }

    const resp = await api.initChatConversation(payload)
    const { conversation, session_token, user, messages, business_hours_id, working_hours_utc_offset } = resp.data.data
    conversation.business_hours_id = business_hours_id
    conversation.working_hours_utc_offset = working_hours_utc_offset

    if (!userStore.userSessionToken && session_token) {
      saveSession(session_token, user, userStore, true)
    }

    chatStore.addConversationToList(conversation)
    chatStore.setCurrentConversation(conversation)
    chatStore.replaceMessages(messages)

    preChatFormSubmitted.value = true
  } catch (error) {
    errorMessage.value = handleHTTPError(error).message
  } finally {
    isInitializing.value = false
  }
}
</script>
