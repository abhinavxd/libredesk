// @vitest-environment jsdom
import { describe, expect, test } from 'vitest'
import {
  TURNSTILE_FAILED,
  attachTurnstileToken,
  needsTurnstileToken,
  requestTurnstileToken
} from './turnstile.js'

describe('needsTurnstileToken', () => {
  test('asks a visitor when a site key is set', () => {
    expect(needsTurnstileToken('site', true)).toBe(true)
    expect(needsTurnstileToken('site', undefined)).toBe(true)
  })

  test('skips signed-in users and a blank site key', () => {
    expect(needsTurnstileToken('site', false)).toBe(false)
    expect(needsTurnstileToken('  ', true)).toBe(false)
    expect(needsTurnstileToken('', true)).toBe(false)
  })
})

describe('requestTurnstileToken', () => {
  test('resolves the token from a fresh render and removes the widget', async () => {
    const removed = []
    const turnstile = {
      render(el, options) {
        options.callback('token-1')
        return 'widget-1'
      },
      remove(id) {
        removed.push(id)
      }
    }

    await expect(requestTurnstileToken('site', { load: async () => turnstile })).resolves.toBe('token-1')
    expect(removed).toEqual(['widget-1'])
    expect(document.body.querySelector('div')).toBeNull()
  })

  test('rejects when the widget reports an error', async () => {
    const turnstile = {
      render(el, options) {
        options['error-callback']()
        return 'widget-1'
      },
      remove() {}
    }

    await expect(requestTurnstileToken('site', { load: async () => turnstile })).rejects.toThrow(TURNSTILE_FAILED)
  })
})

describe('attachTurnstileToken', () => {
  test('adds a token for a visitor and leaves a signed-in payload alone', async () => {
    window.turnstile = {
      render(el, options) {
        options.callback('token-1')
        return 'widget-1'
      },
      remove() {}
    }

    const visitor = await attachTurnstileToken({ message: 'hi' }, { siteKey: 'site', isVisitor: true })
    expect(visitor).toEqual({ message: 'hi', turnstile_token: 'token-1' })

    const signedIn = await attachTurnstileToken({ message: 'hi' }, { siteKey: 'site', isVisitor: false })
    expect(signedIn).toEqual({ message: 'hi' })

    delete window.turnstile
  })
})
