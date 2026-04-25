<template>
  <div v-if="!isSearchRoute" class="h-screen w-full flex flex-col">
    <!-- Table view: show one panel at a time (list OR detail) for full-width tables -->
    <template v-if="viewMode === 'table'">
      <!-- Back button when viewing a conversation -->
      <div
        v-if="hasConversationOpen"
        class="flex items-center gap-2 px-3 py-1.5 border-b bg-background shrink-0"
      >
        <Button variant="ghost" size="sm" class="h-8 gap-1.5" @click="goBack">
          <ArrowLeft class="h-4 w-4" />
          {{ t('conversation.list.backToList') }}
        </Button>
      </div>

      <!-- Full-width list (no conversation open) -->
      <div v-show="!hasConversationOpen" class="flex-1 overflow-y-auto">
        <ConversationList />
      </div>

      <!-- Full-width detail (conversation open) -->
      <div v-show="hasConversationOpen" class="flex-1 overflow-hidden">
        <router-view v-slot="{ Component }">
          <component :is="Component" />
        </router-view>
      </div>
    </template>

    <!-- Card view: original resizable split-panel layout -->
    <template v-else>
      <ResizablePanelGroup
        direction="horizontal"
        class="flex-1"
        @layout="onLayoutChange"
      >
        <ResizablePanel
          :default-size="panelSizes[0]"
          :min-size="20"
          :max-size="45"
        >
          <ConversationList />
        </ResizablePanel>

        <ResizableHandle />

        <ResizablePanel :default-size="panelSizes[1]" :min-size="30">
          <router-view v-slot="{ Component }">
            <component :is="Component" />
          </router-view>
        </ResizablePanel>
      </ResizablePanelGroup>
    </template>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useStorage } from '@vueuse/core'
import { ArrowLeft } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle
} from '@shared-ui/components/ui/resizable'
import ConversationList from '@/features/conversation/list/ConversationList.vue'
import { useViewMode } from '@/composables/useViewMode'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const { viewMode } = useViewMode()

const isSearchRoute = computed(() => route.name === 'search')
const hasConversationOpen = computed(() => !!route.params.uuid)

// Persist panel sizes: [conversationList, conversationDetail]
const panelSizes = useStorage('inboxPanelSizes', [25, 75])

const onLayoutChange = (sizes) => {
  panelSizes.value = sizes
}

const goBack = () => {
  // Prefer router history when available so we land where the user came from
  // (search results, list view, deep link). Fall back to the inbox if no history.
  if (window.history.state?.back) {
    router.back()
  } else {
    router.push({ name: 'inbox', params: { type: route.params.type || 'assigned' } })
  }
}
</script>
