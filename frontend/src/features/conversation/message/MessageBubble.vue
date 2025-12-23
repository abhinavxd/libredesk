<template>
  <div class="flex flex-col text-left" :class="isOutgoing ? 'items-end' : 'items-start'">
    <!-- Sender Name -->
    <div class="mb-1" :class="isOutgoing ? 'pr-[47px]' : 'pl-[47px]'">
      <p class="text-muted-foreground text-sm font-medium">
        {{ getFullName }}
      </p>
    </div>

    <!-- Message Bubble -->
    <div class="flex flex-row gap-2 w-full" :class="{ 'justify-end': isOutgoing }">
      <!-- Avatar (left for incoming) -->
      <Avatar v-if="!isOutgoing" class="cursor-pointer w-8 h-8">
        <AvatarImage :src="getAvatar" />
        <AvatarFallback class="font-medium">
          {{ avatarFallback }}
        </AvatarFallback>
      </Avatar>

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
            class="text-xs cursor-pointer text-muted-foreground px-2 py-1 w-max hover:bg-muted hover:text-primary rounded transition-all"
          >
            {{ showQuotedText ? t('conversation.hideQuotedText') : t('conversation.showQuotedText') }}
          </div>

          <!-- Attachments -->
          <MessageAttachmentPreview :attachments="nonInlineAttachments" />

          <!-- Spinner for Pending Messages (outgoing only) -->
          <Spinner v-if="isOutgoing && message.status === 'pending'" size="w-4 h-4" />

          <!-- Status Icons (outgoing only) -->
          <div v-if="isOutgoing" class="flex items-center space-x-2 mt-2 self-end">
            <Lock :size="10" v-if="isPrivateMessage" class="text-muted-foreground" />
            <Check :size="14" v-if="showCheckCheck" class="text-green-500" />
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
      <Avatar v-if="isOutgoing" class="cursor-pointer w-8 h-8">
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
import { useConversationStore } from '@/stores/conversation'
import { useAppSettingsStore } from '@/stores/appSettings'
import { useI18n } from 'vue-i18n'
import { Lock, RotateCcw, Check } from 'lucide-vue-next'
import { revertCIDToImageSrc } from '@/utils/strings'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { Spinner } from '@/components/ui/spinner'
import { formatMessageTimestamp, formatFullTimestamp } from '@/utils/datetime'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Letter } from 'vue-letter'
import MessageAttachmentPreview from '@/features/conversation/message/attachment/MessageAttachmentPreview.vue'
import MessageEnvelope from './MessageEnvelope.vue'
import api from '@/api'

const props = defineProps({
  message: Object,
  direction: {
    type: String,
    validator: (v) => ['incoming', 'outgoing'].includes(v)
  }
})

const convStore = useConversationStore()
const settingsStore = useAppSettingsStore()
const { t } = useI18n()

// Direction helpers
const isOutgoing = computed(() => props.direction === 'outgoing')

// Participant info
const participant = computed(() => {
  return convStore.conversation?.participants?.[props.message.sender_id] ?? {}
})

const getFullName = computed(() => {
  const firstName = participant.value?.first_name ?? 'User'
  const lastName = participant.value?.last_name ?? ''
  return `${firstName} ${lastName}`
})

const getAvatar = computed(() => {
  return participant.value?.avatar_url || ''
})

const avatarFallback = computed(() => {
  const firstName = participant.value?.first_name ?? (isOutgoing.value ? 'A' : 'U')
  return firstName.toUpperCase().substring(0, 2)
})

// Content sanitization - different processing for incoming vs outgoing
const sanitizedContent = computed(() => {
  let content = props.message.content || ''

  if (isOutgoing.value) {
    return revertCIDToImageSrc(content)
  } else {
    const baseUrl = settingsStore.settings['app.root_url']
    content = props.message.attachments.reduce(
      (acc, { content_id, url }) => acc.replace(new RegExp(`cid:${content_id}`, 'g'), url),
      content
    )
    content = content.replace(/src="\/uploads\//g, `src="${baseUrl}/uploads/`)
    return content
  }
})

const nonInlineAttachments = computed(() =>
  props.message.attachments.filter((attachment) => attachment.disposition !== 'inline')
)

// Bubble classes - conditional based on direction
const bubbleClasses = computed(() => ({
  // Outgoing-specific: private message styling
  'bg-[#FEF1E1] dark:bg-[#4C3A24]': isOutgoing.value && props.message.private,
  'border border-border': isOutgoing.value && !props.message.private,
  'opacity-50 animate-pulse': isOutgoing.value && props.message.status === 'pending',
  'border-red-400': isOutgoing.value && props.message.status === 'failed',
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
const showRetry = computed(() => isOutgoing.value && props.message.status === 'failed')

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