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
  <div v-else class="space-y-1">
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
        <Tooltip>
          <TooltipTrigger asChild>
            <div class="flex gap-1 items-center text-xs text-muted-foreground">
              <span v-if="conversation.created_at">
                {{ getRelativeTime(new Date(conversation.created_at)) }}
              </span>
              <span>•</span>
              <span v-if="conversation.last_message_at">
                {{ getRelativeTime(new Date(conversation.last_message_at)) }}
              </span>
            </div>
          </TooltipTrigger>
          <TooltipContent>
            <div class="space-y-1 text-xs">
              <p>
                {{ $t('globals.terms.createdAt') }}:
                {{ formatFullTimestamp(new Date(conversation.created_at)) }}
              </p>
              <p v-if="conversation.last_message_at">
                {{ $t('globals.terms.lastMessageAt') }}:
                {{ formatFullTimestamp(new Date(conversation.last_message_at)) }}
              </p>
            </div>
          </TooltipContent>
        </Tooltip>
      </div>
    </router-link>
  </div>
</template>

<script setup>
import { useConversationStore } from '@/stores/conversation'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { formatFullTimestamp, getRelativeTime } from '@/utils/datetime'

const conversationStore = useConversationStore()
</script>
