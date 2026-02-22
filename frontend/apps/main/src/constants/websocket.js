export const WS_EVENT = {
    NEW_MESSAGE: 'new_message',
    MESSAGE_PROP_UPDATE: 'message_prop_update',
    CONVERSATION_PROP_UPDATE: 'conversation_prop_update',
    CONVERSATION_SUBSCRIBE: 'conversation_subscribe',
    CONVERSATION_SUBSCRIBED: 'conversation_subscribed',
    TYPING: 'typing',
}

// Message types that should not be queued because they become stale quickly
export const WS_EPHEMERAL_TYPES = [
    WS_EVENT.TYPING,
]