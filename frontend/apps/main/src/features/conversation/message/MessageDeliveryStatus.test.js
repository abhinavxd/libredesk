// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest'
import { createApp, h, nextTick } from 'vue'
import MessageDeliveryStatus from './MessageDeliveryStatus.vue'
import api from '@main/api'
const { fetchMessage } = vi.hoisted(() => ({ fetchMessage: vi.fn() }))
vi.mock('@main/api', () => ({ default: { retryMessage: vi.fn() } }))
vi.mock('@main/stores/user', () => ({ useUserStore: () => ({ userID: 1 }) }))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => ({ fetchMessage }) }))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit: vi.fn() }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
let app, root
const settle = async () => {
  for (let i = 0; i < 10; i++) await nextTick()
}
afterEach(() => {
  app?.unmount()
  root?.remove()
  vi.clearAllMocks()
})
function mount(uncertain) {
  root = document.createElement('div')
  document.body.append(root)
  app = createApp({
    render: () =>
      h(MessageDeliveryStatus, {
        message: {
          uuid: 'mail-A',
          conversation_uuid: 'thread-A',
          sender_id: 1,
          status: 'failed',
          meta: { delivery_uncertain: uncertain }
        }
      })
  })
  app.mount(root)
}
it('requires explicit confirmation before retrying possibly delivered mail', async () => {
  mount(true)
  expect(root.textContent).toContain('conversation.deliveryUncertain')
  root.querySelector('button').click()
  await settle()
  expect(api.retryMessage).not.toHaveBeenCalled()
  const dialog = document.querySelector('[role="alertdialog"]')
  expect(dialog.textContent).toContain('conversation.retryUncertainDescription')
  const confirm = [...dialog.querySelectorAll('button')].find((button) =>
    button.textContent.includes('conversation.retryDelivery')
  )
  confirm.click()
  await settle()
  expect(api.retryMessage).toHaveBeenCalledWith('thread-A', 'mail-A')
  expect(fetchMessage).toHaveBeenCalledWith('thread-A', 'mail-A')
})
it('retries an unambiguously failed message on the original conversation', async () => {
  mount(false)
  root.querySelector('button').click()
  await settle()
  expect(api.retryMessage).toHaveBeenCalledWith('thread-A', 'mail-A')
})
