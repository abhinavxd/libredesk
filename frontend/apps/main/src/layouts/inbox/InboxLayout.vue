<template>
  <ResizablePanelGroup
    v-if="!isSearchRoute"
    ref="panelGroupRef"
    direction="horizontal"
    class="h-full min-h-0 w-full"
    @layout="onLayoutChange"
  >
    <!-- Conversation List Panel -->
    <ResizablePanel
      v-if="!isMobile"
      :order="1"
      :default-size="panelSizes[0]"
      :min-size="listMinSize"
      :max-size="45"
      class="min-w-0 overflow-hidden rounded-xl border border-border bg-background"
    >
      <ConversationList />
    </ResizablePanel>

    <ResizableHandle
      v-if="!isMobile"
      class="w-1 bg-transparent after:w-1 can-hover:hover:after:bg-border"
      :aria-label="i18n.ngettext('Conversation', 'Conversations', 2)"
    />

    <!-- v-show, not v-if: the list keeps its scroll position. -->
    <ConversationList
      v-if="isMobile"
      v-show="isListRoute"
      class="h-full w-full overflow-hidden rounded-xl border border-border bg-background"
    />

    <!-- Conversation Detail Panel -->
    <ResizablePanel
      v-show="!isMobile || !isListRoute"
      :order="2"
      :min-size="isMobile ? 100 : 30"
      class="min-w-0"
    >
      <router-view v-slot="{ Component }">
        <keep-alive>
          <component :is="Component" />
        </keep-alive>
      </router-view>
    </ResizablePanel>
  </ResizablePanelGroup>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useElementSize, useStorage } from '@vueuse/core'
import ConversationList from '@/features/conversation/list/ConversationList.vue'
import { useIsMobile } from '@shared-ui/composables'
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle
} from '@shared-ui/components/ui/resizable'

defineOptions({ name: 'InboxLayout' })

const route = useRoute()
const isMobile = useIsMobile()
const isSearchRoute = computed(() => route.name === 'search')
const panelGroupRef = ref(null)
const { width: panelGroupWidth } = useElementSize(panelGroupRef)
const listMinSize = computed(() =>
  panelGroupWidth.value > 0 ? Math.min(45, (240 / panelGroupWidth.value) * 100) : 20
)

// Every detail route is its list route's name plus `-conversation`.
const isListRoute = computed(() => !String(route.name).endsWith('-conversation'))

// [conversationList, conversationDetail]
const panelSizes = useStorage('inboxPanelSizes', [25, 75])

const onLayoutChange = (sizes) => {
  if (sizes.length === 2) panelSizes.value = sizes
}
</script>
