import { describe, it, expect, vi } from 'vitest'
import { fetchAllPages } from './paged-fetch'

const page = (rows) => ({ data: { data: rows } })
const rowsOf = (n, offset = 0) => Array.from({ length: n }, (_, i) => ({ id: offset + i + 1 }))

const flush = () => new Promise((resolve) => setTimeout(resolve, 0))

describe('fetchAllPages', () => {
  it('stops after one short page', async () => {
    const fetchPage = vi.fn().mockResolvedValue(page(rowsOf(3)))
    const onRows = vi.fn()

    await fetchAllPages(fetchPage, onRows, vi.fn(), 5)
    await flush()

    expect(fetchPage).toHaveBeenCalledTimes(1)
    expect(fetchPage).toHaveBeenCalledWith({ page: 1, page_size: 5 })
    expect(onRows).toHaveBeenCalledWith(rowsOf(3))
  })

  it('resolves after the first page and streams the rest in the background', async () => {
    let releaseSecond
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce(page(rowsOf(2)))
      .mockReturnValueOnce(new Promise((resolve) => (releaseSecond = resolve)))
      .mockResolvedValueOnce(page(rowsOf(1, 4)))
    const seen = []

    await fetchAllPages(fetchPage, (rows) => seen.push(...rows), vi.fn(), 2)
    expect(seen).toHaveLength(2)

    releaseSecond(page(rowsOf(2, 2)))
    await flush()
    expect(seen.map((r) => r.id)).toEqual([1, 2, 3, 4, 5])
    expect(fetchPage).toHaveBeenCalledTimes(3)
    expect(fetchPage).toHaveBeenLastCalledWith({ page: 3, page_size: 2 })
  })

  it('stops on an exactly empty follow-up page', async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce(page(rowsOf(2)))
      .mockResolvedValueOnce(page([]))
    const onRows = vi.fn()

    await fetchAllPages(fetchPage, onRows, vi.fn(), 2)
    await flush()

    expect(fetchPage).toHaveBeenCalledTimes(2)
    expect(onRows).toHaveBeenCalledTimes(1)
  })

  it('rejects when the first page fails and reports background failures via onError', async () => {
    const boom = new Error('boom')
    await expect(fetchAllPages(vi.fn().mockRejectedValue(boom), vi.fn(), vi.fn(), 2)).rejects.toThrow('boom')

    const fetchPage = vi.fn().mockResolvedValueOnce(page(rowsOf(2))).mockRejectedValueOnce(boom)
    const onError = vi.fn()
    await fetchAllPages(fetchPage, vi.fn(), onError, 2)
    await flush()
    expect(onError).toHaveBeenCalledWith(boom)
  })
})
