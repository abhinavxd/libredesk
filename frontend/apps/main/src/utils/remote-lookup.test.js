import { describe, it, expect, vi } from 'vitest'
import { createRemoteLookup } from './remote-lookup'

const rows = (values) => ({ data: { data: values } })

describe('createRemoteLookup', () => {
  it('seeds the list from the first page and only fetches it once', async () => {
    const fetchRows = vi.fn().mockResolvedValue(rows([{ id: 1, name: 'a' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    await Promise.all([lookup.fetchFirstPage(), lookup.fetchFirstPage()])

    expect(fetchRows).toHaveBeenCalledTimes(1)
    expect(lookup.rows.value).toEqual([{ id: 1, name: 'a' }])
  })

  it('returns server search results without replacing the shared list', async () => {
    const fetchRows = vi.fn()
      .mockResolvedValueOnce(rows([{ id: 1, name: 'a' }]))
      .mockResolvedValueOnce(rows([{ id: 9, name: 'zebra' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    await lookup.fetchFirstPage()
    const found = await lookup.search('zeb')

    expect(fetchRows).toHaveBeenLastCalledWith({ page_size: 50, q: 'zeb', page: 1 })
    expect(found).toEqual([{ id: 9, name: 'zebra' }])
    expect(lookup.rows.value).toEqual([{ id: 1, name: 'a' }])
  })

  it('keeps a late first page separate from search results', async () => {
    let resolveFirstPage
    const fetchRows = vi.fn()
      .mockImplementationOnce(() => new Promise((resolve) => { resolveFirstPage = resolve }))
      .mockResolvedValueOnce(rows([{ id: 2, name: 'search result' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    const firstPage = lookup.fetchFirstPage()
    const found = await lookup.search('search')
    resolveFirstPage(rows([{ id: 1, name: 'first page' }]))
    await firstPage

    expect(found).toEqual([{ id: 2, name: 'search result' }])
    expect(lookup.rows.value).toEqual([{ id: 1, name: 'first page' }])
  })

  it('returns independent search results without replacing the shared list', async () => {
    let resolveFirstSearch
    const fetchRows = vi.fn()
      .mockResolvedValueOnce(rows([{ id: 1, name: 'first page' }]))
      .mockImplementationOnce(() => new Promise((resolve) => { resolveFirstSearch = resolve }))
      .mockResolvedValueOnce(rows([{ id: 3, name: 'second search' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    await lookup.fetchFirstPage()
    const firstSearch = lookup.search('first')
    const secondSearch = await lookup.search('second')
    resolveFirstSearch(rows([{ id: 2, name: 'first search' }]))

    expect(await firstSearch).toEqual([{ id: 2, name: 'first search' }])
    expect(secondSearch).toEqual([{ id: 3, name: 'second search' }])
    expect(lookup.rows.value).toEqual([{ id: 1, name: 'first page' }])
  })

  it('keeps pinned ids in the shared list while returning search results separately', async () => {
    const fetchRows = vi.fn()
      .mockResolvedValueOnce(rows([{ id: 7, name: 'pinned' }]))
      .mockResolvedValueOnce(rows([{ id: 1, name: 'a' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    await lookup.ensureIDs([7])
    const found = await lookup.search('a')

    expect(found).toEqual([{ id: 1, name: 'a' }])
    expect(lookup.rows.value).toEqual([{ id: 7, name: 'pinned' }])
  })

  it('resolves ids from the first page instead of fetching them separately', async () => {
    const fetchRows = vi.fn().mockResolvedValue(rows([{ id: 3, name: 'c' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    await lookup.ensureIDs([3])
    await lookup.ensureIDs([3])

    expect(fetchRows).toHaveBeenCalledTimes(1)
    expect(fetchRows).toHaveBeenCalledWith({ page_size: 50, page: 1 })
  })

  it('fetches only the ids the first page does not hold', async () => {
    const fetchRows = vi.fn()
      .mockResolvedValueOnce(rows([{ id: 1, name: 'a' }]))
      .mockResolvedValueOnce(rows([{ id: 9, name: 'z' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    await lookup.ensureIDs([1, 9])

    expect(fetchRows).toHaveBeenCalledTimes(2)
    expect(fetchRows).toHaveBeenLastCalledWith({ page_size: 50, ids: '9' })
    expect(lookup.rows.value).toEqual([{ id: 1, name: 'a' }, { id: 9, name: 'z' }])
  })

  it('returns the cached first page for an empty search', async () => {
    const fetchRows = vi.fn()
      .mockResolvedValueOnce(rows([{ id: 1, name: 'a' }]))
      .mockResolvedValueOnce(rows([{ id: 9, name: 'zebra' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    await lookup.fetchFirstPage()
    await lookup.search('zeb')
    const found = await lookup.search('')

    expect(fetchRows).toHaveBeenCalledTimes(2)
    expect(found).toEqual([{ id: 1, name: 'a' }])
    expect(lookup.rows.value).toEqual([{ id: 1, name: 'a' }])
  })

  it('reports a failed first page and allows a retry', async () => {
    const onError = vi.fn()
    const fetchRows = vi.fn()
      .mockRejectedValueOnce(new Error('boom'))
      .mockResolvedValueOnce(rows([{ id: 1, name: 'a' }]))
    const lookup = createRemoteLookup({ fetchRows, onError })

    await lookup.fetchFirstPage()
    expect(onError).toHaveBeenCalledTimes(1)
    expect(lookup.rows.value).toEqual([])

    await lookup.fetchFirstPage()
    expect(lookup.rows.value).toEqual([{ id: 1, name: 'a' }])
  })

  it('patches a cached row in place', async () => {
    const fetchRows = vi.fn().mockResolvedValue(rows([{ id: 1, availability_status: 'offline' }]))
    const lookup = createRemoteLookup({ fetchRows, onError: vi.fn() })

    await lookup.fetchFirstPage()
    lookup.patch(1, { availability_status: 'online' })

    expect(lookup.rows.value).toEqual([{ id: 1, availability_status: 'online' }])
  })
})
