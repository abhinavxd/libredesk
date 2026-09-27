<template>
  <form @submit="createConversation" novalidate class="flex flex-col flex-1 min-h-0 overflow-y-auto">
    <div class="shrink-0">
      <FormField v-slot="{ componentField }" name="inbox_id">
        <FormItem :class="ROW_CLASS">
          <div class="flex items-center gap-2">
            <FormLabel :class="ROW_LABEL_CLASS">{{ $t('globals.terms.from') }}</FormLabel>
            <Select v-bind="componentField">
              <FormControl>
                <SelectTrigger :class="ROW_INPUT_CLASS">
                  <SelectValue :placeholder="t('placeholders.selectInbox')" />
                </SelectTrigger>
              </FormControl>
              <SelectContent>
                <SelectGroup>
                  <SelectItem
                    v-for="option in inboxStore.emailOptions"
                    :key="option.value"
                    :value="option.value"
                  >
                    {{ option.label }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
          <FormMessage :class="ROW_MESSAGE_CLASS" />
        </FormItem>
      </FormField>

      <FormField name="contact_email">
        <FormItem :class="[ROW_CLASS, 'relative']">
          <div class="flex items-center gap-2">
            <FormLabel :class="ROW_LABEL_CLASS">{{ $t('globals.terms.to') }}</FormLabel>
            <FormControl>
              <Input
                ref="emailInputRef"
                type="email"
                :placeholder="t('conversation.searchContact')"
                v-model="emailQuery"
                :class="ROW_INPUT_CLASS"
                @input="handleSearchContacts"
                @keydown="handleSearchKeydown"
                @blur="clearSearchResults"
                autocomplete="off"
              />
            </FormControl>
            <Button
              v-if="!showCc"
              type="button"
              size="sm"
              variant="ghost"
              :class="RECIPIENT_TOGGLE_CLASS"
              @click="showRecipientField('cc')"
            >
              {{ $t('replyBox.cc') }}
            </Button>
            <Button
              v-if="!showBcc"
              type="button"
              size="sm"
              variant="ghost"
              :class="RECIPIENT_TOGGLE_CLASS"
              @click="showRecipientField('bcc')"
            >
              {{ $t('replyBox.bcc') }}
            </Button>
          </div>
          <FormMessage :class="ROW_MESSAGE_CLASS" />

          <ContactSearchResults
            :results="searchResults"
            :highlighted-index="highlightedIndex"
            @select="selectContact"
          >
            <template #default="{ contact }">
              <div>
                <p class="font-medium">
                  {{ contact.first_name }} {{ contact.last_name }}
                </p>
                <p class="text-xs text-muted-foreground">{{ contact.email }}</p>
                <div
                  v-if="contact.external_user_id"
                  class="flex items-center gap-1 text-xs text-muted-foreground"
                >
                  <IdCard :size="12" class="flex-shrink-0" />
                  <span class="truncate">{{ contact.external_user_id }}</span>
                </div>
              </div>
            </template>
          </ContactSearchResults>
        </FormItem>
      </FormField>

      <FormField
        v-for="field in visibleRecipientFields"
        :key="field.name"
        v-slot="{ componentField }"
        :name="field.name"
      >
        <FormItem :class="ROW_CLASS">
          <div class="flex items-center gap-2">
            <FormLabel :class="ROW_LABEL_CLASS">{{ field.label }}</FormLabel>
            <FormControl>
              <Input
                :ref="(el) => (recipientInputRefs[field.name] = el)"
                type="text"
                :placeholder="t('replyBox.emailAddresess')"
                v-bind="componentField"
                :class="ROW_INPUT_CLASS"
              />
            </FormControl>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              :class="RECIPIENT_TOGGLE_CLASS"
              :aria-label="field.removeLabel"
              @click="hideRecipientField(field.name)"
            >
              <X class="w-4 h-4" />
            </Button>
          </div>
          <FormMessage :class="ROW_MESSAGE_CLASS" />
        </FormItem>
      </FormField>

      <div :class="[ROW_CLASS, 'flex items-start gap-2']">
        <span :class="[ROW_LABEL_CLASS, 'h-9 flex items-center']">{{ $t('globals.terms.name', 1) }}</span>
        <FormField v-slot="{ componentField }" name="first_name">
          <FormItem class="flex-1 min-w-0 space-y-0">
            <FormControl>
              <Input
                type="text"
                :placeholder="t('globals.terms.firstName')"
                :aria-label="t('globals.terms.firstName')"
                v-bind="componentField"
                :disabled="!!selectedContact"
                :class="ROW_INPUT_CLASS"
              />
            </FormControl>
            <FormMessage class="pb-2" />
          </FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" name="last_name">
          <FormItem class="flex-1 min-w-0 space-y-0">
            <FormControl>
              <Input
                type="text"
                :placeholder="t('globals.terms.lastName')"
                :aria-label="t('globals.terms.lastName')"
                v-bind="componentField"
                :disabled="!!selectedContact"
                :class="ROW_INPUT_CLASS"
              />
            </FormControl>
          </FormItem>
        </FormField>
      </div>

      <FormField v-slot="{ componentField }" name="subject">
        <FormItem :class="ROW_CLASS">
          <div class="flex items-center gap-2">
            <FormLabel :class="ROW_LABEL_CLASS">{{ $t('globals.terms.subject') }}</FormLabel>
            <FormControl>
              <Input type="text" v-bind="componentField" :class="ROW_INPUT_CLASS" />
            </FormControl>
          </div>
          <FormMessage :class="ROW_MESSAGE_CLASS" />
        </FormItem>
      </FormField>

      <div :class="[ROW_CLASS, 'grid grid-cols-1 sm:grid-cols-2 sm:gap-4']">
        <FormField v-slot="{ componentField }" name="team_id">
          <FormItem class="flex items-center gap-2 min-w-0 space-y-0">
            <FormLabel :class="ROW_LABEL_CLASS">{{ $t('globals.terms.team', 1) }}</FormLabel>
            <FormControl>
              <SelectTeamCombobox
                v-bind="componentField"
                include-none
                :button-class="isUnset(componentField.modelValue) ? ROW_COMBOBOX_EMPTY_CLASS : ROW_COMBOBOX_CLASS"
              />
            </FormControl>
          </FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" name="agent_id">
          <FormItem class="flex items-center gap-2 min-w-0 space-y-0">
            <FormLabel :class="ROW_LABEL_CLASS">{{ $t('globals.terms.agent', 1) }}</FormLabel>
            <FormControl>
              <SelectAgentCombobox
                v-bind="componentField"
                include-none
                :button-class="isUnset(componentField.modelValue) ? ROW_COMBOBOX_EMPTY_CLASS : ROW_COMBOBOX_CLASS"
              />
            </FormControl>
          </FormItem>
        </FormField>
      </div>
    </div>

    <FormField v-slot="{ componentField }" name="content">
      <FormItem class="flex flex-col flex-1 min-h-40 space-y-0 px-3 pt-2">
        <FormControl class="flex-1 flex flex-col min-h-0">
          <div class="flex flex-col h-full">
            <Editor
              v-model:htmlContent="componentField.modelValue"
              @update:htmlContent="(value) => componentField.onChange(value)"
              :placeholder="isCramped ? t('globals.terms.typeMessage') : t('editor.hint.newLineCtrlK')"
              :insertContent="insertContent"
              :autoFocus="false"
              :enableInlineImages="true"
              class="w-full flex-1 overflow-y-auto min-h-0"
              @send="createConversation"
              @filesDropped="uploadFiles"
            />

            <MacroActionsPreview
              v-if="
                conversationStore.getMacro(MACRO_CONTEXT.NEW_CONVERSATION).actions?.length > 0
              "
              :actions="conversationStore.getMacro(MACRO_CONTEXT.NEW_CONVERSATION)?.actions || []"
              :onRemove="
                (action) =>
                  conversationStore.removeMacroAction(action, MACRO_CONTEXT.NEW_CONVERSATION)
              "
              class="mt-2 flex-shrink-0"
            />

            <ReplyBoxAttachmentPreview
              :attachments="mediaFiles"
              :uploadingFiles="uploadingFiles"
              :onDelete="handleFileDelete"
              v-if="mediaFiles.length > 0 || uploadingFiles.length > 0"
              class="mt-2 flex-shrink-0"
            />
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <div class="flex items-center justify-between gap-2 px-3 py-2 shrink-0">
      <ReplyBoxMenuBar
        :handleFileUpload="handleFileUpload"
        @emojiSelect="handleEmojiSelect"
        :showSendButton="false"
        :showGenerateReply="false"
      />
      <Button type="submit" :disabled="isDisabled" :isLoading="loading">
        {{ $t('globals.messages.send') }}
      </Button>
    </div>
  </form>
</template>

<script setup>
const RECIPIENT_TOGGLE_CLASS = 'shrink-0 px-2 text-muted-foreground'

import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form'
import { z } from 'zod'
import { ref, watch, onUnmounted, nextTick, onMounted, computed } from 'vue'
import ReplyBoxAttachmentPreview from '@/features/conversation/message/attachment/ReplyBoxAttachmentPreview.vue'
import { useConversationStore } from '@/stores/conversation'
import MacroActionsPreview from '@/features/conversation/MacroActionsPreview.vue'
import ReplyBoxMenuBar from '@/features/conversation/ReplyBoxMenuBar.vue'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { MACRO_CONTEXT } from '@main/constants/conversation'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useInboxStore } from '@main/stores/inbox'
import { useUserStore } from '@main/stores/user'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { useI18n } from 'vue-i18n'
import { useFileUpload } from '@/composables/useFileUpload'
import Editor from '@/components/editor/ConversationEditor.vue'
import SelectAgentCombobox from '@main/components/combobox/SelectAgentCombobox.vue'
import SelectTeamCombobox from '@main/components/combobox/SelectTeamCombobox.vue'
import { UserTypeAgent } from '@/constants/user'
import { IdCard, X } from 'lucide-vue-next'
import { validateEmail } from '@shared-ui/utils/string'
import {
  ROW_CLASS,
  ROW_LABEL_CLASS,
  ROW_INPUT_CLASS,
  ROW_COMBOBOX_CLASS,
  ROW_COMBOBOX_EMPTY_CLASS,
  ROW_MESSAGE_CLASS,
  isUnset
} from '@/features/conversation/composerRowClasses.js'
import api from '@/api'
import { useContactSearch } from '@/features/conversation/useContactSearch.js'
import ContactSearchResults from '@/features/conversation/ContactSearchResults.vue'
import { useNewConversationDraft } from '@/features/conversation/useNewConversationDraft.js'
import { hasPendingInlineUpload } from '@main/composables/useInlineImageUpload'
import { useIsComposerCramped } from '@main/composables/useIsComposerCramped'

const emit = defineEmits(['close'])
const props = defineProps({
  initialContact: { type: Object, default: null }
})

const inboxStore = useInboxStore()
const { t } = useI18n()
const isCramped = useIsComposerCramped()
const userStore = useUserStore()
const emitter = useEmitter()
const loading = ref(false)
const emailQuery = ref('')
const conversationStore = useConversationStore()
const insertContent = ref('')
const selectedContact = ref(null)
const emailInputRef = ref(null)
const draft = useNewConversationDraft('email')
const showCc = ref(false)
const showBcc = ref(false)
const recipientInputRefs = {}

const visibleRecipientFields = computed(() =>
  [
    showCc.value && { name: 'cc', label: t('replyBox.cc'), removeLabel: t('replyBox.removeCC') },
    showBcc.value && { name: 'bcc', label: t('replyBox.bcc'), removeLabel: t('replyBox.removeBCC') }
  ].filter(Boolean)
)

const showRecipientField = async (field) => {
  if (field === 'cc') showCc.value = true
  else showBcc.value = true
  await nextTick()
  recipientInputRefs[field]?.$el?.focus()
}

// A hidden field must stay empty, its address would still be sent otherwise.
const hideRecipientField = (field) => {
  if (field === 'cc') showCc.value = false
  else showBcc.value = false
  form.setFieldValue(field, '', false)
}

const splitEmails = (value) =>
  (value || '')
    .split(',')
    .map((e) => e.trim())
    .filter(Boolean)

const handleEmojiSelect = (emoji) => {
  insertContent.value = undefined
  // Force reactivity so the user can select the same emoji multiple times
  nextTick(() => (insertContent.value = emoji))
}

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

const isDisabled = computed(() => {
  if (loading.value || uploadingFiles.value.length > 0) return true
  if (hasPendingInlineUpload(form?.values?.content)) return true
  return false
})

const emailListSchema = z
  .string()
  .refine((v) => splitEmails(v).every(validateEmail), { message: t('validation.invalidEmail') })
  .default('')

const formSchema = z.object({
  subject: z.string().min(1, t('validation.subjectCannotBeEmpty')),
  content: z.string().min(1, t('validation.messageCannotBeEmpty')),
  inbox_id: z
    .any()
    .refine((val) => inboxStore.emailOptions.some((option) => option.value === val), {
      message: t('globals.messages.required')
    }),
  team_id: z.any().optional(),
  agent_id: z.any().optional(),
  contact_email: z.string().email(t('validation.invalidEmail')),
  cc: emailListSchema,
  bcc: emailListSchema,
  first_name: z.string().min(1, t('globals.messages.required')),
  last_name: z.string().optional()
})

onUnmounted(() => {
  clearMediaFiles()
  conversationStore.resetMacro(MACRO_CONTEXT.NEW_CONVERSATION)
})

onMounted(() => {
  restoreDraft()
  if (props.initialContact?.email) selectContact(props.initialContact)
})

defineExpose({ focus: () => emailInputRef.value?.$el?.focus() })

watch(
  () => props.initialContact,
  (contact) => {
    if (contact?.email) selectContact(contact)
  }
)

const form = useForm({
  validationSchema: toTypedSchema(formSchema),
  initialValues: {
    inbox_id: null,
    team_id: null,
    agent_id: userStore.userID ? String(userStore.userID) : null,
    subject: '',
    content: '',
    contact_email: '',
    cc: '',
    bcc: '',
    first_name: '',
    last_name: ''
  }
})

watch(emailQuery, (newVal) => {
  form.setFieldValue('contact_email', newVal, form.submitCount.value > 0)
  if (selectedContact.value && newVal !== selectedContact.value.email) {
    selectedContact.value = null
    form.setFieldValue('first_name', '', false)
    form.setFieldValue('last_name', '', false)
  }
})

const { searchResults, highlightedIndex, handleSearchContacts, handleSearchKeydown, selectContact, clearSearchResults } =
  useContactSearch({
    getQuery: () => emailQuery.value,
    filterResults: (c) => c.email,
    onSelect: (contact) => {
      selectedContact.value = contact
      emailQuery.value = contact.email
      form.setFieldValue('first_name', contact.first_name, false)
      form.setFieldValue('last_name', contact.last_name || '', false)
    }
  })

const createConversation = form.handleSubmit(async (values) => {
  loading.value = true
  try {
    values.inbox_id = Number(values.inbox_id)
    values.team_id = values.team_id && values.team_id !== 'none' ? Number(values.team_id) : null
    values.agent_id =
      values.agent_id && values.agent_id !== 'none' ? Number(values.agent_id) : null
    values.cc = splitEmails(values.cc)
    values.bcc = splitEmails(values.bcc)
    values.attachments = mediaFiles.value.map((file) => file.id)
    if (selectedContact.value?.external_user_id) {
      values.external_user_id = selectedContact.value.external_user_id
    }
    // Form data is a snapshot from search. Never let it overwrite the stored contact.
    values.reuse_contact = true
    values.initiator = UserTypeAgent
    const conversation = await api.createConversation(values)
    const conversationUUID = conversation.data.data.uuid

    const macro = conversationStore.getMacro(MACRO_CONTEXT.NEW_CONVERSATION)
    if (conversationUUID !== '' && macro?.id) {
      try {
        await api.applyMacro(conversationUUID, macro.id, macro.actions)
      } catch (error) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      }
    }
    draft.value = null
    emit('close')
    form.resetForm()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    loading.value = false
  }
})

watch(
  () => conversationStore.getMacro(MACRO_CONTEXT.NEW_CONVERSATION).id,
  () => {
    const content = conversationStore.getMacro(MACRO_CONTEXT.NEW_CONVERSATION).message_content
    if (content) form.setFieldValue('content', content)
  },
  { deep: true }
)

const hasDraftContent = (values) =>
  [values.contact_email, values.subject, values.cc, values.bcc, values.first_name].some((v) => v?.trim()) ||
  hasMessageContent(values.content)

const hasMessageContent = (html) => !!html && (html.replace(/<[^>]*>/g, '').trim() !== '' || html.includes('<img'))

function restoreDraft () {
  if (!draft.value?.values) return
  const { values, contact, attachments = [] } = draft.value
  form.setValues(values, false)
  setMediaFiles([...attachments])
  selectedContact.value = contact || null
  emailQuery.value = values.contact_email || ''
  showCc.value = !!values.cc
  showBcc.value = !!values.bcc
}

watch(
  [() => form.values, selectedContact, mediaFiles],
  ([values, contact, attachments]) => {
    draft.value = hasDraftContent(values) || attachments.length
      ? { values: { ...values }, contact, attachments: [...attachments] }
      : null
  },
  { deep: true }
)
</script>
