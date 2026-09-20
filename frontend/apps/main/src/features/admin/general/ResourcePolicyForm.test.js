// @vitest-environment jsdom
import { expect, it, vi } from 'vitest'
import { createApp, h, nextTick } from 'vue'
import ResourcePolicyForm from './ResourcePolicyForm.vue'
import api from '@main/api'
vi.mock('@main/api', () => ({
  default: { getResourcePolicy: vi.fn(), updateResourcePolicy: vi.fn() }
}))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('@main/stores/conversation', () => ({
  useConversationStore: () => ({ invalidateImageDisplays: vi.fn() })
}))
vi.mock('@main/composables/useEmitter.js', () => ({ useEmitter: () => ({ emit: vi.fn() }) }))
it('does not resubmit the hidden cache limit when saving privacy settings', async () => {
  let policy = { mode: 'block_all', allowed_domains: [], max_cache_bytes: 10 * 2 ** 30 }
  api.getResourcePolicy.mockResolvedValue({ data: { data: { ...policy } } })
  api.updateResourcePolicy.mockImplementation(async (fields) => {
    policy = { ...policy, ...fields }
    return { data: { data: policy } }
  })
  const root = document.createElement('div')
  const app = createApp({ render: () => h(ResourcePolicyForm, { showCacheSize: false }) })
  try {
    app.mount(root)
    for (let i = 0; i < 10; i++) await nextTick()
    policy = { ...policy, max_cache_bytes: 2 ** 30 }
    root
      .querySelector('form')
      .dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    for (let i = 0; i < 10; i++) await nextTick()
    expect(api.updateResourcePolicy).toHaveBeenCalledWith({
      mode: 'block_all',
      allowed_domains: []
    })
    expect(policy.max_cache_bytes).toBe(2 ** 30)
  } finally {
    app.unmount()
  }
})
