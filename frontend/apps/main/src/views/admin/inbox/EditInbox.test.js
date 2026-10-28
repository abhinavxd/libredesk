// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest'
import { createApp, h, nextTick } from 'vue'

const { getInbox, updateInbox, verifyInboxAlias, fetchInboxes, form } = vi.hoisted(() => ({
  getInbox: vi.fn(),
  updateInbox: vi.fn(),
  verifyInboxAlias: vi.fn().mockResolvedValue({}),
  fetchInboxes: vi.fn().mockResolvedValue(),
  form: { props: null }
}))
vi.mock('@/api', () => ({
  default: {
    getInbox,
    updateInbox,
    verifyInboxAlias,
    getAvailableLanguages: vi.fn().mockResolvedValue({ data: { data: [] } })
  }
}))
vi.mock('@/stores/inbox', () => ({ useInboxStore: () => ({ fetchInboxes }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('@/composables/useEmitter', () => ({ useEmitter: () => ({ emit: vi.fn() }) }))
vi.mock('@/features/admin/inbox/EmailInboxForm.vue', () => ({
  default: {
    props: ['initialValues', 'submitForm', 'verifyAlias', 'aliasVerificationState', 'isLoading'],
    setup(props) {
      form.props = props
      return () => h('div')
    }
  }
}))
vi.mock('@/features/admin/inbox/LivechatInboxForm.vue', () => ({ default: { render: () => null } }))
vi.mock('@/features/admin/inbox/WhatsAppInboxForm.vue', () => ({ default: { render: () => null } }))
vi.mock('@shared-ui/components/ui/breadcrumb/index.js', () => ({
  CustomBreadcrumb: { render: () => null }
}))
vi.mock('@/components/button/CopyButton.vue', () => ({ default: { render: () => null } }))

import EditInbox from './EditInbox.vue'

let app
const alias = (email, verification_status) => ({ email, verification_status })
const response = (aliases) => ({
  data: {
    data: {
      id: 1,
      channel: 'email',
      aliases,
      config: { imap: [{ read_interval: '10m' }], smtp: [{}] }
    }
  }
})
const settle = async () => {
  for (let i = 0; i < 10; i++) await nextTick()
}
const mount = async (aliases) => {
  vi.useFakeTimers()
  getInbox.mockResolvedValue(response(aliases))
  app = createApp(EditInbox, { id: '1' })
  app.config.globalProperties.$t = (key) => key
  app.mount(document.createElement('div'))
  await settle()
}
afterEach(() => {
  app?.unmount()
  vi.useRealTimers()
  vi.clearAllMocks()
})

it('refreshes shared sender data after saving aliases', async () => {
  await mount([alias('billing@example.com', 'verified')])
  updateInbox.mockResolvedValue(response([]))
  form.props.submitForm({ imap: {}, smtp: {}, aliases: [] })
  await settle()
  expect(updateInbox).toHaveBeenCalled()
  expect(fetchInboxes).toHaveBeenCalledWith(true)
  expect(form.props.aliasVerificationState).toEqual({})
})

it('refreshes shared senders as each alias verifies while others remain pending', async () => {
  await mount([alias('billing@example.com', 'pending'), alias('sales@example.com', 'pending')])
  getInbox.mockResolvedValue(
    response([alias('billing@example.com', 'verified'), alias('sales@example.com', 'pending')])
  )
  await vi.advanceTimersByTimeAsync(5000)
  expect(fetchInboxes).toHaveBeenCalledWith(true)
  expect(form.props.aliasVerificationState['billing@example.com'].verification_status).toBe(
    'verified'
  )
  expect(vi.getTimerCount()).toBe(1)
  getInbox.mockResolvedValue(
    response([alias('billing@example.com', 'verified'), alias('sales@example.com', 'verified')])
  )
  await vi.advanceTimersByTimeAsync(5000)
  expect(fetchInboxes).toHaveBeenCalledTimes(2)
  expect(vi.getTimerCount()).toBe(0)
})

it('keeps polling past ten minutes, retries errors, and stops on completion or unmount', async () => {
  await mount([alias('billing@example.com', 'unverified')])
  getInbox.mockResolvedValue(response([alias('billing@example.com', 'pending')]))
  await form.props.verifyAlias('billing@example.com')
  await vi.advanceTimersByTimeAsync(10 * 60 * 1000)
  expect(getInbox).toHaveBeenCalledTimes(122)
  expect(vi.getTimerCount()).toBe(1)
  getInbox.mockRejectedValueOnce(new Error('temporary network failure'))
  await vi.advanceTimersByTimeAsync(5000)
  expect(vi.getTimerCount()).toBe(1)
  getInbox.mockResolvedValue(response([alias('billing@example.com', 'verified')]))
  await vi.advanceTimersByTimeAsync(5000)
  expect(form.props.aliasVerificationState['billing@example.com'].verification_status).toBe(
    'verified'
  )
  expect(fetchInboxes).toHaveBeenCalledWith(true)
  expect(vi.getTimerCount()).toBe(0)
  getInbox.mockResolvedValue(response([alias('sales@example.com', 'pending')]))
  await form.props.verifyAlias('sales@example.com')
  expect(vi.getTimerCount()).toBe(1)
  app.unmount()
  app = null
  expect(vi.getTimerCount()).toBe(0)
})

it('does not start checking when the page closes before the inbox loads', async () => {
  vi.useFakeTimers()
  let resolveInbox
  getInbox.mockReturnValue(new Promise((resolve) => (resolveInbox = resolve)))
  app = createApp(EditInbox, { id: '1' })
  app.config.globalProperties.$t = (key) => key
  app.mount(document.createElement('div'))
  app.unmount()
  app = null
  resolveInbox(response([alias('billing@example.com', 'pending')]))
  await settle()
  expect(vi.getTimerCount()).toBe(0)
})

it('does not start checking when the page closes while a verification is sending', async () => {
  await mount([alias('billing@example.com', 'not_verified')])
  let resolveVerify
  verifyInboxAlias.mockReturnValueOnce(new Promise((resolve) => (resolveVerify = resolve)))
  getInbox.mockResolvedValue(response([alias('billing@example.com', 'pending')]))
  const verifying = form.props.verifyAlias('billing@example.com')
  app.unmount()
  app = null
  resolveVerify({})
  await verifying
  expect(vi.getTimerCount()).toBe(0)
})

it('refreshes a failed verification without reloading the page', async () => {
  await mount([alias('billing@example.com', 'not_verified')])
  getInbox.mockResolvedValue(response([alias('billing@example.com', 'failed')]))
  verifyInboxAlias.mockRejectedValueOnce(new Error('sending rejected'))
  await form.props.verifyAlias('billing@example.com')
  expect(form.props.aliasVerificationState['billing@example.com'].verification_status).toBe(
    'failed'
  )
  expect(fetchInboxes).toHaveBeenCalledWith(true)
  expect(vi.getTimerCount()).toBe(0)
})

it('removes a previously verified sender when reverification fails', async () => {
  await mount([alias('billing@example.com', 'verified')])
  getInbox.mockResolvedValue(response([alias('billing@example.com', 'failed')]))
  verifyInboxAlias.mockRejectedValueOnce(new Error('sending rejected'))
  await form.props.verifyAlias('billing@example.com')
  expect(form.props.aliasVerificationState['billing@example.com'].verification_status).toBe(
    'failed'
  )
  expect(fetchInboxes).toHaveBeenCalledWith(true)
})
