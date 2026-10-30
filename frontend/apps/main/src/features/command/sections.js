
import { defineMessage as msg } from '@lingui/core/macro'
export const SECTIONS = {
  BULK: 'bulk',
  CONVERSATION: 'conversation',
  CONTACT: 'contact',
  LIST: 'list',
  ACTIONS: 'actions',
  CREATE: 'create',
  GOTO: 'goto',
  ACCOUNT: 'account',
  CONTACT_RESULTS: 'contact-results',
  CONVERSATION_RESULTS: 'conversation-results'
}

// Contextual sections come first, global ones last.
export const SECTION_ORDER = [
  SECTIONS.BULK,
  SECTIONS.CONVERSATION,
  SECTIONS.CONTACT,
  SECTIONS.LIST,
  SECTIONS.ACTIONS,
  SECTIONS.CREATE,
  SECTIONS.GOTO,
  SECTIONS.ACCOUNT,
  SECTIONS.CONTACT_RESULTS,
  SECTIONS.CONVERSATION_RESULTS
]

export const SECTION_MESSAGES = {
  [SECTIONS.BULK]: msg({ message: 'Selected conversations' }),
  [SECTIONS.CONVERSATION]: msg({ message: '{count, plural, one {Conversation} other {Conversations}}' }),
  [SECTIONS.CONTACT]: msg({ message: '{count, plural, one {Contact} other {Contacts}}' }),
  [SECTIONS.LIST]: msg({ message: 'Conversation list' }),
  [SECTIONS.ACTIONS]: msg({ message: '{count, plural, one {Action} other {Actions}}' }),
  [SECTIONS.CREATE]: msg({ message: 'Create' }),
  [SECTIONS.GOTO]: msg({ message: 'Go to' }),
  [SECTIONS.ACCOUNT]: msg({ message: '{count, plural, one {Account} other {Accounts}}' }),
  [SECTIONS.CONTACT_RESULTS]: msg({ message: '{count, plural, one {Contact} other {Contacts}}' }),
  [SECTIONS.CONVERSATION_RESULTS]: msg({ message: '{count, plural, one {Conversation} other {Conversations}}' })
}

export const SECTION_LABEL_PLURAL = new Set([
  SECTIONS.ACTIONS,
  SECTIONS.CONTACT_RESULTS,
  SECTIONS.CONVERSATION_RESULTS
])
