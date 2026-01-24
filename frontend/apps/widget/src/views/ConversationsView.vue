<template>
  <div class="flex flex-col h-full relative">
    <!-- Header -->
    <WidgetHeader :title="$t('globals.terms.message', 2)" />

    <!-- Messages List -->
    <div class="flex-1 overflow-y-auto pb-20">
      <ConversationsList />
    </div>

    <!-- Floating button with gradient fade -->
    <div v-if="canStartNewConversation" class="absolute bottom-0 inset-x-0">
      <!-- Gradient fade overlay -->
      <div
        class="h-20 bg-gradient-to-t from-background via-background/80 to-transparent pointer-events-none"
      ></div>

      <!-- Floating button -->
      <div class="absolute bottom-4 inset-x-0 mx-auto w-fit z-10">
        <Button @click="startNewConversation">
          {{
            widgetStore.config?.users?.start_conversation_button_text ||
            $t('globals.messages.startNewConversation')
          }}
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import { useChatStore } from '../store/chat.js'
import { useWidgetStore } from '../store/widget.js'
import { useUserStore } from '@widget/store/user.js'
import ConversationsList from '../components/ConversationsList.vue'
import WidgetHeader from '../layouts/WidgetHeader.vue'

const chatStore = useChatStore()
const widgetStore = useWidgetStore()
const userStore = useUserStore()

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

const startNewConversation = () => {
  // Clear current conversation
  chatStore.setCurrentConversation(null)
  chatStore.clearMessages()

  // Navigate directly to chat view
  widgetStore.navigateToChat()
}
</script>
