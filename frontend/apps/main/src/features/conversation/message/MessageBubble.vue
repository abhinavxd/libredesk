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
            :key="sanitizedContent"
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

          <!-- Edit/Delete actions for private notes (author only, hidden once deleted) -->
          <div
            v-if="canEditPrivateNote && !isEditing"
            class="flex items-center gap-3 mt-1.5 self-end"
          >
            <button
              type="button"
              @click="startEdit"
              class="text-xs text-muted-foreground/60 hover:text-foreground transition-colors flex items-center gap-1"
            >
              <Pencil :size="12" />
              <span>{{ t('globals.messages.edit') }}</span>
            </button>
            <button
              type="button"
              @click="confirmDelete"
              class="text-xs text-muted-foreground/60 hover:text-destructive transition-colors flex items-center gap-1"
            >
              <Trash2 :size="12" />
              <span>{{ t('globals.messages.delete') }}</span>
            </button>
          </div>

          <!-- Inline edit textarea -->
          <div v-if="isEditing" class="mt-2 w-full">
            <textarea
              ref="editTextareaEl"
              v-model="editContent"
              class="w-full min-h-[60px] p-2 text-sm border rounded bg-background text-foreground resize-y"
              @keydown.escape="cancelEdit"
            />
            <div class="flex items-center gap-3 mt-1 justify-end">
              <button
                type="button"
                @click="cancelEdit"
                class="text-xs text-muted-foreground hover:text-foreground"
              >
                {{ t('globals.messages.cancel') }}
              </button>
              <button
                type="button"
                @click="saveEdit"
                :disabled="isSavingEdit || !editContent.trim()"
                class="text-xs text-primary font-medium hover:underline disabled:opacity-50"
              >
                {{ isSavingEdit ? t('globals.messages.saving') : t('globals.messages.save') }}
              </button>
            </div>
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
import { computed, nextTick, ref } from 'vue'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { useI18n } from 'vue-i18n'
import { Lock, Mail, RotateCcw, Check, Pencil, Trash2 } from 'lucide-vue-next'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { formatMessageTimestamp, formatFullTimestamp } from '@shared-ui/utils/datetime.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
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

// Edit/delete private notes — only the author of the note can mutate it.
// Hide the controls once meta.deleted is set so an already-tombstoned note
// can't be re-edited or re-deleted.
const emitter = useEmitter()
const isNoteDeleted = computed(() => Boolean(props.message.meta?.deleted))
const isNoteAuthor = computed(() => props.message.sender_id === userStore.userID)
const canEditPrivateNote = computed(
  () => isPrivateMessage.value && isNoteAuthor.value && !isNoteDeleted.value
)

const isEditing = ref(false)
const isSavingEdit = ref(false)
const editContent = ref('')
const editTextareaEl = ref(null)

const startEdit = () => {
  // Strip surrounding tags to get plain text — the textarea is a deliberate
  // simplification (no rich-text editor for notes). Saving wraps in <p> and
  // converts newlines to <br>, mirroring how the message was originally sent.
  const tmp = document.createElement('div')
  tmp.innerHTML = props.message.content || ''
  editContent.value = (tmp.textContent || '').trim()
  isEditing.value = true
  nextTick(() => editTextareaEl.value?.focus())
}

const cancelEdit = () => {
  isEditing.value = false
  editContent.value = ''
}

const saveEdit = async () => {
  const trimmed = editContent.value.trim()
  if (!trimmed) return
  isSavingEdit.value = true
  try {
    const html = '<p>' + trimmed.replace(/\n/g, '<br>') + '</p>'
    await api.updatePrivateNote(convStore.current.uuid, props.message.uuid, html)
    convStore.mergeMessageUpdate({
      conversation_uuid: convStore.current.uuid,
      uuid: props.message.uuid,
      content: html
    })
    isEditing.value = false
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  } finally {
    isSavingEdit.value = false
  }
}

const confirmDelete = async () => {
  if (!window.confirm(t('conversation.deletePrivateNoteConfirm'))) return
  try {
    await api.deletePrivateNote(convStore.current.uuid, props.message.uuid)
    // Mirror the server-side tombstone locally via the store so the bubble
    // updates without a refetch. The server replaces content with
    // "This note was deleted by <name>".
    convStore.mergeMessageUpdate({
      conversation_uuid: convStore.current.uuid,
      uuid: props.message.uuid,
      content: t('conversation.privateNoteDeleted'),
      meta: { ...(props.message.meta || {}), deleted: true }
    })
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  }
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