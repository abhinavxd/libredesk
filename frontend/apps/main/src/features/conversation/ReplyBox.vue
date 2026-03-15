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
        <AlertDialogAction @click="processSend(true)">{{
          $t('replyBox.sendAnyway')
        }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>

  <Dialog :open="openAIKeyPrompt" @update:open="openAIKeyPrompt = false">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader class="space-y-2">
        <DialogTitle>{{ $t('ai.enterOpenAIAPIKey') }}</DialogTitle>
        <DialogDescription>
          {{
            $t('ai.apiKey.description', {
              provider: 'OpenAI'
            })
          }}
        </DialogDescription>
      </DialogHeader>
      <Form v-slot="{ handleSubmit }" as="" keep-values :validation-schema="formSchema">
        <form id="apiKeyForm" @submit="handleSubmit($event, updateProvider)">
          <FormField v-slot="{ componentField }" name="apiKey">
            <FormItem>
              <FormLabel>{{ $t('globals.terms.apiKey') }}</FormLabel>
              <FormControl>
                <Input type="text" placeholder="sk-am1RLw7XUWGX.." v-bind="componentField" />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>
        </form>
        <DialogFooter>
          <Button
            type="submit"
            form="apiKeyForm"
            :is-loading="isOpenAIKeyUpdating"
            :disabled="isOpenAIKeyUpdating"
          >
            {{ $t('globals.messages.save') }}
          </Button>
        </DialogFooter>
      </Form>
    </DialogContent>
  </Dialog>

  <div class="text-foreground bg-background">
    <!-- Fullscreen editor -->
    <Dialog :open="isEditorFullscreen" @update:open="isEditorFullscreen = false">
      <DialogContent
        class="max-w-[60%] max-h-[75%] h-[70%] bg-card text-card-foreground p-4 flex flex-col"
        :class="{ '!bg-private': messageType === 'private_note' }"
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
          @fileUpload="handleFileUpload"
          @fileDelete="handleFileDelete"
          @aiPromptSelected="handleAiPromptSelected"
          class="h-full flex-grow"
        />
      </DialogContent>
    </Dialog>

    <!-- Main Editor non-fullscreen -->
    <div
      class="bg-background text-card-foreground box m-2 px-2 pt-2 flex flex-col"
      :class="{ '!bg-private': messageType === 'private_note' }"
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
        @fileUpload="handleFileUpload"
        @fileDelete="handleFileDelete"
        @aiPromptSelected="handleAiPromptSelected"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, watch, computed, toRaw } from 'vue'
import { useStorage } from '@vueuse/core'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { MACRO_CONTEXT } from '@main/constants/conversation'
import { useUserStore } from '@main/stores/user'
import { useDraftManager } from '@main/composables/useDraftManager'
import api from '@main/api'
import { useI18n } from 'vue-i18n'
import { useConversationStore } from '@main/stores/conversation'
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog'
import { Input } from '@shared-ui/components/ui/input'
import { useEmitter } from '@main/composables/useEmitter'
import { useFileUpload } from '@main/composables/useFileUpload'
import ReplyBoxContent from '@/features/conversation/ReplyBoxContent.vue'
import { UserTypeAgent } from '@/constants/user'
import {
  Form,
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormMessage
} from '@shared-ui/components/ui/form'
import { toTypedSchema } from '@vee-validate/zod'
import * as z from 'zod'

const formSchema = toTypedSchema(
  z.object({
    apiKey: z.string().min(1, 'API key is required')
  })
)

const { t } = useI18n()
const conversationStore = useConversationStore()
const emitter = useEmitter()
const userStore = useUserStore()

// Setup file upload composable
const {
  uploadingFiles,
  handleFileUpload,
  handleFileDelete,
  mediaFiles,
  clearMediaFiles,
  setMediaFiles
} = useFileUpload({
  linkedModel: 'messages'
})

// Setup draft management composable
const currentDraftKey = computed(() => conversationStore.current?.uuid || null)
const {
  htmlContent,
  textContent,
  isLoading: isDraftLoading,
  clearDraft,
  loadedAttachments,
  loadedMacroActions
} = useDraftManager(currentDraftKey, mediaFiles)

// Rest of existing state
const openAIKeyPrompt = ref(false)
const isOpenAIKeyUpdating = ref(false)
const isEditorFullscreen = ref(false)
const isSending = ref(false)
const messageType = useStorage('replyBoxMessageType', 'reply')
const to = ref('')
const cc = ref('')
const bcc = ref('')
const showBcc = ref(false)
const emailErrors = ref([])
const aiPrompts = ref([])
const replyBoxContentRef = ref(null)
const showContactEmailWarning = ref(false)
const mentions = ref([])

/**
 * Fetches AI prompts from the server.
 */
const fetchAiPrompts = async () => {
  try {
    const resp = await api.getAiPrompts()
    aiPrompts.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

fetchAiPrompts()

/**
 * Handles the AI prompt selection event.
 * Sends the selected prompt key and the current text content to the server for completion.
 * Sets the response as the new content in the editor.
 * @param {String} key - The key of the selected AI prompt
 */
const handleAiPromptSelected = async (key) => {
  try {
    const resp = await api.aiCompletion({
      prompt_key: key,
      content: textContent.value
    })
    htmlContent.value = resp.data.data.replace(/\n/g, '<br>')
  } catch (error) {
    // Check if user needs to enter OpenAI API key and has permission to do so.
    if (error.response?.status === 400 && userStore.can('ai:manage')) {
      openAIKeyPrompt.value = true
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

/**
 * updateProvider updates the OpenAI API key.
 * @param {Object} values - The form values containing the API key
 */
const updateProvider = async (values) => {
  try {
    isOpenAIKeyUpdating.value = true
    await api.updateAIProvider({ api_key: values.apiKey, provider: 'openai' })
    openAIKeyPrompt.value = false
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isOpenAIKeyUpdating.value = false
  }
}

/**
 * Returns true if the editor has text content.
 */
const hasTextContent = computed(() => {
  return textContent.value.trim().length > 0
})

/**
 * Processes the send action.
 */
const processSend = async (skipContactEmailCheck = false) => {
  let hasMessageSendingErrored = false
  isEditorFullscreen.value = false

  const hasContent = hasTextContent.value > 0 || mediaFiles.value.length > 0
  const convUUID = conversationStore.current.uuid
  const isPrivate = messageType.value === 'private_note'

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
        to: parsedTo
      })

      // Replace pending message with the real one from API response.
      if (response?.data?.data) {
        conversationStore.replacePendingMessage(convUUID, tempUUID, response.data.data)
      }
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
    clearDraft(currentDraftKey.value)
    conversationStore.resetMacro(MACRO_CONTEXT.REPLY)
    clearMediaFiles()
    emailErrors.value = []
    mentions.value = []
  }
  isSending.value = false
}

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

/**
 * Watch loaded macro actions from draft and update conversation store.
 */
watch(
  loadedMacroActions,
  (actions) => {
    if (actions.length > 0) {
      conversationStore.setMacroActions([...toRaw(actions)], MACRO_CONTEXT.REPLY)
    }
  },
  { deep: true }
)

/**
 * Watch for loaded attachments from draft and restore them to mediaFiles.
 */
watch(
  loadedAttachments,
  (attachments) => {
    if (attachments.length > 0) {
      setMediaFiles([...attachments])
    }
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

// Clear media files and reset macro when conversation changes.
watch(
  () => conversationStore.current?.uuid,
  () => {
    clearMediaFiles()
    conversationStore.resetMacro(MACRO_CONTEXT.REPLY)
    // Focus editor on conversation change
    setTimeout(() => {
      replyBoxContentRef.value?.focus()
    }, 100)
  }
)
</script>
