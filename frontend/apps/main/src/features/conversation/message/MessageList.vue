<template>
  <div class="flex flex-col relative h-full">
    <div ref="threadEl" class="flex-1 overflow-y-auto" @scroll="handleScroll">
      <div class="min-h-full px-4 pb-10">
        <div
          class="text-center mt-3"
          v-if="
            conversationStore.currentConversationHasMoreMessages &&
            !conversationStore.messages.loading
          "
        >
          <Button
            size="sm"
            variant="outline"
            @click="conversationStore.fetchNextMessages"
            class="transition-all duration-200 hover:bg-gray-100 dark:hover:bg-gray-700 hover:scale-105 active:scale-95"
          >
            <RefreshCw size="17" class="mr-2" />
            {{ $t('globals.terms.loadMore') }}
          </Button>
        </div>

        <MessagesSkeleton :count="10" v-if="conversationStore.messages.loading" />

        <TransitionGroup v-else enter-active-class="animate-slide-in" tag="div">
          <div
            v-for="row in messageRows"
            :key="row.message.uuid"
            :data-message-uuid="row.message.uuid"
            :class="[row.spacingClass, { 'my-2': row.message.type === 'activity' }]"
          >
            <div v-if="!row.message.private && row.message.type !== 'activity'">
              <MessageBubble
                :message="row.message"
                :direction="row.message.type"
                :group-with-prev="row.groupWithPrev"
                :group-with-next="row.groupWithNext"
              />
            </div>
            <div v-else-if="isPrivateNote(row.message)">
              <MessageBubble
                :message="row.message"
                direction="outgoing"
                :group-with-prev="row.groupWithPrev"
                :group-with-next="row.groupWithNext"
              />
            </div>
            <div v-else-if="row.message.type === 'activity'">
              <ActivityMessageBubble :message="row.message" />
            </div>
          </div>
        </TransitionGroup>
      </div>

      <!-- Typing indicator -->
      <div v-if="conversationStore.conversation.isTyping" class="px-4 pb-4">
        <TypingIndicator />
      </div>
    </div>

    <!-- Sticky container for the scroll arrow -->
    <ScrollToBottomButton
      :is-at-bottom="isAtBottom"
      :unread-count="unReadMessages"
      @scroll-to-bottom="handleScrollToBottom"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import MessageBubble from './MessageBubble.vue'
import ActivityMessageBubble from './ActivityMessageBubble.vue'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { Button } from '@shared-ui/components/ui/button'
import { RefreshCw, ChevronDown } from 'lucide-vue-next'
import ScrollToBottomButton from '@shared-ui/components/ScrollToBottomButton'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import MessagesSkeleton from './MessagesSkeleton.vue'
import { TypingIndicator } from '@shared-ui/components/TypingIndicator'

const route = useRoute()

const conversationStore = useConversationStore()
const userStore = useUserStore()
const threadEl = ref(null)
const emitter = useEmitter()
const isAtBottom = ref(true)
const unReadMessages = ref(0)
const currentConversationUUID = ref('')

const checkIfAtBottom = () => {
  const thread = threadEl.value
  if (thread) {
    const tolerance = 100
    const isBottom = thread.scrollHeight - thread.scrollTop - thread.clientHeight <= tolerance
    isAtBottom.value = isBottom
  }
}

const handleScroll = () => {
  checkIfAtBottom()
}

const handleScrollToBottom = () => {
  scrollToBottom()
}

const scrollToBottom = () => {
  setTimeout(() => {
    const thread = threadEl.value
    if (thread) {
      thread.scrollTop = thread.scrollHeight
      checkIfAtBottom()
    }
  }, 50)
}

const scrollToMessage = (messageUUID) => {
  if (!messageUUID) {
    scrollToBottom()
    return
  }

  setTimeout(() => {
    const thread = threadEl.value
    const messageEl = thread?.querySelector(`[data-message-uuid="${messageUUID}"]`)
    if (messageEl && thread) {
      // Manual scroll calculation for reliability with variable-height messages
      const messageTop = messageEl.offsetTop
      const threadHeight = thread.clientHeight
      const messageHeight = messageEl.offsetHeight
      // Position message at ~1/3 from top of viewport for better visibility
      const targetScroll = messageTop - threadHeight / 3 + messageHeight / 2
      thread.scrollTop = Math.max(0, targetScroll)

      // Highlight the message briefly
      messageEl.classList.add('highlight-mention')
      setTimeout(() => messageEl.classList.remove('highlight-mention'), 2500)
    } else {
      // Message not found, scroll to bottom instead
      scrollToBottom()
    }
  }, 150)
}

onMounted(() => {
  checkIfAtBottom()
  handleNewMessage()
})

const newMessageHandler = (data) => {
  if (data.conversation_uuid === conversationStore.current.uuid) {
    // Agent's own message - always scroll to bottom
    if (data.message?.sender_id === userStore.userID) {
      scrollToBottom()
    }
    // Customer message - only scroll if already at bottom
    else if (isAtBottom.value) {
      scrollToBottom()
    }
    // Customer message but not at bottom - don't scroll, increment unread
    else {
      unReadMessages.value++
    }
  }
}

const handleNewMessage = () => {
  emitter.on(EMITTER_EVENTS.NEW_MESSAGE, newMessageHandler)
}

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.NEW_MESSAGE, newMessageHandler)
})

watch(
  () => conversationStore.conversationMessages,
  (messages) => {
    // Scroll to bottom when conversation changes and there are new messages.
    // New messages on next db page should not scroll to bottom.
    if (
      messages.length > 0 &&
      conversationStore?.current?.uuid &&
      currentConversationUUID.value !== conversationStore.current.uuid
    ) {
      currentConversationUUID.value = conversationStore.current.uuid
      unReadMessages.value = 0

      // Check if this is a mentioned conversation
      const scrollToUUID = route.query.scrollTo
      if (scrollToUUID) {
        // Mentioned conversation - only scroll to message, NOT to bottom
        scrollToMessage(scrollToUUID)
      } else {
        // Normal conversation - scroll to bottom
        scrollToBottom()
      }
    }
  }
)

// Watch for typing indicator and auto-scroll if user is at bottom
watch(
  () => conversationStore.conversation.isTyping,
  (isTyping) => {
    if (isTyping && isAtBottom.value) {
      scrollToBottom()
    }
  }
)

// Watch for isAtButtom and set unReadMessages to 0
watch(
  () => isAtBottom.value,
  (atBottom) => {
    if (atBottom) {
      unReadMessages.value = 0
    }
  }
)

const isPrivateNote = (message) => {
  return message.type === 'outgoing' && message.private
}

const GROUP_WINDOW_MS = 60_000

const canGroup = (a, b) => {
  if (!a || !b) return false
  if (a.type === 'activity' || b.type === 'activity') return false
  if (a.type !== b.type) return false
  if (Boolean(a.private) !== Boolean(b.private)) return false
  if (a.status === 'failed' || b.status === 'failed') return false

  const aSenderId = a.author?.id ?? a.sender_id
  const bSenderId = b.author?.id ?? b.sender_id
  if (!aSenderId || aSenderId !== bSenderId) return false

  const aBucket = Math.floor(new Date(a.created_at).getTime() / GROUP_WINDOW_MS)
  const bBucket = Math.floor(new Date(b.created_at).getTime() / GROUP_WINDOW_MS)
  return aBucket === bBucket
}

const getSpacingClass = (index, groupWithPrev) => {
  if (index === 0) return 'pt-4'
  return groupWithPrev ? 'mt-1' : 'mt-4'
}

const messageRows = computed(() => {
  const messages = conversationStore.conversationMessages
  return messages.map((message, index) => {
    const groupWithPrev = canGroup(messages[index - 1], message)
    const groupWithNext = canGroup(message, messages[index + 1])
    return {
      message,
      groupWithPrev,
      groupWithNext,
      spacingClass: getSpacingClass(index, groupWithPrev)
    }
  })
})
</script>

<style scoped>
.highlight-mention {
  animation: highlightPulse 2.5s ease-out;
}

@keyframes highlightPulse {
  0% {
    background-color: rgb(251 191 36 / 0.35);
    border-radius: 0.5rem;
  }
  100% {
    background-color: transparent;
  }
}

/* Dark mode highlight - softer yellow */
:global(.dark) .highlight-mention {
  animation: highlightPulseDark 2.5s ease-out;
}

@keyframes highlightPulseDark {
  0% {
    background-color: rgb(250 204 21 / 0.2);
    border-radius: 0.5rem;
  }
  100% {
    background-color: transparent;
  }
}
</style>
