<template>
  <div class="flex flex-col h-full relative">
    <div class="absolute top-2 right-4 z-10">
      <CloseWidgetButton />
    </div>
    
    <div class="flex-1 min-h-0 overflow-y-auto p-4 scrollbar-thin scrollbar-track-transparent scrollbar-thumb-muted-foreground/30 hover:scrollbar-thumb-muted-foreground/50">
      <div class="flex flex-col gap-6">
        <HomeHeader :config="config" />

        <div class="flex flex-col gap-2">
          <!-- Show recent conversation if exists -->
          <RecentConversationCard
            v-if="mostRecentConversation"
            :conversation="mostRecentConversation"
          />

          <!-- Start new conversation button if no recent conversation -->
          <div v-if="canStartConversation && !mostRecentConversation">
            <Button @click="startConversation" class="w-full flex items-center justify-center">
              {{ startButtonText }}
              <ArrowRight class="w-4 h-4 ml-2 inline-block" />
            </Button>
          </div>

          <!-- External Links -->
          <HomeExternalLinks :config="config" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { ArrowRight } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { useWidgetStore } from '@widget/store/widget.js'
import { useChatStore } from '@widget/store/chat.js'
import { useUserStore } from '@widget/store/user.js'
import HomeHeader from '@widget/components/HomeHeader.vue'
import HomeExternalLinks from '@widget/components/HomeExternalLinks.vue'
import RecentConversationCard from '@widget/components/RecentConversationCard.vue'
import CloseWidgetButton from '@widget/components/CloseWidgetButton.vue'

const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const userStore = useUserStore()
const config = computed(() => widgetStore.config)

const mostRecentConversation = computed(() => {
  const conversations = chatStore.getConversations
  if (!conversations || conversations.length === 0) return null
  // Get the most recent conversation (already sorted by last_message.created_at in the store)
  return conversations[0]
})

const canStartConversation = computed(() => {
  const userConfig = userStore.isVisitor ? config.value.visitors : config.value.users
  return userConfig?.prevent_multiple_conversations !== true || !chatStore.hasConversations
})

const startButtonText = computed(() => {
  const isVisitor = userStore.isVisitor
  return isVisitor
    ? config.value.visitors?.start_conversation_button_text || 'Send us a message'
    : config.value.users?.start_conversation_button_text || 'Send us a message'
})

const startConversation = () => {
  chatStore.setCurrentConversation(null)
  widgetStore.navigateToChat()
}
</script>
