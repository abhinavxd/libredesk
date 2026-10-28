<template>
  <div ref="detailContainerRef" class="relative h-full min-h-0">
    <div
      v-if="isLoading"
      class="conv-progress absolute inset-x-0 top-0 h-0.5 z-50 pointer-events-none"
    />
    <ResizablePanelGroup
      v-if="showContent && panelGroupWidth > 0"
      direction="horizontal"
      class="h-full min-h-0 transition-opacity duration-200"
      :class="{ 'opacity-60': isDimmed }"
      :inert="isDimmed"
      @layout="onLayoutChange"
    >
      <!-- Conversation Content Panel -->
      <ResizablePanel
        :order="1"
        :min-size="55"
        class="min-w-0 overflow-hidden rounded-xl border border-border bg-card"
      >
        <Conversation :details-open="useDetailsDrawer ? sheetSidebarOpen : sidebarOpen" />
      </ResizablePanel>

      <!-- Resizable Handle -->
      <ResizableHandle
        v-if="!useDetailsDrawer"
        v-show="sidebarOpen"
        class="w-1 bg-transparent after:w-1 can-hover:hover:after:bg-border"
        :aria-label="$t('copilot.details')"
        @dragging="resizingSidebar = $event"
        @keydown.capture="onSidebarResizeKey"
        @keyup="resizingSidebar = false"
        @blur="resizingSidebar = false"
      />

      <!-- Sidebar Panel (collapsible) -->
      <ResizablePanel
        v-if="!useDetailsDrawer"
        :order="2"
        ref="sidebarPanelRef"
        size-unit="px"
        :default-size="sidebarOpen ? sidebarWidth : 0"
        :min-size="240"
        :collapsible="true"
        :collapsed-size="0"
        class="min-w-0 overflow-hidden rounded-xl border border-border bg-background data-[state=collapsed]:border-0"
        @collapse="onSidebarCollapse"
        @expand="onSidebarExpand"
      >
        <ConversationSideBar />
      </ResizablePanel>
    </ResizablePanelGroup>

    <Sheet
      v-if="showContent && useDetailsDrawer"
      :open="sheetSidebarOpen"
      @update:open="sheetSidebarOpen = $event"
    >
      <SheetContent
        side="right"
        class="w-[90vw] max-w-sm overflow-hidden p-0 [&>button]:hidden"
        :aria-describedby="undefined"
      >
        <SheetTitle class="sr-only">{{ $t('globals.terms.contact') }}</SheetTitle>
        <ConversationSideBar />
      </SheetContent>
    </Sheet>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useElementSize, useMediaQuery, useStorage } from '@vueuse/core'
import { useConversationStore } from '@main/stores/conversation'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import Conversation from '@main/features/conversation/Conversation.vue'
import ConversationSideBar from '@main/features/conversation/sidebar/ConversationSideBar.vue'
import { Sheet, SheetContent, SheetTitle } from '@shared-ui/components/ui/sheet'
import { ResizablePanelGroup, ResizablePanel, ResizableHandle } from '@shared-ui/components/ui/resizable'

const props = defineProps({
  uuid: String
})

const conversationStore = useConversationStore()
const emitter = useEmitter()
const compactViewport = useMediaQuery('(max-width: 1279px)')
const detailContainerRef = ref(null)
const { width: panelGroupWidth } = useElementSize(detailContainerRef)
const useDetailsDrawer = computed(
  () => compactViewport.value || (panelGroupWidth.value > 0 && panelGroupWidth.value < 640)
)
const sidebarPanelRef = ref(null)
const sidebarOpen = useStorage('conversationSidebarOpen', true)
const sidebarWidth = useStorage('conversationSidebarWidth', 320)
const resizingSidebar = ref(false)
const sheetSidebarOpen = ref(false)

watch(useDetailsDrawer, () => {
  sheetSidebarOpen.value = false
})

const showContent = computed(
  () => conversationStore.current || conversationStore.conversation.loading
)

const isLoading = computed(
  () => conversationStore.conversation.loading || conversationStore.messages.loading
)

const isDimmed = computed(() => conversationStore.conversation.loading)

const toggleSidebar = () => {
  if (useDetailsDrawer.value) {
    sheetSidebarOpen.value = !sheetSidebarOpen.value
    return
  }
  if (sidebarOpen.value) {
    sidebarPanelRef.value?.collapse()
  } else {
    sidebarPanelRef.value?.resize(sidebarWidth.value)
  }
}

const onSidebarCollapse = () => {
  sidebarOpen.value = false
}

const onSidebarExpand = () => {
  sidebarOpen.value = true
}

const onLayoutChange = (sizes) => {
  if (sizes.length === 2 && sizes[1] > 0) {
    if (resizingSidebar.value) {
      sidebarWidth.value = sizes[1]
    } else if (sidebarOpen.value) {
      sidebarPanelRef.value?.resize(sidebarWidth.value)
    }
  }
}

const onSidebarResizeKey = (event) => {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
  resizingSidebar.value = true
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
  emitter.off(EMITTER_EVENTS.CONVERSATION_SIDEBAR_TOGGLE, toggleSidebar)
})

const fetchConversation = async (uuid) => {
  await Promise.all([
    conversationStore.fetchConversation(uuid),
    conversationStore.fetchMessages(uuid)
  ])
}

// Initial fetch
onMounted(() => {
  if (props.uuid) fetchConversation(props.uuid)
})

watch(
  () => props.uuid,
  (newUUID, oldUUID) => {
    sheetSidebarOpen.value = false
    if (!newUUID || newUUID === oldUUID) return
    fetchConversation(newUUID)
  }
)
</script>

<style scoped>
.conv-progress {
  background-color: hsl(var(--primary) / 0.4);
  animation: conv-progress-pulse 2.4s ease-in-out infinite;
}

@keyframes conv-progress-pulse {
  0%, 100% { opacity: 0.4; }
  50% { opacity: 1; }
}
</style>
