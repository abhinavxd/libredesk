import { describe, it, expect } from 'vitest'
import { computeLastSeenMessageMap, getAgentFullName, getAgentInitials, formatSeenByTooltip } from './seenBy'

describe('seenBy utility', () => {
  const messages = [
    { uuid: 'msg-1', created_at: '2026-09-10T10:00:00Z' },
    { uuid: 'msg-2', created_at: '2026-09-10T10:05:00Z' },
    { uuid: 'msg-3', created_at: '2026-09-10T10:10:00Z' },
    { uuid: 'msg-4', created_at: '2026-09-10T10:15:00Z' },
    { uuid: 'msg-5', created_at: '2026-09-10T10:20:00Z' },
  ]

  it('maps agent to the last message they read', () => {
    const seenBy = [
      // Agent A last saw at 10:12:00 -> read up to msg-3 (10:10:00), but msg-4 (10:15:00) is newer
      { user_id: 10, first_name: 'Alice', last_name: 'Smith', last_seen_at: '2026-09-10T10:12:00Z' },
      // Agent B last saw at 10:25:00 -> read up to msg-5 (10:20:00)
      { user_id: 20, first_name: 'Bob', last_name: 'Jones', last_seen_at: '2026-09-10T10:25:00Z' },
    ]

    const map = computeLastSeenMessageMap(messages, seenBy, 999)
    expect(map.get('msg-3')).toHaveLength(1)
    expect(map.get('msg-3')[0].first_name).toBe('Alice')

    expect(map.get('msg-5')).toHaveLength(1)
    expect(map.get('msg-5')[0].first_name).toBe('Bob')

    expect(map.has('msg-1')).toBe(false)
    expect(map.has('msg-2')).toBe(false)
    expect(map.has('msg-4')).toBe(false)
  })

  it('maps multiple agents who read the same latest message', () => {
    const seenBy = [
      { user_id: 10, first_name: 'Alice', last_name: 'Smith', last_seen_at: '2026-09-10T10:22:00Z' },
      { user_id: 20, first_name: 'Bob', last_name: 'Jones', last_seen_at: '2026-09-10T10:24:00Z' },
    ]

    const map = computeLastSeenMessageMap(messages, seenBy, 999)
    const msg5Seen = map.get('msg-5')
    expect(msg5Seen).toHaveLength(2)
    expect(msg5Seen.map(a => a.first_name)).toEqual(['Alice', 'Bob'])
  })

  it('excludes the current authenticated user', () => {
    const seenBy = [
      { user_id: 10, first_name: 'Alice', last_name: 'Smith', last_seen_at: '2026-09-10T10:25:00Z' },
      { user_id: 99, first_name: 'Current', last_name: 'User', last_seen_at: '2026-09-10T10:25:00Z' },
    ]

    const map = computeLastSeenMessageMap(messages, seenBy, 99)
    const msg5Seen = map.get('msg-5')
    expect(msg5Seen).toHaveLength(1)
    expect(msg5Seen[0].user_id).toBe(10)
  })

  it('ignores agents who have not seen any visible message', () => {
    const seenBy = [
      // Saw before msg-1 was created
      { user_id: 30, first_name: 'Old', last_name: 'Reader', last_seen_at: '2026-09-10T09:00:00Z' },
    ]

    const map = computeLastSeenMessageMap(messages, seenBy, null)
    expect(map.size).toBe(0)
  })

  it('handles empty inputs gracefully', () => {
    expect(computeLastSeenMessageMap([], [], null).size).toBe(0)
    expect(computeLastSeenMessageMap(null, null, null).size).toBe(0)
    expect(computeLastSeenMessageMap(messages, [], null).size).toBe(0)
  })

  it('computes agent full name and initials', () => {
    expect(getAgentFullName({ first_name: 'John', last_name: 'Doe' })).toBe('John Doe')
    expect(getAgentFullName({ first_name: 'John' })).toBe('John')
    expect(getAgentFullName({})).toBe('Agent')

    expect(getAgentInitials({ first_name: 'John', last_name: 'Doe' })).toBe('JD')
    expect(getAgentInitials({ first_name: 'John' })).toBe('J')
    expect(getAgentInitials({})).toBe('A')
  })

  it('formats tooltip text with and without i18n translator', () => {
    const agent = {
      first_name: 'Jane',
      last_name: 'Doe',
      last_seen_at: '2026-09-10T14:30:00Z',
    }

    const fallbackTooltip = formatSeenByTooltip(agent)
    expect(fallbackTooltip).toMatch(/^Seen by Jane Doe .+ \(.+\)$/)

    const mockT = (key, params) => `${key}:${params.name}:${params.relativeTime}:${params.dateTime}`
    const i18nTooltip = formatSeenByTooltip(agent, mockT)
    expect(i18nTooltip).toContain('conversation.seenBy:Jane Doe:')
  })
})
