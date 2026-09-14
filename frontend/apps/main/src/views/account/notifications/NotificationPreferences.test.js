// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick, ref } from 'vue'

const { getPreferences, savePreferences, refresh, emit } = vi.hoisted(() => ({
  getPreferences: vi.fn(),
  savePreferences: vi.fn().mockResolvedValue({}),
  refresh: vi.fn(),
  emit: vi.fn()
}))
vi.mock('@/api', () => ({ default: { getNotificationPreferences: getPreferences, updateNotificationPreferences: savePreferences } }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('@/composables/useEmitter', () => ({ useEmitter: () => ({ emit }) }))
vi.mock('@shared-ui/utils/http.js', () => ({ handleHTTPError: (error) => error }))
vi.mock('@/composables/usePushNotifications', () => ({
  usePushNotifications: () => ({ supported: ref(true), enabled: ref(false), permission: ref('granted'), refresh })
}))
vi.mock('@shared-ui/components/ui/switch', () => ({
  Switch: {
    props: ['checked', 'disabled'],
    emits: ['update:checked'],
    setup: (props, { emit }) => () => h('button', {
      type: 'button', disabled: props.disabled,
      onClick: () => emit('update:checked', !props.checked)
    })
  }
}))
import NotificationPreferences from './NotificationPreferences.vue'

let app
let root
const settle = async () => { for (let i = 0; i < 10; i++) await nextTick() }
afterEach(() => { app?.unmount(); root?.remove(); vi.clearAllMocks() })

const mount = async () => {
  getPreferences.mockResolvedValue({ data: { data: {
    email_enabled: true, vapid_public_key: 'test-key', preferences: [
      { notification_type: 'new_reply', channel: 'email', enabled: true },
      { notification_type: 'new_reply', channel: 'in_app', enabled: false }
    ]
  } } })
  root = document.createElement('div')
  document.body.append(root)
  app = createApp(NotificationPreferences)
  app.config.globalProperties.$t = (key) => key
  app.mount(root)
  await settle()
}

describe('notification preferences', () => {
  it('saves a channel toggle', async () => {
    await mount()
    const email = root.querySelectorAll('button')[2]
    expect(email).toBeDefined()
    email.click()
    await settle()
    expect(savePreferences).toHaveBeenCalledWith([{ notification_type: 'new_reply', channel: 'email', enabled: false }])
  })

  it('leaves the push subscription to the app shell', async () => {
    await mount()
    expect(root.textContent).toContain('notification.type.newReply')
    expect(refresh).not.toHaveBeenCalled()
  })
})
