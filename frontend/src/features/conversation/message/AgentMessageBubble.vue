<template>
  <div class="flex flex-col items-end text-left">
    <!-- Sender Name -->
    <div class="pr-[47px] mb-1">
      <p class="text-muted-foreground text-sm font-medium">
        {{ getFullName }}
      </p>
    </div>

    <!-- Message Bubble -->
    <div class="flex flex-row gap-2 justify-end">
      <!-- Bubble Wrapper with max 80% width -->
      <div class="w-4/5 flex justify-end">
        <div
          class="flex flex-col justify-end message-bubble relative"
          :class="{
            '!bg-[#FEF1E1]': message.private,
            'bg-white border border-border': !message.private,
            'opacity-50 animate-pulse': message.status === 'pending',
            'bg-red-50 border-red-200': message.status === 'failed'
          }"
        >
          <!-- Message Envelope -->
          <MessageEnvelope :message="message" v-if="showEnvelope" />

          <hr class="mb-2" v-if="showEnvelope" />

          <!-- Message -->
          <div
            v-dompurify-html="messageContent"
            class="whitespace-pre-wrap break-words overflow-wrap-anywhere native-html"
            :class="{ 'mb-3': message.attachments.length > 0 }"
          />

          <!-- Attachments -->
          <MessageAttachmentPreview :attachments="nonInlineAttachments" />

          <!-- Spinner for Pending Messages -->
          <Spinner v-if="message.status === 'pending'" size="w-4 h-4" />

          <!-- Icons -->
          <div class="flex items-center space-x-2 mt-2 self-end">
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

      <!-- Avatar -->
      <Avatar class="cursor-pointer w-8 h-8">
        <AvatarImage :src="getAvatar" />
        <AvatarFallback class="font-medium">
          {{ avatarFallback }}
        </AvatarFallback>
      </Avatar>
    </div>

    <!-- Timestamp tooltip -->
    <div class="pr-[47px]">
      <Tooltip>
        <TooltipTrigger>
          <span class="text-muted-foreground text-xs mt-1">
            {{ format(message.updated_at, 'h:mm a') }}
          </span>
        </TooltipTrigger>
        <TooltipContent>
          {{ format(message.updated_at, "MMMM dd, yyyy 'at' HH:mm") }}
        </TooltipContent>
      </Tooltip>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { format } from 'date-fns'
import { useConversationStore } from '@/stores/conversation'
import { Lock, RotateCcw, Check } from 'lucide-vue-next'
import { revertCIDToImageSrc } from '@/utils/strings'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { Spinner } from '@/components/ui/spinner'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import MessageAttachmentPreview from '@/features/conversation/message/attachment/MessageAttachmentPreview.vue'
import MessageEnvelope from './MessageEnvelope.vue'
import api from '@/api'

const props = defineProps({
  message: Object
})
const convStore = useConversationStore()

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

const messageContent = computed(() => {
  return revertCIDToImageSrc(props.message.content)
})

const nonInlineAttachments = computed(() =>
  props.message.attachments.filter((attachment) => attachment.disposition !== 'inline')
)

const isPrivateMessage = computed(() => {
  return props.message.private
})

const showCheckCheck = computed(() => {
  return props.message.status == 'sent' && !isPrivateMessage.value
})

const showRetry = computed(() => {
  return props.message.status == 'failed'
})

const avatarFallback = computed(() => {
  const firstName = participant.value?.first_name ?? 'A'
  return firstName.toUpperCase().substring(0, 2)
})

const retryMessage = (msg) => {
  api.retryMessage(convStore.current.uuid, msg.uuid)
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

<style scoped>
.overflow-wrap-anywhere {
  overflow-wrap: anywhere;
}
</style>
