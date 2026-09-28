export function formatChannelIdentity(identity, contact) {
  const username = contact?.custom_attributes?.telegram_username
  if (identity.channel === 'telegram' && username) return `@${username} (${identity.identifier})`
  if (identity.channel === 'whatsapp') return `+${identity.identifier}`
  return identity.identifier
}
