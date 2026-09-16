// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick, ref } from 'vue'

const { getPreferences, savePreferences, refresh, enable, disable, emit } = vi.hoisted(() => ({
  getPreferences: vi.fn(),
  savePreferences: vi.fn().mockResolvedValue({}),
  refresh: vi.fn(),
  enable: vi.fn().mockResolvedValue(true),
  disable: vi.fn().mockResolvedValue(),
  emit: vi.fn()
}))
vi.mock('@/api', () => ({
  default: {
    getNotificationPreferences: getPreferences,
    updateNotificationPreferences: savePreferences
  }
}))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('@/composables/useEmitter', () => ({ useEmitter: () => ({ emit }) }))
vi.mock('@shared-ui/utils/http.js', () => ({ handleHTTPError: (error) => error }))
vi.mock('@/composables/usePushNotifications', () => ({
  usePushNotifications: () => ({
    supported: ref(true),
    enabled: ref(false),
    permission: ref('granted'),
    refresh,
    enable,
    disable
  })
}))
vi.mock('@shared-ui/components/ui/switch', () => ({
  Switch: {
    props: ['checked', 'disabled'],
    emits: ['update:checked'],
    setup:
      (props, { emit }) =>
      () =>
        h('button', {
          type: 'button',
          disabled: props.disabled,
          onClick: () => emit('update:checked', !props.checked)
        })
  }
}))
import NotificationPreferences from './NotificationPreferences.vue'

let app
let root
const settle = async () => {
  for (let i = 0; i < 10; i++) await nextTick()
}
afterEach(() => {
  app?.unmount()
  root?.remove()
  vi.clearAllMocks()
})

const mount = async (vapidPublicKey = 'test-key') => {
  getPreferences.mockResolvedValue({
    data: {
      data: {
        email_enabled: true,
        vapid_public_key: vapidPublicKey,
        push_endpoints: ['https://push.example/subscription'],
        preferences: [
          { notification_type: 'new_reply', channel: 'email', enabled: true },
          { notification_type: 'new_reply', channel: 'in_app', enabled: false }
        ]
      }
    }
  })
  root = document.createElement('div')
  document.body.append(root)
  app = createApp(NotificationPreferences)
  app.config.globalProperties.$t = (key) => key
  app.mount(root)
  await settle()
}

describe('notification preferences', () => {
  it('shows a loader until preferences are loaded', async () => {
    let finishLoad
    getPreferences.mockReturnValueOnce(
      new Promise((resolve) => {
        finishLoad = resolve
      })
    )
    root = document.createElement('div')
    document.body.append(root)
    app = createApp(NotificationPreferences)
    app.config.globalProperties.$t = (key) => key
    app.mount(root)
    await nextTick()

    expect(root.querySelector('[role="status"]')).not.toBeNull()
    expect(root.querySelectorAll('button')).toHaveLength(0)

    finishLoad({
      data: {
        data: {
          email_enabled: true,
          vapid_public_key: 'test-key',
          push_endpoints: [],
          preferences: []
        }
      }
    })
    await settle()

    expect(root.querySelector('[role="status"]')).toBeNull()
    expect(root.querySelectorAll('button')).toHaveLength(1)
  })

  it('saves a channel toggle', async () => {
    await mount()
    const email = root.querySelectorAll('button')[2]
    expect(email).toBeDefined()
    email.click()
    await settle()
    expect(savePreferences).toHaveBeenCalledWith([
      { notification_type: 'new_reply', channel: 'email', enabled: false }
    ])
  })

  it('prevents overlapping saves for the same preference', async () => {
    let finishSave
    savePreferences.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finishSave = resolve
        })
    )
    await mount()
    const switches = root.querySelectorAll('button')
    const inApp = switches[1]
    const email = switches[2]

    email.click()
    email.click()
    await nextTick()

    expect(savePreferences).toHaveBeenCalledTimes(1)
    expect(email.disabled).toBe(true)
    expect(inApp.disabled).toBe(false)

    inApp.click()
    await settle()
    expect(savePreferences).toHaveBeenCalledTimes(2)

    finishSave({})
    await settle()
    expect(email.disabled).toBe(false)
  })

  it('restores this browser push state from the saved endpoints', async () => {
    await mount()
    expect(refresh).toHaveBeenCalledWith('test-key', ['https://push.example/subscription'])
  })

  it('disables browser notifications when push is unavailable', async () => {
    await mount('')
    const browserPush = root.querySelector('button')
    expect(browserPush.disabled).toBe(true)
    expect(root.textContent).toContain('notification.browserPush.unavailable')
  })

  it('shows a loader while browser notifications are being enabled', async () => {
    let finishEnable
    enable.mockReturnValueOnce(
      new Promise((resolve) => {
        finishEnable = resolve
      })
    )
    await mount()

    root.querySelector('button').click()
    await nextTick()

    expect(root.querySelector('.animate-dot-flashing')).not.toBeNull()
    finishEnable(true)
    await settle()
    expect(root.querySelector('.animate-dot-flashing')).toBeNull()
  })
})
