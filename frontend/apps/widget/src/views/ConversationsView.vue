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
          {{ startButtonText }}
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import { useChatStore } from '../store/chat.js'
import { useWidgetStore } from '../store/widget.js'
import { useUserStore } from '@widget/store/user.js'
import ConversationsList from '../components/ConversationsList.vue'
import WidgetHeader from '../layouts/WidgetHeader.vue'

const { t } = useI18n()
const chatStore = useChatStore()
const widgetStore = useWidgetStore()
const userStore = useUserStore()

const startButtonText = computed(() => {
  const userConfig = userStore.isVisitor ? widgetStore.config?.visitors : widgetStore.config?.users
  return userConfig?.start_conversation_button_text || t('globals.messages.startNewConversation')
})

const canStartNewConversation = computed(() => {
  const userConfig = userStore.isVisitor ? widgetStore.config?.visitors : widgetStore.config?.users
  // Mirrors the server check, else the button is offered and the send fails.
  if (!userConfig?.allow_start_conversation) return false
  return userConfig?.prevent_multiple_conversations !== true || !chatStore.hasConversations
})

const startNewConversation = () => {
  // Clear current conversation
  chatStore.setCurrentConversation(null)
  chatStore.clearMessages()

  // Navigate directly to chat view
  widgetStore.navigateToChat()
}
</script>
