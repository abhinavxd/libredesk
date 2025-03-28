<template>
  <div class="flex">
    <div class="grow min-w-[300px]">
      <Conversation v-if="conversationStore.current || conversationStore.conversation.loading" />
    </div>
    <div>
      <ConversationSideBarWrapper
        v-if="conversationStore.current || conversationStore.conversation.loading"
      />
    </div>
  </div>
</template>

<script setup>
import { watch, onMounted, onUnmounted } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import Conversation from '@/features/conversation/Conversation.vue'
import ConversationSideBarWrapper from '@/features/conversation/sidebar/ConversationSideBarWrapper.vue'

const props = defineProps({
  uuid: String
})

const conversationStore = useConversationStore()

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

onUnmounted(() => {
  conversationStore.resetCurrentConversation()
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
