// @vitest-environment jsdom
import { describe, test, expect, afterEach } from 'vitest'
import { websocketURL } from './websocket'

describe('websocketURL', () => {
  afterEach(() => {
    document.head.innerHTML = ''
  })

  test('returns the path when no websocket URL is set', () => {
    expect(websocketURL('/ws')).toBe('/ws')
  })

  test('prefixes the path with the websocket URL from the page', () => {
    document.head.innerHTML = '<meta name="libredesk-ws-url" content="wss://desk.example.com:8443">'
    expect(websocketURL('/widget/ws')).toBe('wss://desk.example.com:8443/widget/ws')
  })
})
