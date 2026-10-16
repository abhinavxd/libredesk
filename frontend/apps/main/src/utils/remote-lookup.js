import { ref, computed } from 'vue'

const PAGE_SIZE = 50
const CACHE_TTL_MS = 5 * 60 * 1000
const BATCH_DELAY_MS = 50
const MAX_IDS_PER_REQUEST = 200

// Backs a store whose list lives on the server, ids outside the current page are pinned so their labels still render.
export function createRemoteLookup ({ fetchRows, onError }) {
  const byID = ref({})
  const firstPageIDs = ref([])
  const pinnedIDs = ref([])
  const fetchedAt = new Map()
  const pendingIDs = new Map()
  const queuedIDs = new Set()
  let scheduledBatch = null
  let firstPage = null
  let firstPageFetchedAt = null

  const rows = computed(() => {
    const seen = new Set()
    const out = []
    for (const id of [...firstPageIDs.value, ...pinnedIDs.value]) {
      if (seen.has(id)) continue
      seen.add(id)
      const row = byID.value[id]
      if (row) out.push(row)
    }
    return out
  })

  const merge = (fetched) => {
    const now = Date.now()
    for (const row of fetched) {
      byID.value[row.id] = row
      fetchedAt.set(Number(row.id), now)
    }
    return fetched
  }

  const load = async (params) => {
    const response = await fetchRows({ page_size: PAGE_SIZE, ...params })
    return merge(response?.data?.data || [])
  }

  const fetchFirstPage = () => {
    if (firstPage) return firstPage
    if (firstPageFetchedAt !== null && Date.now() - firstPageFetchedAt < CACHE_TTL_MS) {
      return Promise.resolve(firstPageIDs.value.map((id) => byID.value[id]).filter(Boolean))
    }
    firstPage = load({ page: 1 })
      .then((fetched) => {
        firstPageIDs.value = fetched.map((row) => row.id)
        firstPageFetchedAt = Date.now()
        return fetched
      })
      .catch((error) => {
        onError(error)
      })
      .finally(() => { firstPage = null })
    return firstPage
  }

  const search = async (query) => {
    const term = (query || '').trim()
    if (!term) {
      await fetchFirstPage()
      return firstPageIDs.value.map((id) => byID.value[id]).filter(Boolean)
    }
    try {
      return await load({ q: term, page: 1 })
    } catch (error) {
      onError(error)
      return []
    }
  }

  const fetchIDs = async (ids) => {
    try {
      const fetched = await load({ ids: ids.join(',') })
      const returned = new Set(fetched.map((row) => Number(row.id)))
      const now = Date.now()
      for (const id of ids) {
        fetchedAt.set(id, now)
        if (!returned.has(id)) delete byID.value[id]
      }
    } catch (error) {
      onError(error)
    } finally {
      for (const id of ids) pendingIDs.delete(id)
    }
  }

  const queueIDs = (ids) => {
    for (const id of ids) queuedIDs.add(id)
    if (!scheduledBatch) {
      scheduledBatch = new Promise((resolve) => setTimeout(resolve, BATCH_DELAY_MS))
        .then(async () => {
          const batch = [...queuedIDs]
          queuedIDs.clear()
          scheduledBatch = null
          const requests = []
          for (let i = 0; i < batch.length; i += MAX_IDS_PER_REQUEST) {
            requests.push(fetchIDs(batch.slice(i, i + MAX_IDS_PER_REQUEST)))
          }
          await Promise.all(requests)
        })
    }
    for (const id of ids) pendingIDs.set(id, scheduledBatch)
  }

  const ensureIDs = async (ids) => {
    const wanted = [...new Set((ids || []).map(Number).filter((id) => Number.isInteger(id) && id > 0))]
    if (!wanted.length) return
    pinnedIDs.value = [...new Set([...pinnedIDs.value, ...wanted])]
    if (wanted.some((id) => !fetchedAt.has(id) && !pendingIDs.has(id))) await fetchFirstPage()
    const missing = wanted.filter((id) => (!fetchedAt.has(id) || Date.now() - fetchedAt.get(id) >= CACHE_TTL_MS) && !pendingIDs.has(id))
    if (missing.length) queueIDs(missing)
    await Promise.all(wanted.map((id) => pendingIDs.get(id)).filter(Boolean))
  }

  const patch = (id, changes) => {
    const row = byID.value[id]
    if (row) byID.value[id] = { ...row, ...changes }
  }

  return { rows, fetchFirstPage, search, ensureIDs, patch }
}
