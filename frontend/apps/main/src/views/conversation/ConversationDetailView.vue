<template>
  <div class="relative h-full">
    <div
      v-if="isLoading"
      class="conv-progress absolute inset-x-0 top-0 h-0.5 z-50 pointer-events-none"
    />
    <div
      v-if="showContent"
      class="h-full transition-opacity duration-200"
      :class="{ 'opacity-60': isDimmed }"
      :inert="isDimmed"
    >
      <Conversation />
    </div>
  </div>
</template>

<script setup>
import { computed, watch, onMounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { useDocumentVisibility } from '@vueuse/core'
import { useConversationStore } from '@main/stores/conversation'
import Conversation from '@main/features/conversation/Conversation.vue'

const props = defineProps({
  uuid: String
})

const conversationStore = useConversationStore()
const route = useRoute()

const showContent = computed(
  () => Boolean(conversationStore.current?.uuid)
)

const isLoading = computed(
  () => conversationStore.conversation.loading || conversationStore.messages.loading
)

const isDimmed = computed(() => conversationStore.conversation.loading)

const visibility = useDocumentVisibility()
watch(visibility, (state) => {
  if (state === 'visible' && props.uuid) {
    conversationStore.resyncMail()
  }
})

const fetchConversation = async (uuid) => {
  const [, loaded] = await Promise.all([
    conversationStore.fetchConversation(uuid),
    conversationStore.fetchMessages(uuid)
  ])
  if (loaded && !document.hidden) await conversationStore.updateAssigneeLastSeen(uuid)
}

// Initial fetch
onMounted(() => {
  if (props.uuid) fetchConversation(props.uuid)
})

watch(
  () => props.uuid,
  (newUUID, oldUUID) => {
    if (!newUUID || newUUID === oldUUID) return
    const canTransition =
      oldUUID && !route.query.scrollTo && typeof document.startViewTransition === 'function'
    if (!canTransition) {
      fetchConversation(newUUID)
      return
    }
    const transition = document.startViewTransition(async () => {
      fetchConversation(newUUID)
      await nextTick()
    })
    transition.ready.catch(() => {})
    transition.finished.catch(() => {})
  }
)
</script>

<style scoped>
.conv-progress {
  background-color: hsl(var(--primary) / 0.4);
  animation: conv-progress-pulse 2.4s ease-in-out infinite;
}

@keyframes conv-progress-pulse {
  0%,
  100% {
    opacity: 0.4;
  }
  50% {
    opacity: 1;
  }
}
</style>
