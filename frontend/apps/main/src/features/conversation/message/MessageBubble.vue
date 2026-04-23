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
          <Letter
            v-else
            :html="sanitizedContent"
            :allowedSchemas="['cid', 'https', 'http', 'mailto']"
            class="mb-1 native-html whitespace-pre-wrap break-words"
            :class="{ 'mb-3': message.attachments.length > 0 }"
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

      <!-- Forwarded-to badge under outgoing forwards -->
      <div
        v-if="forwardedToLabel"
        class="flex items-center gap-1 mt-1 pr-[47px] justify-end"
      >
        <span class="text-xs text-muted-foreground/70 italic flex items-center gap-1">
          <Forward :size="12" />
          {{ t('conversation.forwardedTo', { recipients: forwardedToLabel }) }}
        </span>
      </div>

      <!-- Reply-to-forward badge under tagged incoming messages -->
      <div
        v-if="isFromForward"
        class="flex items-center gap-1 mt-1 pl-[47px]"
      >
        <span class="text-xs text-muted-foreground/70 italic flex items-center gap-1">
          <Forward :size="12" />
          {{ t('conversation.internalReplyToForward') }}
        </span>
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

    <!-- Forward action — email-channel, non-private, non-activity messages only. -->
    <div
      v-if="canForward"
      class="flex items-center gap-1 mt-1"
      :class="isOutgoing ? 'pr-[47px] justify-end' : 'pl-[47px]'"
    >
      <button
        type="button"
        @click="forwardMessage"
        class="text-xs text-muted-foreground/60 hover:text-foreground transition-colors flex items-center gap-1"
      >
        <Forward :size="12" />
        <span>{{ t('conversation.forward') }}</span>
      </button>
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
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { useI18n } from 'vue-i18n'
import { Lock, Mail, RotateCcw, Check, Forward } from 'lucide-vue-next'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { formatMessageTimestamp, formatFullTimestamp } from '@shared-ui/utils/datetime.js'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { Letter } from 'vue-letter'
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

// Forward-related: outgoing forward (meta.forwarded_to) OR incoming reply
// to a forward (meta.from_forward, tagged at IMAP intake). Both share the
// private-note bubble color so the internal forward thread visually groups
// distinct from customer-facing messages.
const isForwardRelated = computed(
  () => Boolean(props.message.meta?.forwarded_to) || Boolean(props.message.meta?.from_forward)
)

// Bubble classes - conditional based on direction
const bubbleClasses = computed(() => ({
  // Outgoing-specific: private message styling
  'bg-private': (isOutgoing.value && props.message.private) || isForwardRelated.value,
  'border border-border': isOutgoing.value && !props.message.private && !isForwardRelated.value,
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

// Forward UI: emit a global event so ReplyBox can pick it up and switch
// into forward mode with the original message pre-quoted.
const emitter = useEmitter()
const forwardedToLabel = computed(() => {
  const fwd = props.message.meta?.forwarded_to
  if (Array.isArray(fwd) && fwd.length > 0) return fwd.join(', ')
  return ''
})
const isFromForward = computed(() => Boolean(props.message.meta?.from_forward))
const canForward = computed(
  () =>
    convStore.current?.inbox_channel === 'email' &&
    !props.message.private
)
const forwardMessage = () => {
  emitter.emit(EMITTER_EVENTS.FORWARD_MESSAGE, props.message)
}

// Incoming-only: quoted text toggle
const showQuotedText = ref(false)
const hasQuotedContent = computed(
  () => !isOutgoing.value && sanitizedContent.value.includes('<blockquote')
)
const toggleQuote = () => {
  showQuotedText.value = !showQuotedText.value
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