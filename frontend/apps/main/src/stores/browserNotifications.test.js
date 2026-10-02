// @vitest-environment jsdom
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia, disposePinia } from 'pinia'
import { nextTick, reactive } from 'vue'

const mocks = vi.hoisted(() => ({ user: null, push: vi.fn() }))
vi.mock('./user', () => ({ useUserStore: () => mocks.user }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: mocks.push }) }))
vi.mock('@main/i18n', () => ({ getI18n: () => ({ global: { t: (key) => key } }) }))
import { useBrowserNotificationsStore, mailNotificationContent } from './browserNotifications'

let pinia
let NativeNotification
const incoming = {
  uuid: 'message-1',
  conversation_uuid: 'conversation-1',
  type: 'incoming',
  sender_type: 'contact',
  private: false,
  sender: { first_name: 'Maya', last_name: 'Chen', email: 'maya@example.test' },
  conversation: { correspondent: { first_name: 'Original', last_name: 'Sender' } },
  preview: 'The press kit is ready.\n Can you review it?'
}

beforeEach(() => {
  localStorage.clear()
  mocks.user = reactive({ userID: 42 })
  mocks.push.mockReset()
  NativeNotification = vi.fn(function (title, options) {
    this.title = title
    this.options = options
    this.close = vi.fn()
  })
  NativeNotification.permission = 'default'
  NativeNotification.requestPermission = vi.fn(async () => {
    NativeNotification.permission = 'granted'
    return 'granted'
  })
  vi.stubGlobal('isSecureContext', true)
  vi.stubGlobal('Notification', NativeNotification)
  vi.spyOn(window, 'focus').mockImplementation(() => {})
  pinia = createPinia()
  setActivePinia(pinia)
})

afterEach(() => {
  disposePinia(pinia)
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('browser mail notifications', () => {
  it('requires opt-in, then uses the actual sender and plain message preview', async () => {
    const store = useBrowserNotificationsStore()
    store.notifyNewMessage(incoming)
    expect(NativeNotification).not.toHaveBeenCalled()
    expect(NativeNotification.requestPermission).not.toHaveBeenCalled()
    await store.enable()
    store.notifyNewMessage(incoming)
    expect(NativeNotification).toHaveBeenCalledWith(
      'Maya Chen',
      expect.objectContaining({
        body: 'The press kit is ready. Can you review it?',
        tag: 'libredesk-mail:42:message-1'
      })
    )
    expect(JSON.parse(localStorage.getItem('libredesk.mail-notifications'))).toEqual({ 42: true })
  })

  it('opens the notified conversation when clicked', async () => {
    const store = useBrowserNotificationsStore()
    await store.enable()
    store.notifyNewMessage(incoming)
    const notification = NativeNotification.mock.instances[0]
    notification.onclick()
    expect(window.focus).toHaveBeenCalledOnce()
    expect(notification.close).toHaveBeenCalledOnce()
    expect(mocks.push).toHaveBeenCalledWith({
      name: 'inbox-conversation',
      params: { type: 'all', uuid: 'conversation-1' }
    })
  })

  it('ignores outgoing mail, notes, activity, malformed events and duplicates', async () => {
    const store = useBrowserNotificationsStore()
    await store.enable()
    for (const message of [
      null,
      {},
      { ...incoming, type: 'outgoing' },
      { ...incoming, private: true },
      { ...incoming, sender_type: 'agent' },
      { ...incoming, type: 'activity' }
    ])
      store.notifyNewMessage(message)
    expect(NativeNotification).not.toHaveBeenCalled()
    store.notifyNewMessage(incoming)
    store.notifyNewMessage(incoming)
    expect(NativeNotification).toHaveBeenCalledOnce()
  })

  it.each(['denied', 'default'])('does not opt in when permission is %s', async (permission) => {
    NativeNotification.requestPermission.mockResolvedValue(permission)
    const store = useBrowserNotificationsStore()
    await store.enable()
    store.notifyNewMessage(incoming)
    expect(store.enabled).toBe(false)
    expect(NativeNotification).not.toHaveBeenCalled()
  })

  it('rechecks permissions revoked outside the app and survives native errors', async () => {
    const store = useBrowserNotificationsStore()
    await store.enable()
    NativeNotification.permission = 'denied'
    store.notifyNewMessage(incoming)
    expect(NativeNotification).not.toHaveBeenCalled()
    NativeNotification.permission = 'granted'
    NativeNotification.mockImplementationOnce(function () {
      throw new TypeError('unsupported')
    })
    expect(() => store.notifyNewMessage(incoming)).not.toThrow()
    expect(store.error).toBe(true)
  })

  it('handles unsupported or insecure contexts without requesting permission', async () => {
    vi.stubGlobal('isSecureContext', false)
    const store = useBrowserNotificationsStore()
    await store.enable()
    store.notifyNewMessage(incoming)
    expect(store.supported).toBe(false)
    expect(NativeNotification.requestPermission).not.toHaveBeenCalled()
    expect(NativeNotification).not.toHaveBeenCalled()
  })

  it('closes alerts on disable and does not carry opt-in to another account', async () => {
    const store = useBrowserNotificationsStore()
    await store.enable()
    store.notifyNewMessage(incoming)
    store.disable()
    await nextTick()
    expect(NativeNotification.mock.instances[0].close).toHaveBeenCalled()
    store.notifyNewMessage({ ...incoming, uuid: 'message-2' })
    expect(NativeNotification).toHaveBeenCalledOnce()
    await store.enable()
    mocks.user.userID = 43
    await nextTick()
    expect(store.enabled).toBe(false)
    store.notifyNewMessage({ ...incoming, uuid: 'message-3' })
    expect(NativeNotification).toHaveBeenCalledOnce()
  })

  it('does not opt in another account when permission resolves after an account change', async () => {
    let resolve
    NativeNotification.requestPermission.mockImplementation(
      () =>
        new Promise((r) => {
          resolve = r
        })
    )
    const store = useBrowserNotificationsStore()
    const enabling = store.enable()
    mocks.user.userID = 43
    resolve('granted')
    await enabling
    expect(store.enabled).toBe(false)
  })

  it('bounds long previews and falls back for nameless and attachment-only messages', () => {
    expect(
      mailNotificationContent({ ...incoming, preview: 'x'.repeat(300) }, 'New mail', 'Attachment')
        .body
    ).toHaveLength(241)
    expect(
      mailNotificationContent(
        { ...incoming, sender: { email: 'mail@example.test' }, preview: '' },
        'New mail',
        'Attachment'
      )
    ).toEqual({ title: 'mail@example.test', body: 'Attachment' })
  })
})
