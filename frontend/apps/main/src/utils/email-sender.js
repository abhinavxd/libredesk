export function resolveEmailSender (message, ownedAddresses) {
  const owned = (ownedAddresses || [])
  const ownedSet = new Set(owned.map(address => address.toLowerCase()))
  const candidate = message?.type === 'incoming'
    ? message?.meta?.inbox_address
    : message?.meta?.send_from
  if (candidate && ownedSet.has(candidate.toLowerCase())) return candidate.toLowerCase()
  return owned[0] || ''
}
