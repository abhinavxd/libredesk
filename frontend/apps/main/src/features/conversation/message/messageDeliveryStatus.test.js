import { describe, expect, it } from 'vitest'
import { getMessageDeliveryStatus } from './messageDeliveryStatus.js'

const outgoingMessage = {
  status: 'sent',
  private: false,
  created_at: '2026-09-17T10:00:00Z',
  meta: {}
}

describe('getMessageDeliveryStatus', () => {
  it('shows the WhatsApp delivery lifecycle', () => {
    const conversation = { inbox_channel: 'whatsapp' }

    expect(getMessageDeliveryStatus(outgoingMessage, 'outgoing', conversation)).toBe('sent')
    expect(
      getMessageDeliveryStatus(
        { ...outgoingMessage, meta: { provider_status: 'delivered' } },
        'outgoing',
        conversation
      )
    ).toBe('delivered')
    expect(
      getMessageDeliveryStatus(
        { ...outgoingMessage, meta: { provider_status: 'read' } },
        'outgoing',
        conversation
      )
    ).toBe('read')
  })

  it('keeps live chat read receipts unchanged', () => {
    const unreadConversation = {
      inbox_channel: 'livechat',
      contact_last_seen_at: '2026-09-17T09:59:00Z'
    }
    const readConversation = {
      inbox_channel: 'livechat',
      contact_last_seen_at: '2026-09-17T10:01:00Z'
    }

    expect(getMessageDeliveryStatus(outgoingMessage, 'outgoing', unreadConversation)).toBe('sent')
    expect(getMessageDeliveryStatus(outgoingMessage, 'outgoing', readConversation)).toBe('read')
  })

  it('hides delivery status for messages that were not sent to a contact', () => {
    expect(getMessageDeliveryStatus(outgoingMessage, 'incoming', {})).toBeNull()
    expect(
      getMessageDeliveryStatus({ ...outgoingMessage, status: 'pending' }, 'outgoing', {})
    ).toBeNull()
    expect(
      getMessageDeliveryStatus({ ...outgoingMessage, private: true }, 'outgoing', {})
    ).toBeNull()
  })
})
