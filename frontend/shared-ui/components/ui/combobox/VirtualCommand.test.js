// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick, ref } from 'vue'
import VirtualCommand from './VirtualCommand.vue'

let app
let container

vi.stubGlobal(
  'ResizeObserver',
  class {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
)

afterEach(() => {
  app?.unmount()
  container?.remove()
})

describe('virtual combobox search', () => {
  it('retains every character while updating the search value', async () => {
    const searchTerm = ref('')
    const items = [{ label: 'Europe/Amsterdam', value: 'Europe/Amsterdam' }]
    container = document.createElement('div')
    document.body.append(container)
    app = createApp({
      setup: () => () =>
        h(VirtualCommand, {
          items,
          searchTerm: searchTerm.value,
          'onUpdate:searchTerm': (value) => {
            searchTerm.value = value
          }
        })
    })
    app.mount(container)
    await nextTick()
    const input = container.querySelector('input')
    const text = 'Amsterdam'
    for (let index = 0; index < text.length; index++) {
      input.value += text[index]
      input.dispatchEvent(new Event('input', { bubbles: true }))
      await nextTick()
      expect(input.value).toBe(text.slice(0, index + 1))
      expect(searchTerm.value).toBe(input.value)
    }
    input.value = 'Pacific/Auckland'
    input.dispatchEvent(new Event('input', { bubbles: true }))
    await nextTick()
    expect(input.value).toBe('Pacific/Auckland')
    expect(searchTerm.value).toBe('Pacific/Auckland')
  })
})
