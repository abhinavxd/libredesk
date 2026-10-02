<template>
  <ResizablePanelGroup
    v-if="!isSearchRoute && !isMobile && !isListRoute"
    direction="horizontal"
    class="h-full w-full"
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

  <!-- A list is the primary desktop workspace until a conversation is opened. -->
  <ConversationList v-else-if="!isSearchRoute && !isMobile && addressID" />
  <ConversationPlaceholder v-else-if="!isSearchRoute && !isMobile" />

  <!-- v-show, not v-if: the list keeps its scroll position. -->
  <div v-else-if="!isSearchRoute" class="h-full w-full">
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
import { computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useDocumentVisibility, useIntervalFn, useStorage } from '@vueuse/core'
import ConversationList from '@/features/conversation/list/ConversationList.vue'
import ConversationPlaceholder from '@/features/conversation/ConversationPlaceholder.vue'
import { useIsMobile } from '@shared-ui/composables'
import { useAddressStore } from '@main/stores/address'
import { useConversationStore } from '@main/stores/conversation'
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle
} from '@shared-ui/components/ui/resizable'

defineOptions({ name: 'InboxLayout' })

const route = useRoute()
const router = useRouter()
const isMobile = useIsMobile()
const addressStore = useAddressStore()
const conversationStore = useConversationStore()
const isSearchRoute = computed(() => route.name === 'search')
const addressID = computed(() => route.params.addressID)

// Every detail route is its list route's name plus `-conversation`.
const isListRoute = computed(() => !String(route.name).endsWith('-conversation'))

// [conversationList, conversationDetail]
const panelSizes = useStorage('inboxPanelSizes', [25, 75])

const onLayoutChange = (sizes) => {
  panelSizes.value = sizes
}

let lastFetchedAddressID = ''

const hasCurrentList = () =>
  conversationStore.conversations.initialized &&
  conversationStore.conversations.addressID === Number(addressID.value)

async function fetchForCurrentRoute() {
  await addressStore.fetchAddresses()
  if (!addressID.value) {
    const first = addressStore.addresses[0]
    if (first) router.replace({ name: 'address-inbox', params: { addressID: first.id } })
    return
  }

  if (!addressStore.get(addressID.value)) {
    router.replace({ name: 'address-inbox' })
    return
  }

  if (String(addressID.value) === lastFetchedAddressID && hasCurrentList()) {
    conversationStore.refreshConversationList()
    return
  }
  lastFetchedAddressID = String(addressID.value)
  conversationStore.fetchConversationsList(true, Number(addressID.value))
}

onMounted(fetchForCurrentRoute)
watch(addressID, fetchForCurrentRoute)

const visibility = useDocumentVisibility()
const { pause, resume } = useIntervalFn(() => conversationStore.refreshConversationList(), 120000)
watch(visibility, (value) => {
  if (value === 'visible') {
    conversationStore.refreshConversationList()
    resume()
  } else {
    pause()
  }
})
</script>
