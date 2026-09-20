// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick } from 'vue'
import MessageImagePermissions from './MessageImagePermissions.vue'
import api from '@main/api'

vi.mock('@main/api', () => ({ default: { allowMessageImages: vi.fn() } }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: key => key }) }))

let app
let root
afterEach(() => {
  app?.unmount()
  root?.remove()
  vi.clearAllMocks()
})

function mount(display, onUpdated = vi.fn()) {
  root = document.createElement('div')
  document.body.appendChild(root)
  app = createApp({ render: () => h(MessageImagePermissions, {
    message: { uuid: 'message', conversation_uuid: 'conversation', display }, onUpdated
  }) })
  app.mount(root)
}

describe('image permission controls', () => {
  it('offers no overrides under block_all', () => {
    mount({ can_allow: false, blocked_images: 2, sender: 'sender@example.com' })
    expect(root.querySelectorAll('button')).toHaveLength(0)
    expect(api.allowMessageImages).not.toHaveBeenCalled()
  })

  it('waits for a user action and the server result before changing display', async () => {
    let resolve
    api.allowMessageImages.mockImplementation(() => new Promise(r => { resolve = r }))
    const updated = vi.fn()
    mount({ can_allow: true, blocked_images: 2, sender: 'sender@example.com' }, updated)
    expect(api.allowMessageImages).not.toHaveBeenCalled()
    root.querySelectorAll('button')[1].click()
    await nextTick()
    expect(api.allowMessageImages).toHaveBeenCalledWith('conversation', 'message', 'sender')
    expect([...root.querySelectorAll('button')].every(button => button.disabled)).toBe(true)
    expect(updated).not.toHaveBeenCalled()
    resolve({ data: { data: { uuid: 'message', display: { html: '<p>Allowed</p>' } } } })
    await nextTick()
    await nextTick()
    expect(updated).toHaveBeenCalledWith({ uuid: 'message', display: { html: '<p>Allowed</p>' } }, 'sender')
  })

  it('does not offer sender trust without a sender address', () => {
    mount({ can_allow: true, blocked_images: 1 })
    expect(root.querySelectorAll('button')).toHaveLength(1)
  })

  it('allows existing sender trust to be revoked', async () => {
    api.allowMessageImages.mockResolvedValue({ data: { data: {} } })
    mount({ can_allow: true, blocked_images: 0, sender_trusted: true })
    root.querySelector('button').click()
    await nextTick()
    expect(api.allowMessageImages).toHaveBeenCalledWith('conversation', 'message', 'revoke-sender')
  })
})
