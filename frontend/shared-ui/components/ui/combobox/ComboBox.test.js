// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest'
import { createApp, createSSRApp, h, nextTick } from 'vue'
import { renderToString } from 'vue/server-renderer'
import ComboBox from './ComboBox.vue'

describe('combobox trigger', () => {
  it.each([
    [{}, 'Select timezone'],
    [{ modelValue: 'UTC' }, 'UTC (UTC+00:00)'],
    [{ defaultLabel: 'Saved timezone' }, 'Saved timezone']
  ])('shows the selected label or fallback for %j', async (props, label) => {
    const app = createSSRApp({
      render: () =>
        h(ComboBox, {
          items: [{ value: 'UTC', label: 'UTC (UTC+00:00)' }],
          placeholder: 'Select timezone',
          ...props
        })
    })
    app.config.globalProperties.i18n = { _: (message) => message }
    const html = await renderToString(app)
    const container = document.createElement('div')
    container.innerHTML = html
    expect(container.querySelector('button[role="combobox"]').textContent.trim()).toBe(label)
  })

  it('forwards blur from the trigger and virtual search input', async () => {
    vi.stubGlobal(
      'ResizeObserver',
      class {
        observe() {}
        unobserve() {}
        disconnect() {}
      }
    )
    const onBlur = vi.fn()
    const container = document.createElement('div')
    document.body.append(container)
    const app = createApp({
      render: () =>
        h(ComboBox, {
          items: [{ value: 'UTC', label: 'UTC (UTC+00:00)' }],
          placeholder: 'Select timezone',
          virtual: true,
          onBlur
        })
    })
    app.config.globalProperties.i18n = { _: (message) => message }
    try {
      app.mount(container)
      const trigger = container.querySelector('button[role="combobox"]')
      const triggerBlur = new FocusEvent('blur')
      trigger.dispatchEvent(triggerBlur)
      expect(onBlur).toHaveBeenCalledExactlyOnceWith(triggerBlur)
      trigger.click()
      await nextTick()
      const input = document.querySelector('input[role="combobox"]')
      const inputBlur = new FocusEvent('blur')
      onBlur.mockClear()
      input.dispatchEvent(inputBlur)
      expect(onBlur).toHaveBeenCalledExactlyOnceWith(inputBlur)
    } finally {
      app.unmount()
      container.remove()
      vi.unstubAllGlobals()
    }
  })
})
