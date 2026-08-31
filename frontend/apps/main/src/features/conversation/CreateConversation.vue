<template>
  <Dialog v-model:open="dialogOpen">
    <DialogContent class="max-w-5xl w-full h-[90vh] flex flex-col">
      <DialogHeader>
        <DialogTitle>{{ $t('conversation.newConversation') }}</DialogTitle>
        <DialogDescription />
      </DialogHeader>

      <form @submit="createConversation" class="flex flex-col flex-1 overflow-hidden">
        <div class="space-y-4 pb-2 flex-shrink-0">
          <CreateConversationFields
            v-model:email-query="emailQuery"
            layout="grid"
            :inbox-store="inboxStore"
            :u-store="uStore"
            :team-store="teamStore"
            :search-results="searchResults"
            :highlighted-index="highlightedIndex"
            :selected-contact="selectedContact"
            :handle-search-contacts="handleSearchContacts"
            :handle-search-keydown="handleSearchKeydown"
            :select-contact="selectContact"
          />
        </div>

        <div class="flex-1 flex flex-col min-h-0 mt-4">
          <FormField v-slot="{ componentField }" name="content">
            <FormItem class="flex flex-col h-full">
              <FormLabel>{{ $t('globals.terms.message') }}</FormLabel>
              <FormControl class="flex-1 flex flex-col min-h-0">
                <div class="flex flex-col h-full">
                  <Editor
                    v-model:htmlContent="componentField.modelValue"
                    @update:htmlContent="(value) => componentField.onChange(value)"
                    :placeholder="t('editor.hint.newLineCtrlK')"
                    :insertContent="insertContent"
                    :autoFocus="false"
                    :enableInlineImages="true"
                    class="w-full flex-1 overflow-y-auto p-2 box min-h-0"
                    @send="createConversation"
                    @filesDropped="uploadFiles"
                  />

                  <MacroActionsPreview
                    v-if="conversationStore.getMacro(MACRO_CONTEXT.NEW_CONVERSATION).actions?.length > 0"
                    :actions="conversationStore.getMacro(MACRO_CONTEXT.NEW_CONVERSATION)?.actions || []"
                    :onRemove="(action) => conversationStore.removeMacroAction(action, MACRO_CONTEXT.NEW_CONVERSATION)"
                    class="mt-2 flex-shrink-0"
                  />

                  <ReplyBoxAttachmentPreview
                    v-if="mediaFiles.length > 0 || uploadingFiles.length > 0"
                    :attachments="mediaFiles"
                    :uploadingFiles="uploadingFiles"
                    :onDelete="handleFileDelete"
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
</template>

<script setup>
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription
} from '@shared-ui/components/ui/dialog'
import { Button } from '@shared-ui/components/ui/button'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form'
import { useI18n } from 'vue-i18n'
import { watch, nextTick } from 'vue'
import ReplyBoxAttachmentPreview from '@/features/conversation/message/attachment/ReplyBoxAttachmentPreview.vue'
import MacroActionsPreview from '@/features/conversation/MacroActionsPreview.vue'
import ReplyBoxMenuBar from '@/features/conversation/ReplyBoxMenuBar.vue'
import CreateConversationFields from '@/features/conversation/CreateConversationFields.vue'
import Editor from '@/components/editor/ConversationEditor.vue'
import { useCreateConversationForm } from '@main/composables/useCreateConversationForm'

const dialogOpen = defineModel({
  required: false,
  default: () => false
})

const { t } = useI18n()

const {
  inboxStore,
  uStore,
  teamStore,
  conversationStore,
  loading,
  searchResults,
  emailQuery,
  insertContent,
  selectedContact,
  highlightedIndex,
  uploadingFiles,
  mediaFiles,
  isDisabled,
  handleEmojiSelect,
  handleSearchContacts,
  handleSearchKeydown,
  selectContact,
  handleFileUpload,
  handleFileDelete,
  uploadFiles,
  createConversation,
  MACRO_CONTEXT
} = useCreateConversationForm({
  onSuccess: async () => {
    dialogOpen.value = false
  }
})

watch(dialogOpen, (open) => {
  if (!open) return
  nextTick(() => document.querySelector('[data-new-ticket-email]')?.focus())
})
</script>
