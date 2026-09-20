// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick } from 'vue'
import ResourcePolicyForm from './ResourcePolicyForm.vue'
import api from '@/api'

const invalidate = vi.hoisted(() => vi.fn())
vi.mock('@/api', () => ({ default: { getResourcePolicy: vi.fn(), updateResourcePolicy: vi.fn() } }))
vi.mock('@/stores/conversation', () => ({ useConversationStore: () => ({ invalidateImageDisplays: invalidate }) }))
vi.mock('@/composables/useEmitter.js', () => ({ useEmitter: () => ({ emit: vi.fn() }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: key => key }) }))

let app
let root
afterEach(() => {
  app?.unmount()
  root?.remove()
  vi.clearAllMocks()
})

describe('image policy cache refresh', () => {
  it('invalidates cached displays only after the new policy is saved', async () => {
    api.getResourcePolicy.mockResolvedValue({ data: { data: { mode: 'block_all', allowed_domains: [] } } })
    let finish
    api.updateResourcePolicy.mockImplementation(() => new Promise(resolve => { finish = resolve }))
    root = document.createElement('div')
    document.body.appendChild(root)
    app = createApp({ render: () => h(ResourcePolicyForm) })
    app.mount(root)
    await nextTick()
    await nextTick()
    const select = root.querySelector('select')
    expect([...select.options].map(option => option.value)).toEqual(['block_all', 'allowlist', 'load_on_receipt'])
    select.value = 'load_on_receipt'
    select.dispatchEvent(new Event('change', { bubbles: true }))
    await nextTick()
    expect(root.querySelector('textarea')).toBeNull()
    expect(root.querySelector('input[type=number]').value).toBe('10')
    root.querySelector('form').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    await nextTick()
    expect(api.updateResourcePolicy).toHaveBeenCalledWith({ mode: 'load_on_receipt', allowed_domains: [], max_cache_bytes: 10 * 2 ** 30 })
    expect(invalidate).not.toHaveBeenCalled()
    finish({ data: { data: { allowed_domains: [] } } })
    await nextTick()
    await nextTick()
    expect(invalidate).toHaveBeenCalledOnce()
  })

  it('loads and saves the cache size in GiB without changing its byte value', async () => {
    const policy = { mode: 'allowlist', allowed_domains: [], max_cache_bytes: 2 * 2 ** 30 }
    api.getResourcePolicy.mockResolvedValue({ data: { data: policy } })
    api.updateResourcePolicy.mockImplementation(async value => ({ data: { data: value } }))
    root = document.createElement('div')
    document.body.appendChild(root)
    app = createApp({ render: () => h(ResourcePolicyForm) })
    app.mount(root)
    await nextTick()
    await nextTick()
    const input = root.querySelector('input[type=number]')
    expect(input.value).toBe('2')
    input.value = '0.5'
    input.dispatchEvent(new Event('input', { bubbles: true }))
    await nextTick()
    await nextTick()
    root.querySelector('form').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    await nextTick()
    expect(api.updateResourcePolicy).toHaveBeenCalledWith({ ...policy, max_cache_bytes: 2 ** 29 })
  })

  it.each(['0', '-1', ''])('rejects invalid cache size %s without saving', async value => {
    api.getResourcePolicy.mockResolvedValue({ data: { data: { mode: 'block_all', allowed_domains: [], max_cache_bytes: 10 * 2 ** 30 } } })
    root = document.createElement('div')
    document.body.appendChild(root)
    app = createApp({ render: () => h(ResourcePolicyForm) })
    app.mount(root)
    await nextTick()
    await nextTick()
    const input = root.querySelector('input[type=number]')
    input.value = value
    input.dispatchEvent(new Event('input', { bubbles: true }))
    await nextTick()
    await nextTick()
    root.querySelector('form').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    await nextTick()
    expect(api.updateResourcePolicy).not.toHaveBeenCalled()
    expect(root.querySelector('[role=alert]').textContent).toBe('admin.resourcePolicy.invalidCacheSize')
  })

})
