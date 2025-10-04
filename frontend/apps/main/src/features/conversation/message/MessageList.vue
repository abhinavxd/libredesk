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

        <TransitionGroup v-else enter-active-class="animate-slide-in" tag="div" class="space-y-4">
          <div
            v-for="(message, index) in conversationStore.conversationMessages"
            :key="message.uuid"
            :class="{
              'my-2': message.type === 'activity',
              'pt-4': index === 0
            }"
          >
            <div v-if="!message.private">
              <ContactMessageBubble :message="message" v-if="message.type === 'incoming'" />
              <AgentMessageBubble :message="message" v-if="message.type === 'outgoing'" />
            </div>
            <div v-else-if="isPrivateNote(message)">
              <AgentMessageBubble :message="message" v-if="message.type === 'outgoing'" />
            </div>
            <div v-else-if="message.type === 'activity'">
              <ActivityMessageBubble :message="message" />
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
import { ref, onMounted, watch } from 'vue'
import ContactMessageBubble from './ContactMessageBubble.vue'
import ActivityMessageBubble from './ActivityMessageBubble.vue'
import AgentMessageBubble from './AgentMessageBubble.vue'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { Button } from '@shared-ui/components/ui/button'
import { RefreshCw } from 'lucide-vue-next'
import ScrollToBottomButton from '@shared-ui/components/ScrollToBottomButton'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import MessagesSkeleton from './MessagesSkeleton.vue'
import { TypingIndicator } from '@shared-ui/components/TypingIndicator'

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

onMounted(() => {
  checkIfAtBottom()
  handleNewMessage()
})

const handleNewMessage = () => {
  emitter.on(EMITTER_EVENTS.NEW_MESSAGE, (data) => {
    if (data.conversation_uuid === conversationStore.current.uuid) {
      if (data.message?.sender_id === userStore.userID) {
        scrollToBottom()
      } else if (isAtBottom.value) {
        // If user is at bottom, scroll to show new message
        scrollToBottom()
      } else {
        // If user is not at bottom, increment unread counter but do not scroll
        unReadMessages.value++
      }
    }
  })
}

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
      scrollToBottom()
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
</script>
