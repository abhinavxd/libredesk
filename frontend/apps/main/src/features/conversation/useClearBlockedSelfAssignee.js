import { computed, watch } from 'vue'
import { useUserStore } from '@main/stores/user'
import { useAppSettingsStore } from '@main/stores/appSettings'

// Clears the assignee whenever it is the current user and the server would reject them as away and reassigning.
export const useClearBlockedSelfAssignee = (getAgentId, clearAgentId) => {
  const userStore = useUserStore()
  const appSettingsStore = useAppSettingsStore()

  const isSelfBlocked = computed(
    () =>
      appSettingsStore.settings['app.block_assign_away_reassigning'] === true &&
      userStore.user.availability_status === 'away_and_reassigning'
  )

  watch(
    [isSelfBlocked, getAgentId],
    ([blocked, agentId]) => {
      if (blocked && userStore.userID && String(agentId) === String(userStore.userID)) clearAgentId()
    },
    { immediate: true }
  )
}
