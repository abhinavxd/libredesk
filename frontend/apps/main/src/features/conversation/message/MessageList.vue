<template>
  <div class="flex flex-col relative h-full">
    <div ref="threadEl" class="flex-1 overflow-y-auto [overflow-anchor:none]" @scroll="handleScroll">
      <div ref="contentEl" class="min-h-full px-4 pb-10">
        <div
          v-if="showLoadMore"
          class="text-center mt-3"
        >
          <Button
            size="sm"
            variant="outline"
            @click="loadMore"
            :disabled="conversationStore.messages.fetching"
            class="transition-all duration-200 hover:bg-accent hover:scale-105 active:scale-95"
          >
            <Loader2
              v-if="conversationStore.messages.fetching"
              size="17"
              class="mr-2 animate-spin"
            />
            <RefreshCw v-else size="17" class="mr-2" />
            {{ $t('globals.terms.loadMore') }}
          </Button>
        </div>

        <MessagesSkeleton :count="10" v-if="conversationStore.messages.loading" />

        <TransitionGroup v-else enter-active-class="animate-slide-in" tag="div">
          <div
            v-for="row in messageRows"
            :key="row.message.uuid"
            v-scroll-target="row.message.uuid"
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
            <div v-else-if="row.message.type === 'outgoing' && row.message.private">
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
      :is-at-bottom="!hasUserScrolled"
      :unread-count="unReadMessages"
      @scroll-to-bottom="handleScrollToBottom"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import MessageBubble from './MessageBubble.vue'
import ActivityMessageBubble from './ActivityMessageBubble.vue'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { Button } from '@shared-ui/components/ui/button'
import { RefreshCw, Loader2 } from 'lucide-vue-next'
import ScrollToBottomButton from '@shared-ui/components/ScrollToBottomButton'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import MessagesSkeleton from './MessagesSkeleton.vue'
import { TypingIndicator } from '@shared-ui/components/TypingIndicator'
import { useStickyScroll } from '@shared-ui/composables'

const route = useRoute()

const conversationStore = useConversationStore()
const userStore = useUserStore()
const threadEl = ref(null)
const contentEl = ref(null)
const emitter = useEmitter()
const unReadMessages = ref(0)
let currentConversationUUID = ''

const { hasUserScrolled, scrollToBottom, scrollToOffset, handleScroll } = useStickyScroll(threadEl, contentEl, {
  onArriveBottom: () => { unReadMessages.value = 0 }
})

const handleScrollToBottom = () => {
  hasUserScrolled.value = false
  scrollToBottom()
}

const vScrollTarget = {
  mounted (el, binding) {
    if (binding.value !== route.query.scrollTo || !threadEl.value) return
    hasUserScrolled.value = true
    scrollToOffset(Math.max(0, el.offsetTop - threadEl.value.clientHeight / 3 + el.offsetHeight / 2))
    el.classList.add('highlight-mention')
    setTimeout(() => el.classList.remove('highlight-mention'), 2500)
  }
}

const newMessageHandler = (data) => {
  if (data.conversation_uuid !== conversationStore.current.uuid) return
  if (data.message?.sender_id === userStore.userID) {
    hasUserScrolled.value = false
    return
  }
  if (hasUserScrolled.value) unReadMessages.value++
}

onMounted(() => {
  emitter.on(EMITTER_EVENTS.NEW_MESSAGE, newMessageHandler)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.NEW_MESSAGE, newMessageHandler)
})

watch(
  () => conversationStore.current?.uuid,
  (newUUID) => {
    if (!newUUID || newUUID === currentConversationUUID) return
    currentConversationUUID = newUUID
    unReadMessages.value = 0
    hasUserScrolled.value = !!route.query.scrollTo
  }
)

watch(
  () => conversationStore.conversationMessages.length,
  (newLen, oldLen) => {
    if (oldLen === 0 && newLen > 0 && !route.query.scrollTo) {
      hasUserScrolled.value = false
      nextTick(scrollToBottom)
    }
  }
)

// Watch for typing indicator and auto-scroll if user is at bottom
watch(
  () => conversationStore.conversation.isTyping,
  (isTyping) => {
    if (isTyping && !hasUserScrolled.value) scrollToBottom()
  }
)

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

const showLoadMore = computed(
  () => conversationStore.currentConversationHasMoreMessages && !conversationStore.messages.loading
)

const loadMore = async () => {
  const thread = threadEl.value
  if (!thread) return
  const prevHeight = thread.scrollHeight
  const prevTop = thread.scrollTop
  await conversationStore.fetchNextMessages()
  await nextTick()
  thread.scrollTop = thread.scrollHeight - prevHeight + prevTop
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
