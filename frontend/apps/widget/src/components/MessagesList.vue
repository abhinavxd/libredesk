<template>
  <div>
    <!-- Loading State -->
    <div v-if="isLoadingConversations" class="py-8">
      <Spinner size="md" :text="$t('globals.terms.loading')" />
    </div>

    <!-- Empty State -->
    <div v-else-if="!hasConversations" class="flex flex-col items-center justify-center py-12 px-4">
      <MessageCircleDashed class="w-10 h-10 text-muted-foreground mb-4" />
      <h4 class="text-sm text-muted-foreground mb-2">No messages yet</h4>
    </div>

    <!-- List of conversations -->
    <div v-else class="divide-y divide-border">
      <div
        v-for="conversation in getConversations"
        :key="conversation.uuid"
        @click="setCurrentConversation(conversation.uuid)"
        class="p-4 hover:bg-accent/50 cursor-pointer transition-colors"
      >
        <div class="flex items-center justify-between gap-3">
          <div class="flex-1 min-w-0">
            <h4 class="font-medium text-foreground text-sm truncate mb-1">
              {{ conversation.last_message.content }}
            </h4>
            <div
              class="text-xs text-muted-foreground flex items-center gap-1"
              v-if="conversation.last_message && conversation.last_message.author"
            >
              <span v-if="conversation.last_message?.author?.id !== userStore.userID">
                {{
                  conversation.last_message?.author.first_name +
                  ' ' +
                  conversation.last_message?.author.last_name
                }}
              </span>
              <span v-else>{{ $t('globals.terms.you') }}</span>
              <span>•</span>
              <span>{{ getConversationTime(conversation.last_message.created_at) }}</span>
            </div>
          </div>

          <div class="flex items-center justify-center flex-shrink-0 gap-2">
            <UnreadCountBadge :count="conversation.unread_message_count" />
            <ArrowRight class="w-4 h-4" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { MessageCircleDashed, ArrowRight } from 'lucide-vue-next'
import { useChatStore } from '../store/chat.js'
import { useWidgetStore } from '../store/widget.js'
import { useUserStore } from '@widget/store/user.js'
import { useRelativeTime } from '@widget/composables/useRelativeTime.js'
import UnreadCountBadge from './UnreadCountBadge.vue'
import { Spinner } from '@shared-ui/components/ui/spinner'

const chatStore = useChatStore()
const widgetStore = useWidgetStore()
const userStore = useUserStore()

const hasConversations = computed(() => chatStore.hasConversations)
const getConversations = computed(() => chatStore.getConversations)
const isLoadingConversations = computed(() => chatStore.isLoadingConversations)

const getConversationTime = (timestamp) => {
  return useRelativeTime(new Date(timestamp)).value
}

const setCurrentConversation = async (conversationUUID) => {
  // Navigate to chat view
  widgetStore.navigateToChat()
  // Fetch conversation and messages and set it as current conversation.
  const fetched = await chatStore.loadConversation(conversationUUID)
  // If fetch fails, navigate back to message list.
  if (!fetched) {
    widgetStore.navigateToMessages()
  }
}

onMounted(() => {
  chatStore.fetchConversations()
})
</script>
