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

export const SECTION_LABEL_KEYS = {
  [SECTIONS.BULK]: 'command.section.selectedConversations',
  [SECTIONS.CONVERSATION]: 'globals.terms.conversation',
  [SECTIONS.CONTACT]: 'globals.terms.contact',
  [SECTIONS.LIST]: 'command.section.list',
  [SECTIONS.ACTIONS]: 'globals.terms.action',
  [SECTIONS.CREATE]: 'globals.messages.create',
  [SECTIONS.GOTO]: 'command.section.goTo',
  [SECTIONS.ACCOUNT]: 'globals.terms.account',
  [SECTIONS.CONTACT_RESULTS]: 'globals.terms.contact',
  [SECTIONS.CONVERSATION_RESULTS]: 'globals.terms.conversation'
}

export const SECTION_LABEL_PLURAL = new Set([
  SECTIONS.ACTIONS,
  SECTIONS.CONTACT_RESULTS,
  SECTIONS.CONVERSATION_RESULTS
])
