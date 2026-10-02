// Helpers for the Users settings screen. Each person holds one role; Admins
// see every address, everyone else only the addresses granted to them.

export const BUILT_IN_ROLES = ['Admin', 'Agent', 'Contributor']

export const primaryRole = (roles = []) =>
  BUILT_IN_ROLES.find((role) => roles.includes(role)) || roles[0] || ''

// Roles a person holds that the screen cannot represent, from older releases.
export const legacyRoles = (roles = []) => {
  const primary = primaryRole(roles)
  return roles.filter((role) => role !== primary || !BUILT_IN_ROLES.includes(role))
}

const sortedIDs = (ids = []) => [...new Set(ids.map(Number))].sort((a, b) => a - b)

export const accessDraft = (user) => ({
  role: primaryRole(user?.roles),
  address_ids: sortedIDs(user?.address_ids),
  enabled: Boolean(user?.enabled)
})

export function describeAccessChanges(user, draft, addressLabel = (id) => String(id)) {
  const before = accessDraft(user)
  const after = { ...draft, address_ids: sortedIDs(draft.address_ids) }
  const changes = []
  const rolesChanged = before.role !== after.role || legacyRoles(user?.roles).length > 0
  if (rolesChanged && after.role) {
    changes.push({ kind: 'role', from: (user?.roles || []).join(', ') || '—', to: after.role })
  }
  const added = after.address_ids.filter((id) => !before.address_ids.includes(id))
  const removed = before.address_ids.filter((id) => !after.address_ids.includes(id))
  if (added.length) changes.push({ kind: 'added', list: added.map(addressLabel) })
  if (removed.length) changes.push({ kind: 'removed', list: removed.map(addressLabel) })
  if (before.enabled !== after.enabled) changes.push({ kind: after.enabled ? 'enabled' : 'disabled' })
  return changes
}

export const hasAccessChanges = (user, draft) => describeAccessChanges(user, draft).length > 0

export function filterUsers(users, query) {
  const needle = String(query || '').trim().toLowerCase()
  if (!needle) return users
  return users.filter((user) =>
    [user.first_name, user.last_name, user.email, ...(user.roles || [])]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(needle))
  )
}

export const fullName = (user) => [user?.first_name, user?.last_name].filter(Boolean).join(' ').trim()

export const initials = (user) =>
  [user?.first_name, user?.last_name]
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase())
    .join('')
    .slice(0, 2) || '?'
