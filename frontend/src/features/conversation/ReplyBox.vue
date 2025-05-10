<template>
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
            {{ $t('globals.buttons.save') }}
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
        :class="{ '!bg-[#FEF1E1]': messageType === 'private_note' }"
        @escapeKeyDown="isEditorFullscreen = false"
        :hide-close-button="true"
      >
        <ReplyBoxContent
          v-if="isEditorFullscreen"
          :isFullscreen="true"
          :aiPrompts="aiPrompts"
          :isSending="isSending"
          :uploadingFiles="uploadingFiles"
          :clearEditorContent="clearEditorContent"
          :contentToSet="contentToSet"
          v-model:htmlContent="htmlContent"
          v-model:textContent="textContent"
          v-model:selectedText="selectedText"
          v-model:isBold="isBold"
          v-model:isItalic="isItalic"
          v-model:cursorPosition="cursorPosition"
          v-model:to="to"
          v-model:cc="cc"
          v-model:bcc="bcc"
          v-model:emailErrors="emailErrors"
          v-model:messageType="messageType"
          v-model:showBcc="showBcc"
          @toggleFullscreen="isEditorFullscreen = true"
          @send="processSend"
          @fileUpload="handleFileUpload"
          @inlineImageUpload="handleInlineImageUpload"
          @fileDelete="handleOnFileDelete"
          @aiPromptSelected="handleAiPromptSelected"
          class="h-full flex-grow"
        />
      </DialogContent>
    </Dialog>

    <!-- Main Editor non-fullscreen -->
    <div
      class="bg-card text-card-foreground box m-2 px-2 pt-2 flex flex-col"
      :class="{ '!bg-[#FEF1E1]': messageType === 'private_note' }"
      v-if="!isEditorFullscreen"
    >
      <ReplyBoxContent
        :isFullscreen="false"
        :aiPrompts="aiPrompts"
        :isSending="isSending"
        :uploadingFiles="uploadingFiles"
        :clearEditorContent="clearEditorContent"
        :contentToSet="contentToSet"
        v-model:htmlContent="htmlContent"
        v-model:textContent="textContent"
        v-model:selectedText="selectedText"
        v-model:isBold="isBold"
        v-model:isItalic="isItalic"
        v-model:cursorPosition="cursorPosition"
        v-model:to="to"
        v-model:cc="cc"
        v-model:bcc="bcc"
        v-model:emailErrors="emailErrors"
        v-model:messageType="messageType"
        v-model:showBcc="showBcc"
        @toggleFullscreen="isEditorFullscreen = true"
        @send="processSend"
        @fileUpload="handleFileUpload"
        @inlineImageUpload="handleInlineImageUpload"
        @fileDelete="handleOnFileDelete"
        @aiPromptSelected="handleAiPromptSelected"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick, watch, computed } from 'vue'
import { transformImageSrcToCID } from '@/utils/strings'
import { handleHTTPError } from '@/utils/http'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useUserStore } from '@/stores/user'
import api from '@/api'
import { useI18n } from 'vue-i18n'
import { useConversationStore } from '@/stores/conversation'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { useEmitter } from '@/composables/useEmitter'
import ReplyBoxContent from '@/features/conversation/ReplyBoxContent.vue'
import {
  Form,
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormMessage
} from '@/components/ui/form'
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
const openAIKeyPrompt = ref(false)
const isOpenAIKeyUpdating = ref(false)

// Shared state between the two editor components.
const clearEditorContent = ref(false)
const isEditorFullscreen = ref(false)
const isSending = ref(false)
const messageType = ref('reply')
const to = ref('')
const cc = ref('')
const bcc = ref('')
const showBcc = ref(false)
const emailErrors = ref([])
const aiPrompts = ref([])
const uploadingFiles = ref([])
const htmlContent = ref('')
const textContent = ref('')
const selectedText = ref('')
const isBold = ref(false)
const isItalic = ref(false)
const cursorPosition = ref(0)
const contentToSet = ref('')

onMounted(async () => {
  await fetchAiPrompts()
})

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
    contentToSet.value = JSON.stringify({
      content: resp.data.data.replace(/\n/g, '<br>'),
      timestamp: Date.now()
    })
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
      description: t('globals.messages.savedSuccessfully', {
        name: t('globals.terms.apiKey')
      })
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
 * Handles the file upload process when files are selected.
 * Uploads each file to the server and adds them to the conversation's mediaFiles.
 * @param {Event} event - The file input change event containing selected files
 */
const handleFileUpload = (event) => {
  const files = Array.from(event.target.files)
  uploadingFiles.value = files

  for (const file of files) {
    api
      .uploadMedia({
        files: file,
        inline: false,
        linked_model: 'messages'
      })
      .then((resp) => {
        conversationStore.conversation.mediaFiles.push(resp.data.data)
        uploadingFiles.value = uploadingFiles.value.filter((f) => f.name !== file.name)
      })
      .catch((error) => {
        uploadingFiles.value = uploadingFiles.value.filter((f) => f.name !== file.name)
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      })
  }
}

// Inline image upload is not supported yet.
const handleInlineImageUpload = (event) => {
  for (const file of event.target.files) {
    api
      .uploadMedia({
        files: file,
        inline: true,
        linked_model: 'messages'
      })
      .then((resp) => {
        const imageData = {
          src: resp.data.data.url,
          alt: resp.data.data.filename,
          title: resp.data.data.uuid
        }
        conversationStore.conversation.mediaFiles.push(resp.data.data)
        return imageData
      })
      .catch((error) => {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      })
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
const processSend = async () => {
  let hasAPIErrored = false
  isEditorFullscreen.value = false
  try {
    isSending.value = true

    // Send message if there is text content in the editor.
    if (hasTextContent.value > 0) {
      // Replace inline image url with cid.
      const message = transformImageSrcToCID(htmlContent.value)

      // Check which images are still in editor before sending.
      const parser = new DOMParser()
      const doc = parser.parseFromString(htmlContent.value, 'text/html')
      const inlineImageUUIDs = Array.from(doc.querySelectorAll('img.inline-image'))
        .map((img) => img.getAttribute('title'))
        .filter(Boolean)

      // TODO: Inline images are not supported yet, this is some old boilerplate code.
      conversationStore.conversation.mediaFiles = conversationStore.conversation.mediaFiles.filter(
        (file) =>
          // Keep if:
          // 1. Not an inline image OR
          // 2. Is an inline image that exists in editor
          file.disposition !== 'inline' || inlineImageUUIDs.includes(file.uuid)
      )

      await api.sendMessage(conversationStore.current.uuid, {
        private: messageType.value === 'private_note',
        message: message,
        attachments: conversationStore.conversation.mediaFiles.map((file) => file.id),
        // Convert email addresses to array and remove empty strings.
        cc: cc.value
          .split(',')
          .map((email) => email.trim())
          .filter((email) => email),
        bcc: bcc.value
          ? bcc.value
              .split(',')
              .map((email) => email.trim())
              .filter((email) => email)
          : [],
        to: to.value
          ? to.value
              .split(',')
              .map((email) => email.trim())
              .filter((email) => email)
          : []
      })
    }

    // Apply macro actions if any.
    // For macro errors just show toast and clear the editor.
    if (conversationStore.conversation?.macro?.actions?.length > 0) {
      try {
        await api.applyMacro(
          conversationStore.current.uuid,
          conversationStore.conversation.macro.id,
          conversationStore.conversation.macro.actions
        )
      } catch (error) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      }
    }
  } catch (error) {
    hasAPIErrored = true
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    // If API has NOT errored clear state.
    if (hasAPIErrored === false) {
      // Clear editor.
      clearEditorContent.value = true

      // Clear macro.
      conversationStore.resetMacro()

      // Clear media files.
      conversationStore.resetMediaFiles()

      // Clear any email errors.
      emailErrors.value = []

      nextTick(() => {
        clearEditorContent.value = false
      })
    }
    isSending.value = false
  }
  // Update assignee last seen timestamp.
  api.updateAssigneeLastSeen(conversationStore.current.uuid)
}

/**
 * Handles the file delete event.
 * Removes the file from the conversation's mediaFiles.
 * @param {String} uuid - The UUID of the file to delete
 */
const handleOnFileDelete = (uuid) => {
  conversationStore.conversation.mediaFiles = conversationStore.conversation.mediaFiles.filter(
    (item) => item.uuid !== uuid
  )
}

/**
 * Watches for changes in the conversation's macro id and update message content.
 */
watch(
  () => conversationStore.conversation.macro.id,
  () => {
    // Setting timestamp, so the same macro can be set again.
    contentToSet.value = JSON.stringify({
      content: conversationStore.conversation.macro.message_content,
      timestamp: Date.now()
    })
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
</script>
