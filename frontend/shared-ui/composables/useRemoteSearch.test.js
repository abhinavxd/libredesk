import { describe, it, expect, vi, afterEach } from 'vitest'
import { useRemoteSearch } from './useRemoteSearch'

describe('useRemoteSearch', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('keeps picker results independent when another picker is cleared', async () => {
    vi.useFakeTimers()
    const first = useRemoteSearch(vi.fn().mockResolvedValue([{ value: '1', label: 'one' }]), 10)
    const second = useRemoteSearch(vi.fn().mockResolvedValue([{ value: '2', label: 'two' }]), 10)

    first.update('one')
    second.update('two')
    await vi.runAllTimersAsync()
    first.update('')

    expect(first.results.value).toBeNull()
    expect(second.results.value).toEqual([{ value: '2', label: 'two' }])
  })

  it('ignores a pending response after the picker is cleared', async () => {
    vi.useFakeTimers()
    let resolveSearch
    const search = vi.fn().mockReturnValue(new Promise((resolve) => { resolveSearch = resolve }))
    const lookup = useRemoteSearch(search, 10)

    lookup.update('old')
    await vi.advanceTimersByTimeAsync(10)
    lookup.update('')
    resolveSearch([{ value: '1', label: 'old' }])
    await Promise.resolve()

    expect(lookup.results.value).toBeNull()
    expect(lookup.searching.value).toBe(false)
  })

  it('cancels a debounced search when the picker is cleared', async () => {
    vi.useFakeTimers()
    const search = vi.fn()
    const lookup = useRemoteSearch(search, 10)

    lookup.update('old')
    lookup.update('')
    await vi.runAllTimersAsync()

    expect(search).not.toHaveBeenCalled()
  })
})
