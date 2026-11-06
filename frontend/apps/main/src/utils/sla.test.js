import { afterEach, describe, expect, test, vi } from 'vitest'
import { calculateSla } from './sla'

afterEach(() => vi.useRealTimers())

describe('calculateSla', () => {
  test.each([
    ['2026-10-15T12:30:00Z', '30m'],
    ['2026-10-16T12:00:00Z', '24h'],
    ['2026-10-17T12:00:00Z', '2d'],
    ['2026-11-16T12:00:00Z', '1mo'],
    ['2027-10-15T12:00:00Z', '1y'],
    ['2027-11-18T12:00:00Z', '1y'],
    ['2028-10-15T12:00:00Z', '2y']
  ])('formats remaining time until %s as %s', (dueAt, value) => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-10-15T12:00:00Z'))
    expect(calculateSla(dueAt)).toEqual({ status: 'remaining', value })
  })

  test.each([
    ['2026-10-15T11:30:00Z', '30m'],
    ['2026-10-13T12:00:00Z', '2d'],
    ['2026-09-15T12:00:00Z', '30d'],
    ['2026-08-15T12:00:00Z', '2mo'],
    ['2025-09-11T12:00:00Z', '1y'],
    ['2024-10-15T12:00:00Z', '2y']
  ])('formats overdue time since %s as %s', (dueAt, value) => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-10-15T12:00:00Z'))
    expect(calculateSla(dueAt)).toEqual({ status: 'overdue', value })
  })

  test('uses the actual completion time for completed targets', () => {
    expect(calculateSla('2027-11-18T12:00:00Z', '2026-10-15T12:00:00Z'))
      .toEqual({ status: 'hit', value: '1y' })
    expect(calculateSla('2026-10-15T12:00:00Z', '2027-11-18T12:00:00Z'))
      .toEqual({ status: 'overdue', value: '1y' })
  })
})
