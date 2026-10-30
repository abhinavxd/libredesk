/**
 * Picks the toast for a finished conversation delete.
 *
 * The mailbox purge moves the conversation's mails to the mailbox's Trash, where a mistaken delete
 * can still be undone from a mail client, so the toast says where they went. Two outcomes are worth
 * flagging: mails the purge could not reach, which the next inbox scan imports again as a brand new
 * conversation, and mails it had to delete outright because the account has no Trash folder.
 *
 * @param {object|undefined} result The delete API payload.
 * @param {string[]|undefined} result.unpurged_message_ids Message-IDs the purge could not remove.
 * @param {object|undefined} result.mail_purge Counts of what the purge did, absent when none ran.
 * @returns {{ key: string, count: number, variant: string|undefined }} i18n key, plural count and toast variant.
 */
export function deletionToast (result) {
  const unpurged = Array.isArray(result?.unpurged_message_ids) ? result.unpurged_message_ids.length : 0
  if (unpurged > 0) {
    return { key: 'conversation.deletedMailsRemaining', count: unpurged, variant: 'warning' }
  }

  const expunged = result?.mail_purge?.expunged ?? 0
  if (expunged > 0) {
    return { key: 'conversation.deletedMailsExpunged', count: expunged, variant: 'warning' }
  }

  const movedToTrash = result?.mail_purge?.moved_to_trash ?? 0
  if (movedToTrash > 0) {
    return { key: 'conversation.deletedMailsTrashed', count: movedToTrash, variant: undefined }
  }

  return { key: 'conversation.deleted', count: 0, variant: undefined }
}
