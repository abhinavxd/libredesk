<template>
  <div class="relative">
    <!-- Button to toggle sidebar when closed -->
    <button
      v-if="!conversationSidebarOpen"
      @click="toggleSidebar"
      class="absolute right-0 top-16 p-2 rounded-l-full bg-background text-primary hover:bg-opacity-90 transition-all duration-200 border shadow hover:scale-105 z-50"
    >
      <ChevronLeft size="16" />
    </button>

    <!-- Sidebar container -->
    <div
      class="h-screen border-l transition-all duration-300 ease-in-out"
      :class="conversationSidebarOpen ? 'w-[16rem] 2xl:w-[20rem]' : 'w-0 border-0'"
    >
      <div
        class="h-full overflow-y-auto overflow-x-hidden transition-opacity ease-in-out"
        :class="[
          conversationSidebarOpen
            ? 'opacity-100 duration-300'
            : 'opacity-0 duration-1000'
        ]"
      >
        <ConversationSideBar />
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted } from 'vue'
import { ChevronLeft } from 'lucide-vue-next'
import ConversationSideBar from './ConversationSideBar.vue'
import { useEmitter } from '@/composables/useEmitter'
import { useStorage } from '@vueuse/core'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'

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
