<template>
  <ConversationPlaceholder v-if="['inbox', 'team-inbox', 'view-inbox'].includes(route.name)" />
  <router-view />
</template>

<script setup>
import { computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useConversationStore } from '../../stores/conversation'
import { CONVERSATION_LIST_TYPE, CONVERSATION_DEFAULT_STATUSES } from '../../constants/conversation'
import ConversationPlaceholder from '@/features/conversation/ConversationPlaceholder.vue'

const route = useRoute()
const type = computed(() => route.params.type)
const teamID = computed(() => route.params.teamID)
const viewID = computed(() => route.params.viewID)

const conversationStore = useConversationStore()

// Init conversations list based on route params
onMounted(() => {
  // Fetch list based on type
  if (type.value) {
    // Set list status if not already set
    if (!conversationStore.getListStatus) {
      conversationStore.setListStatus(CONVERSATION_DEFAULT_STATUSES.OPEN, false)
    }
    conversationStore.fetchConversationsList(true, type.value)
  }
  // Fetch team list.
  if (teamID.value) {
    // Set list status if not already set
    if (!conversationStore.getListStatus) {
      conversationStore.setListStatus(CONVERSATION_DEFAULT_STATUSES.OPEN, false)
    }
    conversationStore.fetchConversationsList(
      true,
      CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED,
      teamID.value
    )
  }
  // Fetch view list.
  if (viewID.value) {
    // Empty out list status as views are already filtered.
    conversationStore.setListStatus('', false)
    conversationStore.fetchConversationsList(true, CONVERSATION_LIST_TYPE.VIEW, 0, [], viewID.value)
  }
})

// Refetch when route params change
watch(
  [type, teamID, viewID],
  ([newType, newTeamID, newViewID], [oldType, oldTeamID, oldViewID]) => {
    if (newType !== oldType && newType) {
      // Set list status if not already set
      if (!conversationStore.getListStatus) {
        conversationStore.setListStatus(CONVERSATION_DEFAULT_STATUSES.OPEN, false)
      }
      conversationStore.fetchConversationsList(true, newType)
    }
    if (newTeamID !== oldTeamID && newTeamID) {
      // Set list status if not already set
      if (!conversationStore.getListStatus) {
        conversationStore.setListStatus(CONVERSATION_DEFAULT_STATUSES.OPEN, false)
      }
      conversationStore.fetchConversationsList(
        true,
        CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED,
        newTeamID
      )
    }
    if (newViewID !== oldViewID && newViewID) {
      // Empty out list status as views are already filtered.
      conversationStore.setListStatus('', false)
      conversationStore.fetchConversationsList(true, CONVERSATION_LIST_TYPE.VIEW, 0, [], newViewID)
    }
  }
)
</script>
