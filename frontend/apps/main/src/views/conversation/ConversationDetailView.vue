<template>
  <div class="relative h-full">
    <ResizablePanelGroup
      v-if="showContent"
      direction="horizontal"
      class="h-full"
      @layout="onLayoutChange"
    >
      <!-- Conversation Content Panel -->
      <ResizablePanel :default-size="sidebarOpen ? panelSizes[0] : 100" :min-size="40">
        <Conversation />
      </ResizablePanel>

      <!-- Resizable Handle -->
      <ResizableHandle />

      <!-- Sidebar Panel (collapsible) -->
      <ResizablePanel
        ref="sidebarPanelRef"
        class="transition-all duration-300 ease-in-out"
        :default-size="panelSizes[1]"
        :min-size="15"
        :max-size="40"
        :collapsible="true"
        :collapsed-size="0"
        @collapse="onSidebarCollapse"
        @expand="onSidebarExpand"
      >
        <div class="h-full overflow-y-auto overflow-x-hidden">
          <ConversationSideBar />
        </div>
      </ResizablePanel>
    </ResizablePanelGroup>

    <!-- Toggle button when sidebar is collapsed -->
    <button
      v-if="showContent && !sidebarOpen"
      @click="toggleSidebar"
      class="absolute right-0 top-16 p-2 rounded-l-full bg-sidebar text-sidebar-foreground hover:bg-opacity-90 transition-all duration-200 border shadow hover:scale-105 z-50"
    >
      <ChevronLeft size="16" />
    </button>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useStorage } from '@vueuse/core'
import { ChevronLeft } from 'lucide-vue-next'
import { useConversationStore } from '@main/stores/conversation'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import Conversation from '@main/features/conversation/Conversation.vue'
import ConversationSideBar from '@main/features/conversation/sidebar/ConversationSideBar.vue'
import { ResizablePanelGroup, ResizablePanel, ResizableHandle } from '@shared-ui/components/ui/resizable'

const props = defineProps({
  uuid: String
})

const conversationStore = useConversationStore()
const emitter = useEmitter()
const sidebarPanelRef = ref(null)
const sidebarOpen = useStorage('conversationSidebarOpen', true)
const panelSizes = useStorage('conversationDetailPanelSizes', [70, 30])

const showContent = computed(
  () => conversationStore.current || conversationStore.conversation.loading
)

const toggleSidebar = () => {
  if (sidebarOpen.value) {
    sidebarPanelRef.value?.collapse()
  } else {
    sidebarPanelRef.value?.expand()
  }
}

const onSidebarCollapse = () => {
  sidebarOpen.value = false
}

const onSidebarExpand = () => {
  sidebarOpen.value = true
}

const onLayoutChange = (sizes) => {
  if (sidebarOpen.value && sizes.length === 2) {
    panelSizes.value = sizes
  }
}

// Listen to emitter events for toggle (from sidebar contact)
onMounted(() => {
  emitter.on(EMITTER_EVENTS.CONVERSATION_SIDEBAR_TOGGLE, toggleSidebar)

  // Sync initial collapsed state from localStorage
  nextTick(() => {
    if (!sidebarOpen.value && sidebarPanelRef.value) {
      sidebarPanelRef.value.collapse()
    }
  })
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.CONVERSATION_SIDEBAR_TOGGLE)
})

const fetchConversation = async (uuid) => {
  await Promise.all([
    conversationStore.fetchConversation(uuid),
    conversationStore.fetchMessages(uuid),
    conversationStore.fetchParticipants(uuid)
  ])
  await conversationStore.updateAssigneeLastSeen(uuid)
}

// Initial fetch
onMounted(() => {
  if (props.uuid) fetchConversation(props.uuid)
})

// Watcher for UUID changes
watch(
  () => props.uuid,
  (newUUID, oldUUID) => {
    if (newUUID && newUUID !== oldUUID) {
      fetchConversation(newUUID)
    }
  }
)
</script>
