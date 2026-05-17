<template>
  <div class="flex flex-col text-left" :class="isOutgoing ? 'items-end' : 'items-start'">
    <!-- Sender Name -->
    <div
      v-if="!groupWithPrev"
      class="mb-1 flex items-center gap-1"
      :class="isOutgoing ? 'pr-[47px]' : 'pl-[47px]'"
    >
      <router-link
        v-if="!isOutgoing"
        :to="{ name: 'contact-detail', params: { id: message.author?.id } }"
        class="cursor-pointer text-muted-foreground text-sm font-medium hover:underline hover:text-primary transition-colors duration-200"
      >
        {{ getFullName }}
      </router-link>
      <router-link
        v-else-if="canManageUsers"
        :to="{ name: 'edit-agent', params: { id: message.author?.id } }"
        class="cursor-pointer text-muted-foreground text-sm font-medium hover:underline hover:text-primary transition-colors duration-200"
      >
        {{ getFullName }}
      </router-link>
      <p v-else class="text-muted-foreground text-sm font-medium">
        {{ getFullName }}
      </p>
    </div>

    <!-- Message Bubble -->
    <div class="flex flex-row gap-2 w-full" :class="{ 'justify-end': isOutgoing }">
      <!-- Avatar (left for incoming) -->
      <template v-if="!isOutgoing">
        <router-link
          v-if="!groupWithPrev"
          :to="{ name: 'contact-detail', params: { id: message.author?.id } }"
          class="flex-shrink-0"
        >
          <Avatar class="cursor-pointer w-8 h-8 hover:opacity-80 transition-opacity">
            <AvatarImage :src="getAvatar" />
            <AvatarFallback class="font-medium">
              {{ avatarFallback }}
            </AvatarFallback>
          </Avatar>
        </router-link>
        <div v-else class="w-8 flex-shrink-0" />
      </template>

      <!-- Bubble Wrapper with max 80% width -->
      <div
        class="w-4/5"
        :class="{ 'flex justify-end': isOutgoing }"
        style="contain: inline-size"
      >
        <div
          class="flex flex-col justify-end message-bubble"
          :class="bubbleClasses"
        >
          <!-- Message Envelope -->
          <MessageEnvelope :message="message" v-if="showEnvelope" />

          <hr class="mb-2" v-if="showEnvelope" />

          <!-- Message Content -->
          <div
            ref="contentWrapperEl"
            class="relative"
            :class="{ 'max-h-[400px] overflow-hidden': isExpandable && !isExpanded }"
          >
            <div
              v-if="message.content_type === 'text'"
              class="mb-1 native-html whitespace-pre-wrap"
              :class="{ 'mb-3': message.attachments.length > 0 }"
            >
              {{ sanitizedContent }}
            </div>
            <div v-else ref="messageContentEl" @click="onMessageContentClick">
              <Letter
                :html="sanitizedContent"
                :allowedSchemas="['cid', 'https', 'http', 'mailto']"
                :allowed-css-properties="extendedCssProperties"
                class="mb-1 native-html break-words"
                :class="{ 'mb-3': message.attachments.length > 0 }"
              />
            </div>

            <div
              v-if="isExpandable && !isExpanded"
              class="absolute left-0 right-0 bottom-0 h-24 flex items-end justify-center pointer-events-none"
              :class="
                message.private
                  ? 'bg-gradient-to-t from-private via-private/90 to-transparent'
                  : 'bg-gradient-to-t from-background via-background/90 to-transparent'
              "
            >
              <button
                type="button"
                @click="isExpanded = true"
                class="pointer-events-auto flex items-center gap-1.5 text-xs font-medium text-foreground bg-accent hover:bg-accent/80 border border-border rounded-full px-3 py-1 mb-1 transition-colors duration-200"
              >
                <Maximize2 :size="12" />
                {{ t('globals.terms.expand') }}
              </button>
            </div>
          </div>

          <ImageLightbox
            v-model="inlineLightboxOpen"
            :images="inlineImages"
            :start-index="inlineLightboxIndex"
          />

          <!-- Quoted Text Toggle (incoming only) -->
          <div
            v-if="!isOutgoing && hasQuotedContent"
            @click="toggleQuote"
            class="text-xs cursor-pointer text-muted-foreground px-2 py-1 w-max hover:bg-muted hover:text-primary rounded transition-colors duration-200"
          >
            {{ showQuotedText ? t('conversation.hideQuotedText') : t('conversation.showQuotedText') }}
          </div>

          <!-- Attachments -->
          <BubbleAttachmentPreview :attachments="nonInlineAttachments" />

          <!-- CSAT Response -->
          <CSATResponseDisplay :message="message" />

          <!-- Spinner for Pending Messages (outgoing only) -->
          <Spinner v-if="isOutgoing && message.status === 'pending'" size="sm" />

          <!-- Status Icons (outgoing only) -->
          <div v-if="isOutgoing" class="flex items-center space-x-2 mt-2 self-end">
            <Lock :size="10" v-if="isPrivateMessage" class="text-muted-foreground" />
            <Check :size="14" v-if="showCheckCheck" class="text-green-500" />
            <Tooltip v-if="message.meta?.continuity_emailed">
              <TooltipTrigger>
                <Mail :size="12" class="text-muted-foreground" />
              </TooltipTrigger>
              <TooltipContent>
                <p>{{ t('conversation.sentViaEmail') }}</p>
              </TooltipContent>
            </Tooltip>
            <RotateCcw
              size="10"
              @click="retryMessage(message)"
              class="cursor-pointer text-muted-foreground hover:text-foreground transition-colors duration-200"
              v-if="showRetry"
            />
          </div>
        </div>
      </div>

      <!-- Avatar (right for outgoing) -->
      <template v-if="isOutgoing">
        <div v-if="groupWithPrev" class="w-8 flex-shrink-0" />
        <router-link
          v-else-if="canManageUsers"
          :to="{ name: 'edit-agent', params: { id: message.author?.id } }"
          class="flex-shrink-0"
        >
          <Avatar class="cursor-pointer w-8 h-8 hover:opacity-80 transition-opacity">
            <AvatarImage :src="getAvatar" />
            <AvatarFallback class="font-medium">
              {{ avatarFallback }}
            </AvatarFallback>
          </Avatar>
        </router-link>
        <Avatar v-else class="w-8 h-8">
          <AvatarImage :src="getAvatar" />
          <AvatarFallback class="font-medium">
            {{ avatarFallback }}
          </AvatarFallback>
        </Avatar>
      </template>
    </div>

    <!-- Timestamp tooltip -->
    <div v-if="!groupWithNext" :class="isOutgoing ? 'pr-[47px]' : 'pl-[47px]'">
      <Tooltip>
        <TooltipTrigger>
          <span class="text-muted-foreground text-xs mt-1">
            {{ formatMessageTimestamp(message.created_at) }}
          </span>
        </TooltipTrigger>
        <TooltipContent>
          <p>{{ formatFullTimestamp(message.created_at) }}</p>
        </TooltipContent>
      </Tooltip>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted, nextTick } from 'vue'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { useI18n } from 'vue-i18n'
import { Lock, Mail, RotateCcw, Check, Maximize2 } from 'lucide-vue-next'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { formatMessageTimestamp, formatFullTimestamp } from '@shared-ui/utils/datetime.js'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { Letter } from 'vue-letter'
import { allowedCssProperties } from 'lettersanitizer'
import ImageLightbox from '@/components/ImageLightbox.vue'
import BubbleAttachmentPreview from '@main/features/conversation/message/attachment/BubbleAttachmentPreview.vue'
import MessageEnvelope from './MessageEnvelope.vue'
import CSATResponseDisplay from './CSATResponseDisplay.vue'
import api from '@main/api'
import { containsQuoteMarkers } from '@shared-ui/utils/quotedContent.js'

const extendedCssProperties = [...allowedCssProperties, 'transform', 'transform-origin']

const COLLAPSE_THRESHOLD_PX = 400

const contentWrapperEl = ref(null)
const isExpandable = ref(false)
const isExpanded = ref(false)

const measureExpandable = () => {
  const el = contentWrapperEl.value
  if (!el) return
  isExpandable.value = el.scrollHeight > COLLAPSE_THRESHOLD_PX
}

onMounted(async () => {
  await nextTick()
  measureExpandable()

  // Email HTML images change height after initial paint - re-measure on load.
  const imgs = contentWrapperEl.value?.querySelectorAll?.('img') ?? []
  imgs.forEach((img) => {
    if (!img.complete) img.addEventListener('load', measureExpandable, { once: true })
  })
})

const props = defineProps({
  message: Object,
  direction: {
    type: String,
    validator: (v) => ['incoming', 'outgoing'].includes(v)
  },
  groupWithPrev: {
    type: Boolean,
    default: false
  },
  groupWithNext: {
    type: Boolean,
    default: false
  }
})

const convStore = useConversationStore()
const { t } = useI18n()
const userStore = useUserStore()

const isSystemUser = computed(() => props.message.author?.email === 'System')
const canManageUsers = computed(() => !isSystemUser.value && userStore.can('users:manage'))

const isOutgoing = computed(() => props.direction === 'outgoing')

const getFullName = computed(() => {
  const author = props.message.author ?? {}
  const firstName = author.first_name ?? 'User'
  const lastName = author.last_name ?? ''
  return `${firstName} ${lastName}`.trim()
})

const getAvatar = computed(() => {
  return props.message.author?.avatar_url || ''
})

const avatarFallback = computed(() => {
  const firstName = props.message.author?.first_name ?? (isOutgoing.value ? 'A' : 'U')
  return firstName.toUpperCase().substring(0, 2)
})

const sanitizedContent = computed(() => {
  if (props.message.meta?.is_csat) {
    return t('globals.messages.pleaseRateConversation')
  }
  return props.message.content || ''
})

const nonInlineAttachments = computed(() =>
  props.message.attachments.filter((attachment) => attachment.disposition !== 'inline')
)

const bubbleClasses = computed(() => ({
  'bg-private': isOutgoing.value && props.message.private,
  'border border-border': isOutgoing.value && !props.message.private,
  'opacity-50 animate-pulse': isOutgoing.value && props.message.status === 'pending',
  'border-destructive': isOutgoing.value && props.message.status === 'failed',
  relative: isOutgoing.value,
  'show-quoted-text': !isOutgoing.value && showQuotedText.value,
  'hide-quoted-text': !isOutgoing.value && !showQuotedText.value
}))

const isPrivateMessage = computed(() => isOutgoing.value && props.message.private)
const showCheckCheck = computed(
  () => isOutgoing.value && props.message.status === 'sent' && !isPrivateMessage.value
)
const showRetry = computed(() => isOutgoing.value && props.message.status === 'failed' && props.message.sender_id === userStore.userID)

const retryMessage = (msg) => {
  api.retryMessage(convStore.current.uuid, msg.uuid)
}

const showQuotedText = ref(false)
const hasQuotedContent = computed(
  () => !isOutgoing.value && containsQuoteMarkers(sanitizedContent.value)
)
const toggleQuote = () => {
  showQuotedText.value = !showQuotedText.value
}

// Enumerate from rendered DOM (not HTML source) to inherit vue-letter's
// sanitization and dodge regex parsing of attributes containing '>'.
const messageContentEl = ref(null)
const inlineLightboxOpen = ref(false)
const inlineLightboxIndex = ref(0)
const inlineImages = ref([])

// Re-walk on click instead of caching - cheaper than watching sanitizedContent
// and always reflects what the user actually sees.
const refreshInlineImages = () => {
  const root = messageContentEl.value
  if (!root) {
    inlineImages.value = []
    return
  }
  inlineImages.value = Array.from(root.querySelectorAll('img'))
    .map((el) => ({ url: el.getAttribute('src'), name: el.getAttribute('alt') || '' }))
    .filter((img) => img.url)
}

const onMessageContentClick = (event) => {
  // closest('img') so clicks on <a><img></a> wrappers still resolve.
  const img = event.target?.closest?.('img')
  if (!img || !messageContentEl.value?.contains(img)) return

  // Suppress anchor navigation so the lightbox can take over.
  const wrappingAnchor = img.closest('a')
  if (wrappingAnchor && messageContentEl.value.contains(wrappingAnchor)) {
    event.preventDefault()
  }

  refreshInlineImages()
  const src = img.getAttribute('src')
  const idx = inlineImages.value.findIndex((entry) => entry.url === src)
  inlineLightboxIndex.value = idx >= 0 ? idx : 0
  inlineLightboxOpen.value = true
}

const showEnvelope = computed(() => {
  return (
    props.message.meta?.from?.length ||
    props.message.meta?.to?.length ||
    props.message.meta?.cc?.length ||
    props.message.meta?.bcc?.length ||
    props.message.meta?.subject
  )
})
</script>

<style scoped lang="scss">
.native-html :deep(img) {
  max-width: 100%;
  height: auto;
  cursor: zoom-in;
}
</style>