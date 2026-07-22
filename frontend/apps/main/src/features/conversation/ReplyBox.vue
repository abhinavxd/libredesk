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
import { convertTextToHtml } from '@shared-ui/utils/string'
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
  handleFileUpload,
  handleFileDelete,
  uploadFiles,
  mediaFiles,
  clearMediaFiles,
  setMediaFiles
} = useFileUpload({
  linkedModel: 'messages'
})

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
  loadedMacroActions
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
    htmlContent.value = convertTextToHtml(resp.data.data || '')
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
  runAiGeneration(() => api.aiCompletion({ prompt_key: key, content: textContent.value }))

const handleGenerateReply = () =>
  runAiGeneration((uuid) => api.aiGenerateReply({ conversation_uuid: uuid, instruction: textContent.value }))

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
  let tempUUID = null

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

    tempUUID = conversationStore.addPendingMessage(
      convUUID,
      savedContent,
      isPrivate,
      author,
      mediaFiles.value,
      textContent.value,
      meta
    )

    // Clear editor immediately.
    htmlContent.value = ''

    try {
      isSending.value = true
      const response = await api.sendMessage(convUUID, {
        sender_type: UserTypeAgent,
        private: isPrivate,
        message: savedContent,
        attachments: mediaFiles.value.map((file) => file.id),
        mentions: isPrivate ? mentions.value : [],
        cc: parsedCC,
        bcc: parsedBCC,
        to: parsedTo,
        echo_id: isPrivate ? '' : tempUUID
      })

      // Private notes are sent immediately so replace immediately.
      if (isPrivate && response?.data?.data) {
        conversationStore.replacePendingMessage(convUUID, tempUUID, response.data.data)
      }

      notificationStore.markAssignmentAsReadForConversation(convUUID)
    } catch (error) {
      hasMessageSendingErrored = true
      // Remove pending message and restore editor content.
      conversationStore.removePendingMessage(convUUID, tempUUID)
      htmlContent.value = savedContent
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  // Apply macro actions if any.
  if (!hasMessageSendingErrored) {
    const macroID = conversationStore.getMacro(MACRO_CONTEXT.REPLY)?.id
    const macroActions = conversationStore.getMacro(MACRO_CONTEXT.REPLY)?.actions || []
    if (macroID > 0 && macroActions.length > 0) {
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

// Reset first so a loaded draft never inherits the previous conversation's macro id/message_content (drafts store only actions).
watch(
  loadedMacroActions,
  (actions) => {
    conversationStore.resetMacro(MACRO_CONTEXT.REPLY)
    if (actions.length) conversationStore.setMacroActions([...toRaw(actions)], MACRO_CONTEXT.REPLY)
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
