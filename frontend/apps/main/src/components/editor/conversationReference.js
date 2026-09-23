import api from '@main/api'

export const MIN_REFERENCE_QUERY_LENGTH = 3
const SUGGESTION_LIMIT = 10

export const createLatestConversationSuggestionFetcher = (fetchSuggestions) => {
  let latestRequestID = 0

  return async (query) => {
    const requestID = ++latestRequestID
    const suggestions = await fetchSuggestions(query)
    return requestID === latestRequestID ? suggestions : []
  }
}

export const getConversationSuggestions = async (query, search = api.searchConversations) => {
  const reference = query.trim()
  if (reference.length < MIN_REFERENCE_QUERY_LENGTH) return []

  const response = await search({ query: reference, page_size: SUGGESTION_LIMIT })
  return (response.data?.data?.results || [])
    .filter((conversation) => conversation.reference_number === reference)
    .map((conversation) => ({
      id: conversation.uuid,
      label: conversation.reference_number,
      subject: conversation.subject,
      status: conversation.status
    }))
}
