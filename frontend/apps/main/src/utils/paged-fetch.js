const PAGE_SIZE = 200

// Resolves once the first page is applied; the remaining pages stream in the background.
export async function fetchAllPages (fetchPage, onRows, onError, pageSize = PAGE_SIZE) {
  const count = await loadPage(fetchPage, 1, pageSize, onRows)
  if (count < pageSize) return
  ;(async () => {
    for (let page = 2; ; page++) {
      const batch = await loadPage(fetchPage, page, pageSize, onRows)
      if (batch < pageSize) return
    }
  })().catch(onError)
}

async function loadPage (fetchPage, page, pageSize, onRows) {
  const response = await fetchPage({ page, page_size: pageSize })
  const rows = response?.data?.data || []
  if (rows.length) onRows(rows)
  return rows.length
}
