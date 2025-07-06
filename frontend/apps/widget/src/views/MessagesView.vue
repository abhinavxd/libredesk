<template>
  <div class="flex flex-col h-full">
    <!-- Header -->
    <div class="flex items-center justify-center p-4 border-b">
      <h3 class="text-base font-semibold text-foreground">Messages</h3>
    </div>

    <!-- Conversations List -->
    <div class="flex-1 overflow-y-auto">
      <!-- Loading State -->
      <div v-if="isLoadingConversations" class="flex items-center justify-center py-8">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>

      <!-- Empty State -->
      <div
        v-else-if="!hasConversations"
        class="flex flex-col items-center justify-center py-12 px-4"
      >
        <MessageCircleDashed class="w-10 h-10 text-muted-foreground mb-4" />
        <h4 class="text-sm text-muted-foreground mb-2">No messages yet</h4>
      </div>

      <!-- Conversations -->
      <div v-else class="divide-y divide-border">
        <div
          v-for="conversation in getConversations"
          :key="conversation.uuid"
          @click="openConversation(conversation)"
          class="p-4 hover:bg-accent/50 cursor-pointer transition-colors"
        >
          <div class="flex items-center justify-between gap-3">
            <div class="flex-1 min-w-0">
              <h4 class="font-medium text-foreground text-sm truncate mb-1">
                {{ conversation.last_message }}
              </h4>
              <div class="text-xs text-muted-foreground">
                {{ widgetStore.config?.brand_name }}
              </div>
            </div>

            <div class="flex flex-col items-end gap-1 flex-shrink-0">
              <span
                v-if="conversation.unread_message_count > 0"
                class="bg-primary text-primary-foreground text-xs font-medium rounded-full px-2 py-1"
              >
                {{
                  conversation.unread_message_count > 10 ? '9+' : conversation.unread_message_count
                }}
              </span>
              <span class="text-xs text-muted-foreground">
                {{ getConversationTime(conversation.last_message_at) }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- New Conversation Button -->
    <div class="p-4 border-border mx-auto" v-if="canStartNewConversation">
      <Button @click="startNewConversation">
        {{ widgetStore.config?.users?.start_conversation_button_text || 'Start new conversation' }}
      </Button>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { MessageCircleDashed } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { useChatStore } from '../store/chat.js'
import { useWidgetStore } from '../store/widget.js'
import { useUserStore } from '@widget/store/user.js'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'

const chatStore = useChatStore()
const widgetStore = useWidgetStore()
const userStore = useUserStore()
const timeUpdateTrigger = ref(0)

const hasConversations = computed(() => chatStore.hasConversations)
const getConversations = computed(() => chatStore.getConversations)
const isLoadingConversations = computed(() => chatStore.isLoadingConversations)

const getConversationTime = computed(() => {
  timeUpdateTrigger.value // Force reactivity
  return (timestamp) => getRelativeTime(timestamp)
})

const canStartNewConversation = computed(() => {
  const isVisitor = userStore.isVisitor
  if (isVisitor) {
    if (widgetStore.config?.visitors?.prevent_multiple_conversations) {
      return !chatStore.hasConversations
    }
    return widgetStore.config?.visitors?.allow_start_conversation ?? true
  } else {
    if (widgetStore.config?.users?.prevent_multiple_conversations) {
      return !chatStore.hasConversations
    }
    return widgetStore.config?.users?.allow_start_conversation ?? true
  }
})

const openConversation = (conversation) => {
  // Set the current conversation in chat store.
  chatStore.openConversation(conversation)

  // Navigate to chat view
  widgetStore.navigateToChat()
}

const startNewConversation = () => {
  // Clear current conversation
  chatStore.setCurrentConversation(null)
  chatStore.clearMessages()

  // Navigate directly to chat view
  widgetStore.navigateToChat()
}

onMounted(() => {
  chatStore.fetchConversations()

  // Update relative times every minute
  const timeUpdateTimer = setInterval(() => {
    timeUpdateTrigger.value++
  }, 60000)

  // Cleanup on unmount
  onUnmounted(() => {
    if (timeUpdateTimer) {
      clearInterval(timeUpdateTimer)
    }
  })
})
</script>
