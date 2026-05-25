<template>
  <ConversationPlaceholder v-if="['inbox', 'team-inbox', 'view-inbox'].includes(route.name)" />
  <router-view />
</template>

<script setup>
import { computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useIntervalFn, useDocumentVisibility } from '@vueuse/core'
import { useConversationStore } from '../../stores/conversation'
import { CONVERSATION_LIST_TYPE, CONVERSATION_DEFAULT_STATUSES } from '../../constants/conversation'
import ConversationPlaceholder from '@/features/conversation/ConversationPlaceholder.vue'

const route = useRoute()
const type = computed(() => route.params.type)
const teamID = computed(() => route.params.teamID)
const viewID = computed(() => route.params.viewID)

const conversationStore = useConversationStore()

let lastFetchedKey = ''

const fetchForCurrentRoute = () => {
  if (!type.value && !teamID.value && !viewID.value) return

  const key = `${type.value || ''}|${teamID.value || ''}|${viewID.value || ''}`
  if (key === lastFetchedKey) return
  lastFetchedKey = key

  if (viewID.value) {
    conversationStore.setListStatus('', false)
    conversationStore.fetchConversationsList(true, CONVERSATION_LIST_TYPE.VIEW, 0, [], viewID.value)
    return
  }

  if (!conversationStore.getListStatus) {
    conversationStore.setListStatus(CONVERSATION_DEFAULT_STATUSES.OPEN, false)
  }
  if (type.value) {
    conversationStore.fetchConversationsList(true, type.value)
  } else {
    conversationStore.fetchConversationsList(true, CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED, teamID.value)
  }
}

onMounted(fetchForCurrentRoute)

const visibility = useDocumentVisibility()
const { pause, resume } = useIntervalFn(
  () => conversationStore.refreshConversationList(),
  120000
)
watch(visibility, v => {
  if (v === 'visible') {
    conversationStore.refreshConversationList()
    resume()
  } else {
    pause()
  }
})

watch([type, teamID, viewID], fetchForCurrentRoute)
</script>
