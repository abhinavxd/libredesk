// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick, ref } from 'vue'
import { useConversationRead } from './useConversationRead'

let app
let host
let intersection
let focused
let hidden
let markerBottom
let acknowledge
let conversationUUID
let message
let loading

const settle = async () => {
  for (let i = 0; i < 5; i++) await nextTick()
}

const mount = async () => {
  host = document.createElement('div')
  document.body.append(host)
  app = createApp({
    setup () {
      const root = ref(document.createElement('div'))
      const marker = ref(document.createElement('div'))
      root.value.getBoundingClientRect = () => ({ top: 0, bottom: 500, height: 500 })
      marker.value.getBoundingClientRect = () => ({ top: markerBottom - 1, bottom: markerBottom, height: 1 })
      useConversationRead({ root, marker, conversationUUID, message, loading, acknowledge })
      return () => h('div')
    }
  })
  app.mount(host)
  await settle()
}

beforeEach(() => {
  focused = true
  hidden = false
  markerBottom = 490
  acknowledge = vi.fn().mockResolvedValue()
  conversationUUID = ref('conversation-a')
  message = ref({ uuid: 'message-a', id: 1 })
  loading = ref(false)
  vi.spyOn(document, 'hasFocus').mockImplementation(() => focused)
  vi.spyOn(document, 'hidden', 'get').mockImplementation(() => hidden)
  vi.stubGlobal('IntersectionObserver', class {
    constructor (callback) { intersection = callback }
    observe () {}
    disconnect () {}
  })
})

afterEach(() => {
  app?.unmount()
  host?.remove()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('conversation read acknowledgement', () => {
  it('acknowledges the visible message once', async () => {
    await mount()
    intersection()
    window.dispatchEvent(new Event('focus'))
    await settle()
    expect(acknowledge).toHaveBeenCalledExactlyOnceWith('conversation-a', 'message-a')
  })

  it('keeps a scrolled-up new reply unread until its end is visible', async () => {
    markerBottom = 900
    await mount()
    expect(acknowledge).not.toHaveBeenCalled()
    message.value = { uuid: 'message-b', id: 2 }
    await settle()
    expect(acknowledge).not.toHaveBeenCalled()
    markerBottom = 490
    intersection()
    await settle()
    expect(acknowledge).toHaveBeenCalledExactlyOnceWith('conversation-a', 'message-b')
  })

  it('waits for focus and visibility, even if the conversation is at the bottom', async () => {
    focused = false
    await mount()
    expect(acknowledge).not.toHaveBeenCalled()
    focused = true
    hidden = true
    window.dispatchEvent(new Event('focus'))
    await settle()
    expect(acknowledge).not.toHaveBeenCalled()
    hidden = false
    document.dispatchEvent(new Event('visibilitychange'))
    await settle()
    expect(acknowledge).toHaveBeenCalledExactlyOnceWith('conversation-a', 'message-a')
  })

  it('waits for opening and loading to finish', async () => {
    loading.value = true
    await mount()
    expect(acknowledge).not.toHaveBeenCalled()
    markerBottom = 900
    loading.value = false
    await settle()
    expect(acknowledge).not.toHaveBeenCalled()
  })

  it('never includes a newer unseen reply in an in-flight acknowledgement', async () => {
    let finish
    acknowledge.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
    await mount()
    expect(acknowledge).toHaveBeenCalledWith('conversation-a', 'message-a')
    markerBottom = 900
    message.value = { uuid: 'message-b', id: 2 }
    await settle()
    finish()
    await settle()
    expect(acknowledge).toHaveBeenCalledTimes(1)
    markerBottom = 490
    intersection()
    await settle()
    expect(acknowledge).toHaveBeenLastCalledWith('conversation-a', 'message-b')
  })

  it('does not treat a failed request as a successful acknowledgement', async () => {
    acknowledge.mockRejectedValueOnce(new Error('offline'))
    await mount()
    window.dispatchEvent(new Event('focus'))
    await settle()
    expect(acknowledge).toHaveBeenCalledTimes(2)
  })
})
