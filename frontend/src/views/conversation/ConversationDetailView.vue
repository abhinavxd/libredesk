<template>
  <div class="flex">
    <Conversation
      class="flex-auto min-w-[300px]"
      v-if="conversationStore.current || conversationStore.conversation.loading"
    />
    <ConversationSideBarWrapper
      class="flex-none"
      v-if="conversationStore.current || conversationStore.conversation.loading"
    />
  </div>
</template>

<script setup>
import { watch, onMounted } from 'vue'
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
