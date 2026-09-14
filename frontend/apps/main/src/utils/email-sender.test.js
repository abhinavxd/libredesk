import { describe, expect, test } from 'vitest'
import { resolveEmailSender } from './email-sender'

const owned = ['support@example.com', 'billing@example.com']

describe('resolveEmailSender', () => {
  test('uses the resolved incoming inbox address', () => {
    expect(resolveEmailSender({ type: 'incoming', meta: { inbox_address: 'billing@example.com' } }, owned))
      .toBe('billing@example.com')
  })

  test('preserves the latest outgoing selection', () => {
    expect(resolveEmailSender({ type: 'outgoing', meta: { send_from: 'billing@example.com' } }, owned))
      .toBe('billing@example.com')
  })

  test('rejects stale or arbitrary metadata and falls back to primary', () => {
    expect(resolveEmailSender({ type: 'outgoing', meta: { send_from: 'attacker@example.net' } }, owned))
      .toBe('support@example.com')
  })
})
