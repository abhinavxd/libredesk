<template>
  <div
    class="flex flex-col text-left"
    :class="
      isEmailMessage
        ? 'w-full items-stretch email-thread-message'
        : isOutgoing
          ? 'items-end'
          : 'items-start'
    "
  >
    <!-- Sender Name -->
    <div
      v-if="!groupWithPrev && !isEmailMessage"
      class="mb-1 flex items-center gap-1"
      :class="isOutgoing ? 'pr-2 md:pr-[47px]' : 'pl-10 md:pl-[47px]'"
    >
      <p class="text-muted-foreground text-sm font-medium">{{ getFullName }}</p>
    </div>

    <!-- Message Bubble -->
    <div
      class="flex w-full flex-row gap-2 group"
      :class="{ 'justify-end': isOutgoing && !isEmailMessage }"
    >
      <!-- Avatar (left for incoming) -->
      <template v-if="!isOutgoing && !isEmailMessage">
        <div v-if="!groupWithPrev" class="flex-shrink-0">
          <Avatar class="cursor-pointer w-8 h-8 hover:opacity-80 transition-opacity">
            <AvatarImage :src="getAvatar" />
            <AvatarFallback class="font-medium">
              {{ avatarFallback }}
            </AvatarFallback>
          </Avatar>
        </div>
        <div v-else class="w-8 flex-shrink-0" />
      </template>

      <div
        class="min-w-0"
        :class="[
          isEmailMessage ? 'w-full' : 'w-full md:w-4/5',
          { 'flex justify-end items-center gap-2': isOutgoing && !isEmailMessage }
        ]"
        style="contain: inline-size"
      >
        <!-- Delete note menu (private notes, appears on hover, left of bubble) -->
        <div
          v-if="canDeleteNote"
          class="flex-shrink-0 transition-opacity duration-200 can-hover:opacity-0 can-hover:group-hover:opacity-100 focus-within:!opacity-100"
        >
          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <Button variant="ghost" class="w-8 h-8 p-0 text-muted-foreground">
                <MoreHorizontal class="w-4 h-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                class="text-destructive focus:text-destructive"
                @click="alertOpen = true"
              >
                <Trash2 class="mr-2 h-4 w-4" />
                {{ t('conversation.deletePrivateNote') }}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        <div
          class="min-w-0 flex w-full flex-col"
          :class="isEmailMessage ? 'email-message-card' : isOutgoing ? 'items-end' : 'items-start'"
        >
          <!-- Conventional email header: transport details and date sit above the body. -->
          <div v-if="isEmailMessage" class="email-message-header">
            <MessageEnvelope :message="message" class="mb-0 min-w-0 flex-1" />
            <Tooltip>
              <TooltipTrigger>
                <time class="email-message-date" :datetime="message.created_at">
                  {{ formatMessageTimestamp(message.created_at) }}
                </time>
              </TooltipTrigger>
              <TooltipContent>
                <p>{{ formatFullTimestamp(message.created_at) }}</p>
              </TooltipContent>
            </Tooltip>
            <MessageConversationActions v-if="showConversationActions" />
          </div>

          <!-- Keep email transport details outside the bubble for other message types. -->
          <MessageEnvelope v-else-if="showEnvelope" :message="message" />

          <div class="flex flex-col justify-end message-bubble" :class="bubbleClasses">
            <div v-if="isDeleted" class="text-sm italic text-muted-foreground">
              {{ message.content }}
            </div>
            <template v-else>
              <!-- Message Content -->
              <div
                ref="contentWrapperEl"
                class="relative"
                :class="{ 'max-h-[400px] overflow-hidden': isExpandable && !isExpanded }"
              >
                <div
                  v-if="message.content_type === 'text' || message.meta?.is_csat"
                  class="mb-1 native-html whitespace-pre-wrap"
                  :class="{ 'mb-3': message.attachments.length > 0 }"
                >
                  {{ messageText }}
                </div>
                <div
                  v-else
                  :class="{
                    'email-light-canvas': isEmailMessage
                  }"
                >
                  <SafeMessageContent
                    :message="message"
                    :show-blocked-notice="false"
                    :show-quoted-text="isOutgoing || showQuotedText"
                    @image-click="onImageClick"
                    @resize="measureExpandable"
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
                      : isOutgoing
                        ? 'bg-gradient-to-t from-secondary via-secondary/90 to-transparent'
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
                class="text-xs cursor-pointer text-muted-foreground px-2 py-1 w-max hover:bg-muted hover:text-foreground rounded-md transition-colors duration-200"
              >
                {{
                  showQuotedText
                    ? t('conversation.hideQuotedText')
                    : t('conversation.showQuotedText')
                }}
              </div>

              <!-- Attachments -->
              <BubbleAttachmentPreview :attachments="nonInlineAttachments" />

              <!-- CSAT Response -->

              <!-- Spinner for Pending Messages (outgoing only) -->

              <!-- Status Icons (outgoing only) -->
              <div v-if="isOutgoing" class="flex items-center space-x-2 mt-2 self-end">
                <Lock :size="12" v-if="isPrivateMessage" class="text-muted-foreground" />

                <MessageDeliveryStatus v-if="!isPrivateMessage" :message="message" />
                <Tooltip v-if="message.meta?.continuity_emailed">
                  <TooltipTrigger>
                    <Mail :size="12" class="text-muted-foreground" />
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>{{ t('conversation.sentViaEmail') }}</p>
                  </TooltipContent>
                </Tooltip>
              </div>
            </template>
          </div>
        </div>
      </div>

      <!-- Avatar (right for outgoing) -->
      <template v-if="isOutgoing && !isEmailMessage">
        <div v-if="groupWithPrev" class="w-8 flex-shrink-0" />
        <Avatar v-else class="w-8 h-8">
          <AvatarImage :src="getAvatar" />
          <AvatarFallback class="font-medium">
            {{ avatarFallback }}
          </AvatarFallback>
        </Avatar>
      </template>
    </div>

    <!-- Timestamp tooltip -->
    <div v-if="!groupWithNext && !isEmailMessage" :class="isOutgoing ? 'pr-[47px]' : 'pl-[47px]'">
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

  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ t('conversation.deletePrivateNoteConfirmation') }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="deleteNote">{{
          t('globals.messages.delete')
        }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { computed, ref, onMounted, nextTick, watch } from 'vue'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { useI18n } from 'vue-i18n'
import { Lock, Mail, Maximize2, Trash2, MoreHorizontal } from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem
} from '@shared-ui/components/ui/dropdown-menu'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import { Button } from '@shared-ui/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import MessageDeliveryStatus from './MessageDeliveryStatus.vue'
import { formatMessageTimestamp, formatFullTimestamp } from '@shared-ui/utils/datetime.js'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import SafeMessageContent from '@shared-ui/components/SafeMessageContent.vue'
import ImageLightbox from '@/components/ImageLightbox.vue'
import BubbleAttachmentPreview from '@main/features/conversation/message/attachment/BubbleAttachmentPreview.vue'
import MessageEnvelope from './MessageEnvelope.vue'
import MessageConversationActions from './MessageConversationActions.vue'

import { containsQuoteMarkers } from '@shared-ui/utils/quotedContent.js'

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
})

const props = defineProps({
  message: Object,
  showConversationActions: Boolean,
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

const alertOpen = ref(false)

const deleteNote = () => {
  const conversationUUID = convStore.current?.uuid
  if (!conversationUUID) return
  convStore.deleteMessage(conversationUUID, props.message.uuid)
  alertOpen.value = false
}

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

const isEmailMessage = computed(
  () => convStore.current?.inbox_channel === 'email' && !props.message.private
)

const messageText = computed(() => {
  if (props.message.meta?.is_csat) {
    return t('globals.messages.pleaseRateConversation')
  }
  return props.message.content || ''
})

const nonInlineAttachments = computed(() =>
  props.message.attachments.filter(
    (attachment) => attachment.unavailable || attachment.disposition !== 'inline'
  )
)

const bubbleClasses = computed(() => ({
  'email-message-bubble': isEmailMessage.value,
  '!w-full':
    props.message.content_type !== 'text' && typeof props.message.display?.html === 'string',
  'bg-private': isOutgoing.value && props.message.private,
  'bg-secondary border border-border': isOutgoing.value && !props.message.private,
  'opacity-50 animate-pulse': isOutgoing.value && props.message.status === 'pending',
  'border-destructive': isOutgoing.value && props.message.status === 'failed',
  relative: isOutgoing.value,
  'show-quoted-text': !isOutgoing.value && showQuotedText.value,
  'hide-quoted-text': !isOutgoing.value && !showQuotedText.value
}))

const isPrivateMessage = computed(() => isOutgoing.value && props.message.private)
const isDeleted = computed(() => !!props.message.meta?.deleted_at)
const canDeleteNote = computed(
  () =>
    isPrivateMessage.value &&
    !isDeleted.value &&
    (props.message.sender_id === userStore.userID || userStore.hasAdminRole)
)
const showQuotedText = ref(false)
const hasQuotedContent = computed(
  () => !isOutgoing.value && containsQuoteMarkers(props.message.display?.html)
)
const toggleQuote = () => {
  showQuotedText.value = !showQuotedText.value
}

const inlineLightboxOpen = ref(false)
const inlineLightboxIndex = ref(0)
const inlineImages = ref([])

watch(
  () => [props.message.uuid, props.message.content, props.message.display?.html],
  async () => {
    inlineLightboxOpen.value = false
    inlineImages.value = []
    showQuotedText.value = false
    await nextTick()
    measureExpandable()
  }
)

const onImageClick = ({ images, index }) => {
  inlineImages.value = images
  inlineLightboxIndex.value = index
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
