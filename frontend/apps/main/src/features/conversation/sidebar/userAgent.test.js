import { describe, test, expect } from 'vitest'
import { formatUserAgent } from './userAgent.js'

describe('formatUserAgent', () => {
  test('formats an app product token', () => {
    expect(formatUserAgent('ExampleApp/1.0.0 (android)')).toEqual({
      label: 'ExampleApp 1.0.0 · android',
      isMobile: true
    })
  })

  test('flags desktop platforms as not mobile', () => {
    expect(formatUserAgent('ExampleApp/1.2.3 (web)')).toEqual({
      label: 'ExampleApp 1.2.3 · web',
      isMobile: false
    })
  })

  test('handles a multi-word app name', () => {
    expect(formatUserAgent('Example Desk App/2.0 (ios)')).toEqual({
      label: 'Example Desk App 2.0 · ios',
      isMobile: true
    })
  })

  test('keeps a browser UA as raw text', () => {
    const ua =
      'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36'
    expect(formatUserAgent(ua)).toEqual({ label: ua, isMobile: false })
  })

  test('does not mistake a short browser UA for an app', () => {
    const ua = 'Mozilla/5.0 (Windows NT 10.0)'
    expect(formatUserAgent(ua)).toEqual({ label: ua, isMobile: false })
  })

  test('detects a mobile browser UA', () => {
    expect(
      formatUserAgent('Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 Chrome/120 Mobile')
    ).toMatchObject({ isMobile: true })
  })

  test('shows nothing for machine agents', () => {
    expect(formatUserAgent('curl/8.5.0')).toBeNull()
    expect(formatUserAgent('Wget/1.21.4')).toBeNull()
    expect(formatUserAgent('python-requests/2.31.0')).toBeNull()
    expect(formatUserAgent('ExampleBot/1.0 (+https://example.com/bot)')).toBeNull()
    expect(formatUserAgent('Mozilla/5.0 (compatible; examplebot/2.1)')).toBeNull()
  })

  test('shows nothing when there is no UA', () => {
    expect(formatUserAgent('')).toBeNull()
    expect(formatUserAgent('   ')).toBeNull()
    expect(formatUserAgent(null)).toBeNull()
    expect(formatUserAgent(undefined)).toBeNull()
  })
})
