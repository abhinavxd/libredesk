import { describe, expect, it, vi } from 'vitest'
import { conversationReferenceHref, getTicketSuggestions } from './ticketReference'

describe('ticket reference suggestions', () => {
  it('does not search until three characters are entered', async () => {
    const search = vi.fn()
    await expect(getTicketSuggestions('10', search)).resolves.toEqual([])
    expect(search).not.toHaveBeenCalled()
  })

  it('maps a conversation result to a link suggestion', async () => {
    const search = vi
      .fn()
      .mockResolvedValue({
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
    await expect(getTicketSuggestions('108', search)).resolves.toEqual([
      {
        id: 'conversation-uuid',
        label: '108',
        subject: 'Payment failed',
        status: 'Open',
        href: '/inboxes/all/conversation/conversation-uuid'
      }
    ])
  })

  it('builds a stable conversation link', () => {
    expect(conversationReferenceHref('abc')).toBe('/inboxes/all/conversation/abc')
  })
})
