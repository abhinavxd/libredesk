import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('@main/api', () => ({
  default: {
    searchMacros: vi.fn(),
    getMacro: vi.fn()
  }
}))

vi.mock('@main/composables/useEmitter', () => ({
  useEmitter: () => ({ emit: vi.fn(), on: vi.fn(), off: vi.fn() })
}))

const userMock = vi.hoisted(() => ({ allowed: null }))

vi.mock('@main/stores/user', () => ({
  useUserStore: () => ({
    teams: [],
    userID: 1,
    can: (permission) => userMock.allowed === null || userMock.allowed.includes(permission)
  })
}))

import api from '@main/api'
import { useMacroStore } from '@main/stores/macro'

const compactMacro = (overrides = {}) => ({
  id: 1,
  name: 'Macro',
  actions: [],
  visibility: 'all',
  visible_when: ['replying'],
  has_message_content: true,
  user_id: null,
  team_id: null,
  usage_count: 0,
  ...overrides
})

describe('macro store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    userMock.allowed = null
  })

  it('searches with the query and the current view', async () => {
    api.searchMacros.mockResolvedValue({ data: { data: [compactMacro()] } })
    const store = useMacroStore()
    store.setCurrentView('replying')

    await store.searchMacros('refund')

    expect(api.searchMacros).toHaveBeenCalledWith({ q: 'refund', view: 'replying' })
    expect(store.searchResults).toHaveLength(1)
  })

  it('loads macros without sending an empty query', async () => {
    api.searchMacros.mockResolvedValue({ data: { data: [compactMacro()] } })
    const store = useMacroStore()
    store.setCurrentView('replying')

    await store.searchMacros('  ')

    expect(api.searchMacros).toHaveBeenCalledWith({ view: 'replying' })
    expect(store.searchResults).toHaveLength(1)
  })

  it('keeps only the newest search response', async () => {
    let resolveFirst
    api.searchMacros
      .mockReturnValueOnce(new Promise((r) => (resolveFirst = r)))
      .mockResolvedValueOnce({ data: { data: [compactMacro({ id: 2, name: 'newer' })] } })
    const store = useMacroStore()

    const first = store.searchMacros('a')
    const second = store.searchMacros('ab')
    resolveFirst({ data: { data: [compactMacro({ id: 1, name: 'stale' })] } })
    await Promise.all([first, second])

    expect(store.searchResults.map((m) => m.name)).toEqual(['newer'])
    expect(store.searchLoading).toBe(false)
  })

  it('does not restore search results after clearing the cache', async () => {
    let resolveSearch
    api.searchMacros.mockReturnValue(new Promise((resolve) => { resolveSearch = resolve }))
    const store = useMacroStore()

    const search = store.searchMacros('old')
    store.clearCache()
    resolveSearch({ data: { data: [compactMacro()] } })
    await search

    expect(store.searchResults).toEqual([])
  })

  it('drops macros with no actions and no message content from the options', async () => {
    api.searchMacros.mockResolvedValue({
      data: {
        data: [
          compactMacro({ id: 1, name: 'content only', has_message_content: true }),
          compactMacro({ id: 2, name: 'actions only', has_message_content: false, actions: [{ type: 'add_tags', value: ['x'] }] }),
          compactMacro({ id: 3, name: 'empty', has_message_content: false })
        ]
      }
    })
    const store = useMacroStore()
    await store.searchMacros()

    expect(store.macroOptions.map((m) => m.id)).toEqual([1, 2])
  })

  it('fetches macro content once and serves it from the cache after', async () => {
    api.getMacro.mockResolvedValue({ data: { data: { id: 7, message_content: '<p>hi</p>' } } })
    const store = useMacroStore()

    const first = await store.fetchMacroContent(7)
    const second = await store.fetchMacroContent(7)

    expect(first).toBe('<p>hi</p>')
    expect(second).toBe('<p>hi</p>')
    expect(api.getMacro).toHaveBeenCalledTimes(1)
    expect(store.macroContents[7]).toBe('<p>hi</p>')
  })

  it('dedupes concurrent fetches for the same macro', async () => {
    let resolve
    api.getMacro.mockReturnValue(new Promise((r) => (resolve = r)))
    const store = useMacroStore()

    const a = store.fetchMacroContent(7)
    const b = store.fetchMacroContent(7)
    resolve({ data: { data: { id: 7, message_content: '<p>hi</p>' } } })

    expect(await a).toBe('<p>hi</p>')
    expect(await b).toBe('<p>hi</p>')
    expect(api.getMacro).toHaveBeenCalledTimes(1)
  })

  it('rejects and caches nothing when the fetch fails', async () => {
    api.getMacro.mockRejectedValue(new Error('boom'))
    const store = useMacroStore()

    await expect(store.fetchMacroContent(7)).rejects.toThrow('boom')
    expect(store.macroContents[7]).toBeUndefined()

    api.getMacro.mockResolvedValue({ data: { data: { id: 7, message_content: '<p>hi</p>' } } })
    expect(await store.fetchMacroContent(7)).toBe('<p>hi</p>')
    expect(api.getMacro).toHaveBeenCalledTimes(2)
  })

  it('does not restore macro content after clearing the cache', async () => {
    let resolveContent
    api.getMacro.mockReturnValue(new Promise((resolve) => { resolveContent = resolve }))
    const store = useMacroStore()

    const fetch = store.fetchMacroContent(7)
    store.clearCache()
    resolveContent({ data: { data: { message_content: '<p>old</p>' } } })
    await fetch

    expect(store.macroContents[7]).toBeUndefined()
  })

  it('restores filtered actions when permissions change', () => {
    userMock.allowed = ['messages:write']
    const store = useMacroStore()
    store.searchResults = [
      compactMacro({
        name: 'Reply and note',
        has_message_content: false,
        actions: [{ type: 'send_reply' }, { type: 'send_private_note' }]
      })
    ]

    expect(store.macroOptions[0].actions).toEqual([{ type: 'send_reply' }])
    expect(store.searchResults[0].actions).toHaveLength(2)

    userMock.allowed = ['messages:write', 'messages:write_private']
    store.searchResults = [...store.searchResults]

    expect(store.macroOptions[0].actions).toEqual([
      { type: 'send_reply' },
      { type: 'send_private_note' }
    ])
  })

  it('clears results and cached content on clearCache', async () => {
    api.searchMacros.mockResolvedValue({ data: { data: [compactMacro()] } })
    api.getMacro.mockResolvedValue({ data: { data: { id: 7, message_content: '<p>hi</p>' } } })
    const store = useMacroStore()

    await store.searchMacros()
    await store.fetchMacroContent(7)
    store.clearCache()

    expect(store.searchResults).toEqual([])
    expect(store.macroContents[7]).toBeUndefined()
  })
})
