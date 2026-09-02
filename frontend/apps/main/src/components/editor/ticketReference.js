import api from '@main/api'

export const MIN_REFERENCE_QUERY_LENGTH = 3

export const conversationReferenceHref = (uuid) => {
  const path = `/inboxes/all/conversation/${uuid}`
  const origin = typeof window !== 'undefined' ? window.location.origin : ''
  return origin ? `${origin}${path}` : path
}

export const getTicketSuggestions = async (query, search = api.searchConversations) => {
  const reference = query.trim()
  if (reference.length < MIN_REFERENCE_QUERY_LENGTH) return []

  const response = await search({ query: reference })
  return (response.data?.data || []).map((conversation) => ({
    id: conversation.uuid,
    label: conversation.reference_number,
    subject: conversation.subject,
    status: conversation.status,
    href: conversationReferenceHref(conversation.uuid)
  }))
}
