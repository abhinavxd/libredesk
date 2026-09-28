import { describe, expect, it } from 'vitest'
import { formatChannelIdentity } from './channel-identity'

describe('formatChannelIdentity', () => {
  it('shows the Telegram username with the chat ID', () => {
    const contact = { custom_attributes: { telegram_username: 'jane' } }
    expect(formatChannelIdentity({ channel: 'telegram', identifier: '42' }, contact)).toBe(
      '@jane (42)'
    )
  })
  it('falls back to the Telegram chat ID without a username', () => {
    expect(formatChannelIdentity({ channel: 'telegram', identifier: '42' }, {})).toBe('42')
    expect(formatChannelIdentity({ channel: 'telegram', identifier: '42' })).toBe('42')
  })
  it('prefixes WhatsApp numbers with a plus', () => {
    expect(formatChannelIdentity({ channel: 'whatsapp', identifier: '15550100' }, {})).toBe(
      '+15550100'
    )
  })
  it('shows other identifiers as they are', () => {
    const contact = { custom_attributes: { telegram_username: 'jane' } }
    expect(formatChannelIdentity({ channel: 'livechat', identifier: 'abc' }, contact)).toBe('abc')
  })
})
