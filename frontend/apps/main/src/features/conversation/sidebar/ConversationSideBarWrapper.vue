<template>
  <div class="h-screen">
    <SidebarProvider :style="{ '--sidebar-width': '20rem' }" :open="conversationSidebarOpen">
      <Sidebar side="right" collapsible="offcanvas" variant="sidebar" :collapseOnMobile="false">
        <SidebarContent>
          <SidebarGroup style="padding: 0">
            <SidebarMenu>
              <SidebarMenuItem>
                <ConversationSideBar />
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>
        <button
          v-if="!conversationSidebarOpen"
          @click="toggleSidebar"
          class="absolute right-full top-16 p-2 rounded-l-full bg-background text-primary hover:bg-opacity-90 transition-all duration-200 border shadow hover:scale-105"
        >
          <ChevronLeft size="16"/>
        </button>
      </Sidebar>
    </SidebarProvider>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted } from 'vue'
import { ChevronLeft } from 'lucide-vue-next'
import ConversationSideBar from './ConversationSideBar.vue'
import { useEmitter } from '../../../composables/useEmitter'
import { useStorage } from '@vueuse/core'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarMenu,
  SidebarMenuItem,
  SidebarProvider
} from '@shared-ui/components/ui/sidebar'

const emitter = useEmitter()
const conversationSidebarOpen = useStorage('conversationSidebarOpen', true)

const toggleSidebar = () => {
  conversationSidebarOpen.value = !conversationSidebarOpen.value
}

onMounted(() => {
  emitter.on(EMITTER_EVENTS.CONVERSATION_SIDEBAR_TOGGLE, toggleSidebar)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.CONVERSATION_SIDEBAR_TOGGLE)
})
</script>
