// @vitest-environment jsdom
import { afterEach, describe, expect, it } from 'vitest'
import { createApp, h } from 'vue'
import { setTimeFormat } from '@shared-ui/utils/datetime.js'
import TableDate from './TableDate.vue'

const afternoon = new Date(2026, 7, 28, 15, 46, 9)

let app
let container

function mountDate(value) {
  container = document.createElement('div')
  document.body.appendChild(container)
  app = createApp({ render: () => h(TableDate, { value }) })
  app.mount(container)
  return container.querySelector('time')
}

afterEach(() => {
  app?.unmount()
  container?.remove()
  setTimeFormat('12h')
})

describe('TableDate', () => {
  it('renders the 12-hour layout by default', () => {
    const el = mountDate(afternoon)
    expect(el.textContent).toBe('28 Aug 2026, 3:46 PM')
    expect(el.getAttribute('title')).toMatch(/^28 Aug 2026, 3:46:09 PM GMT/)
  })

  it('follows the 24-hour desk setting', () => {
    setTimeFormat('24h')
    const el = mountDate(afternoon)
    expect(el.textContent).toBe('28 Aug 2026, 15:46')
    expect(el.getAttribute('title')).toMatch(/^28 Aug 2026, 15:46:09 GMT/)
  })

  it('renders nothing for an empty value', () => {
    expect(mountDate(null)).toBeNull()
  })
})
