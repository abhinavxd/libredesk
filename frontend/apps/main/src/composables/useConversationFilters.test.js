import { describe, expect, it, vi } from 'vitest'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('@/stores/conversation', () => ({
  useConversationStore: () => ({ statusOptions: [], priorityOptions: [] })
}))
vi.mock('@/stores/inbox', () => ({ useInboxStore: () => ({ options: [] }) }))
vi.mock('@/stores/sla', () => ({ useSlaStore: () => ({ options: [] }) }))
vi.mock('@/stores/customAttributes', () => ({
  useCustomAttributeStore: () => ({
    contactAttributeOptions: [
      { key: 'shared', label: 'Contact plan', data_type: 'text', values: [] }
    ],
    conversationAttributeOptions: [
      { key: 'shared', label: 'Conversation plan', data_type: 'checkbox', values: [] }
    ]
  })
}))

import { useConversationFilters } from './useConversationFilters'

describe('automation filter configuration', () => {
  it('keeps contact and conversation attributes with identical keys separate', () => {
    const { contactCustomAttributes, conversationCustomAttributes } = useConversationFilters()
    expect(contactCustomAttributes.value.shared.type).toBe('text')
    expect(conversationCustomAttributes.value.shared.type).toBe('boolean')
    expect(contactCustomAttributes.value.shared.label).toBe('Contact plan')
    expect(conversationCustomAttributes.value.shared.label).toBe('Conversation plan')
  })

  it('exposes message and contact conditions on updates without exposing transient recipients', () => {
    const { conversationFilters, newConversationFilters } = useConversationFilters()
    for (const field of [
      'subject',
      'content',
      'contact_email',
      'channel',
      'tags',
      'within_business_hours'
    ]) {
      expect(conversationFilters.value[field]).toBeDefined()
      expect(newConversationFilters.value[field]).toBeDefined()
    }
    expect(newConversationFilters.value.to).toBeDefined()
    expect(conversationFilters.value.to).toBeUndefined()
  })

  it('offers only stored status categories and supported channels', () => {
    const { conversationFilters } = useConversationFilters()
    expect(conversationFilters.value.status_category.options.map((option) => option.value)).toEqual(
      ['open', 'waiting', 'resolved']
    )
    expect(conversationFilters.value.channel.options.map((option) => option.value)).toEqual([
      'email',
      'livechat',
      'whatsapp'
    ])
  })

  it('allows checking missing time values and missing ratings explicitly', () => {
    const { conversationFilters } = useConversationFilters()
    for (const field of [
      'hours_waiting',
      'hours_since_last_message',
      'hours_since_last_incoming',
      'hours_since_last_activity',
      'hours_until_sla_deadline',
      'csat_rating'
    ]) {
      expect(conversationFilters.value[field].operators).toContain('set')
      expect(conversationFilters.value[field].operators).toContain('not set')
    }
  })
})
