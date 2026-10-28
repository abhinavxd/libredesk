import { afterEach, describe, expect, it, vi } from 'vitest'
import { timeZoneOptions } from './timezones.js'

afterEach(() => {
  vi.restoreAllMocks()
  vi.useRealTimers()
  vi.resetModules()
})

describe('timezone options', () => {
  it('keeps every timezone available when browser timezone data is older', async () => {
    const DateTimeFormat = Intl.DateTimeFormat
    vi.spyOn(Intl, 'DateTimeFormat').mockImplementation(function (locale, options) {
      if (options.timeZone === 'America/Coyhaique') {
        throw new RangeError('Unsupported timezone')
      }
      return new DateTimeFormat(locale, options)
    })
    vi.resetModules()
    const { timeZoneOptions: options } = await import('./timezones.js')
    expect(options.map((option) => option.value)).toEqual(
      timeZoneOptions.map((option) => option.value)
    )
    expect(options.find((option) => option.value === 'America/Coyhaique')).toEqual({
      label: 'America/Coyhaique',
      value: 'America/Coyhaique'
    })
    expect(options.find((option) => option.value === 'UTC').label).toBe('UTC (UTC+00:00)')
  })

  it.each([
    ['2026-01-15T12:00:00Z', 'Europe/Amsterdam', 'UTC+01:00'],
    ['2026-07-15T12:00:00Z', 'Europe/Amsterdam', 'UTC+02:00'],
    ['2026-01-15T12:00:00Z', 'Asia/Kathmandu', 'UTC+05:45'],
    ['2026-01-15T12:00:00Z', 'America/St_Johns', 'UTC-03:30'],
    ['2026-01-15T12:00:00Z', 'UTC', 'UTC+00:00']
  ])('shows the current offset on %s for %s', async (date, zone, offset) => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(date))
    vi.resetModules()
    const { timeZoneOptions: options } = await import('./timezones.js')
    expect(options.find((option) => option.value === zone).label).toBe(`${zone} (${offset})`)
  })

  it('offers unique timezone values that JavaScript can format', () => {
    const values = timeZoneOptions.map((option) => option.value)
    expect(values.length).toBeGreaterThan(400)
    expect(new Set(values).size).toBe(values.length)
    expect(values[0]).toBe('UTC')
    for (const timeZone of values) {
      expect(() => new Intl.DateTimeFormat('en', { timeZone }).format()).not.toThrow()
    }
  })

  it('includes missing cities and preserves previous timezone names', () => {
    const values = timeZoneOptions.map((option) => option.value)
    expect(values).toEqual(
      expect.arrayContaining([
        'Europe/Amsterdam',
        'Europe/Brussels',
        'Europe/Zurich',
        'Europe/Lisbon',
        'Asia/Kathmandu',
        'Asia/Kolkata',
        'America/Buenos_Aires',
        'America/Argentina/Buenos_Aires',
        'Antarctica/Casey',
        'Asia/Karachi',
        'Australia/Adelaide',
        'America/Phoenix'
      ])
    )
  })
})
