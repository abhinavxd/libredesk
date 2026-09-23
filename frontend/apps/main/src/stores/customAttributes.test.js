import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/api', () => ({
  default: { getCustomAttributes: vi.fn() }
}))

vi.mock('@/composables/useEmitter', () => ({
  useEmitter: () => ({ emit: vi.fn() })
}))

import api from '@/api'
import { useCustomAttributeStore } from '@/stores/customAttributes'

describe('custom attribute store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('refreshes attributes after the initial cached fetch', async () => {
    api.getCustomAttributes
      .mockResolvedValueOnce({ data: { data: [{ id: 1, applies_to: 'conversation' }] } })
      .mockResolvedValueOnce({
        data: { data: [{ id: 2, name: 'Plan', applies_to: 'contact' }] }
      })
    const store = useCustomAttributeStore()

    await store.fetchCustomAttributes()
    await store.fetchCustomAttributes()
    await store.refreshCustomAttributes()

    expect(api.getCustomAttributes).toHaveBeenCalledTimes(2)
    expect(store.contactAttributeOptions).toEqual([
      { id: 2, name: 'Plan', applies_to: 'contact', label: 'Plan', value: '2' }
    ])
  })
})
