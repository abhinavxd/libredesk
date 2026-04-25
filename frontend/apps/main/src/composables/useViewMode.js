import { useStorage } from '@vueuse/core'

// Persisted across reload + synced across tabs via the storage event.
const viewMode = useStorage('conversationViewMode', 'card')

export function useViewMode () {
  function setViewMode (mode) {
    if (mode !== 'card' && mode !== 'table') return
    viewMode.value = mode
  }
  return { viewMode, setViewMode }
}
