<template>
  <div class="flex flex-col text-left" :class="isOutgoing ? 'items-end' : 'items-start'">
    <!-- Sender Name -->
    <div class="mb-1 flex items-center gap-1" :class="isOutgoing ? 'pr-[47px]' : 'pl-[47px]'">
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
      <router-link
        v-if="!isOutgoing"
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
              class="mb-1 native-html whitespace-pre-wrap break-words"
              :class="{ 'mb-3': message.attachments.length > 0 }"
            />
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
          <MessageAttachmentPreview :attachments="nonInlineAttachments" />

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
      <router-link
        v-if="isOutgoing && canManageUsers"
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
      <Avatar v-else-if="isOutgoing" class="w-8 h-8">
        <AvatarImage :src="getAvatar" />
        <AvatarFallback class="font-medium">
          {{ avatarFallback }}
        </AvatarFallback>
      </Avatar>
    </div>

    <!-- Timestamp tooltip -->
    <div :class="isOutgoing ? 'pr-[47px]' : 'pl-[47px]'">
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
import { computed, ref } from 'vue'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { useI18n } from 'vue-i18n'
import { Lock, Mail, RotateCcw, Check } from 'lucide-vue-next'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { formatMessageTimestamp, formatFullTimestamp } from '@shared-ui/utils/datetime.js'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { Letter } from 'vue-letter'
import ImageLightbox from '@/components/ImageLightbox.vue'
import MessageAttachmentPreview from '@main/features/conversation/message/attachment/MessageAttachmentPreview.vue'
import MessageEnvelope from './MessageEnvelope.vue'
import CSATResponseDisplay from './CSATResponseDisplay.vue'
import api from '@main/api'

const props = defineProps({
  message: Object,
  direction: {
    type: String,
    validator: (v) => ['incoming', 'outgoing'].includes(v)
  }
})

const convStore = useConversationStore()
const { t } = useI18n()
const userStore = useUserStore()

const isSystemUser = computed(() => props.message.author?.email === 'System')
const canManageUsers = computed(() => !isSystemUser.value && userStore.can('users:manage'))

// Direction helpers
const isOutgoing = computed(() => props.direction === 'outgoing')

// Author info from message
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

// Bubble classes - conditional based on direction
const bubbleClasses = computed(() => ({
  // Outgoing-specific: private message styling
  'bg-private': isOutgoing.value && props.message.private,
  'border border-border': isOutgoing.value && !props.message.private,
  'opacity-50 animate-pulse': isOutgoing.value && props.message.status === 'pending',
  'border-destructive': isOutgoing.value && props.message.status === 'failed',
  relative: isOutgoing.value,
  // Incoming-specific: quoted text visibility
  'show-quoted-text': !isOutgoing.value && showQuotedText.value,
  'hide-quoted-text': !isOutgoing.value && !showQuotedText.value
}))

// Outgoing-only computed properties
const isPrivateMessage = computed(() => isOutgoing.value && props.message.private)
const showCheckCheck = computed(
  () => isOutgoing.value && props.message.status === 'sent' && !isPrivateMessage.value
)
const showRetry = computed(() => isOutgoing.value && props.message.status === 'failed' && props.message.sender_id === userStore.userID)

const retryMessage = (msg) => {
  api.retryMessage(convStore.current.uuid, msg.uuid)
}

// Incoming-only: quoted text toggle
const showQuotedText = ref(false)
const hasQuotedContent = computed(
  () => !isOutgoing.value && sanitizedContent.value.includes('<blockquote')
)
const toggleQuote = () => {
  showQuotedText.value = !showQuotedText.value
}

// Inline image lightbox: click an <img> in the rendered email body to open it.
// We enumerate images from the rendered DOM rather than the HTML source so we
// inherit vue-letter's sanitization and don't have to parse HTML with regex
// (which trips on attributes containing '>' and similar edge cases).
const messageContentEl = ref(null)
const inlineLightboxOpen = ref(false)
const inlineLightboxIndex = ref(0)
const inlineImages = ref([])

// Re-walk the rendered <img> set on click. Cheaper than maintaining a watcher
// on sanitizedContent, and always reflects what the user actually sees.
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
  // Walk up so clicks on nested wrappers (e.g. <a><img></a>) still resolve.
  const img = event.target?.closest?.('img')
  if (!img || !messageContentEl.value?.contains(img)) return

  // If the image is inside an anchor, suppress the navigation so the
  // lightbox can take over.
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

// Envelope visibility (both directions)
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