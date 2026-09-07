import { beforeEach, describe, expect, test, vi } from 'vitest'

const searchContacts = vi.fn()
const emit = vi.fn()

vi.mock('vue', async (importOriginal) => ({
  ...(await importOriginal()),
  onUnmounted: vi.fn()
}))
vi.mock('@/api', () => ({ default: { searchContacts } }))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit }) }))

const { useContactSearch } = await import('./useContactSearch.js')

const deferred = () => {
  let resolve
  const promise = new Promise((res) => {
    resolve = res
  })
  return { promise, resolve }
}

describe('useContactSearch', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    searchContacts.mockReset()
    emit.mockReset()
  })

  test('ignores a previous query that resolves last', async () => {
    let query = 'first'
    const first = deferred()
    const second = deferred()
    searchContacts.mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise)
    const search = useContactSearch({ getQuery: () => query, onSelect: vi.fn() })

    search.handleSearchContacts()
    await vi.advanceTimersByTimeAsync(300)
    query = 'second'
    search.handleSearchContacts()
    await vi.advanceTimersByTimeAsync(300)

    second.resolve({ data: { data: [{ id: 2 }] } })
    await vi.waitFor(() => expect(search.searchResults.value).toEqual([{ id: 2 }]))
    first.resolve({ data: { data: [{ id: 1 }] } })
    await vi.waitFor(() => expect(search.searchResults.value).toEqual([{ id: 2 }]))
  })

  test('keeps Escape from closing the parent dialog while results are open', () => {
    const search = useContactSearch({ getQuery: () => 'query', onSelect: vi.fn() })
    search.searchResults.value = [{ id: 1 }]
    const event = { key: 'Escape', stopPropagation: vi.fn() }

    search.handleSearchKeydown(event)

    expect(event.stopPropagation).toHaveBeenCalledOnce()
    expect(search.searchResults.value).toEqual([])
  })

  test('dismisses results and invalidates a request when focus leaves', async () => {
    const pending = deferred()
    searchContacts.mockReturnValueOnce(pending.promise)
    const search = useContactSearch({ getQuery: () => 'query', onSelect: vi.fn() })
    search.handleSearchContacts()
    await vi.advanceTimersByTimeAsync(300)

    search.clearSearchResults()
    pending.resolve({ data: { data: [{ id: 1 }] } })
    await Promise.resolve()

    expect(search.searchResults.value).toEqual([])
  })
})
