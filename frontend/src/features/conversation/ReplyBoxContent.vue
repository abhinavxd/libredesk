<template>
  <!-- Set fixed width only when not in fullscreen. -->
  <div class="flex flex-col h-full" :class="{ 'max-h-[600px]': !isFullscreen }">
    <!-- Message type toggle -->
    <div
      class="flex justify-between items-center"
      :class="{ 'mb-4': !isFullscreen, 'border-b border-border pb-4': isFullscreen }"
    >
      <Tabs v-model="messageType" class="rounded-lg border">
        <TabsList class="bg-muted p-1 rounded-lg">
          <TabsTrigger
            value="reply"
            class="px-3 py-1 rounded-lg transition-colors duration-200"
            :class="{ 'bg-background text-foreground': messageType === 'reply' }"
          >
            {{ $t('replyBox.reply') }}
          </TabsTrigger>
          <TabsTrigger
            value="private_note"
            class="px-3 py-1 rounded-lg transition-colors duration-200"
            :class="{ 'bg-background text-foreground': messageType === 'private_note' }"
          >
            {{ $t('replyBox.privateNote') }}
          </TabsTrigger>
        </TabsList>
      </Tabs>
      <span
        class="text-muted-foreground hover:text-foreground transition-colors duration-200 cursor-pointer"
        variant="ghost"
        @click="toggleFullscreen"
      >
        <component
          :is="isFullscreen ? Minimize2 : Maximize2"
          :size="isFullscreen ? '18' : '15'"
          :class="{ 'mr-2': !isFullscreen, 'mr-1 mb-2': isFullscreen }"
        />
      </span>
    </div>

    <!-- To, CC, and BCC fields -->
    <div
      :class="['space-y-3', isFullscreen ? 'p-4 border-b border-border' : 'mb-4']"
      v-if="messageType === 'reply'"
    >
      <div class="flex items-center space-x-2">
        <label class="w-12 text-sm font-medium text-muted-foreground">To:</label>
        <Input
          type="text"
          :placeholder="t('replyBox.emailAddresess')"
          v-model="to"
          class="flex-grow px-3 py-2 text-sm border rounded-md focus:ring-2 focus:ring-ring"
          @blur="validateEmails('to')"
        />
      </div>
      <div class="flex items-center space-x-2">
        <label class="w-12 text-sm font-medium text-muted-foreground">CC:</label>
        <Input
          type="text"
          :placeholder="t('replyBox.emailAddresess')"
          v-model="cc"
          class="flex-grow px-3 py-2 text-sm border rounded-md focus:ring-2 focus:ring-ring"
          @blur="validateEmails('cc')"
        />
        <Button
          size="sm"
          @click="toggleBcc"
          class="text-sm bg-secondary text-secondary-foreground hover:bg-secondary/80"
        >
          {{ showBcc ? 'Remove BCC' : 'BCC' }}
        </Button>
      </div>
      <div v-if="showBcc" class="flex items-center space-x-2">
        <label class="w-12 text-sm font-medium text-muted-foreground">BCC:</label>
        <Input
          type="text"
          :placeholder="t('replyBox.emailAddresess')"
          v-model="bcc"
          class="flex-grow px-3 py-2 text-sm border rounded-md focus:ring-2 focus:ring-ring"
          @blur="validateEmails('bcc')"
        />
      </div>
    </div>

    <!-- email errors -->
    <div
      v-if="emailErrors.length > 0"
      class="mb-4 px-2 py-1 bg-destructive/10 border border-destructive text-destructive rounded"
    >
      <p v-for="error in emailErrors" :key="error" class="text-sm">{{ error }}</p>
    </div>

    <!-- Main tiptap editor -->
    <div class="flex-grow flex flex-col overflow-hidden">
      <Editor
        v-model:selectedText="selectedText"
        v-model:isBold="isBold"
        v-model:isItalic="isItalic"
        v-model:htmlContent="htmlContent"
        v-model:textContent="textContent"
        v-model:cursorPosition="cursorPosition"
        :placeholder="editorPlaceholder"
        :aiPrompts="aiPrompts"
        @aiPromptSelected="handleAiPromptSelected"
        :contentToSet="contentToSet"
        @send="handleSend"
        :clearContent="clearEditorContent"
        :setInlineImage="setInlineImage"
        :insertContent="insertContent"
      />
    </div>

    <!-- Macro preview -->
    <MacroActionsPreview
      v-if="conversationStore.conversation?.macro?.actions?.length > 0"
      :actions="conversationStore.conversation.macro.actions"
      :onRemove="conversationStore.removeMacroAction"
      class="mt-2"
    />

    <!-- Attachments preview -->
    <AttachmentsPreview
      :attachments="attachments"
      :uploadingFiles="uploadingFiles"
      :onDelete="handleOnFileDelete"
      v-if="attachments.length > 0 || uploadingFiles.length > 0"
      class="mt-2"
    />

    <!-- Editor menu bar with send button -->
    <ReplyBoxMenuBar
      class="mt-1 shrink-0"
      :isFullscreen="isFullscreen"
      :handleFileUpload="handleFileUpload"
      :handleInlineImageUpload="handleInlineImageUpload"
      :isBold="isBold"
      :isItalic="isItalic"
      :isSending="isSending"
      @toggleBold="toggleBold"
      @toggleItalic="toggleItalic"
      :enableSend="enableSend"
      :handleSend="handleSend"
      @emojiSelect="handleEmojiSelect"
    />
  </div>
</template>

<script setup>
import { ref, computed, nextTick } from 'vue'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { Maximize2, Minimize2 } from 'lucide-vue-next'
import Editor from './ConversationTextEditor.vue'
import { useConversationStore } from '@/stores/conversation'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useEmitter } from '@/composables/useEmitter'
import AttachmentsPreview from '@/features/conversation/message/attachment/AttachmentsPreview.vue'
import MacroActionsPreview from '@/features/conversation/MacroActionsPreview.vue'
import ReplyBoxMenuBar from '@/features/conversation/ReplyBoxMenuBar.vue'
import { useI18n } from 'vue-i18n'
import { validateEmail } from '@/utils/strings'

// Define models for two-way binding
const messageType = defineModel('messageType', { default: 'reply' })
const to = defineModel('to', { default: '' })
const cc = defineModel('cc', { default: '' })
const bcc = defineModel('bcc', { default: '' })
const showBcc = defineModel('showBcc', { default: false })
const emailErrors = defineModel('emailErrors', { default: () => [] })
const htmlContent = defineModel('htmlContent', { default: '' })
const textContent = defineModel('textContent', { default: '' })
const selectedText = defineModel('selectedText', { default: '' })
const isBold = defineModel('isBold', { default: false })
const isItalic = defineModel('isItalic', { default: false })
const cursorPosition = defineModel('cursorPosition', { default: 0 })

const props = defineProps({
  isFullscreen: {
    type: Boolean,
    default: false
  },
  aiPrompts: {
    type: Array,
    required: true
  },
  isSending: {
    type: Boolean,
    required: true
  },
  uploadingFiles: {
    type: Array,
    required: true
  },
  clearEditorContent: {
    type: Boolean,
    required: true
  },
  contentToSet: {
    type: String,
    default: null
  }
})

const emit = defineEmits([
  'toggleFullscreen',
  'send',
  'fileUpload',
  'inlineImageUpload',
  'fileDelete',
  'aiPromptSelected'
])

const conversationStore = useConversationStore()
const emitter = useEmitter()
const { t } = useI18n()

const insertContent = ref(null)
const setInlineImage = ref(null)
const editorPlaceholder = t('replyBox.editor.placeholder')

const toggleBcc = async () => {
  showBcc.value = !showBcc.value
  await nextTick()
  // If hiding BCC field, clear the content
  if (!showBcc.value) {
    bcc.value = ''
  }
}

const toggleFullscreen = () => {
  emit('toggleFullscreen')
}

const toggleBold = () => {
  isBold.value = !isBold.value
}

const toggleItalic = () => {
  isItalic.value = !isItalic.value
}

const attachments = computed(() => {
  return conversationStore.conversation.mediaFiles.filter(
    (upload) => upload.disposition === 'attachment'
  )
})

const enableSend = computed(() => {
  return (
    (textContent.value.trim().length > 0 ||
      conversationStore.conversation?.macro?.actions?.length > 0) &&
    emailErrors.value.length === 0 &&
    !props.uploadingFiles.length
  )
})

/**
 * Validate email addresses in the To, CC, and BCC fields
 * @param {string} field - 'to', 'cc', or 'bcc'
 */
const validateEmails = (field) => {
  console.log('Validating emails for field:', field)
  const emails = field === 'to' ? to.value : field === 'cc' ? cc.value : bcc.value
  const emailList = emails
    .split(',')
    .map((e) => e.trim())
    .filter((e) => e !== '')

  const invalidEmails = emailList.filter((email) => !validateEmail(email))

  console.log('Invalid emails:', invalidEmails)

  // Clear existing errors
  emailErrors.value = []

  // Add new error if there are invalid emails
  if (invalidEmails.length > 0) {
    emailErrors.value = [
      ...emailErrors.value,
      `${t('replyBox.invalidEmailsIn')} '${field}': ${invalidEmails.join(', ')}`
    ]
  }
}

/**
 * Send the reply or private note
 */
const handleSend = async () => {
  validateEmails('to')
  validateEmails('cc')
  validateEmails('bcc')
  if (emailErrors.value.length > 0) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('replyBox.correctEmailErrors')
    })
    return
  }
  emit('send')
}

const handleFileUpload = (event) => {
  emit('fileUpload', event)
}

const handleInlineImageUpload = (event) => {
  emit('inlineImageUpload', event)
}

const handleOnFileDelete = (uuid) => {
  emit('fileDelete', uuid)
}

const handleEmojiSelect = (emoji) => {
  insertContent.value = undefined
  // Force reactivity so the user can select the same emoji multiple times
  nextTick(() => (insertContent.value = emoji))
}

const handleAiPromptSelected = (key) => {
  emit('aiPromptSelected', key)
}
</script>
