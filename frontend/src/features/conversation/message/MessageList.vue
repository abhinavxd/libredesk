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
            <div v-if="!message.private && message.type !== 'activity'">
              <MessageBubble :message="message" :direction="message.type" />
            </div>
            <div v-else-if="isPrivateNote(message)">
              <MessageBubble :message="message" direction="outgoing" />
            </div>
            <div v-else-if="message.type === 'activity'">
              <ActivityMessageBubble :message="message" />
            </div>
          </div>
        </TransitionGroup>
      </div>
    </div>

    <!-- Sticky container for the scroll arrow -->
    <Transition
      enter-active-class="transition ease-out duration-200"
      enter-from-class="opacity-0 translate-y-1"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition ease-in duration-150"
      leave-from-class="opacity-100 translate-y-0"
      leave-to-class="opacity-0 translate-y-1"
    >
      <div v-show="!isAtBottom" class="absolute bottom-5 right-6 z-10">
        <button
          @click="handleScrollToBottom"
          class="w-10 h-10 rounded-full flex items-center justify-center shadow-lg border bg-background text-primary transition-colors duration-200 hover:bg-gray-100 dark:hover:bg-gray-700"
        >
          <ChevronDown size="18" />
        </button>
        <span
          v-if="unReadMessages > 0"
          class="absolute -top-1 -right-1 min-w-[20px] h-5 px-1.5 rounded-full bg-green-500 text-secondary text-xs font-medium flex items-center justify-center"
        >
          {{ unReadMessages }}
        </span>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import MessageBubble from './MessageBubble.vue'
import ActivityMessageBubble from './ActivityMessageBubble.vue'
import { useConversationStore } from '@/stores/conversation'
import { useUserStore } from '@/stores/user'
import { Button } from '@/components/ui/button'
import { RefreshCw, ChevronDown } from 'lucide-vue-next'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import MessagesSkeleton from './MessagesSkeleton.vue'

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
  unReadMessages.value = 0
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
      } else if (!isAtBottom.value) {
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

const isPrivateNote = (message) => {
  return message.type === 'outgoing' && message.private
}
</script>
