<template>
  <div class="flex flex-col items-start">
    <!-- Sender Name -->
    <div class="pl-[47px] mb-1">
      <p class="text-muted-foreground text-sm font-medium">
        {{ getFullName }}
      </p>
    </div>

    <!-- Message Bubble -->
    <div class="flex flex-row gap-2 w-full">
      <!-- Avatar -->
      <Avatar class="cursor-pointer w-8 h-8">
        <AvatarImage :src="getAvatar" />
        <AvatarFallback class="font-medium">
          {{ avatarFallback }}
        </AvatarFallback>
      </Avatar>

      <!-- Message Content -->
      <div class="w-4/5">
        <div
          class="flex flex-col justify-end message-bubble"
          :class="{
            'show-quoted-text': showQuotedText,
            'hide-quoted-text': !showQuotedText
          }"
        >
          <MessageEnvelope :message="message" v-if="showEnvelope" />

          <hr class="mb-2" v-if="showEnvelope" />

          <!-- Message Text -->
          <Letter
            :html="sanitizedMessageContent"
            :allowedSchemas="['cid', 'https', 'http', 'mailto']"
            class="mb-1 native-html"
            :class="{ 'mb-3': message.attachments.length > 0 }"
          />

          <!-- Quoted Text Toggle -->
          <div
            v-if="hasQuotedContent"
            @click="toggleQuote"
            class="text-xs cursor-pointer text-muted-foreground px-2 py-1 w-max hover:bg-muted hover:text-primary rounded-md transition-all"
          >
            {{
              showQuotedText ? t('conversation.hideQuotedText') : t('conversation.showQuotedText')
            }}
          </div>

          <!-- Attachments -->
          <MessageAttachmentPreview :attachments="nonInlineAttachments" />
        </div>
      </div>
    </div>

    <!-- Timestamp -->
    <div class="pl-[47px]">
      <Tooltip>
        <TooltipTrigger>
          <span class="text-muted-foreground text-xs mt-1">
            {{ format(message.updated_at, 'h:mm a') }}
          </span>
        </TooltipTrigger>
        <TooltipContent>
          <p>
            {{ format(message.updated_at, "MMMM dd, yyyy 'at' HH:mm") }}
          </p>
        </TooltipContent>
      </Tooltip>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { format } from 'date-fns'
import { useConversationStore } from '@/stores/conversation'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Letter } from 'vue-letter'
import { useAppSettingsStore } from '@/stores/appSettings'
import { useI18n } from 'vue-i18n'
import MessageAttachmentPreview from '@/features/conversation/message/attachment/MessageAttachmentPreview.vue'
import MessageEnvelope from './MessageEnvelope.vue'

const props = defineProps({
  message: Object
})

const convStore = useConversationStore()
const settingsStore = useAppSettingsStore()
const showQuotedText = ref(false)
const { t } = useI18n()

const getAvatar = computed(() => {
  return convStore.current?.contact?.avatar_url || ''
})
const sanitizedMessageContent = computed(() => {
  let content = props.message.content || ''
  const baseUrl = settingsStore.settings['app.root_url']

  // Replace CID with URL for inline attachments from the message.
  content = props.message.attachments.reduce(
    (acc, { content_id, url }) => acc.replace(new RegExp(`cid:${content_id}`, 'g'), url),
    content
  )

  // Add base URL to all img src starting with /uploads/ as vue-letter does not allow relative URLs.
  content = content.replace(/src="\/uploads\//g, `src="${baseUrl}/uploads/`)

  return content
})

const hasQuotedContent = computed(() => sanitizedMessageContent.value.includes('<blockquote'))

const toggleQuote = () => {
  showQuotedText.value = !showQuotedText.value
}

const nonInlineAttachments = computed(() =>
  props.message.attachments.filter((attachment) => attachment.disposition !== 'inline')
)

const getFullName = computed(() => {
  const contact = convStore.current?.contact || {}
  return `${contact.first_name || ''} ${contact.last_name || ''}`.trim()
})

const avatarFallback = computed(() => {
  const contact = convStore.current?.contact || {}
  return (contact.first_name || '').toUpperCase().substring(0, 2)
})

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
