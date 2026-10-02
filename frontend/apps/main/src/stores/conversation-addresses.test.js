// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('vue-router', () => ({ useRouter: () => ({ currentRoute: { value: { params: {} } } }) }))
vi.mock('@main/websocket', () => ({
  subscribeToConversation: vi.fn(),
  sendTypingIndicator: vi.fn(),
  subscribeListReplace: vi.fn()
}))
vi.mock('@main/i18n', () => ({ getI18n: () => ({ global: { t: (key) => key } }) }))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit: vi.fn(), on: vi.fn(), off: vi.fn() }) }))

const { addressRequest } = vi.hoisted(() => ({ addressRequest: vi.fn() }))
vi.mock('@main/api', () => ({ default: { getAddressConversations: addressRequest } }))

import { useConversationStore } from './conversation'

const response = (uuid, addressID) => ({
  data: {
    data: {
      results: [{ uuid, address_id: addressID, status: 'Open' }],
      page: 1,
      total: 1,
      total_pages: 1
    }
  }
})

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('separate address lists', () => {
  it('keeps arrivals in their own address and preserves scope on refresh', async () => {
    addressRequest.mockResolvedValue(response('billing', 6))
    const store = useConversationStore()
    await store.fetchConversationsList(false, 6)
    expect(addressRequest).toHaveBeenCalledWith(6, expect.any(Object))
    store.handleConvPush({ uuid: 'support', address_id: 7, status: 'Open' })
    expect(store.conversationsList.map((row) => row.uuid)).toEqual(['billing'])
    store.handleConvPush({ uuid: 'billing-new', address_id: 6, status: 'Open' })
    expect(store.conversationsList.map((row) => row.uuid)).toEqual(['billing-new', 'billing'])
    await store.fetchFirstPageConversations()
    expect(addressRequest.mock.lastCall[0]).toBe(6)
  })

  it('drops stale responses after switching addresses', async () => {
    let finishBilling
    addressRequest.mockImplementationOnce(
      () => new Promise((resolve) => { finishBilling = resolve })
    )
    addressRequest.mockResolvedValueOnce(response('support', 7))
    const store = useConversationStore()
    const billing = store.fetchConversationsList(false, 6)
    await store.fetchConversationsList(false, 7)
    finishBilling(response('billing', 6))
    await billing
    expect(store.conversationsList.map((row) => row.uuid)).toEqual(['support'])
    expect(store.conversations.addressID).toBe(7)
  })
})
