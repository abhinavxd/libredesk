// @vitest-environment jsdom
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import api from '@widget/api/index.js'
import { useProactiveStore } from './proactive.js'

vi.mock('@widget/api/index.js', () => ({ default: { nextCampaign: vi.fn(), campaignEvent: vi.fn() } }))
vi.mock('@widget/store/widget.js', () => ({ useWidgetStore: () => ({ config: { has_campaigns: true }, isOpen: false }) }))
vi.mock('@widget/store/chat.js', () => ({ useChatStore: () => ({ getConversations: [] }) }))

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  window.history.replaceState({}, '', '/?parent_origin=https://example.com')
})

describe('campaign delivery', () => {
  test('sends the requested page URL with a delayed response', async () => {
    const store = useProactiveStore()
    store.setBrowserKey('00000000-0000-4000-8000-000000000001')
    store.setSessionKey('00000000-0000-4000-8000-000000000002')
    let resolve
    api.nextCampaign.mockImplementationOnce(() => new Promise(done => { resolve = done }))
    const post = vi.spyOn(window.parent, 'postMessage').mockImplementation(() => {})
    const delivery = { id: 'delivery' }

    const request = store.next({ url: 'https://example.com/pricing' }, 7 /* pageGeneration */)
    resolve({ data: { data: delivery } })
    await request

    expect(post).toHaveBeenCalledWith({
      type: 'SHOW_CAMPAIGN', delivery, url: 'https://example.com/pricing', pageGeneration: 7
    }, 'https://example.com')
    await store.event('dropped', delivery.id)
    expect(store.invitation).toBeNull()
    expect(api.campaignEvent).not.toHaveBeenCalled()
  })
})
