<template>
  <AlertDialog :open="showContactEmailWarning" @update:open="showContactEmailWarning = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('replyBox.contactEmailMissing') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{
            $t('replyBox.contactEmailMissingDescription', {
              email: conversationStore.current?.contact?.email
            })
          }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction @click="processSend(true, true, deferredStatus)">{{
          $t('replyBox.sendAnyway')
        }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>

  <AlertDialog :open="showMissingTagsWarning" @update:open="showMissingTagsWarning = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('replyBox.missingTagsTitle') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ $t('replyBox.missingTagsDescription') }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction @click="processSend(false, true, deferredStatus)">{{
          $t('replyBox.sendAnyway')
        }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>

  <div class="text-foreground bg-background">
    <!-- Fullscreen editor -->
    <Dialog :open="isEditorFullscreen" @update:open="isEditorFullscreen = false">
      <DialogContent
        class="max-w-[60%] max-h-[75%] h-[70%] bg-card text-card-foreground p-4 flex flex-col"
        :class="{ '!bg-private': messageType === 'private_note', 'ai-generating': isGenerating }"
        @escapeKeyDown="isEditorFullscreen = false"
        :hide-close-button="true"
      >
        <ReplyBoxContent
          v-if="isEditorFullscreen"
          :isFullscreen="true"
          :aiPrompts="aiPrompts"
          :isSending="isSending"
          :isDraftLoading="isDraftLoading"
          :uploadingFiles="uploadingFiles"
          :uploadedFiles="mediaFiles"
          v-model:htmlContent="htmlContent"
          v-model:textContent="textContent"
          v-model:to="to"
          v-model:cc="cc"
          v-model:bcc="bcc"
          v-model:emailErrors="emailErrors"
          v-model:messageType="messageType"
          v-model:showBcc="showBcc"
          v-model:mentions="mentions"
          @toggleFullscreen="isEditorFullscreen = !isEditorFullscreen"
          @send="processSend"
          @sendAndSetStatus="processSendAndSetStatus"
          @fileUpload="handleFileUpload"
          @fileDelete="handleFileDelete"
          @filesDropped="uploadFiles"
          @aiPromptSelected="handleAiPromptSelected"
          :isGenerating="isGenerating"
          @generateReply="handleGenerateReply"
          class="h-full flex-grow"
        />
      </DialogContent>
    </Dialog>

    <!-- Main Editor non-fullscreen -->
    <div
      class="bg-background text-card-foreground box m-2 px-2 pt-2 flex flex-col relative"
      :class="{ '!bg-private': messageType === 'private_note', 'ai-generating': isGenerating }"
      v-if="!isEditorFullscreen"
    >
      <ReplyBoxContent
        ref="replyBoxContentRef"
        :isFullscreen="false"
        :aiPrompts="aiPrompts"
        :isSending="isSending"
        :isDraftLoading="isDraftLoading"
        :uploadingFiles="uploadingFiles"
        :uploadedFiles="mediaFiles"
        v-model:htmlContent="htmlContent"
        v-model:textContent="textContent"
        v-model:to="to"
        v-model:cc="cc"
        v-model:bcc="bcc"
        v-model:emailErrors="emailErrors"
        v-model:messageType="messageType"
        v-model:showBcc="showBcc"
        v-model:mentions="mentions"
        @toggleFullscreen="isEditorFullscreen = !isEditorFullscreen"
        @send="processSend"
        @sendAndSetStatus="processSendAndSetStatus"
        @fileUpload="handleFileUpload"
        @fileDelete="handleFileDelete"
        @filesDropped="uploadFiles"
        @aiPromptSelected="handleAiPromptSelected"
        :isGenerating="isGenerating"
        @generateReply="handleGenerateReply"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, watch, computed, toRaw, onMounted, onUnmounted } from 'vue'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { MACRO_CONTEXT } from '@main/constants/conversation'
import { WHATSAPP_CHANNEL, isWhatsAppWindowOpen } from '@main/features/conversation/whatsappTemplate'
import { useUserStore } from '@main/stores/user'
import { useDraftManager } from '@main/composables/useDraftManager'
import api from '@main/api'
import { useI18n } from 'vue-i18n'
import { useConversationStore } from '@main/stores/conversation'
import { useInboxStore } from '@main/stores/inbox'
import { useAiPromptStore } from '@main/stores/aiPrompt'
import { useNotificationStore } from '@main/stores/notification'
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
import { Dialog, DialogContent } from '@shared-ui/components/ui/dialog'
import { useEmitter } from '@main/composables/useEmitter'
import { useFileUpload } from '@main/composables/useFileUpload'
import { hasInlineImage, hasPendingInlineUpload } from '@main/composables/useInlineImageUpload'
import ReplyBoxContent from '@/features/conversation/ReplyBoxContent.vue'
import { UserTypeAgent } from '@/constants/user'

const { t } = useI18n()
const conversationStore = useConversationStore()
const notificationStore = useNotificationStore()
const inboxStore = useInboxStore()
const emitter = useEmitter()
const userStore = useUserStore()

// Setup file upload composable
const {
  uploadingFiles,
  handleFileUpload: _handleFileUpload,
  handleFileDelete,
  uploadFiles: _uploadFiles,
  mediaFiles,
  clearMediaFiles,
  setMediaFiles
} = useFileUpload({
  linkedModel: 'messages'
})

// Meta's per-type upload caps, less the 2% headroom the backend applies. Mirrors maxMediaBytes in internal/inbox/channel/whatsapp.
const WA_MAX_FILE_MB = { image: 5 * 0.98, video: 16 * 0.98, audio: 16 * 0.98, document: 100 * 0.98 }

function waMediaType(contentType = '') {
  if (['image/jpeg', 'image/png'].includes(contentType)) return 'image'
  if (['video/mp4', 'video/3gpp', 'video/3gp'].includes(contentType)) return 'video'
  if (contentType.startsWith('audio/')) return 'audio'
  return 'document'
}

// WhatsApp takes one attachment per message, so a multi-file reply is sent as several messages, the caption riding the first.
function buildWhatsAppReplyParts(content, files) {
  const parts = []
  let caption = content
  if (hasTextContent.value && waMediaType(files[0].content_type) === 'audio') {
    parts.push({ content: caption, attachments: [] })
    caption = ''
  }
  files.forEach((file, i) => parts.push({ content: i === 0 ? caption : '', attachments: [file] }))
  return parts
}

function validateWhatsAppFiles(files) {
  if (conversationStore.current?.inbox_channel !== WHATSAPP_CHANNEL) return files
  const valid = []
  for (const file of files) {
    const maxMB = WA_MAX_FILE_MB[waMediaType(file.type)]
    if (file.size > maxMB * 1024 * 1024) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: t('conversation.whatsapp.fileSizeExceeded', {
          name: file.name,
          size: Math.floor(maxMB)
        })
      })
    } else {
      valid.push(file)
    }
  }
  return valid
}

const handleFileUpload = (event) => {
  const files = validateWhatsAppFiles(Array.from(event.target.files))
  if (files.length) _handleFileUpload({ target: { files } })
}

const uploadFiles = (files) => {
  const valid = validateWhatsAppFiles(Array.from(files))
  if (valid.length) _uploadFiles(valid)
}

const messageType = ref('reply')
const currentConversationUUID = computed(() => conversationStore.current?.uuid || null)
watch(
  currentConversationUUID,
  async (uuid, prevUuid) => {
    if (prevUuid) conversationStore.setSelectedDraftType(prevUuid, messageType.value)
    if (!uuid) {
      messageType.value = 'reply'
      return
    }
    messageType.value = conversationStore.resolveDraftType(uuid)
    // Prefetch may still be in flight on first load; re-resolve once drafts land.
    await conversationStore.draftsReady
    if (uuid !== currentConversationUUID.value) return
    messageType.value = conversationStore.resolveDraftType(uuid)
  },
  { immediate: true }
)

// Setup draft management composable, keyed per conversation and message type.
const {
  htmlContent,
  textContent,
  isLoading: isDraftLoading,
  clearDraft,
  loadedAttachments,
  loadedMacroActions,
  loadedMacroID
} = useDraftManager(currentConversationUUID, messageType, mediaFiles)

// Rest of existing state
const isEditorFullscreen = ref(false)
const isSending = ref(false)
const isGenerating = ref(false)
const to = ref('')
const cc = ref('')
const bcc = ref('')
const showBcc = ref(false)
const emailErrors = ref([])
const aiPromptStore = useAiPromptStore()
const aiPrompts = computed(() => aiPromptStore.prompts)
const replyBoxContentRef = ref(null)
const showContactEmailWarning = ref(false)
const showMissingTagsWarning = ref(false)
const deferredStatus = ref(null)
const mentions = ref([])

aiPromptStore.fetchPrompts()

const runAiGeneration = async (requestFn) => {
  if (isGenerating.value) return
  const uuid = currentConversationUUID.value
  if (!uuid) return
  isGenerating.value = true
  try {
    const resp = await requestFn(uuid)
    if (uuid !== currentConversationUUID.value) return
    htmlContent.value = resp.data.data || ''
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isGenerating.value = false
  }
}

const handleAiPromptSelected = (key) =>
  runAiGeneration(() => api.aiCompletion({ prompt_key: key, content: htmlContent.value }))

const handleGenerateReply = () =>
  runAiGeneration((uuid) =>
    api.aiGenerateReply({ conversation_uuid: uuid, instruction: textContent.value })
  )

// Copilot's "Insert into reply" replaces the draft with its answer (already HTML from the panel),
// forcing reply mode so a private note in progress does not silently receive customer-facing text.
const handleCopilotInsertReply = (html) => {
  if (!html) return
  if (messageType.value === 'private_note') messageType.value = 'reply'
  htmlContent.value = html
}

onMounted(() => {
  emitter.on(EMITTER_EVENTS.COPILOT_INSERT_REPLY, handleCopilotInsertReply)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.COPILOT_INSERT_REPLY, handleCopilotInsertReply)
})

/**
 * Returns true if the editor has text content.
 */
const hasTextContent = computed(() => {
  return textContent.value.trim().length > 0
})

const processSend = async (skipContactEmailCheck = false, skipMissingTagsCheck = false, statusToSet = null) => {
  let hasMessageSendingErrored = false
  isEditorFullscreen.value = false

  const html = htmlContent.value
  if (hasPendingInlineUpload(html)) return
  const hasContent = hasTextContent.value || hasInlineImage(html) || mediaFiles.value.length > 0
  const convUUID = conversationStore.current.uuid
  const isPrivate = messageType.value === 'private_note'

  const currentInbox = inboxStore.inboxes.find(
    (i) => i.id === conversationStore.current.inbox_id
  )
  if (
    !isPrivate &&
    !skipMissingTagsCheck &&
    currentInbox?.prompt_tags_on_reply &&
    !(conversationStore.current.tags?.length > 0)
  ) {
    deferredStatus.value = statusToSet
    showMissingTagsWarning.value = true
    return
  }

  if (!isPrivate && conversationStore.current.inbox_channel === WHATSAPP_CHANNEL) {
    if (!isWhatsAppWindowOpen(conversationStore.current)) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: t('conversation.whatsapp.windowClosed.description')
      })
      return
    }
  }

  if (!isPrivate && conversationStore.current.inbox_channel === 'email') {
    // Require at least one recipient in `to`.
    if (!to.value.trim()) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: t('replyBox.toRequired')
      })
      return
    }

    // Warn if the contact's email is not in any recipient field.
    if (!skipContactEmailCheck) {
      const contactEmail = conversationStore.current.contact?.email?.toLowerCase()
      if (contactEmail) {
        const allRecipients = [to.value, cc.value, bcc.value].join(',').toLowerCase()
        if (
          !allRecipients
            .split(',')
            .map((e) => e.trim())
            .includes(contactEmail)
        ) {
          deferredStatus.value = statusToSet
          showContactEmailWarning.value = true
          return
        }
      }
    }
  }
  let tempUUIDs = []

  // Add pending message to cache for instant display.
  if (hasContent) {
    const savedContent = htmlContent.value
    const author = {
      id: userStore.userID,
      first_name: userStore.firstName,
      last_name: userStore.lastName,
      avatar_url: userStore.avatar,
      type: 'agent'
    }
    const parsedTo =
      !isPrivate && to.value
        ? to.value
            .split(',')
            .map((e) => e.trim())
            .filter(Boolean)
        : []
    const parsedCC =
      !isPrivate && cc.value
        ? cc.value
            .split(',')
            .map((e) => e.trim())
            .filter(Boolean)
        : []
    const parsedBCC =
      !isPrivate && bcc.value
        ? bcc.value
            .split(',')
            .map((e) => e.trim())
            .filter(Boolean)
        : []
    const meta = {}
    if (parsedTo.length) meta.to = parsedTo
    if (parsedCC.length) meta.cc = parsedCC
    if (parsedBCC.length) meta.bcc = parsedBCC

    const isWhatsAppReply =
      !isPrivate && conversationStore.current.inbox_channel === WHATSAPP_CHANNEL
    const parts =
      isWhatsAppReply && mediaFiles.value.length
        ? buildWhatsAppReplyParts(savedContent, mediaFiles.value)
        : [{ content: savedContent, attachments: mediaFiles.value }]

    tempUUIDs = parts.map((part, i) =>
      conversationStore.addPendingMessage(
        convUUID,
        part.content,
        isPrivate,
        author,
        part.attachments,
        i === 0 ? textContent.value : '',
        meta
      )
    )

    // Clear editor immediately.
    htmlContent.value = ''

    isSending.value = true
    for (const [i, part] of parts.entries()) {
      try {
        const response = await api.sendMessage(convUUID, {
          sender_type: UserTypeAgent,
          private: isPrivate,
          message: part.content,
          attachments: part.attachments.map((file) => file.id),
          mentions: isPrivate ? mentions.value : [],
          cc: parsedCC,
          bcc: parsedBCC,
          to: parsedTo,
          echo_id: isPrivate ? '' : tempUUIDs[i]
        })

        if (isPrivate && response?.data?.data) {
          conversationStore.replacePendingMessage(convUUID, tempUUIDs[i], response.data.data)
        }
      } catch (error) {
        hasMessageSendingErrored = true
        // Drop the bubbles for everything still unsent; parts already accepted stay in the timeline.
        tempUUIDs.slice(i).forEach((uuid) => conversationStore.removePendingMessage(convUUID, uuid))
        // Already-accepted attachments must not be resent on retry.
        const sentIDs = new Set(parts.slice(0, i).flatMap((p) => p.attachments.map((f) => f.id)))
        if (sentIDs.size) setMediaFiles(mediaFiles.value.filter((f) => !sentIDs.has(f.id)))
        if (i === 0) htmlContent.value = savedContent
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
        break
      }
    }

    if (!hasMessageSendingErrored) {
      notificationStore.markAssignmentAsReadForConversation(convUUID)
    }
  }

  // Apply macro actions if any.
  if (!hasMessageSendingErrored) {
    const macroID = conversationStore.getMacro(MACRO_CONTEXT.REPLY)?.id
    const macroActions = conversationStore.getMacro(MACRO_CONTEXT.REPLY)?.actions || []
    if (macroID > 0) {
      try {
        await api.applyMacro(convUUID, macroID, macroActions)
      } catch (error) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      }
    }
  }

  // Clear state on success.
  if (!hasMessageSendingErrored) {
    clearDraft(convUUID, isPrivate ? 'private_note' : 'reply')
    conversationStore.resetMacro(MACRO_CONTEXT.REPLY)
    clearMediaFiles()
    emailErrors.value = []
    mentions.value = []
    if (statusToSet) conversationStore.updateStatus(statusToSet)
  }
  isSending.value = false
}

const processSendAndSetStatus = (status) => processSend(false, false, status)

/**
 * Watches for changes in the conversation's macro id and update message content.
 */
watch(
  () => conversationStore.getMacro('reply').id,
  (newId) => {
    // No macro set.
    if (!newId) return

    // If macro has message content, set it in the editor.
    if (conversationStore.getMacro('reply').message_content) {
      htmlContent.value = conversationStore.getMacro('reply').message_content
    }
  },
  { deep: true }
)

// Reset first so a loaded draft never inherits the previous conversation's macro (drafts store no message_content).
watch(
  [loadedMacroID, loadedMacroActions],
  ([id, actions]) => {
    conversationStore.resetMacro(MACRO_CONTEXT.REPLY)
    if (id > 0) conversationStore.setMacro({ id, actions: [...toRaw(actions)] }, MACRO_CONTEXT.REPLY)
    else if (actions.length) conversationStore.setMacroActions([...toRaw(actions)], MACRO_CONTEXT.REPLY)
  },
  { deep: true }
)

/**
 * Watch for loaded attachments from draft and restore them to mediaFiles.
 */
watch(
  loadedAttachments,
  (attachments) => {
    setMediaFiles([...attachments])
  },
  { deep: true }
)

// Initialize to, cc, and bcc fields with the current conversation's values.
watch(
  () => conversationStore.currentCC,
  (newVal) => {
    cc.value = newVal?.join(', ') || ''
  },
  { deep: true, immediate: true }
)

watch(
  () => conversationStore.currentTo,
  (newVal) => {
    to.value = newVal?.join(', ') || ''
  },
  { immediate: true }
)

watch(
  () => conversationStore.currentBCC,
  (newVal) => {
    const newBcc = newVal?.join(', ') || ''
    bcc.value = newBcc
    // Only show BCC field if it has content
    if (newBcc.length > 0) {
      showBcc.value = true
    }
  },
  { deep: true, immediate: true }
)

// Media files and macro state are restored per draft by the draft manager; resetting here would race ahead of the save and drop them.
watch(
  () => conversationStore.current?.uuid,
  () => {
    setTimeout(() => {
      replyBoxContentRef.value?.focus()
    }, 100)
  }
)
</script>

<style scoped>
/* While the AI drafts a reply, a point of light orbits the reply box: a bright
   comet head that fades to a transparent tail, with its glow travelling along. */
@property --ai-angle {
  syntax: '<angle>';
  initial-value: 0deg;
  inherits: false;
}

.ai-generating {
  box-shadow: 0 6px 22px -10px hsl(var(--primary) / 0.28);
}

.ai-generating::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  padding: 1.5px;
  background: conic-gradient(
    from var(--ai-angle),
    hsl(var(--primary)) 0deg,
    hsl(var(--primary) / 0) 90deg,
    hsl(var(--primary) / 0) 180deg,
    hsl(var(--primary)) 180deg,
    hsl(var(--primary) / 0) 270deg,
    hsl(var(--primary) / 0) 360deg
  );
  filter: drop-shadow(0 0 5px hsl(var(--primary) / 0.5));
  -webkit-mask:
    linear-gradient(#000 0 0) content-box,
    linear-gradient(#000 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  animation: ai-border-spin 2.4s linear infinite;
  pointer-events: none;
  z-index: 20;
}

@keyframes ai-border-spin {
  to {
    --ai-angle: 360deg;
  }
}

@media (prefers-reduced-motion: reduce) {
  /* Steady even glow so the active state stays legible without motion. */
  .ai-generating {
    box-shadow: 0 0 0 1.5px hsl(var(--primary) / 0.4);
  }
  .ai-generating::after {
    display: none;
  }
}
</style>
