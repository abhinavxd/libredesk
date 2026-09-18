import { format, formatDistanceToNow } from 'date-fns'

/**
 * Computes a map of message UUID to the list of agents who last saw up to that message.
 *
 * @param {Array} messages - List of messages sorted in chronological order (oldest to newest)
 * @param {Array} seenByList - List of seen_by agent objects from conversation
 * @param {number|string|null} currentUserID - ID of currently authenticated agent (to exclude self)
 * @returns {Map<string, Array>} Map of message UUID to array of agents
 */
export function computeLastSeenMessageMap(messages = [], seenByList = [], currentUserID = null) {
  const map = new Map()
  if (!Array.isArray(messages) || messages.length === 0) return map
  if (!Array.isArray(seenByList) || seenByList.length === 0) return map

  const teammates = currentUserID
    ? seenByList.filter(agent => String(agent.user_id) !== String(currentUserID))
    : seenByList

  for (const agent of teammates) {
    if (!agent?.last_seen_at) continue
    const seenTime = new Date(agent.last_seen_at).getTime()
    if (isNaN(seenTime)) continue

    // Find the latest message whose created_at is <= agent's last_seen_at
    for (let i = messages.length - 1; i >= 0; i--) {
      const msg = messages[i]
      if (!msg?.created_at || !msg?.uuid) continue
      const msgTime = new Date(msg.created_at).getTime()
      if (msgTime <= seenTime) {
        if (!map.has(msg.uuid)) {
          map.set(msg.uuid, [])
        }
        map.get(msg.uuid).push(agent)
        break
      }
    }
  }

  return map
}

/**
 * Returns full name for an agent.
 */
export function getAgentFullName(agent) {
  if (!agent) return 'Agent'
  const first = agent.first_name || ''
  const last = agent.last_name || ''
  return `${first} ${last}`.trim() || 'Agent'
}

/**
 * Returns 1-2 character initials for an agent.
 */
export function getAgentInitials(agent) {
  if (!agent) return 'A'
  const first = agent.first_name?.charAt(0)?.toUpperCase() || ''
  const last = agent.last_name?.charAt(0)?.toUpperCase() || ''
  return `${first}${last}` || 'A'
}

/**
 * Formats hover tooltip text for seen_by badge.
 * e.g., "Seen by Jane Doe 2 days ago (16 Sep, 03:00 PM)"
 */
export function formatSeenByTooltip(agent, t) {
  if (!agent?.last_seen_at) return ''
  const date = new Date(agent.last_seen_at)
  if (isNaN(date.getTime())) return ''

  const name = getAgentFullName(agent)
  const relativeTime = formatDistanceToNow(date, { addSuffix: true })
  const dateTime = format(date, 'd MMM, hh:mm a')

  if (typeof t === 'function') {
    return t('conversation.seenBy', { name, relativeTime, dateTime })
  }

  return `Seen by ${name} ${relativeTime} (${dateTime})`
}
