<template>
  <div class="flex flex-col relative flex-1 min-h-0">
    <!-- Loading conversation overlay -->
    <div v-if="isLoadingConversation" class="absolute inset-0 bg-background/80 backdrop-blur-sm z-10">
      <Spinner size="md" :text="$t('globals.terms.loading')" absolute />
    </div>
    
    <div
      class="flex-1 min-h-0 overflow-y-auto p-4 flex flex-col gap-4 scrollbar-thin scrollbar-track-transparent scrollbar-thumb-muted-foreground/30 hover:scrollbar-thumb-muted-foreground/50"
      ref="messagesContainer"
      @scroll="handleScroll"
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
        v-for="message in chatStore.getCurrentConversationMessages"
        :key="message.uuid"
        :class="[
          'flex flex-col animate-slide-in',
          message.author.type === 'contact' || message.author.type === 'visitor'
            ? 'items-end'
            : 'items-start'
        ]"
      >
        <!-- CSAT Message Bubble -->
        <CSATMessageBubble
          v-if="message.meta?.is_csat"
          :message="message"
          @submitted="handleCSATSubmitted"
        />

        <!-- Regular Message Bubble -->
        <div
          v-else
          :class="[
            'max-w-[85%] px-4 py-3 rounded-2xl text-sm leading-5 break-words transition-all duration-200',
            message.author.type === 'contact' || message.author.type === 'visitor'
              ? [
                  'text-primary-foreground rounded-br-sm',
                  message.status === 'sending'
                    ? 'bg-primary/60'
                    : message.status === 'failed'
                      ? 'bg-destructive/60'
                      : 'bg-primary'
                ]
              : 'bg-background text-foreground rounded-bl-sm border border-border'
          ]"
        >
          <!-- Message content rendered using vue-letter -->
          <Letter
            :html="message.content"
            :allowedSchemas="['cid', 'https', 'http', 'mailto']"
            class="mb-1 native-html"
          />
          <!-- Show attachments if available -->
          <MessageAttachment :attachments="message.attachments" />
        </div>

        <!-- Message metadata -->
        <div class="text-xs text-muted-foreground mt-1 flex items-center gap-2">
          <!-- Agent name and time for agent messages -->
          <span v-if="message.author.type === 'agent'">
            {{ message.author.first_name }} {{ message.author.last_name }}
            •
            {{ getMessageTime(message.created_at) }}
          </span>

          <!-- Delivery status for user messages -->
          <span
            v-else-if="message.author.type === 'contact' || message.author.type === 'visitor'"
            class="flex items-center gap-1"
          >
            <span v-if="message.status === 'sending'" class="flex items-center gap-1">
              <div
                class="w-3 h-3 border border-current border-t-transparent rounded-full animate-spin"
              ></div>
              {{ $t('globals.messages.sending') }}
            </span>
            <span v-else>
              {{ getMessageTime(message.created_at) }}
            </span>
          </span>
        </div>
      </div>

      <!-- Typing Indicator -->
      <div v-if="isTyping" class="flex flex-col items-start">
        <div
          class="max-w-[85%] px-4 py-3 rounded-2xl text-sm leading-5 bg-background text-foreground rounded-bl-sm border border-border"
        >
          <TypingIndicator />
        </div>
      </div>
    </div>

    <!-- Sticky scroll to bottom button -->
    <ScrollToBottomButton
      :is-at-bottom="isAtBottom"
      :unread-count="unreadMessages"
      @scroll-to-bottom="handleScrollToBottom"
    />
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, watch } from 'vue'
import { useWidgetStore } from '../store/widget.js'
import { useChatStore } from '../store/chat.js'
import { useRelativeTime } from '@widget/composables/useRelativeTime.js'
import { Letter } from 'vue-letter'
import ScrollToBottomButton from '@shared-ui/components/ScrollToBottomButton'
import ChatIntro from './ChatIntro.vue'
import NoticeBanner from './NoticeBanner.vue'
import MessageAttachment from './MessageAttachment.vue'
import CSATMessageBubble from './CSATMessageBubble.vue'
import { TypingIndicator } from '@shared-ui/components/TypingIndicator'
import { Spinner } from '@shared-ui/components/ui/spinner'

const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const messagesContainer = ref(null)
const isAtBottom = ref(true)
const unreadMessages = ref(0)
const currentConversationUUID = ref('')

const config = computed(() => widgetStore.config)
const isTyping = computed(() => chatStore.isTyping)
const isLoadingConversation = computed(() => chatStore.isLoadingConversation)

const getMessageTime = (timestamp) => {
  return useRelativeTime(new Date(timestamp)).value
}

// handleCSATSubmitted updates the local message state when CSAT feedback is submitted.
const handleCSATSubmitted = ({ message_uuid, rating, feedback }) => {
  const currentMessage = chatStore.getCurrentConversationMessages.find(
    (m) => m.uuid === message_uuid
  )
  const updatedMeta = {
    ...currentMessage.meta,
    csat_submitted: true,
    is_csat: true
  }

  // Add submitted rating and feedback to meta if provided
  if (rating > 0) {
    updatedMeta.submitted_rating = rating
  }
  if (feedback && feedback.trim()) {
    updatedMeta.submitted_feedback = feedback.trim()
  }

  chatStore.replaceMessage(chatStore.currentConversation.uuid, message_uuid, {
    ...currentMessage,
    meta: updatedMeta
  })
}

const checkIfAtBottom = () => {
  const container = messagesContainer.value
  if (container) {
    const tolerance = 100
    const isBottom =
      container.scrollHeight - container.scrollTop - container.clientHeight <= tolerance
    isAtBottom.value = isBottom
  }
}

const handleScroll = () => {
  checkIfAtBottom()
}

const handleScrollToBottom = () => {
  unreadMessages.value = 0
  scrollToBottom()
}

const scrollToBottom = () => {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
      checkIfAtBottom()
    }
  })
}

onMounted(() => {
  // Check initial scroll position
  checkIfAtBottom()

  // Scroll to bottom on mount
  setTimeout(() => {
    scrollToBottom()
  }, 200)

  // Update conversation last seen timestamp.
  chatStore.updateCurrentConversationLastSeen()
})

// Only auto-scroll for user's own messages or when at bottom
watch(
  () => chatStore.getCurrentConversationMessages,
  (newMessages, oldMessages) => {
    if (!newMessages || newMessages.length === 0) return

    // Check if this is a new conversation
    const currentConvUUID = chatStore.currentConversation?.uuid
    if (currentConvUUID && currentConversationUUID.value !== currentConvUUID) {
      currentConversationUUID.value = currentConvUUID
      unreadMessages.value = 0
      scrollToBottom()
      return
    }

    // Check if new messages were added
    if (oldMessages && newMessages.length > oldMessages.length) {
      const newMessage = newMessages[newMessages.length - 1]

      // Auto-scroll if:
      // 1. Message is from current user (contact/visitor), OR
      // 2. User is already at the bottom
      if (
        newMessage.author?.type === 'contact' ||
        newMessage.author?.type === 'visitor' ||
        isAtBottom.value
      ) {
        scrollToBottom()
      } else {
        // User is scrolled up and agent sent message - show unread count
        unreadMessages.value++
      }
    }
  },
  { deep: true }
)

// Watch for typing indicator and auto-scroll if user is at bottom
watch(
  () => chatStore.isTyping,
  (isTyping) => {
    if (isTyping && isAtBottom.value) {
      scrollToBottom()
    }
  }
)
</script>
