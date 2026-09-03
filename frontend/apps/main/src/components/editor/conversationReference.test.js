import { describe, expect, it, vi } from 'vitest'
import { getConversationSuggestions } from './conversationReference'

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
})
