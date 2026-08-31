<template>
  <ResizablePanelGroup
    v-if="!isSearchRoute && !isMobile"
    direction="horizontal"
    class="h-full w-full min-h-0"
    @layout="onLayoutChange"
  >
    <!-- Conversation List Panel -->
    <ResizablePanel :default-size="panelSizes[0]" :min-size="20" :max-size="45">
      <ConversationList />
    </ResizablePanel>

    <ResizableHandle />

    <!-- Conversation Detail Panel -->
    <ResizablePanel :default-size="panelSizes[1]" :min-size="30">
      <router-view v-slot="{ Component }">
        <keep-alive>
          <component :is="Component" />
        </keep-alive>
      </router-view>
    </ResizablePanel>
  </ResizablePanelGroup>

  <!-- v-show, not v-if: the list keeps its scroll position. -->
  <div v-else-if="!isSearchRoute" class="h-full w-full min-h-0">
    <ConversationList v-show="isListRoute" />
    <div v-show="!isListRoute" class="h-full">
      <router-view v-slot="{ Component }">
        <keep-alive>
          <component :is="Component" />
        </keep-alive>
      </router-view>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useStorage } from '@vueuse/core'
import { useIsMobileLayout } from '@main/composables/useIsMobileLayout'
import ConversationList from '@/features/conversation/list/ConversationList.vue'
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle
} from '@shared-ui/components/ui/resizable'

defineOptions({ name: 'InboxLayout' })

const route = useRoute()
const isMobile = useIsMobileLayout()
const isSearchRoute = computed(() => route.name === 'search')
const isListRoute = computed(() => !String(route.name).endsWith('-conversation'))

const panelSizes = useStorage('inboxPanelSizes', [25, 75])

const onLayoutChange = (sizes) => {
  panelSizes.value = sizes
}
</script>
