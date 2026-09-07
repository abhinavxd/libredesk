import { ref } from 'vue'
import { useTimeoutFn } from '@vueuse/core'

export function useRemoteSearch (search, delay) {
  const results = ref(null)
  const searching = ref(false)
  let version = 0
  let disposed = false
  let pendingSearch = null

  const runSearch = async () => {
    const { query, currentVersion } = pendingSearch
    try {
      const found = await search(query)
      if (!disposed && currentVersion === version) results.value = found || []
    } catch {
      if (!disposed && currentVersion === version) results.value = []
    } finally {
      if (!disposed && currentVersion === version) searching.value = false
    }
  }

  const { start: startSearch, stop: stopSearch } = useTimeoutFn(runSearch, delay, {
    immediate: false
  })

  const update = (term) => {
    const query = (term || '').trim()
    const currentVersion = ++version
    stopSearch()
    if (!query) {
      results.value = null
      searching.value = false
      return
    }
    searching.value = true
    pendingSearch = { query, currentVersion }
    startSearch()
  }

  const dispose = () => {
    disposed = true
    version++
    stopSearch()
    searching.value = false
  }

  return {
    results,
    searching,
    update,
    dispose
  }
}
