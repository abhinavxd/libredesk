import { ref, onUnmounted } from 'vue'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@/api'

export function useContactSearch({ getQuery, filterResults, onSelect }) {
  const emitter = useEmitter()
  const searchResults = ref([])
  const highlightedIndex = ref(-1)
  let timeoutId = null

  onUnmounted(() => clearTimeout(timeoutId))

  const handleSearchContacts = () => {
    clearTimeout(timeoutId)
    timeoutId = setTimeout(async () => {
      const query = getQuery().trim()
      if (query.length < 3) {
        searchResults.value.splice(0)
        return
      }
      try {
        const resp = await api.searchContacts({ query })
        const results = resp.data.data
        searchResults.value = filterResults ? results.filter(filterResults) : [...results]
        highlightedIndex.value = -1
      } catch (error) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
        searchResults.value.splice(0)
      }
    }, 300)
  }

  const handleSearchKeydown = (e) => {
    if (!searchResults.value.length) return
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      highlightedIndex.value = Math.min(highlightedIndex.value + 1, searchResults.value.length - 1)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      highlightedIndex.value = Math.max(highlightedIndex.value - 1, 0)
    } else if (e.key === 'Enter' && highlightedIndex.value >= 0) {
      e.preventDefault()
      selectContact(searchResults.value[highlightedIndex.value])
    } else if (e.key === 'Escape') {
      searchResults.value.splice(0)
      highlightedIndex.value = -1
    }
  }

  const selectContact = (contact) => {
    onSelect(contact)
    searchResults.value.splice(0)
    highlightedIndex.value = -1
  }

  return { searchResults, highlightedIndex, handleSearchContacts, handleSearchKeydown, selectContact }
}
