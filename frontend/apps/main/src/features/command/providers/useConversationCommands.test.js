import { describe, it, expect } from 'vitest'
import { formatSnoozeDuration } from '@main/features/command/providers/useConversationCommands'

describe('formatSnoozeDuration', () => {
  it('keeps minutes below an hour', () => {
    expect(formatSnoozeDuration(30)).toBe('30m')
  })

  it('uses hours only for exact hours', () => {
    expect(formatSnoozeDuration(60)).toBe('1h')
    expect(formatSnoozeDuration(180)).toBe('3h')
  })

  it('keeps minutes when the duration is not a whole hour', () => {
    expect(formatSnoozeDuration(90)).toBe('90m')
    expect(formatSnoozeDuration(1445)).toBe('1445m')
  })
})
