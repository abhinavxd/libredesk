import { describe, test, expect } from 'vitest'
import { deletionToast } from './conversation-delete'

describe('deletionToast', () => {
  test('plain confirmation when every mail was purged', () => {
    expect(deletionToast([])).toEqual({ key: 'conversation.deleted', count: 0, variant: undefined })
  })

  test('plain confirmation when the API reported nothing', () => {
    expect(deletionToast(undefined)).toEqual({ key: 'conversation.deleted', count: 0, variant: undefined })
    expect(deletionToast(null)).toEqual({ key: 'conversation.deleted', count: 0, variant: undefined })
  })

  test('warns with the count when mails were left on the server', () => {
    expect(deletionToast(['a@example.com'])).toEqual({
      key: 'conversation.deletedMailsRemaining',
      count: 1,
      variant: 'warning'
    })
    expect(deletionToast(['a@example.com', 'b@example.com'])).toEqual({
      key: 'conversation.deletedMailsRemaining',
      count: 2,
      variant: 'warning'
    })
  })
})
