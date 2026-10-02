import { ref, watch } from 'vue'

// A warning is advisory: composing remains possible, and each new remote reply
// can be reviewed without treating our own optimistic echo or older pages as new.
export function useReplyAwareness({ uuid, messageType, hasDraft, messages, ready, userID }) {
  const newerReply = ref(false)
  let key = ''
  let known = null
  let newestAt = 0
  const mail = () =>
    messages.value.filter(
      (message) =>
        ['incoming', 'outgoing'].includes(message.type) &&
        !message.private &&
        !message.uuid?.startsWith('pending-') &&
        message.status !== 'failed'
    )
  watch(
    [uuid, messageType, hasDraft, messages, ready],
    () => {
      const nextKey = `${uuid.value}::${messageType.value}`
      if (key !== nextKey || !hasDraft.value || messageType.value !== 'reply') {
        key = nextKey
        known = null
        newestAt = 0
        newerReply.value = false
      }
      if (!uuid.value || !ready.value || !hasDraft.value || messageType.value !== 'reply') return
      const current = mail()
      if (known) {
        if (
          current.some(
            (message) =>
              !known.has(message.uuid) &&
              new Date(message.created_at).getTime() >= newestAt &&
              message.sender_id !== userID.value
          )
        )
          newerReply.value = true
      }
      known = new Set([...(known || []), ...current.map((message) => message.uuid)])
      newestAt = Math.max(
        newestAt,
        0,
        ...current.map((message) => new Date(message.created_at).getTime() || 0)
      )
    },
    { immediate: true }
  )
  return {
    newerReply,
    acknowledgeReply: () => {
      newerReply.value = false
    }
  }
}
