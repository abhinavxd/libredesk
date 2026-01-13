<template>
  <ResizablePanelGroup
    v-if="!isSearchRoute"
    direction="horizontal"
    class="h-screen w-full"
    @layout="onLayoutChange"
  >
    <!-- Conversation List Panel -->
    <ResizablePanel
      :default-size="panelSizes[0]"
      :min-size="25"
      :max-size="35"
      class="overflow-y-auto"
    >
      <ConversationList />
    </ResizablePanel>

    <ResizableHandle />

    <!-- Conversation Detail Panel -->
    <ResizablePanel :default-size="panelSizes[1]" :min-size="30">
      <router-view v-slot="{ Component }">
        <component :is="Component" />
      </router-view>
    </ResizablePanel>
  </ResizablePanelGroup>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useStorage } from '@vueuse/core'
import ConversationList from '@/features/conversation/list/ConversationList.vue'
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle
} from '@/components/ui/resizable'

const route = useRoute()
const isSearchRoute = computed(() => route.name === 'search')

// Persist panel sizes: [conversationList, conversationDetail]
const panelSizes = useStorage('inboxPanelSizes', [25, 75])

const onLayoutChange = (sizes) => {
  panelSizes.value = sizes
}
</script>
