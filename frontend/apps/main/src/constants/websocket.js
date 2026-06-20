export const WS_EVENT = {
    NEW_MESSAGE: 'new_message',
    NEW_CONVERSATION: 'new_conversation',
    MESSAGE_UPDATE: 'message_update',
    CONVERSATION_UPDATE: 'conversation_update',
    CONTACT_UPDATE: 'contact_update',
    CONVERSATION_SUBSCRIBE: 'conversation_subscribe',
    LIST_SUBSCRIBE_REPLACE: 'list_subscribe_replace',
    TYPING: 'typing',
    NEW_NOTIFICATION: 'new_notification',
    AGENT_AVAILABILITY_UPDATE: 'agent_availability_update',
}

// Message types that should not be queued because they become stale quickly
export const WS_EPHEMERAL_TYPES = [
    WS_EVENT.TYPING,
]