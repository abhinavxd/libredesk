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

    <!-- Escape hatch out of an in-progress guided form -->
    <div v-if="isInGuidedForm" class="px-4 py-1.5 border-t">
      <button
        type="button"
        class="text-xs text-muted-foreground hover:text-foreground underline cursor-pointer"
        @click="skipGuidedForm"
      >
        {{ $t('widget.talkToHuman') }}
      </button>
    </div>

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
import api from '@widget/api/index.js'
import { initConversation } from '@widget/composables/useChatInit.js'
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

// Lets a visitor bail out of an in-progress guided form and reach a human directly, instead of
// being stuck answering questions until something matches a branch.
const isInGuidedForm = computed(
  () =>
    chatStore.currentConversation?.assignee?.type === 'guided_form_bot' &&
    !!chatStore.currentConversation?.guided_form_allow_skip
)

const skipGuidedForm = async () => {
  if (!chatStore.currentConversation?.uuid) return
  try {
    await api.skipGuidedForm(chatStore.currentConversation.uuid)
  } catch (error) {
    errorMessage.value = handleHTTPError(error).message
  }
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

// Handle pre-chat form submission - init chat with form data and message.
const handlePreChatFormSubmit = async ({ formData, message }) => {
  // No message and nothing to proactively start a conversation with (no guided form) - just
  // skip to an empty chat and wait for the visitor to type.
  if (!message && !config.value?.has_guided_form) {
    preChatFormSubmitted.value = true
    return
  }

  isInitializing.value = true
  errorMessage.value = ''

  try {
    const payload = {}
    if (message) payload.message = message
    if (Object.keys(formData).length > 0) {
      payload.form_data = formData
    }

    await initConversation(payload)
    preChatFormSubmitted.value = true
  } catch (error) {
    errorMessage.value = handleHTTPError(error).message
  } finally {
    isInitializing.value = false
  }
}
</script>
