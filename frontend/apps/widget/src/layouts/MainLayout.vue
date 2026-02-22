<template>
  <div class="flex flex-col h-full">
    <Tabs v-model="widgetStore.currentView" class="flex flex-col h-full">
      <div class="flex-1 min-h-0">
        <TabsContent value="home" class="h-full mt-0">
          <HomeView />
        </TabsContent>
        <TabsContent value="messages" class="h-full mt-0">
          <ConversationsView v-if="!widgetStore.isChatView" />
          <ChatView v-else />
        </TabsContent>
      </div>
      <TabsList class="grid grid-cols-2 border-t rounded-none">
        <TabsTrigger value="home" class="flex gap-1">
          <Home class="w-5 h-5" />
          <span class="text-xs">{{ $t('globals.terms.home') }}</span>
        </TabsTrigger>
        <TabsTrigger value="messages" class="flex gap-1">
          <MessageCircle class="w-5 h-5" />
          <span class="text-xs">{{ $t('globals.terms.message', 2) }}</span>
        </TabsTrigger>
      </TabsList>
      <div
        v-if="widgetStore.config?.show_powered_by !== false"
        class="text-center flex items-center justify-center"
      >
        <span class="text-[10px] text-muted-foreground"
          >Powered by <a href="https://libredesk.io" target="_blank">Libredesk</a></span
        >
      </div>

      <!-- Network Connection Banner -->
      <ConnectionBanner />
    </Tabs>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@shared-ui/components/ui/tabs'
import HomeView from '@widget/views/HomeView.vue'
import { Home, MessageCircle } from 'lucide-vue-next'
import ChatView from '@widget/views/ChatView.vue'
import ConversationsView from '@widget/views/ConversationsView.vue'
import ConnectionBanner from '@widget/components/ConnectionBanner.vue'
import { useWidgetStore } from '@widget/store/widget.js'
import { useChatStore } from '@widget/store/chat.js'

const widgetStore = useWidgetStore()
const chatStore = useChatStore()

onMounted(async () => {
  if (widgetStore.config?.direct_to_conversation) {
    await chatStore.fetchConversations()
    if (chatStore.hasConversations) {
      const latest = chatStore.getConversations[0]
      await chatStore.loadConversation(latest.uuid)
    }
    // Always navigate to chat - shows latest convo OR new convo screen
    widgetStore.navigateToChat()
  }
})
</script>
