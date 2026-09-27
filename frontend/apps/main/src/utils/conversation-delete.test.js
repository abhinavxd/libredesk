import { describe, test, expect } from 'vitest'
import { deletionToast } from './conversation-delete'

describe('deletionToast', () => {
  test('plain confirmation when no mail was purged', () => {
    expect(deletionToast({ unpurged_message_ids: [] })).toEqual({
      key: 'conversation.deleted',
      count: 0,
      variant: undefined
    })
  })

  test('plain confirmation when the API reported nothing', () => {
    expect(deletionToast(undefined)).toEqual({ key: 'conversation.deleted', count: 0, variant: undefined })
    expect(deletionToast(null)).toEqual({ key: 'conversation.deleted', count: 0, variant: undefined })
    expect(deletionToast({})).toEqual({ key: 'conversation.deleted', count: 0, variant: undefined })
  })

  test('names the mailbox Trash when the mails were moved there', () => {
    expect(
      deletionToast({
        unpurged_message_ids: [],
        mail_purge: { moved_to_trash: 3, expunged: 0, not_found: 0, not_purged: 0, failed: 0, trash_mailbox: 'Trash' }
      })
    ).toEqual({ key: 'conversation.deletedMailsTrashed', count: 3, variant: undefined })
  })

  test('warns when the mails had to be deleted outright', () => {
    expect(
      deletionToast({
        unpurged_message_ids: [],
        mail_purge: { moved_to_trash: 0, expunged: 2, not_found: 0, not_purged: 0, failed: 0 }
      })
    ).toEqual({ key: 'conversation.deletedMailsExpunged', count: 2, variant: 'warning' })
  })

  test('an irreversible delete is reported even next to a move', () => {
    expect(
      deletionToast({
        unpurged_message_ids: [],
        mail_purge: { moved_to_trash: 1, expunged: 1, not_found: 0, not_purged: 0, failed: 0 }
      })
    ).toEqual({ key: 'conversation.deletedMailsExpunged', count: 1, variant: 'warning' })
  })

  test('mails left on the server outrank every other outcome', () => {
    expect(
      deletionToast({
        unpurged_message_ids: ['a@example.com', 'b@example.com'],
        mail_purge: { moved_to_trash: 1, expunged: 0, not_found: 1, not_purged: 1, failed: 0 }
      })
    ).toEqual({ key: 'conversation.deletedMailsRemaining', count: 2, variant: 'warning' })
  })
})
