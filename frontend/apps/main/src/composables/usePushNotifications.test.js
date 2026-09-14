import { describe, expect, it, vi } from 'vitest'
import { createPushNotifications } from './usePushNotifications'

const makeBrowser = ({ permission = 'default', subscription = null } = {}) => {
  const pushManager = {
    getSubscription: vi.fn().mockResolvedValue(subscription),
    subscribe: vi.fn()
  }
  const registration = { pushManager }
  const browser = {
    Notification: {
      permission,
      requestPermission: vi.fn().mockResolvedValue(permission)
    },
    navigator: {
      serviceWorker: {
        register: vi.fn().mockResolvedValue(registration),
        getRegistration: vi.fn().mockResolvedValue(registration),
        ready: Promise.resolve(registration)
      }
    },
    PushManager: function PushManager () {}
  }
  return { browser, registration, pushManager }
}

const subscription = {
  endpoint: 'https://push.example/subscription',
  toJSON: () => ({
    endpoint: 'https://push.example/subscription',
    keys: { p256dh: 'public-key', auth: 'auth-secret' }
  }),
  unsubscribe: vi.fn().mockResolvedValue(true)
}

describe('createPushNotifications', () => {
  it('reports unsupported browsers without registering a worker', async () => {
    const api = { createPushSubscription: vi.fn() }
    const state = createPushNotifications(api, {})

    await state.refresh('vapid-key')

    expect(state.supported.value).toBe(false)
    expect(api.createPushSubscription).not.toHaveBeenCalled()
  })

  it('refreshes an existing subscription without requesting permission', async () => {
    const { browser } = makeBrowser({ permission: 'granted', subscription })
    const api = { createPushSubscription: vi.fn().mockResolvedValue({}) }
    const state = createPushNotifications(api, browser)

    await state.refresh('vapid-key')

    expect(browser.Notification.requestPermission).not.toHaveBeenCalled()
    expect(api.createPushSubscription).toHaveBeenCalledWith({
      endpoint: subscription.endpoint,
      p256dh: 'public-key',
      auth: 'auth-secret'
    })
    expect(state.enabled.value).toBe(true)
  })

  it('subscribes only after permission is granted', async () => {
    const { browser, pushManager } = makeBrowser({ permission: 'default' })
    browser.Notification.requestPermission.mockResolvedValue('granted')
    pushManager.subscribe.mockResolvedValue(subscription)
    const api = { createPushSubscription: vi.fn().mockResolvedValue({}) }
    const state = createPushNotifications(api, browser)

    await state.enable('AQID')

    expect(pushManager.subscribe).toHaveBeenCalledWith({
      userVisibleOnly: true,
      applicationServerKey: new Uint8Array([1, 2, 3])
    })
    expect(state.enabled.value).toBe(true)
  })

  it('does not subscribe when permission is denied', async () => {
    const { browser, pushManager } = makeBrowser({ permission: 'default' })
    browser.Notification.requestPermission.mockResolvedValue('denied')
    const state = createPushNotifications({}, browser)

    await state.enable('AQID')

    expect(pushManager.subscribe).not.toHaveBeenCalled()
    expect(state.permission.value).toBe('denied')
  })

  it('waits for the first worker to become active before subscribing', async () => {
    const { browser, registration, pushManager } = makeBrowser({ permission: 'granted' })
    const { promise, resolve } = Promise.withResolvers()
    browser.navigator.serviceWorker.ready = promise
    pushManager.subscribe.mockResolvedValue(subscription)
    const api = { createPushSubscription: vi.fn().mockResolvedValue({}) }
    const state = createPushNotifications(api, browser)

    const enabling = state.enable('AQID')
    await vi.waitFor(() => expect(browser.navigator.serviceWorker.register).toHaveBeenCalledOnce())

    expect(pushManager.subscribe).not.toHaveBeenCalled()
    expect(api.createPushSubscription).not.toHaveBeenCalled()
    expect(state.enabled.value).toBe(false)

    resolve(registration)
    await expect(enabling).resolves.toBe(true)
    expect(pushManager.subscribe).toHaveBeenCalledOnce()
    expect(api.createPushSubscription).toHaveBeenCalledOnce()
    expect(state.enabled.value).toBe(true)
  })

  it('unsubscribes the current browser and deletes its endpoint', async () => {
    const { browser } = makeBrowser({ permission: 'granted', subscription })
    const api = {
      createPushSubscription: vi.fn().mockResolvedValue({}),
      deletePushSubscription: vi.fn().mockResolvedValue({})
    }
    const state = createPushNotifications(api, browser)
    await state.refresh('vapid-key')

    await state.disable()

    expect(subscription.unsubscribe).toHaveBeenCalled()
    expect(api.deletePushSubscription).toHaveBeenCalledWith(subscription.endpoint)
    expect(state.enabled.value).toBe(false)
  })

  it('unsubscribes the browser when clearing the server association fails', async () => {
    subscription.unsubscribe.mockClear()
    const { browser } = makeBrowser({ permission: 'granted', subscription })
    const api = { deletePushSubscription: vi.fn().mockRejectedValue(new Error('offline')) }
    const state = createPushNotifications(api, browser)

    await expect(state.disable()).rejects.toThrow('offline')

    expect(subscription.unsubscribe).toHaveBeenCalledOnce()
  })
})
