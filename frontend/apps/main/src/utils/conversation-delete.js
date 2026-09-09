/**
 * Picks the toast for a finished conversation delete.
 *
 * A purge that could not reach every mail is worth flagging: those mails are still on the mail
 * server, and the next inbox scan imports them again as a brand new conversation.
 *
 * @param {string[]|undefined} unpurgedMessageIDs Message-IDs the API could not remove from the mailbox.
 * @returns {{ key: string, count: number, variant: string|undefined }} i18n key, plural count and toast variant.
 */
export function deletionToast (unpurgedMessageIDs) {
  const count = Array.isArray(unpurgedMessageIDs) ? unpurgedMessageIDs.length : 0
  if (count === 0) {
    return { key: 'conversation.deleted', count: 0, variant: undefined }
  }
  return { key: 'conversation.deletedMailsRemaining', count, variant: 'warning' }
}
