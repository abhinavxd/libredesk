

import { ref } from 'vue'
import { useI18n } from '@shared-ui/i18n'
import { useConversationStore } from '@/stores/conversation'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { TAG_ACTION } from '@/constants/conversation'
import api from '@/api'

const bulkLoading = ref(false)

export function useBulkActions() {
  const conversationStore = useConversationStore()
  const emitter = useEmitter()
  const i18n = useI18n()

  const runBulkAction = async (actionFn) => {
    const uuids = [...conversationStore.selectedUUIDs]
    bulkLoading.value = true
    const results = await Promise.allSettled(uuids.map((uuid) => actionFn(uuid)))
    bulkLoading.value = false

    const hasFailures = results.some((r) => r.status === 'rejected')

    conversationStore.clearSelection()
    conversationStore.fetchFirstPageConversations()
    conversationStore.fetchSidebarCounts({ force: true })

    if (hasFailures) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        title: i18n._('{count, plural, one {Error} other {Errors}}', 1),
        description: i18n._('Some conversations couldn\'\'t be updated')
      })
    } else {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        description: i18n._('Conversations updated')
      })
    }
  }

  const bulkAssign = (assigneeType, assigneeValue) => {
    if (assigneeValue === 'none') {
      return runBulkAction((uuid) => api.removeAssignee(uuid, assigneeType))
    }
    const assigneeId = parseInt(assigneeValue, 10)
    return runBulkAction((uuid) =>
      api.updateAssignee(uuid, assigneeType, { assignee_id: assigneeId })
    )
  }

  const bulkAddTag = (tag) =>
    runBulkAction((uuid) => conversationStore.updateConversationTags(uuid, TAG_ACTION.ADD, [tag]))

  const bulkUpdateStatus = (status) =>
    runBulkAction((uuid) => api.updateConversationStatus(uuid, { status }))

  return { bulkLoading, runBulkAction, bulkAssign, bulkAddTag, bulkUpdateStatus }
}
