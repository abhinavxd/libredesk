import { onMounted, watch } from 'vue'
import { useEventListener, useIntersectionObserver } from '@vueuse/core'

export function useConversationRead ({ root, marker, conversationUUID, message, loading, acknowledge }) {
  let pending = false
  let acknowledged = ''

  const acknowledgeVisible = async () => {
    const uuid = conversationUUID.value
    const latest = message.value
    if (pending || loading.value || !uuid || !latest?.uuid ||
      document.hidden || !document.hasFocus() || !root.value || !marker.value) return
    const key = `${uuid}:${latest.uuid}`
    if (key === acknowledged) return
    const viewport = root.value.getBoundingClientRect()
    const end = marker.value.getBoundingClientRect()
    if (viewport.height <= 0 || end.height <= 0 || end.top < viewport.top || end.bottom > viewport.bottom) return
    pending = true
    try {
      await acknowledge(uuid, latest.uuid)
      acknowledged = key
    } catch {
      return
    } finally {
      pending = false
    }
    if (message.value?.uuid !== latest.uuid || conversationUUID.value !== uuid) acknowledgeVisible()
  }

  watch([conversationUUID, message, loading], acknowledgeVisible, { flush: 'post' })
  useIntersectionObserver(marker, acknowledgeVisible, { root, threshold: 1 })
  useEventListener(window, 'focus', acknowledgeVisible)
  useEventListener(document, 'visibilitychange', acknowledgeVisible)
  useEventListener(root, 'scroll', acknowledgeVisible)
  onMounted(acknowledgeVisible)
}
