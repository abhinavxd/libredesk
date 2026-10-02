export const SECTIONS = {
  CONVERSATION: 'conversation',
  CONTACT: 'contact',
  LIST: 'list',
  ACTIONS: 'actions',
  CREATE: 'create',
  GOTO: 'goto'
}

// Contextual sections come first, global ones last.
export const SECTION_ORDER = [
  SECTIONS.CONVERSATION,
  SECTIONS.CONTACT,
  SECTIONS.LIST,
  SECTIONS.ACTIONS,
  SECTIONS.CREATE,
  SECTIONS.GOTO
]

export const SECTION_LABEL_KEYS = {
  [SECTIONS.CONVERSATION]: 'globals.terms.conversation',
  [SECTIONS.CONTACT]: 'globals.terms.correspondent',
  [SECTIONS.LIST]: 'command.section.list',
  [SECTIONS.ACTIONS]: 'globals.terms.action',
  [SECTIONS.CREATE]: 'globals.messages.create',
  [SECTIONS.GOTO]: 'command.section.goTo'
}

export const SECTION_LABEL_PLURAL = new Set([SECTIONS.ACTIONS])
