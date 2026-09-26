import { describe, test, expect, afterEach } from 'vitest'
import { format } from 'date-fns'
import {
  setTimeFormat,
  timePattern,
  formatMessageTimestamp,
  formatFullTimestamp,
  formatDateTime
} from './datetime'

const afternoon = new Date(2025, 0, 5, 14, 7, 9)
const morning = new Date(2025, 0, 5, 9, 7, 9)

afterEach(() => setTimeFormat('12h'))

describe('12-hour time format (default)', () => {
  test('formatMessageTimestamp', () => {
    expect(formatMessageTimestamp(afternoon)).toBe('5 Jan, 02:07 PM')
  })

  test('formatFullTimestamp', () => {
    expect(formatFullTimestamp(afternoon)).toBe('5 Jan 2025, 02:07 PM')
  })

  test('formatDateTime', () => {
    expect(formatDateTime(afternoon)).toBe('Jan 5, 2025, 2:07:09 PM')
  })

  test('timePattern renders like the previous 12-hour patterns', () => {
    expect(format(afternoon, timePattern())).toBe(format(afternoon, 'h:mm a'))
    expect(format(afternoon, `PP, ${timePattern()}`)).toBe(format(afternoon, 'PPp'))
    expect(format(afternoon, `PPP ${timePattern()}`)).toBe(format(afternoon, 'PPP p'))
    expect(formatDateTime(afternoon)).toBe(format(afternoon, `PP, ${timePattern(true)}`))
  })

  test('unknown values fall back to 12-hour', () => {
    setTimeFormat('24h')
    setTimeFormat(undefined)
    expect(formatMessageTimestamp(afternoon)).toBe('5 Jan, 02:07 PM')
    setTimeFormat('bogus')
    expect(format(afternoon, timePattern())).toBe('2:07 PM')
  })
})

describe('24-hour time format', () => {
  test('formatMessageTimestamp', () => {
    setTimeFormat('24h')
    expect(formatMessageTimestamp(afternoon)).toBe('5 Jan, 14:07')
    expect(formatMessageTimestamp(morning)).toBe('5 Jan, 09:07')
  })

  test('formatFullTimestamp', () => {
    setTimeFormat('24h')
    expect(formatFullTimestamp(afternoon)).toBe('5 Jan 2025, 14:07')
  })

  test('formatDateTime', () => {
    setTimeFormat('24h')
    expect(formatDateTime(afternoon)).toBe('Jan 5, 2025, 14:07:09')
  })

  test('timePattern', () => {
    setTimeFormat('24h')
    expect(format(afternoon, timePattern())).toBe('14:07')
    expect(format(morning, timePattern(true))).toBe('09:07:09')
  })
})
