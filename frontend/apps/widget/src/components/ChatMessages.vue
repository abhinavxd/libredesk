<template>
  <div class="flex flex-col relative flex-1 min-h-0">
    <!-- Loading conversation overlay -->
    <div v-if="isLoadingConversation" class="absolute inset-0 bg-background z-10" role="status">
      <Spinner size="md" :text="$t('globals.terms.loading')" absolute />
    </div>
    <div
      class="flex-1 min-h-0 overflow-y-auto [overflow-anchor:none] scrollbar-thin scrollbar-track-transparent scrollbar-thumb-muted-foreground/30 hover:scrollbar-thumb-muted-foreground/50"
      ref="messagesContainer"
      @scroll="handleScroll"
    >
      <div ref="contentEl" class="p-4 flex flex-col gap-4">
        <!-- Chat Intro -->
        <ChatIntro v-if="!props.showPreChatForm" :introText="config.chat_introduction" />

        <!-- Notice -->
        <NoticeBanner
          v-if="config.notice_banner.enabled === true && !props.showPreChatForm"
          :noticeText="config.notice_banner.text"
        />

        <ProactiveMessage
          v-if="proactive.pending && !chatStore.currentConversation?.uuid"
          :pending="proactive.pending"
        />

        <!-- Messages -->
        <TransitionGroup tag="div" enter-active-class="animate-slide-in" class="flex flex-col">
          <div
            v-for="{ message, groupWithPrev, groupWithNext } in messageRows"
            :key="message.uuid"
            :data-message-uuid="message.uuid"
            :class="[
              'flex flex-col',
              groupWithPrev ? 'mt-1' : 'mt-4 first:mt-0',
              message.author.type === 'contact' || message.author.type === 'visitor'
                ? 'items-end'
                : 'items-start'
            ]"
          >
            <CSATMessageBubble
              v-if="message.meta?.is_csat"
              :message="message"
              @submitted="handleCSATSubmitted"
            />

            <div
              v-else
              :class="[
                'max-w-[85%] px-4 py-3 rounded-2xl text-sm leading-5 break-words transition-all duration-200',
                message.author.type === 'contact' || message.author.type === 'visitor'
                  ? [
                      'text-primary-foreground',
                      message.status === 'sending' || message.status === 'uploading'
                        ? 'bg-primary/60'
                        : message.status === 'failed'
                          ? 'bg-destructive/60'
                          : 'bg-primary'
                    ]
                  : 'bg-muted text-foreground',
                {
                  'show-quoted-text': isQuotedTextVisible(message.uuid),
                  'hide-quoted-text': !isQuotedTextVisible(message.uuid)
                }
              ]"
            >
              <span v-if="message.content_type === 'text'" class="whitespace-pre-wrap">{{
                message.content
              }}</span>
              <Letter
                v-else
                :html="message.content"
                :allowedSchemas="['cid', 'https', 'http', 'mailto']"
                :allowed-css-properties="extendedCssProperties"
                class="native-html"
              />
              <div
                v-if="containsQuoteMarkers(message.content)"
                @click="toggleQuote(message.uuid)"
                @keydown.enter.prevent="toggleQuote(message.uuid)"
                @keydown.space.prevent="toggleQuote(message.uuid)"
                tabindex="0"
                role="button"
                :aria-expanded="isQuotedTextVisible(message.uuid)"
                :class="[
                  'text-xs cursor-pointer px-2 py-1 w-max rounded-md transition-all mt-1',
                  message.author.type === 'contact' || message.author.type === 'visitor'
                    ? 'text-primary-foreground/70 hover:bg-primary-foreground/10 hover:text-primary-foreground'
                    : 'text-muted-foreground hover:bg-muted hover:text-primary'
                ]"
              >
                {{
                  isQuotedTextVisible(message.uuid)
                    ? t('conversation.hideQuotedText')
                    : t('conversation.showQuotedText')
                }}
              </div>
              <MessageAttachment class="mt-1" :attachments="message.attachments" />

              <PreChatForm
                v-if="message.uuid === pendingHandoffMessageUUID"
                handoff-mode
                :is-submitting="props.handoffFormSubmitting"
                class="mt-3 border-t border-border pt-3"
                @submit="emit('handoff-form-submit', { ...$event, message })"
              />
            </div>

            <ReplyButtons
              v-if="message.uuid === suggestedReplyMessageUUID"
              :replies="message.meta.suggested_replies"
              class="mt-2 max-w-[85%]"
              @select="emit('suggested-reply', $event)"
            />

            <div
              v-if="!groupWithNext"
              class="text-[10px] text-muted-foreground mt-1 flex items-center gap-2"
            >
              <span v-if="message.author.type === 'agent'">
                {{ message.author.first_name }} {{ message.author.last_name }}
                •
                {{ getMessageTime(message.created_at) }}
              </span>

              <span
                v-else-if="message.author.type === 'contact' || message.author.type === 'visitor'"
                class="flex items-center gap-1"
              >
                <span
                  v-if="message.status === 'sending' || message.status === 'uploading'"
                  class="flex items-center gap-1"
                >
                  <div
                    class="w-3 h-3 border border-current border-t-transparent rounded-full animate-spin"
                  ></div>
                  <span v-if="message.status === 'sending'">
                    {{ $t('globals.messages.sending') }}
                  </span>
                  <span v-if="message.status === 'uploading'">
                    {{ $t('globals.messages.uploading') }}
                  </span>
                </span>
                <span v-else>
                  {{ getMessageTime(message.created_at) }}
                </span>
              </span>
            </div>
          </div>
        </TransitionGroup>

        <!-- Typing Indicator -->
        <div v-if="isTyping" class="flex flex-col items-start">
          <div class="max-w-[85%] px-4 py-3 rounded-2xl text-sm leading-5 bg-muted text-foreground">
            <TypingIndicator />
          </div>
        </div>
      </div>
    </div>

    <!-- Sticky scroll to bottom button -->
    <ScrollToBottomButton
      :is-at-bottom="!hasUserScrolled"
      :unread-count="unreadMessages"
      @scroll-to-bottom="handleScrollToBottom"
    />
  </div>
</template>

<script setup>
import { ref, computed, nextTick, watch } from 'vue'
import { useDocumentVisibility, useDebounceFn } from '@vueuse/core'
import { useWidgetStore } from '@widget/store/widget.js'
import { useChatStore } from '@widget/store/chat.js'
import { useRelativeTime } from '@widget/composables/useRelativeTime.js'
import { useI18n } from 'vue-i18n'
import { Letter } from 'vue-letter'
import { allowedCssProperties } from 'lettersanitizer'
import ScrollToBottomButton from '@shared-ui/components/ScrollToBottomButton'
import ChatIntro from './ChatIntro.vue'
import ProactiveMessage from './ProactiveMessage.vue'
import { useProactiveStore } from '@widget/store/proactive.js'
import NoticeBanner from './NoticeBanner.vue'
import MessageAttachment from './MessageAttachment.vue'
import CSATMessageBubble from './CSATMessageBubble.vue'
import PreChatForm from './PreChatForm.vue'
import ReplyButtons from './ReplyButtons.vue'
import { TypingIndicator } from '@shared-ui/components/TypingIndicator'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { containsQuoteMarkers } from '@shared-ui/utils/quotedContent.js'
import { useStickyScroll } from '@shared-ui/composables'

const extendedCssProperties = [...allowedCssProperties, 'transform', 'transform-origin']

const props = defineProps({
  showPreChatForm: {
    type: Boolean,
    default: false
  },
  handoffFormSubmitting: {
    type: Boolean,
    default: false
  }
})
const emit = defineEmits(['suggested-reply', 'handoff-form-submit'])

const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const proactive = useProactiveStore()
const messagesContainer = ref(null)
const contentEl = ref(null)
const unreadMessages = ref(0)
const currentConversationUUID = ref('')
const quotedTextState = ref({})
const { t } = useI18n()

const { hasUserScrolled, scrollToBottom, scrollToOffset, handleScroll } = useStickyScroll(
  messagesContainer,
  contentEl,
  {
    onArriveBottom: () => {
      unreadMessages.value = 0
    }
  }
)

const config = computed(() => widgetStore.config)
const isTyping = computed(() => chatStore.isTyping)
const isLoadingConversation = computed(() => chatStore.isLoadingConversation)
const GROUP_WINDOW_MS = 60_000
const canGroup = (a, b) => {
  if (!a || !b) return false
  if (a.meta?.is_csat || b.meta?.is_csat) return false
  if (a.status === 'failed' || b.status === 'failed') return false
  if (!a.author?.id || a.author.id !== b.author?.id) return false
  const aBucket = Math.floor(new Date(a.created_at).getTime() / GROUP_WINDOW_MS)
  const bBucket = Math.floor(new Date(b.created_at).getTime() / GROUP_WINDOW_MS)
  return aBucket === bBucket
}
const messageRows = computed(() => {
  const messages = chatStore.getCurrentConversationMessages
  return messages.map((message, index) => ({
    message,
    groupWithPrev: canGroup(messages[index - 1], message),
    groupWithNext: canGroup(message, messages[index + 1])
  }))
})
const suggestedReplyMessageUUID = computed(() => {
  const lastMessage = chatStore.getCurrentConversationLastMessage
  if (
    lastMessage?.author?.type === 'ai_assistant' &&
    Array.isArray(lastMessage.meta?.suggested_replies) &&
    lastMessage.meta.suggested_replies.length > 0
  ) {
    return lastMessage.uuid
  }
  return ''
})
const pendingHandoffMessageUUID = computed(() => {
  const lastMessage = chatStore.getCurrentConversationLastMessage
  return lastMessage?.meta?.handoff_form_pending ? lastMessage.uuid : ''
})

const getMessageTime = (timestamp) => {
  return useRelativeTime(new Date(timestamp)).value
}

const isQuotedTextVisible = (messageUuid) => {
  return quotedTextState.value[messageUuid] || false
}

const toggleQuote = (messageUuid) => {
  quotedTextState.value[messageUuid] = !quotedTextState.value[messageUuid]
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

const handleScrollToBottom = () => {
  hasUserScrolled.value = false
  scrollToBottom()
}

// Debounced version for tab-switch and widget-open triggers only.
// New message and conversation switch call the store function directly.
const debouncedUpdateLastSeen = useDebounceFn(() => {
  // Make sure widget is open and there's a convo loaded.
  if (widgetStore.isOpen && !document.hidden && chatStore.currentConversation?.uuid) {
    chatStore.updateCurrentConversationLastSeen()
  }
}, 2000)

const visibility = useDocumentVisibility()
watch(visibility, (state) => {
  if (state === 'visible' && widgetStore.isOpen && chatStore.currentConversation?.uuid) {
    debouncedUpdateLastSeen()
  }
})

// Conversation switch - reset scroll state and update last seen.
watch(
  () => chatStore.currentConversation?.uuid,
  (newUUID) => {
    if (!newUUID || currentConversationUUID.value === newUUID) return
    currentConversationUUID.value = newUUID
    unreadMessages.value = 0
    hasUserScrolled.value = false
    nextTick(() => {
      const id = chatStore.previewUnreadUUID
      chatStore.previewUnreadUUID = null
      const container = messagesContainer.value
      const target = id && contentEl.value?.querySelector(`[data-message-uuid="${id}"]`)
      const offset = target
        ? target.getBoundingClientRect().top -
          container.getBoundingClientRect().top +
          container.scrollTop
        : 0
      if (target && offset > 0 && container.scrollHeight - container.clientHeight - offset > 1) {
        hasUserScrolled.value = true
        scrollToOffset(offset)
      } else scrollToBottom()
    })
    if (widgetStore.isOpen && !chatStore.isLoadingConversation) {
      chatStore.updateCurrentConversationLastSeen()
    }
  },
  { immediate: true }
)

// New message arrival - update last seen for agent messages, increment unread if user scrolled up, force-stick for own messages.
watch(
  () => chatStore.getCurrentConversationMessages.length,
  (newLen, oldLen) => {
    if (oldLen === 0 && newLen > 0) {
      hasUserScrolled.value = false
      nextTick(scrollToBottom)
      return
    }
    if (!oldLen || !widgetStore.isOpen) return
    if (newLen <= oldLen) return
    const newMessage = chatStore.getCurrentConversationLastMessage
    const isOwnMessage =
      newMessage.author?.type === 'contact' || newMessage.author?.type === 'visitor'

    if (!isOwnMessage && !document.hidden) {
      chatStore.updateCurrentConversationLastSeen()
    }

    if (isOwnMessage) {
      hasUserScrolled.value = false
    } else if (hasUserScrolled.value) {
      unreadMessages.value++
    }
  }
)

// Widget opening - direct_to_conversation case where messages load while widget is hidden.
watch(
  () => widgetStore.isOpen,
  (isOpen) => {
    if (isOpen && chatStore.currentConversation?.uuid) {
      chatStore.updateCurrentConversationLastSeen()
      if (!hasUserScrolled.value) scrollToBottom()
    }
  }
)
</script>
