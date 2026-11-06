// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
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
    app.config.globalProperties.$t = (key) => key
    const html = await renderToString(app)
    const container = document.createElement('div')
    container.innerHTML = html
    expect(container.querySelector('button[role="combobox"]').textContent.trim()).toBe(label)
  })
})
