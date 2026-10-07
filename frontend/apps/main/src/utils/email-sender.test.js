import { describe, expect, test } from 'vitest'
import { resolveEmailSender, sendableAddresses } from './email-sender'

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

describe('sendableAddresses', () => {
  test('matches normalized sender selections to mixed-case address options', () => {
    const addresses = sendableAddresses('Support <Support@Example.COM>', [
      { email: 'Billing@Example.COM', verification_status: 'verified' }
    ])
    expect(addresses).toEqual(['support@example.com', 'billing@example.com'])
    const selected = resolveEmailSender({ type: 'incoming', meta: { inbox_address: 'Support@Example.COM' } }, addresses)
    expect(addresses).toContain(selected)
  })
  test('lists the primary address and verified aliases only', () => {
    const aliases = [
      { email: 'billing@example.com', verification_status: 'verified' },
      { email: 'sales@example.com', verification_status: 'pending' }
    ]
    expect(sendableAddresses('Support <support@example.com>', aliases))
      .toEqual(['support@example.com', 'billing@example.com'])
  })
})
