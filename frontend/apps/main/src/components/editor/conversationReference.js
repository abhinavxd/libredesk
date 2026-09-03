import api from '@main/api'

export const MIN_REFERENCE_QUERY_LENGTH = 3
const SUGGESTION_LIMIT = 10

export const getConversationSuggestions = async (query, search = api.searchConversations) => {
  const reference = query.trim()
  if (reference.length < MIN_REFERENCE_QUERY_LENGTH) return []

  const response = await search({ query: reference })
  return (response.data?.data || []).slice(0, SUGGESTION_LIMIT).map((conversation) => ({
    id: conversation.uuid,
    label: conversation.reference_number,
    subject: conversation.subject,
    status: conversation.status
  }))
}
