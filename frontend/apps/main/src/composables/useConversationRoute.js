import { useRoute } from 'vue-router'

// Builds the route descriptor needed to open a conversation while staying within
// the user's current list context (regular inbox, team inbox, or saved view).
export function useConversationRoute () {
  const route = useRoute()

  const buildConversationRoute = (conversation) => {
    const baseRoute = route.name.includes('team')
      ? 'team-inbox-conversation'
      : route.name.includes('view')
        ? 'view-inbox-conversation'
        : 'inbox-conversation'
    return {
      name: baseRoute,
      params: {
        uuid: conversation.uuid,
        ...(baseRoute === 'team-inbox-conversation' && { teamID: route.params.teamID }),
        ...(baseRoute === 'view-inbox-conversation' && { viewID: route.params.viewID })
      },
      query: conversation.mentioned_message_uuid
        ? { scrollTo: conversation.mentioned_message_uuid }
        : {}
    }
  }

  return { buildConversationRoute }
}
