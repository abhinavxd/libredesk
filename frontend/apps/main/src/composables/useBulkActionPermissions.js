import { computed } from 'vue'
import { useUserStore } from '@/stores/user'
import { permissions as p } from '@/constants/permissions'

export function useBulkActionPermissions () {
  const userStore = useUserStore()

  const canAssignAgent = computed(() => userStore.can(p.CONVERSATIONS_UPDATE_USER_ASSIGNEE))
  const canAssignTeam = computed(() => userStore.can(p.CONVERSATIONS_UPDATE_TEAM_ASSIGNEE))
  const canUpdateStatus = computed(() => userStore.can(p.CONVERSATIONS_UPDATE_STATUS))
  const canUpdateTags = computed(() => userStore.can(p.CONVERSATIONS_UPDATE_TAGS))

  const canBulkAct = computed(
    () => canAssignAgent.value || canAssignTeam.value || canUpdateStatus.value || canUpdateTags.value
  )

  return { canAssignAgent, canAssignTeam, canUpdateStatus, canUpdateTags, canBulkAct }
}
