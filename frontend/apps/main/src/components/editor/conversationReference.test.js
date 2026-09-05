import { describe, expect, it, vi } from 'vitest'
import * as conversationReference from './conversationReference'

const { getConversationSuggestions } = conversationReference

describe('conversation reference suggestions', () => {
  it('does not search until three characters are entered', async () => {
    const search = vi.fn()
    await expect(getConversationSuggestions('10', search)).resolves.toEqual([])
    expect(search).not.toHaveBeenCalled()
  })

  it('maps a conversation result to a suggestion', async () => {
    const search = vi.fn().mockResolvedValue({
      data: {
        data: [
          {
            uuid: 'conversation-uuid',
            reference_number: '108',
            subject: 'Payment failed',
            status: 'Open'
          }
        ]
      }
    })
    await expect(getConversationSuggestions('108', search)).resolves.toEqual([
      { id: 'conversation-uuid', label: '108', subject: 'Payment failed', status: 'Open' }
    ])
  })

  it('caps the suggestion list', async () => {
    const rows = Array.from({ length: 30 }, (_, i) => ({ uuid: `u${i}`, reference_number: `${i}` }))
    const search = vi.fn().mockResolvedValue({ data: { data: rows } })
    await expect(getConversationSuggestions('108', search)).resolves.toHaveLength(10)
  })

  it('discards a response superseded by a newer query', async () => {
    let resolveFirst
    let resolveSecond
    const fetchSuggestions = vi
      .fn()
      .mockImplementationOnce(() => new Promise((resolve) => (resolveFirst = resolve)))
      .mockImplementationOnce(() => new Promise((resolve) => (resolveSecond = resolve)))
    const getLatestSuggestions =
      conversationReference.createLatestConversationSuggestionFetcher(fetchSuggestions)

    const firstRequest = getLatestSuggestions('108')
    const secondRequest = getLatestSuggestions('109')

    resolveSecond([{ label: '109' }])
    await expect(secondRequest).resolves.toEqual([{ label: '109' }])
    resolveFirst([{ label: '108' }])
    await expect(firstRequest).resolves.toEqual([])
  })
})
