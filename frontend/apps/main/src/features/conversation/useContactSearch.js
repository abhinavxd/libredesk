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
  let searchSequence = 0

  const clearSearchResults = () => {
    clearTimeout(timeoutId)
    searchSequence++
    searchResults.value.splice(0)
    highlightedIndex.value = -1
  }

  onUnmounted(clearSearchResults)

  const handleSearchContacts = () => {
    clearTimeout(timeoutId)
    const sequence = ++searchSequence
    timeoutId = setTimeout(async () => {
      const query = getQuery().trim()
      if (query.length < 3) {
        searchResults.value.splice(0)
        return
      }
      try {
        const resp = await api.searchContacts({ query })
        if (sequence !== searchSequence) return
        const results = resp.data.data
        searchResults.value = filterResults ? results.filter(filterResults) : [...results]
        highlightedIndex.value = -1
      } catch (error) {
        if (sequence !== searchSequence) return
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
      e.stopPropagation()
      clearSearchResults()
    }
  }

  const selectContact = (contact) => {
    onSelect(contact)
    clearSearchResults()
  }

  return {
    searchResults,
    highlightedIndex,
    handleSearchContacts,
    handleSearchKeydown,
    selectContact,
    clearSearchResults
  }
}
