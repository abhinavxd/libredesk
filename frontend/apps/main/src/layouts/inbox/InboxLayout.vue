<template>
  <template v-if="!isSearchRoute">
    <!-- Desktop: list and detail side by side in a resizable splitter. -->
    <ResizablePanelGroup
      v-if="!isMobile"
      direction="horizontal"
      class="h-screen w-full"
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

    <!-- Mobile: one pane at a time, list pushes to detail. The splitter's
         minimum sizes are percentages of the window, so at 390px they would
         yield a ~205px thread next to a ~117px sidebar; no class work fixes a
         percentage floor, hence a separate render path.

         Both panes stay mounted and are toggled with v-show: the router-view
         hosts InboxView, which owns the conversation-list fetching, and
         keeping the list mounted preserves its scroll position on the way
         back from a conversation. -->
    <div v-else class="h-screen w-full">
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
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useStorage } from '@vueuse/core'
import ConversationList from '@/features/conversation/list/ConversationList.vue'
import { useIsMobile } from '@main/composables/useIsMobile'
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle
} from '@shared-ui/components/ui/resizable'

defineOptions({ name: 'InboxLayout' })

const route = useRoute()
const isMobile = useIsMobile()
const isSearchRoute = computed(() => route.name === 'search')

// The three list routes; their `*-conversation` children are the detail routes.
const isListRoute = computed(() => ['inbox', 'team-inbox', 'view-inbox'].includes(route.name))

// Persist panel sizes: [conversationList, conversationDetail]. Only read by the
// desktop branch, so the stored desktop geometry never reaches the mobile one.
const panelSizes = useStorage('inboxPanelSizes', [25, 75])

const onLayoutChange = (sizes) => {
  panelSizes.value = sizes
}
</script>
