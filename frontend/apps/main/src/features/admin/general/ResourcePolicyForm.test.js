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
    root.querySelector('form').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    await nextTick()
    expect(api.updateResourcePolicy).toHaveBeenCalledWith({ mode: 'load_on_receipt', allowed_domains: [] })
    expect(invalidate).not.toHaveBeenCalled()
    finish({ data: { data: { allowed_domains: [] } } })
    await nextTick()
    await nextTick()
    expect(invalidate).toHaveBeenCalledOnce()
  })
})
