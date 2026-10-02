export function addressLabel(address) {
  const displayName = String(address?.display_name || '').trim()
  if (displayName) return displayName

  const email = String(address?.address || '').trim()
  const localPart = email.split('@')[0]
  const humanized = localPart.replace(/[._+-]+/g, ' ').trim()

  if (!humanized) return email

  return humanized.replace(/(^|\s)\S/g, (character) => character.toUpperCase())
}
