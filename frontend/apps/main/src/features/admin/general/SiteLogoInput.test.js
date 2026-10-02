// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest'
import { createApp, h, nextTick } from 'vue'
import SiteLogoInput from './SiteLogoInput.vue'

const { uploadSiteLogo } = vi.hoisted(() => ({ uploadSiteLogo: vi.fn() }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('../../../api', () => ({ default: { uploadSiteLogo } }))

let app
afterEach(() => app?.unmount())

const mountInput = (modelValue, onUpdate) => {
  const root = document.createElement('div')
  app = createApp({
    render: () => h(SiteLogoInput, { modelValue, 'onUpdate:modelValue': onUpdate })
  })
  app.config.globalProperties.emitter = { emit: vi.fn() }
  app.mount(root)
  return root
}

it('previews the Fernmail fern until a logo is uploaded, then emits its URL', async () => {
  uploadSiteLogo.mockResolvedValue({ data: { data: { url: '/uploads/abc' } } })
  const onUpdate = vi.fn()
  const root = mountInput('', onUpdate)
  expect(root.querySelector('img').getAttribute('src')).toBe('/images/fern.svg')
  expect(root.textContent).toContain('admin.general.siteLogo.default')
  expect(root.textContent).not.toContain('admin.general.siteLogo.remove')

  const input = root.querySelector('input[type="file"]')
  const file = new File(['png'], 'logo.png', { type: 'image/png' })
  Object.defineProperty(input, 'files', { value: [file] })
  input.dispatchEvent(new Event('change'))
  await vi.waitFor(() => expect(onUpdate).toHaveBeenCalledWith('/uploads/abc'))
  expect(uploadSiteLogo.mock.calls[0][0].get('files')).toBe(file)
})

it('shows the saved logo and clears it on remove', async () => {
  const onUpdate = vi.fn()
  const root = mountInput('/uploads/abc', onUpdate)
  expect(root.querySelector('img').getAttribute('src')).toBe('/uploads/abc')
  const remove = [...root.querySelectorAll('button')].find((b) =>
    b.textContent.includes('admin.general.siteLogo.remove')
  )
  remove.click()
  await nextTick()
  expect(onUpdate).toHaveBeenCalledWith('')
})
