<template>
  <div class="flex flex-col h-full relative">
    <div
      v-if="widgetStore.currentView === 'home'"
      class="absolute right-2 z-20"
      :class="widgetStore.isMobileFullScreen ? 'top-[max(1rem,env(safe-area-inset-top))]' : 'top-2'"
    >
      <CloseWidgetButton :class="textColorClass" />
    </div>
    <Tabs
      :modelValue="widgetStore.currentView"
      @update:modelValue="handleTabChange"
      class="flex flex-col h-full"
    >
      <main class="flex-1 min-h-0">
        <TabsContent value="home" class="h-full mt-0">
          <HomeView />
        </TabsContent>
        <TabsContent value="messages" class="h-full mt-0">
          <ConversationsView v-if="!widgetStore.isChatView" />
          <ChatView v-else />
        </TabsContent>
        <TabsContent value="help" class="h-full mt-0"><HelpView /></TabsContent>
      </main>
      <TabsList
        v-if="!widgetStore.isChatView"
        class="flex h-auto bg-background border-t rounded-none p-0"
      >
        <TabsTrigger value="home" class="nav-tab">
          <House class="w-5 h-5" aria-hidden="true" />
          <span class="text-xs font-medium">{{ $t('globals.terms.home') }}</span>
        </TabsTrigger>
        <TabsTrigger value="messages" class="nav-tab">
          <MessagesSquare class="w-5 h-5" aria-hidden="true" />
          <span class="text-xs font-medium">{{ $t('globals.terms.message', 2) }}</span>
        </TabsTrigger>
        <TabsTrigger v-if="help.available && help.data.audience.tab" value="help" class="nav-tab">
          <CircleQuestionMark class="w-5 h-5" aria-hidden="true" />
          <span class="text-xs font-medium">{{ $t('globals.terms.help') }}</span>
        </TabsTrigger>
      </TabsList>
      <div
        v-if="widgetStore.config?.show_powered_by !== false && widgetStore.isChatView"
        class="flex items-center justify-center pb-1.5"
      >
        <a
          href="https://libredesk.io"
          target="_blank"
          rel="noopener noreferrer"
          class="text-[10px] text-muted-foreground/70 hover:text-muted-foreground transition-colors no-underline"
        >
          Powered by <span class="font-medium">libredesk</span>
        </a>
      </div>
      <ConnectionBanner />
    </Tabs>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@shared-ui/components/ui/tabs'
import HomeView from '@widget/views/HomeView.vue'
import { House, MessagesSquare, CircleQuestionMark } from 'lucide-vue-next'
import HelpView from '@widget/views/HelpView.vue'
import { useHelpStore } from '@widget/store/help.js'
import ChatView from '@widget/views/ChatView.vue'
import ConversationsView from '@widget/views/ConversationsView.vue'
import ConnectionBanner from '@widget/components/ConnectionBanner.vue'
import CloseWidgetButton from '@widget/components/CloseWidgetButton.vue'
import { useHeaderTheme } from '@widget/composables/useHeaderTheme.js'
import { useWidgetStore } from '@widget/store/widget.js'

const widgetStore = useWidgetStore()
const help = useHelpStore()
const { textColorClass } = useHeaderTheme(computed(() => widgetStore.config))

const handleTabChange = (value) => {
  if (value === 'home') {
    widgetStore.navigateToHome()
  } else if (value === 'messages') {
    widgetStore.navigateToMessages()
  } else if (value === 'help') {
    widgetStore.navigateToHelp()
  }
}
</script>

<style scoped>
.nav-tab {
  @apply flex flex-col items-center justify-center gap-1 w-full px-0 py-4
         rounded-none shadow-none cursor-pointer transition-colors
         text-muted-foreground;
}
.nav-tab[data-state='active'] {
  @apply bg-transparent shadow-none text-primary;
}
</style>
