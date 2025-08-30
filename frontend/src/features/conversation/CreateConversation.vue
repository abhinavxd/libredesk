<template>
  <div>
    <Dialog v-model:open="dialogOpen">
      <DialogContent class="max-w-5xl w-full h-[90vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>
            {{
              $t('globals.messages.new', {
                name: $t('globals.terms.conversation').toLowerCase()
              })
            }}
          </DialogTitle>
          <DialogDescription />
        </DialogHeader>
        <form @submit="createConversation" class="flex flex-col flex-1 overflow-hidden">
          <!-- Form Fields Section -->
          <div class="space-y-4 pb-2 flex-shrink-0">
            <div class="space-y-2">
              <FormField name="contact_email">
                <FormItem class="relative">
                  <FormLabel>{{ $t('globals.terms.email') }}</FormLabel>
                  <FormControl>
                    <Input
                      type="email"
                      :placeholder="t('conversation.searchContact')"
                      v-model="emailQuery"
                      @input="handleSearchContacts"
                      autocomplete="off"
                    />
                  </FormControl>
                  <FormMessage />

                  <ul
                    v-if="searchResults.length"
                    class="border rounded p-2 max-h-60 overflow-y-auto absolute w-full z-50 shadow bg-background"
                  >
                    <li
                      v-for="contact in searchResults"
                      :key="contact.email"
                      @click="selectContact(contact)"
                      class="cursor-pointer p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-800"
                    >
                      {{ contact.first_name }} {{ contact.last_name }} ({{ contact.email }})
                    </li>
                  </ul>
                </FormItem>
              </FormField>

              <!-- Name Group -->
              <div class="grid grid-cols-2 gap-4">
                <FormField v-slot="{ componentField }" name="first_name">
                  <FormItem>
                    <FormLabel>{{ $t('globals.terms.firstName') }}</FormLabel>
                    <FormControl>
                      <Input type="text" placeholder="" v-bind="componentField" required />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>

                <FormField v-slot="{ componentField }" name="last_name">
                  <FormItem>
                    <FormLabel>{{ $t('globals.terms.lastName') }}</FormLabel>
                    <FormControl>
                      <Input type="text" placeholder="" v-bind="componentField" />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <!-- Subject and Inbox Group -->
              <div class="grid grid-cols-2 gap-4">
                <FormField v-slot="{ componentField }" name="subject">
                  <FormItem>
                    <FormLabel>{{ $t('globals.terms.subject') }}</FormLabel>
                    <FormControl>
                      <Input type="text" placeholder="" v-bind="componentField" />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>

                <FormField v-slot="{ componentField }" name="inbox_id">
                  <FormItem>
                    <FormLabel>{{ $t('globals.terms.inbox') }}</FormLabel>
                    <FormControl>
                      <Select v-bind="componentField">
                        <SelectTrigger>
                          <SelectValue
                            :placeholder="
                              t('globals.messages.select', { name: t('globals.terms.inbox') })
                            "
                          />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            <SelectItem
                              v-for="option in inboxStore.options"
                              :key="option.value"
                              :value="option.value"
                            >
                              {{ option.label }}
                            </SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <!-- Assignment Group -->
              <div class="grid grid-cols-2 gap-4">
                <!-- Set assigned team -->
                <FormField v-slot="{ componentField }" name="team_id">
                  <FormItem>
                    <FormLabel>
                      {{
                        $t('globals.messages.assign', {
                          name: t('globals.terms.team').toLowerCase()
                        })
                      }}
                      ({{ $t('globals.terms.optional').toLowerCase() }})
                    </FormLabel>
                    <FormControl>
                      <SelectComboBox
                        v-bind="componentField"
                        :items="[{ value: 'none', label: 'None' }, ...teamStore.options]"
                        :placeholder="
                          t('globals.messages.select', { name: t('globals.terms.team') })
                        "
                        type="team"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>

                <!-- Set assigned agent -->
                <FormField v-slot="{ componentField }" name="agent_id">
                  <FormItem>
                    <FormLabel>
                      {{
                        $t('globals.messages.assign', {
                          name: t('globals.terms.agent').toLowerCase()
                        })
                      }}
                      ({{ $t('globals.terms.optional').toLowerCase() }})
                    </FormLabel>
                    <FormControl>
                      <SelectComboBox
                        v-bind="componentField"
                        :items="[{ value: 'none', label: 'None' }, ...uStore.options]"
                        :placeholder="
                          t('globals.messages.select', { name: t('globals.terms.agent') })
                        "
                        type="user"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>
            </div>
          </div>

          <!-- Message Editor Section -->
          <div class="flex-1 flex flex-col min-h-0 mt-4">
            <FormField v-slot="{ componentField }" name="content">
              <FormItem class="flex flex-col h-full">
                <FormLabel>{{ $t('globals.terms.message') }}</FormLabel>
                <FormControl class="flex-1 flex flex-col min-h-0">
                  <div class="flex flex-col h-full">
                    <Editor
                      v-model:htmlContent="componentField.modelValue"
                      @update:htmlContent="(value) => componentField.onChange(value)"
                      :placeholder="t('editor.newLine') + t('editor.ctrlK')"
                      :insertContent="insertContent"
                      :autoFocus="false"
                      class="w-full flex-1 overflow-y-auto p-2 box min-h-0"
                      @send="createConversation"
                    />

                    <!-- Macro preview -->
                    <MacroActionsPreview
                      v-if="conversationStore.getMacro('new-conversation').actions?.length > 0"
                      :actions="conversationStore.getMacro('new-conversation')?.actions || []"
                      :onRemove="
                        (action) => conversationStore.removeMacroAction(action, 'new-conversation')
                      "
                      class="mt-2 flex-shrink-0"
                    />

                    <!-- Attachments preview -->
                    <AttachmentsPreview
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
          </div>

          <DialogFooter class="mt-4 pt-2 flex items-center !justify-between w-full flex-shrink-0">
            <ReplyBoxMenuBar
              :handleFileUpload="handleFileUpload"
              @emojiSelect="handleEmojiSelect"
              :showSendButton="false"
            />
            <Button type="submit" :disabled="isDisabled" :isLoading="loading">
              {{ $t('globals.messages.submit') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { z } from 'zod'
import { ref, watch, onUnmounted, nextTick, onMounted, computed } from 'vue'
import AttachmentsPreview from '@/features/conversation/message/attachment/AttachmentsPreview.vue'
import { useConversationStore } from '@/stores/conversation'
import MacroActionsPreview from '@/features/conversation/MacroActionsPreview.vue'
import ReplyBoxMenuBar from '@/features/conversation/ReplyBoxMenuBar.vue'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { useInboxStore } from '@/stores/inbox'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { useI18n } from 'vue-i18n'
import { useFileUpload } from '@/composables/useFileUpload'
import Editor from '@/components/editor/TextEditor.vue'
import { useMacroStore } from '@/stores/macro'
import SelectComboBox from '@/components/combobox/SelectCombobox.vue'
import { UserTypeAgent } from '@/constants/user'
import api from '@/api'

const dialogOpen = defineModel({
  required: false,
  default: () => false
})

const inboxStore = useInboxStore()
const { t } = useI18n()
const uStore = useUsersStore()
const teamStore = useTeamStore()
const emitter = useEmitter()
const loading = ref(false)
const searchResults = ref([])
const emailQuery = ref('')
const conversationStore = useConversationStore()
const macroStore = useMacroStore()
let timeoutId = null
const insertContent = ref('')

const handleEmojiSelect = (emoji) => {
  insertContent.value = undefined
  // Force reactivity so the user can select the same emoji multiple times
  nextTick(() => (insertContent.value = emoji))
}

const { uploadingFiles, handleFileUpload, handleFileDelete, mediaFiles, clearMediaFiles } =
  useFileUpload({
    linkedModel: 'messages'
  })

const isDisabled = computed(() => {
  if (loading.value || uploadingFiles.value.length > 0) {
    return true
  }
  return false
})

const formSchema = z.object({
  subject: z.string().min(
    1,
    t('globals.messages.cannotBeEmpty', {
      name: t('globals.terms.subject')
    })
  ),
  content: z.string().min(
    1,
    t('globals.messages.cannotBeEmpty', {
      name: t('globals.terms.message')
    })
  ),
  inbox_id: z.any().refine((val) => inboxStore.options.some((option) => option.value === val), {
    message: t('globals.messages.required')
  }),
  team_id: z.any().optional(),
  agent_id: z.any().optional(),
  contact_email: z.string().email(t('globals.messages.invalidEmailAddress')),
  first_name: z.string().min(1, t('globals.messages.required')),
  last_name: z.string().optional()
})

onUnmounted(() => {
  clearTimeout(timeoutId)
  clearMediaFiles()
  conversationStore.resetMacro('new-conversation')
  emitter.emit(EMITTER_EVENTS.SET_NESTED_COMMAND, {
    command: null,
    open: false
  })
})

onMounted(() => {
  macroStore.setCurrentView('starting_conversation')
  emitter.emit(EMITTER_EVENTS.SET_NESTED_COMMAND, {
    command: 'apply-macro-to-new-conversation',
    open: false
  })
})

const form = useForm({
  validationSchema: toTypedSchema(formSchema),
  initialValues: {
    inbox_id: null,
    team_id: null,
    agent_id: null,
    subject: '',
    content: '',
    contact_email: '',
    first_name: '',
    last_name: ''
  }
})

watch(emailQuery, (newVal) => {
  form.setFieldValue('contact_email', newVal)
})

const handleSearchContacts = async () => {
  clearTimeout(timeoutId)
  timeoutId = setTimeout(async () => {
    const query = emailQuery.value.trim()

    if (query.length < 3) {
      searchResults.value.splice(0)
      return
    }

    try {
      const resp = await api.searchContacts({ query })
      searchResults.value = [...resp.data.data]
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
      searchResults.value.splice(0)
    }
  }, 300)
}

const selectContact = (contact) => {
  emailQuery.value = contact.email
  form.setFieldValue('first_name', contact.first_name)
  form.setFieldValue('last_name', contact.last_name || '')
  searchResults.value.splice(0)
}

const createConversation = form.handleSubmit(async (values) => {
  loading.value = true
  try {
    // Convert ids to numbers if they are not already
    values.inbox_id = Number(values.inbox_id)
    values.team_id = values.team_id ? Number(values.team_id) : null
    values.agent_id = values.agent_id ? Number(values.agent_id) : null
    // Array of attachment ids.
    values.attachments = mediaFiles.value.map((file) => file.id)
    // Initiator of this conversation is always agent
    values.initiator = UserTypeAgent
    const conversation = await api.createConversation(values)
    const conversationUUID = conversation.data.data.uuid

    // Get macro from context, and set if any actions are available.
    const macro = conversationStore.getMacro('new-conversation')
    if (conversationUUID !== '' && macro?.id && macro?.actions?.length > 0) {
      try {
        await api.applyMacro(conversationUUID, macro.id, macro.actions)
      } catch (error) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      }
    }
    dialogOpen.value = false
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

/**
 * Watches for changes in the macro id and update message content.
 */
watch(
  () => conversationStore.getMacro('new-conversation').id,
  () => {
    form.setFieldValue('content', conversationStore.getMacro('new-conversation').message_content)
  },
  { deep: true }
)
</script>
