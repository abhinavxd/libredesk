export const getMessageDeliveryStatus = (message, direction, conversation) => {
  if (direction !== 'outgoing' || message?.status !== 'sent' || message?.private) return null

  if (conversation?.inbox_channel === 'whatsapp') {
    const providerStatus = message.meta?.provider_status
    if (providerStatus === 'read' || providerStatus === 'delivered') return providerStatus
  }

  if (
    conversation?.inbox_channel === 'livechat' &&
    conversation.contact_last_seen_at &&
    new Date(message.created_at) <= new Date(conversation.contact_last_seen_at)
  ) {
    return 'read'
  }

  return 'sent'
}
