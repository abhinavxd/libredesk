<template>
  <div class="flex flex-col h-full">
    <Tabs v-model="widgetStore.currentView" class="flex flex-col h-full">
      <div class="flex-1 min-h-0">
        <TabsContent value="home" class="h-full mt-0">
          <HomeView />
        </TabsContent>
        <TabsContent value="messages" class="h-full mt-0">
          <MessagesView v-if="!widgetStore.isChatView" />
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
      <div class="text-center flex items-center justify-center py-1">
        <span class="text-[10px] text-muted-foreground"
          >Powered by <a href="https://libredesk.io" target="_blank">Libredesk</a></span
        >
      </div>
    </Tabs>
  </div>
</template>

<script setup>
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@shared-ui/components/ui/tabs'
import HomeView from '@widget/views/HomeView.vue'
import { Home, MessageCircle } from 'lucide-vue-next'
import ChatView from '@widget/views/ChatView.vue'
import MessagesView from '@widget/views/MessagesView.vue'
import { useWidgetStore } from '@widget/store/widget.js'
const widgetStore = useWidgetStore()
</script>
