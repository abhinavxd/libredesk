// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest'
import { createApp, h, nextTick } from 'vue'
import ResourceMonitor from './ResourceMonitor.vue'
import api from '@main/api'
vi.mock('@main/api', () => ({
  default: {
    getResourceUsage: vi.fn(),
    getResourceLimits: vi.fn(),
    getResourcePolicy: vi.fn(),
    updateResourceLimits: vi.fn(),
    updateResourcePolicy: vi.fn()
  }
}))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('@main/layouts/admin/AdminSplitLayout.vue', () => ({
  default: {
    setup(_, { slots }) {
      return () => h('div', slots.content?.())
    }
  }
}))
vi.mock('@main/composables/useEmitter.js', () => ({ useEmitter: () => ({ emit: vi.fn() }) }))
let app, root
const flush = async () => {
  for (let i = 0; i < 10; i++) await nextTick()
}
afterEach(() => {
  app?.unmount()
  root?.remove()
  vi.useRealTimers()
  vi.clearAllMocks()
})
it('preserves edits across automatic and manual refreshes and saves the draft', async () => {
  vi.useFakeTimers()
  api.getResourceUsage.mockResolvedValue({ data: { data: null } })
  api.getResourceLimits.mockResolvedValue({
    data: { data: { max_incoming_message_size: 100 * 2 ** 20, max_storage_bytes: 0 } }
  })
  const policy = { mode: 'block_all', allowed_domains: [], max_cache_bytes: 10 * 2 ** 30 }
  api.getResourcePolicy.mockResolvedValue({ data: { data: policy } })
  api.updateResourceLimits.mockResolvedValue({})
  api.updateResourcePolicy.mockResolvedValue({ data: { data: policy } })
  root = document.createElement('div')
  document.body.appendChild(root)
  app = createApp(ResourceMonitor)
  app.mount(root)
  await flush()
  const inputs = root.querySelectorAll('input')
  inputs[1].value = '2'
  inputs[1].dispatchEvent(new Event('input', { bubbles: true }))
  await flush()
  await vi.advanceTimersByTimeAsync(60000)
  await flush()
  expect(inputs[1].value).toBe('2')
  root.querySelector('button').click()
  await flush()
  expect(inputs[1].value).toBe('2')
  expect(api.getResourceLimits).toHaveBeenCalledTimes(1)
  root.querySelector('form').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
  await flush()
  expect(api.updateResourceLimits).toHaveBeenCalledWith({
    max_incoming_message_size: 100 * 2 ** 20,
    max_storage_bytes: 2 * 2 ** 30
  })
})

it('saving a budget preserves newer image privacy settings', async () => {
  api.getResourceUsage.mockResolvedValue({ data: { data: null } })
  api.getResourceLimits.mockResolvedValue({
    data: { data: { max_incoming_message_size: 104857600, max_storage_bytes: 0 } }
  })
  let policy = { mode: 'load_on_receipt', allowed_domains: [], max_cache_bytes: 10 * 2 ** 30 }
  api.getResourcePolicy.mockImplementation(async () => ({ data: { data: { ...policy } } }))
  api.updateResourceLimits.mockResolvedValue({})
  api.updateResourcePolicy.mockImplementation(async (fields) => {
    policy = { ...policy, ...fields }
    return { data: { data: policy } }
  })
  root = document.createElement('div')
  document.body.appendChild(root)
  app = createApp(ResourceMonitor)
  app.mount(root)
  await flush()
  policy = { ...policy, mode: 'block_all' }
  root.querySelector('form').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
  await flush()
  expect(api.updateResourcePolicy).toHaveBeenCalledWith({ max_cache_bytes: 10 * 2 ** 30 })
  expect(policy.mode).toBe('block_all')
})
