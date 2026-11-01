<template>
  <!-- Set fixed width only when not in fullscreen. -->
  <div class="flex flex-col h-full" :class="{ 'max-h-[600px]': !isFullscreen }">
    <!-- Message type toggle -->
    <div
      class="flex items-center justify-between"
      :class="{ 'mb-4': !isFullscreen, 'border-b border-border pb-4': isFullscreen }"
    >
      <Tabs v-model="messageType" class="rounded-lg">
        <TabsList>
          <TabsTrigger
            v-if="canSendReply"
            value="reply"
            class="max-md:py-2.5"
          >
            {{ i18n.ngettext('Reply', 'Replies', 1) }}
          </TabsTrigger>
          <TabsTrigger
            v-if="canSendPrivateNote"
            value="private_note"
            class="max-md:py-2.5"
          >
            {{ i18n.ngettext('Private note', 'Private notes', 1) }}
          </TabsTrigger>
        </TabsList>
      </Tabs>
      <div class="flex items-center">
        <Button
          v-if="!isFullscreen"
          type="button"
          class="text-muted-foreground"
          variant="ghost"
          :aria-label="i18n._('Collapse')"
          @click="emit('minimize')"
        >
          <Minus class="translate-y-1" />
        </Button>
        <Button
          class="text-muted-foreground max-md:h-11 max-md:w-11 max-md:p-0"
          variant="ghost"
          @click="toggleFullscreen"
        >
          <component :is="isFullscreen ? Minimize2 : Maximize2" />
        </Button>
      </div>
    </div>

    <!-- To, CC, and BCC fields -->
    <div v-if="conversationStore.current.inbox_channel === 'email'">
      <div
        :class="['grid grid-cols-[max-content_minmax(0,1fr)] gap-x-2 gap-y-3', isFullscreen ? 'border-b border-border p-4' : 'mb-3']"
        v-if="messageType === 'reply'"
      >
        <div v-if="conversationStore.currentFromOptions.length > 1" :class="RECIPIENT_ROW_CLASS">
          <label :for="`${recipientId}-from`" :class="RECIPIENT_LABEL_CLASS">{{ i18n._('From') }}</label>
          <div class="flex min-w-0 items-center gap-2">
            <Select v-model="sendFrom">
              <SelectTrigger :id="`${recipientId}-from`" :class="RECIPIENT_INPUT_CLASS"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="address in conversationStore.currentFromOptions" :key="address" :value="address">
                  {{ address }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <div :class="RECIPIENT_ROW_CLASS">
          <label :for="`${recipientId}-to`" :class="RECIPIENT_LABEL_CLASS">{{ i18n._('To') }}</label>
          <div class="flex min-w-0 items-center gap-2">
            <Input
              :id="`${recipientId}-to`"
              type="text"
              :placeholder="i18n._('Email addresses separated by comma')"
              v-model="to"
              :class="RECIPIENT_INPUT_CLASS"
              @blur="validateEmails"
            />
            <Button
              v-if="!showCc"
              type="button"
              size="sm"
              variant="ghost"
              :class="RECIPIENT_TOGGLE_CLASS"
              @click="showRecipientField('cc')"
            >
              {{ i18n._('CC') }}
            </Button>
            <Button
              v-if="!showBcc"
              type="button"
              size="sm"
              variant="ghost"
              :class="RECIPIENT_TOGGLE_CLASS"
              @click="showRecipientField('bcc')"
            >
              {{ i18n._('BCC') }}
            </Button>
          </div>
        </div>
        <div v-if="showCc" :class="RECIPIENT_ROW_CLASS">
          <label :for="`${recipientId}-cc`" :class="RECIPIENT_LABEL_CLASS">{{ i18n._('CC') }}</label>
          <div class="flex min-w-0 items-center gap-2">
            <Input
              ref="ccInputRef"
              :id="`${recipientId}-cc`"
              type="text"
              :placeholder="i18n._('Email addresses separated by comma')"
              v-model="cc"
              :class="RECIPIENT_INPUT_CLASS"
              @blur="validateEmails"
            />
            <Button
              type="button"
              size="sm"
              variant="ghost"
              :class="RECIPIENT_TOGGLE_CLASS"
              :aria-label="i18n._('Remove CC')"
              @click="hideRecipientField('cc')"
            >
              <X class="w-4 h-4" />
            </Button>
          </div>
        </div>
        <div v-if="showBcc" :class="RECIPIENT_ROW_CLASS">
          <label :for="`${recipientId}-bcc`" :class="RECIPIENT_LABEL_CLASS">{{ i18n._('BCC') }}</label>
          <div class="flex min-w-0 items-center gap-2">
            <Input
              ref="bccInputRef"
              :id="`${recipientId}-bcc`"
              type="text"
              :placeholder="i18n._('Email addresses separated by comma')"
              v-model="bcc"
              :class="RECIPIENT_INPUT_CLASS"
              @blur="validateEmails"
            />
            <Button
              type="button"
              size="sm"
              variant="ghost"
              :class="RECIPIENT_TOGGLE_CLASS"
              :aria-label="i18n._('Remove BCC')"
              @click="hideRecipientField('bcc')"
            >
              <X class="w-4 h-4" />
            </Button>
          </div>
        </div>
      </div>

      <!-- email errors -->
      <div
        v-if="emailErrors.length > 0"
        class="mb-3 rounded-md border border-destructive bg-destructive/10 px-3 py-2 text-destructive"
      >
        <p v-for="error in emailErrors" :key="error" class="text-sm">{{ error }}</p>
      </div>
    </div>

    <!-- Main tiptap editor -->
    <div class="flex-grow flex flex-col overflow-hidden">
      <Editor
        ref="editorRef"
        v-model:htmlContent="htmlContent"
        v-model:textContent="textContent"
        :message-type="messageType"
        :placeholder="isCramped ? i18n._('Type your message...') : i18n._('Shift + Enter to add a new line. Ctrl + Enter to send. Ctrl + K to open command bar.')"
        :insertContent="insertContent"
        :autoFocus="false"
        :disabled="isDraftLoading"
        :enableMentions="messageType === 'private_note'"
        :enableConversationReferences="messageType === 'private_note'"
        :enableInlineImages="conversationStore.current.inbox_channel === 'email'"
        :getSuggestions="getSuggestions"
        :getConversationSuggestions="getConversationSuggestions"
        @aiGenerationChange="emit('aiGenerationChange', $event)"
        @send="handleSend"
        @mentionsChanged="handleMentionsChanged"
        @filesDropped="handleFilesDropped"
        @toggleMessageType="toggleMessageType"
      />
    </div>

    <!-- Macro preview -->
    <MacroActionsPreview
      v-if="conversationStore.getMacro(MACRO_CONTEXT.REPLY)?.actions?.length > 0"
      :actions="conversationStore.getMacro(MACRO_CONTEXT.REPLY).actions"
      :onRemove="(action) => conversationStore.removeMacroAction(action, MACRO_CONTEXT.REPLY)"
      class="mt-2"
    />

    <!-- Attachments preview -->
    <ReplyBoxAttachmentPreview
      :attachments="uploadedFiles"
      :uploadingFiles="uploadingFiles"
      :onDelete="handleOnFileDelete"
      v-if="uploadedFiles.length > 0 || uploadingFiles.length > 0"
      class="mt-2"
    />

    <!-- Editor menu bar with send button -->
    <ReplyBoxMenuBar
      class="mt-2"
      :isFullscreen="isFullscreen"
      :isWhatsApp="isWhatsAppReply"
      :showWhatsAppTemplate="isWhatsAppReply"
      :handleFileUpload="handleFileUpload"
      :isSending="isSending"
      :enableSend="enableSend"
      :handleSend="handleSend"
      :handleSendAndSetStatus="handleSendAndSetStatus"
      :isGenerating="isGenerating"
      :showGenerateReply="messageType !== 'private_note'"
      @emojiSelect="handleEmojiSelect"
      @generateReply="$emit('generateReply')"
    />
  </div>
</template>

<script setup>
const RECIPIENT_ROW_CLASS = 'col-span-2 grid grid-cols-subgrid items-center gap-x-2'
const RECIPIENT_LABEL_CLASS = 'whitespace-nowrap text-right text-sm text-muted-foreground'
const RECIPIENT_INPUT_CLASS =
  'min-w-0 flex-1 border-input bg-card px-2 py-2 text-sm shadow-none focus-visible:ring-1 focus-visible:ring-ring'
const RECIPIENT_TOGGLE_CLASS = 'shrink-0 px-2 text-muted-foreground'

import { ref, computed, nextTick, watch, useId } from 'vue'
import { useDebounceFn } from '@vueuse/core'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { MACRO_CONTEXT } from '@main/constants/conversation'
import { Maximize2, Minimize2, Minus, X } from 'lucide-vue-next'
import Editor from '@main/components/editor/ConversationEditor.vue'
import { hasInlineImage, hasPendingInlineUpload } from '@main/composables/useInlineImageUpload'
import { useConversationStore } from '@main/stores/conversation'
import { WHATSAPP_CHANNEL } from '@main/features/conversation/whatsappTemplate'
import { useIsComposerCramped } from '@main/composables/useIsComposerCramped'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import { Tabs, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@shared-ui/components/ui/select'
import { useEmitter } from '@main/composables/useEmitter'
import ReplyBoxAttachmentPreview from '@/features/conversation/message/attachment/ReplyBoxAttachmentPreview.vue'
import MacroActionsPreview from '@/features/conversation/MacroActionsPreview.vue'
import ReplyBoxMenuBar from '@/features/conversation/ReplyBoxMenuBar.vue'
import { useI18n } from '@shared-ui/i18n'
import { validateEmail } from '@shared-ui/utils/string'
import { useMacroStore } from '@main/stores/macro'
import api from '@main/api'
import {
  createLatestConversationSuggestionFetcher,
  getConversationSuggestions as fetchConversationSuggestions
} from '@main/components/editor/conversationReference'

const MENTION_LIMIT = 10
const MENTION_DEBOUNCE_MS = 250

const messageType = defineModel('messageType', { default: 'reply' })
const to = defineModel('to', { default: '' })
const cc = defineModel('cc', { default: '' })
const bcc = defineModel('bcc', { default: '' })
const sendFrom = defineModel('sendFrom', { default: '' })
const showCc = defineModel('showCc', { default: false })
const showBcc = defineModel('showBcc', { default: false })
const emailErrors = defineModel('emailErrors', { default: () => [] })
const htmlContent = defineModel('htmlContent', { default: '' })
const textContent = defineModel('textContent', { default: '' })
const mentions = defineModel('mentions', { default: () => [] })
const macroStore = useMacroStore()

// Get suggestions for the mention dropdown
const fetchSuggestions = async (query) => {
  // Mentions run their own query so typing here never disturbs the shared agent and team pickers.
  const [agentsResponse, teamsResponse] = await Promise.all([
    api.getUsersCompact({ q: query, page_size: MENTION_LIMIT, type: 'agent', enabled: true }),
    api.getTeamsCompact({ q: query, page_size: MENTION_LIMIT })
  ])

  const users = (agentsResponse?.data?.data || [])
    .map((u) => ({
      id: u.id,
      type: 'agent',
      label: `${u.first_name} ${u.last_name}`.trim(),
      avatar_url: u.avatar_url
    }))

  const teams = (teamsResponse?.data?.data || []).map((t) => ({
    id: t.id,
    type: 'team',
    label: t.name,
    emoji: t.emoji
  }))

  return [...users, ...teams].slice(0, MENTION_LIMIT)
}

const debouncedFetchSuggestions = useDebounceFn(fetchSuggestions, MENTION_DEBOUNCE_MS)

const getSuggestions = async (query) => {
  if (messageType.value !== 'private_note') return []
  return (await debouncedFetchSuggestions(query)) || []
}

const debouncedFetchConversationSuggestions = useDebounceFn(fetchConversationSuggestions, MENTION_DEBOUNCE_MS)
const fetchLatestConversationSuggestions = createLatestConversationSuggestionFetcher(
  debouncedFetchConversationSuggestions
)

const getConversationSuggestions = async (query) => {
  if (messageType.value !== 'private_note') return []
  const messageTypeAtRequest = messageType.value
  const suggestions = (await fetchLatestConversationSuggestions(query)) || []
  return messageType.value === messageTypeAtRequest ? suggestions : []
}

// Handle mentions changed from editor
const handleMentionsChanged = (newMentions) => {
  mentions.value = newMentions
}

const toggleMessageType = () => {
  if (props.isGenerating || !props.canSendReply || !props.canSendPrivateNote) return
  messageType.value = messageType.value === 'private_note' ? 'reply' : 'private_note'
}

const props = defineProps({
  isFullscreen: {
    type: Boolean,
    default: false
  },
  isSending: {
    type: Boolean,
    required: true
  },
  uploadingFiles: {
    type: Array,
    required: true
  },
  uploadedFiles: {
    type: Array,
    required: false,
    default: () => []
  },
  isDraftLoading: {
    type: Boolean,
    required: false,
    default: false
  },
  isGenerating: {
    type: Boolean,
    default: false
  },
  canSendReply: {
    type: Boolean,
    required: true
  },
  canSendPrivateNote: {
    type: Boolean,
    required: true
  }
})

const emit = defineEmits([
  'toggleFullscreen',
  'minimize',
  'send',
  'sendAndSetStatus',
  'fileUpload',
  'inlineImageUpload',
  'fileDelete',
  'filesDropped',
  'aiGenerationChange',
  'generateReply'
])

const conversationStore = useConversationStore()
const isWhatsAppReply = computed(
  () =>
    conversationStore.current?.inbox_channel === WHATSAPP_CHANNEL &&
    messageType.value !== 'private_note'
)
const isCramped = useIsComposerCramped()
const emitter = useEmitter()
const i18n = useI18n()
const recipientId = useId()
const insertContent = ref(null)
const editorRef = ref(null)
const ccInputRef = ref(null)
const bccInputRef = ref(null)

const showRecipientField = async (field) => {
  if (field === 'cc') showCc.value = true
  else showBcc.value = true
  await nextTick()
  const input = field === 'cc' ? ccInputRef.value : bccInputRef.value
  input?.$el?.focus()
}

// A hidden field must stay empty, its address would still be sent otherwise.
const hideRecipientField = async (field) => {
  if (field === 'cc') {
    showCc.value = false
    cc.value = ''
  } else {
    showBcc.value = false
    bcc.value = ''
  }
  await nextTick()
  validateEmails()
}

const toggleFullscreen = () => {
  emit('toggleFullscreen')
}

const enableSend = computed(() => {
  const html = htmlContent.value
  return (
    !hasPendingInlineUpload(html) &&
    (textContent.value.trim().length > 0 ||
      hasInlineImage(html) ||
      conversationStore.getMacro('reply')?.actions?.length > 0 ||
      props.uploadedFiles.length > 0) &&
    emailErrors.value.length === 0 &&
    !props.uploadingFiles.length &&
    !props.isDraftLoading
  )
})

/**
 * Validates email addresses in To, CC, and BCC fields.
 * Populates `emailErrors` with invalid emails grouped by field.
 */
const validateEmails = async () => {
  emailErrors.value = []
  await nextTick()

  const fields = ['to', 'cc', 'bcc']
  const values = { to: to.value, cc: cc.value, bcc: bcc.value }

  fields.forEach((field) => {
    const invalid = values[field]
      .split(',')
      .map((e) => e.trim())
      .filter((e) => e && !validateEmail(e))

    if (invalid.length)
      emailErrors.value.push(`${i18n._('Invalid email(s) in')} '${field}': ${invalid.join(', ')}`)
  })
}

const validateBeforeSend = async () => {
  await validateEmails()
  if (emailErrors.value.length > 0) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: i18n._('Please correct the email errors')
    })
    return false
  }
  return true
}

/**
 * Send the reply or private note
 */
const handleSend = async () => {
  if (!(await validateBeforeSend())) return
  emit('send')
}

/**
 * Send the reply or private note and set conversation status
 */
const handleSendAndSetStatus = async (status) => {
  if (!(await validateBeforeSend())) return
  emit('sendAndSetStatus', status)
}

const handleFileUpload = (event) => {
  emit('fileUpload', event)
}

const handleFilesDropped = (files) => {
  emit('filesDropped', files)
}

const handleOnFileDelete = (uuid) => {
  emit('fileDelete', uuid)
}

const handleEmojiSelect = (emoji) => {
  insertContent.value = undefined
  // Force reactivity so the user can select the same emoji multiple times
  nextTick(() => (insertContent.value = emoji))
}

// Watch and update macro view based on message type this filters our macros.
watch(
  messageType,
  (newType) => {
    if (newType === 'reply') {
      macroStore.setCurrentView('replying')
    } else if (newType === 'private_note') {
      macroStore.setCurrentView('adding_private_note')
    }
  },
  { immediate: true }
)

watch(
  [messageType, () => props.isDraftLoading, () => conversationStore.current?.uuid],
  async () => {
    await nextTick()
    if (props.isDraftLoading || document.activeElement?.matches('input, textarea')) return
    editorRef.value?.focus()
  },
  { immediate: true, flush: 'post' }
)

// Expose focus method for parent components
const focus = () => {
  editorRef.value?.focus()
}
defineExpose({ focus })
</script>
