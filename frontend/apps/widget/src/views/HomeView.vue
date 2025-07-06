<template>
  <div class="flex flex-col p-4 h-full gap-5">
    <div class="p-2">
      <!-- Logo Section -->
      <img v-if="config.logo_url" :src="config.logo_url" alt="Logo" class="max-h-8 max-w-full" />

      <!-- Welcome Message -->
      <div class="mt-24 font-medium text-4xl">
        <h2>{{ config.greeting_message || 'Hi there' }}</h2>
        <p class="text-muted-foreground mt-2">
          {{ config.introduction_message || 'How can we help?' }}
        </p>
      </div>
    </div>

    <!-- Start Conversation Button -->
    <div class="mt-3">
      <div>
        <Button
          v-if="canStartConversation"
          @click="startConversation"
          class="w-full h-12"
          size="lg"
        >
          {{ getStartButtonText() }}
          <ArrowRight class="w-4 h-4 ml-2" />
        </Button>
      </div>
    </div>

    <!-- External Links -->
    <div v-if="config.external_links?.length" class="mt-2 space-y-2">
      <a
        v-for="link in config.external_links"
        :key="link.url"
        :href="link.url"
        target="_blank"
        rel="noopener noreferrer"
        class="block no-underline"
      >
        <Card class="hover:bg-accent transition-colors cursor-pointer rounded-md">
          <CardContent class="p-4">
            <div class="flex justify-between items-center">
              <span class="text-sm text-primary font-medium">
                {{ link.text }}
              </span>
              <ExternalLink size="18" class="text-muted-foreground" />
            </div>
          </CardContent>
        </Card>
      </a>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { ArrowRight, ExternalLink } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Card, CardContent } from '@shared-ui/components/ui/card'
import { useWidgetStore } from '../store/widget.js'
import { useChatStore } from '../store/chat.js'
import { useUserStore } from '@widget/store/user.js'

const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const userStore = useUserStore()
const config = computed(() => widgetStore.config)

const canStartConversation = computed(() => {
  const userConfig = userStore.isVisitor ? config.value.visitors : config.value.users
  return userConfig?.prevent_multiple_conversations !== true || !chatStore.hasConversations
})

const getStartButtonText = () => {
  const isVisitor = userStore.isVisitor
  return isVisitor
    ? config.value.visitors?.start_conversation_button_text || 'Send us a message'
    : config.value.users?.start_conversation_button_text || 'Send us a message'
}

onMounted(() => {
  chatStore.fetchConversations()
})

const startConversation = () => {
  // Clear current conversation for new one
  chatStore.setCurrentConversation(null)
  chatStore.clearMessages()
  // Navigate to messages tab and then to chat view
  widgetStore.navigateToMessages()
  // Use nextTick to ensure we're on messages tab first, then navigate to chat
  setTimeout(() => {
    widgetStore.navigateToChat()
  }, 0)
}
</script>
