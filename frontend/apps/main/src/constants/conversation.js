export const CONVERSATION_LIST_TYPE = {
  ASSIGNED: 'assigned',
  UNASSIGNED: 'unassigned',
  TEAM_UNASSIGNED: 'team_unassigned',
  VIEW: 'view',
  ALL: 'all',
  MENTIONED: 'mentioned'
}

export const CONVERSATION_DEFAULT_STATUSES = {
  OPEN: 'Open',
  SNOOZED: 'Snoozed',
  RESOLVED: 'Resolved',
  CLOSED: 'Closed',
}

export const CONVERSATION_DEFAULT_STATUSES_LIST = Object.values(CONVERSATION_DEFAULT_STATUSES);

export const MACRO_CONTEXT = {
  REPLY: 'reply',
  NEW_CONVERSATION: 'new-conversation'
}