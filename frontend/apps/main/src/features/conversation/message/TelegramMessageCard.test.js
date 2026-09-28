// @vitest-environment jsdom
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { createApp, h, nextTick } from 'vue'
import TelegramMessageCard from './TelegramMessageCard.vue'

const { save, push, emit, can } = vi.hoisted(() => ({
  save: vi.fn(),
  push: vi.fn(),
  emit: vi.fn(),
  can: vi.fn()
}))
vi.mock('@main/api', () => ({ default: { saveTelegramContact: save } }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push }) }))
vi.mock('@main/stores/user', () => ({ useUserStore: () => ({ can }) }))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit }) }))
vi.mock('@shared-ui/utils/http', () => ({ handleHTTPError: (error) => error }))
vi.mock('@shared-ui/components/ui/button', () => ({
  Button: {
    setup:
      (_, { slots }) =>
      () =>
        h('button', slots.default?.())
  }
}))

let app, root
const mount = (meta) => {
  app = createApp(TelegramMessageCard, {
    message: { uuid: 'message', conversation_uuid: 'conversation', meta }
  })
  app.config.globalProperties.$t = (key) => key
  app.mount(root)
}
const contact = { first_name: 'Alice', last_name: 'Example', phone_number: '+123456789' }
beforeEach(() => {
  vi.clearAllMocks()
  can.mockReturnValue(true)
  save.mockResolvedValue({ data: { data: { id: 42 } } })
  root = document.createElement('div')
  document.body.append(root)
})
afterEach(() => {
  app?.unmount()
  root.remove()
})

it('saves a shared contact and opens the returned contact', async () => {
  mount({ telegram_contact: contact })
  expect(root.textContent).toContain('Alice Example')
  expect(root.textContent).toContain('+123456789')
  expect(root.querySelector('button').type).toBe('button')
  root.querySelector('button').click()
  await vi.waitFor(() =>
    expect(push).toHaveBeenCalledWith({ name: 'contact-detail', params: { id: 42 } })
  )
  expect(save).toHaveBeenCalledWith('conversation', 'message')
})
it('hides contact saving without write permission', () => {
  can.mockReturnValue(false)
  mount({ telegram_contact: contact })
  expect(can).toHaveBeenCalledWith('contacts:write')
  expect(root.querySelector('button')).toBeNull()
})
it('prevents duplicate saves while the request is pending', async () => {
  let resolve
  save.mockImplementation(
    () =>
      new Promise((done) => {
        resolve = done
      })
  )
  mount({ telegram_contact: contact })
  root.querySelector('button').click()
  root.querySelector('button').click()
  await nextTick()
  expect(save).toHaveBeenCalledOnce()
  expect(root.querySelector('button').disabled).toBe(true)
  resolve({ data: { data: { id: 42 } } })
  await vi.waitFor(() => expect(root.querySelector('button').disabled).toBe(false))
})
it('shows a failed save and allows retry', async () => {
  save.mockRejectedValueOnce(new Error('Unable to save contact. Try again.'))
  mount({ telegram_contact: contact })
  root.querySelector('button').click()
  await vi.waitFor(() =>
    expect(emit).toHaveBeenCalledWith(expect.any(String), {
      variant: 'destructive',
      description: 'Unable to save contact. Try again.'
    })
  )
  expect(push).not.toHaveBeenCalled()
  root.querySelector('button').click()
  await vi.waitFor(() => expect(push).toHaveBeenCalledOnce())
})
it('renders a venue and its coordinates as an external map link', () => {
  mount({
    telegram_location: {
      title: 'Office',
      address: 'Main Street',
      location: { latitude: 12.5, longitude: -4.2 }
    }
  })
  expect(root.textContent).toContain('Office')
  expect(root.textContent).toContain('Main Street')
  const link = root.querySelector('a')
  expect(link.href).toBe('https://maps.google.com/?q=12.5,-4.2')
  expect(link.rel).toBe('noopener noreferrer')
  expect(link.target).toBe('_blank')
})
it('renders a plain location at zero coordinates', () => {
  mount({ telegram_location: { location: { latitude: 0, longitude: 0 } } })
  expect(root.textContent).toContain('globals.terms.location')
  expect(root.querySelector('a').href).toBe('https://maps.google.com/?q=0,0')
})
