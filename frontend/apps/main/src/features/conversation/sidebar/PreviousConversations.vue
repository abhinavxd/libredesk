<template>
  <div
    v-if="
      conversationStore.current?.previous_conversations?.length === 0 ||
      conversationStore.conversation?.loading
    "
    class="text-center text-sm text-muted-foreground py-4"
  >
    {{ $t('conversation.sidebar.noPreviousConvo') }}
  </div>
  <div v-else class="space-y-3">
    <router-link
      v-for="conversation in conversationStore.current.previous_conversations"
      :key="conversation.uuid"
      :to="{
        name: 'inbox-conversation',
        params: {
          uuid: conversation.uuid,
          type: 'assigned'
        }
      }"
      class="block p-2 rounded hover:bg-muted"
    >
      <div class="flex items-center justify-between">
        <div class="flex flex-col">
          <span class="font-medium text-sm">
            {{ conversation.contact.first_name }} {{ conversation.contact.last_name }}
          </span>
          <span class="text-xs text-muted-foreground truncate max-w-[200px]">
            {{ conversation.last_message }}
          </span>
        </div>
        <span class="text-xs text-muted-foreground" v-if="conversation.last_message_at">
          {{ format(new Date(conversation.last_message_at), 'h') + ' h' }}
        </span>
      </div>
    </router-link>
  </div>
</template>

<script setup>
import { useConversationStore } from '../../../stores/conversation'
import { format } from 'date-fns'

const conversationStore = useConversationStore()
</script>
