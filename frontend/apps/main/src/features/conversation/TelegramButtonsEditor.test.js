// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest'
import { createApp, h, nextTick, ref } from 'vue'
import TelegramButtonsEditor from './TelegramButtonsEditor.vue'

let app, root, editor, buttons, validated
const empty = vi.fn()

const mount = (initial, submitted = false) => {
  buttons = ref(initial)
  validated = ref(submitted)
  editor = ref(null)
  root = document.createElement('div')
  document.body.append(root)
  app = createApp({
    render: () =>
      buttons.value.length
        ? h(TelegramButtonsEditor, {
            ref: editor,
            modelValue: buttons.value,
            'onUpdate:modelValue': (value) => (buttons.value = value),
            validated: validated.value,
            onEmpty: empty
          })
        : null
  })
  app.config.globalProperties.$t = (key) => key
  app.mount(root)
}

const fill = async (input, value) => {
  input.value = value
  input.dispatchEvent(new Event('input', { bubbles: true }))
  await nextTick()
}

afterEach(() => {
  app?.unmount()
  root?.remove()
  vi.clearAllMocks()
})

it('waits for submission before showing errors, including while typing', async () => {
  mount([{ text: '', url: '' }])
  await fill(root.querySelectorAll('input')[1], 'https:/')
  expect(root.querySelector('[role="alert"]')).toBeNull()
  expect(root.querySelector('[aria-invalid="true"]')).toBeNull()
  validated.value = true
  await nextTick()
  expect(root.querySelectorAll('[aria-invalid="true"]')).toHaveLength(2)
})

it('focuses the first invalid field and connects its error description', () => {
  mount(
    [
      { text: 'Docs', url: 'broken' },
      { text: '', url: '' }
    ],
    true
  )
  editor.value.focusError()
  const link = root.querySelectorAll('input')[1]
  expect(document.activeElement).toBe(link)
  expect(document.getElementById(link.getAttribute('aria-describedby')).textContent).toContain(
    'conversation.telegram.error.buttonLink'
  )
  expect(root.querySelector('input').getAttribute('aria-invalid')).toBe('false')
})

it('clears individual errors as fields are corrected after submission', async () => {
  mount([{ text: '', url: 'broken' }], true)
  const [text, link] = root.querySelectorAll('input')
  await fill(text, 'Docs')
  expect(text.getAttribute('aria-invalid')).toBe('false')
  expect(link.getAttribute('aria-invalid')).toBe('true')
  await fill(link, 'https://example.com')
  expect(root.querySelector('[role="alert"]')).toBeNull()
  expect(link.hasAttribute('aria-describedby')).toBe(false)
})

it('identifies a callback label that exceeds the byte limit', async () => {
  mount([{ text: '😀'.repeat(17), url: '' }], true)
  expect(root.textContent).toContain('conversation.telegram.error.buttonTextTooLong')
  await fill(root.querySelector('input'), '😀'.repeat(16))
  expect(root.querySelector('[role="alert"]')).toBeNull()
})

it('keeps focus beside a removed row and focuses newly added rows in order', async () => {
  mount([
    { text: 'First', url: '' },
    { text: 'Second', url: '' },
    { text: 'Third', url: '' }
  ])
  root.querySelectorAll('button')[1].click()
  await nextTick()
  expect(buttons.value.map((button) => button.text)).toEqual(['First', 'Third'])
  expect(document.activeElement.value).toBe('Third')
  buttons.value.push({ text: 'Fourth', url: '' })
  await nextTick()
  editor.value.focus()
  expect(document.activeElement.value).toBe('Fourth')
})

it('hands focus back to the composer when the last row is removed', async () => {
  mount([{ text: 'First', url: '' }])
  const remove = root.querySelector('button')
  expect(remove.type).toBe('button')
  remove.click()
  await nextTick()
  expect(buttons.value).toEqual([])
  expect(empty).toHaveBeenCalledOnce()
})
