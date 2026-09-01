<template>
  <div class="flex flex-col h-full w-full min-h-0 min-w-0 bg-background">
    <div
      v-if="isLoading"
      class="h-0.5 shrink-0 bg-primary/40 animate-pulse"
    />

    <div
      v-if="showContent"
      class="zendesk-ticket-shell flex-1 min-h-0 min-w-0"
    >
      <ZendeskTicketChrome />

      <div
        class="zendesk-ticket-workspace flex-1 min-h-0 min-w-0"
        :class="{ 'context-collapsed': contextCollapsed }"
      >
        <ZendeskTicketProperties />

        <div class="zendesk-ticket-center">
          <div class="px-4 py-2.5 border-b shrink-0 bg-background flex items-start justify-between gap-3">
          <div class="min-w-0">
            <h2 class="zendesk-title truncate leading-snug">
              {{ conversationStore.current?.subject || t('zendesk.noSubject') }}
            </h2>
            <p
              v-if="conversationStore.current?.inbox_name"
              class="zendesk-meta mt-0.5"
            >
              {{ t('zendesk.viaInbox', { inbox: conversationStore.current.inbox_name }) }}
            </p>
          </div>
          <ConversationViewers class="shrink-0 mt-0.5" />
        </div>

        <div class="flex flex-col flex-1 min-h-0 overflow-hidden">
          <MessageList class="flex-1 overflow-y-auto" />
        </div>

        <div class="zendesk-reply-stack shrink-0">
          <ReplyBox />
          <ZendeskSubmitBar />
        </div>
      </div>

      <ZendeskCustomerContext />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, watch, onMounted } from 'vue'
import { useStorage } from '@vueuse/core'
import { useDocumentVisibility } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { useConversationStore } from '@main/stores/conversation'
import MessageList from '@/features/conversation/message/MessageList.vue'
import ReplyBox from '@/features/conversation/ReplyBox.vue'
import ZendeskTicketProperties from './ZendeskTicketProperties.vue'
import ZendeskCustomerContext from './ZendeskCustomerContext.vue'
import ZendeskTicketChrome from './ZendeskTicketChrome.vue'
import ZendeskSubmitBar from './ZendeskSubmitBar.vue'
import ConversationViewers from '@/features/conversation/ConversationViewers.vue'

const props = defineProps({
  uuid: String
})

const { t } = useI18n()
const conversationStore = useConversationStore()
const contextCollapsed = useStorage('libredesk_zendesk_context_collapsed', false)

const showContent = computed(
  () => conversationStore.current || conversationStore.conversation.loading
)

const isLoading = computed(
  () => conversationStore.conversation.loading || conversationStore.messages.loading
)

const fetchConversation = async (uuid) => {
  await Promise.all([
    conversationStore.fetchConversation(uuid),
    conversationStore.fetchMessages(uuid)
  ])
  await conversationStore.updateAssigneeLastSeen(uuid)
}

onMounted(() => {
  if (props.uuid) fetchConversation(props.uuid)
})

watch(
  () => props.uuid,
  (newUUID, oldUUID) => {
    if (!newUUID || newUUID === oldUUID) return
    fetchConversation(newUUID)
  }
)

const visibility = useDocumentVisibility()
watch(visibility, (state) => {
  if (state === 'visible' && props.uuid) {
    conversationStore.updateAssigneeLastSeen(props.uuid)
  }
})
</script>
