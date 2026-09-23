import { beforeEach, describe, expect, test, vi } from 'vitest'
import { ref } from 'vue'
import api from '@/api'
import { useHelpCenterArticles } from './useHelpCenterArticles'

vi.mock('@/api', () => ({
  default: {
    getHelpCenterTree: vi.fn()
  }
}))

const treeResponse = (tree) => ({ data: { data: { tree } } })
const deferred = () => {
  let resolve
  let reject
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

beforeEach(() => vi.clearAllMocks())

describe('useHelpCenterArticles', () => {
  test('keeps the empty state local when no help center is selected', () => {
    const result = useHelpCenterArticles(ref(0))

    expect(api.getHelpCenterTree).not.toHaveBeenCalled()
    expect(result.tree.value).toEqual([])
    expect(result.articles.value).toEqual([])
    expect(result.failed.value).toBe(false)
  })

  test('exposes only published collections and articles', async () => {
    api.getHelpCenterTree.mockResolvedValue(
      treeResponse([
        {
          id: 1,
          is_published: true,
          articles: [
            { id: 11, title: 'Published', status: 'published' },
            { id: 12, title: 'Draft', status: 'draft' }
          ],
          children: [
            {
              id: 2,
              is_published: true,
              articles: [{ id: 21, title: 'Nested', status: 'published' }],
              children: []
            },
            {
              id: 3,
              is_published: false,
              articles: [{ id: 31, title: 'Hidden', status: 'published' }],
              children: []
            }
          ]
        }
      ])
    )

    const result = useHelpCenterArticles(ref(7))

    await vi.waitFor(() => expect(result.articles.value).toHaveLength(2))
    expect(result.articles.value).toEqual([
      expect.objectContaining({ id: 11, collection_id: 1 }),
      expect.objectContaining({ id: 21, collection_id: 2 })
    ])
    expect(result.tree.value[0].article_count).toBe(2)
  })

  test('shows the failed empty state when loading fails', async () => {
    api.getHelpCenterTree.mockRejectedValue(new Error('network'))

    const result = useHelpCenterArticles(ref(7))

    await vi.waitFor(() => expect(result.failed.value).toBe(true))
    expect(result.tree.value).toEqual([])
    expect(result.articles.value).toEqual([])
  })

  test('ignores an older response after the selected help center changes', async () => {
    const first = deferred()
    const second = deferred()
    api.getHelpCenterTree.mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise)
    const helpCenterID = ref(1)
    const result = useHelpCenterArticles(helpCenterID)

    helpCenterID.value = 2
    second.resolve(
      treeResponse([
        {
          id: 2,
          is_published: true,
          articles: [{ id: 22, title: 'Current', status: 'published' }],
          children: []
        }
      ])
    )
    await vi.waitFor(() => expect(result.articles.value[0]?.id).toBe(22))

    first.resolve(
      treeResponse([
        {
          id: 1,
          is_published: true,
          articles: [{ id: 11, title: 'Stale', status: 'published' }],
          children: []
        }
      ])
    )
    await Promise.resolve()
    expect(result.articles.value[0].id).toBe(22)
  })
})
