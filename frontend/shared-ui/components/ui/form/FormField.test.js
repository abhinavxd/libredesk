// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick } from 'vue'
import { useForm } from 'vee-validate'
import { FormField } from './index'
import { Input } from '@shared-ui/components/ui/input'

let app
let container

function mountForm({ initialName = '', native = false } = {}) {
  container = document.createElement('div')
  document.body.append(container)
  app = createApp({
    setup() {
      const form = useForm({ initialValues: { name: initialName } })
      return () =>
        h('form', { onSubmit: form.handleSubmit(() => {}) }, [
          h(
            FormField,
            {
              name: 'name',
              rules: (value) => String(value || '').length >= 2 || 'Name needs two characters'
            },
            {
              default: (field) =>
                h('div', [
                  native
                    ? h('input', { ...field.field, 'aria-label': 'Name' })
                    : h(Input, { ...field.componentField, 'aria-label': 'Name' }),
                  h('p', { role: 'alert' }, field.errorMessage || '')
                ])
            }
          ),
          h('button', { type: 'submit' }, 'Save')
        ])
    }
  })
  app.mount(container)
}

async function typeName(value) {
  const input = container.querySelector('input')
  input.value = value
  input.dispatchEvent(new Event('input', { bubbles: true }))
  await nextTick()
}

afterEach(() => {
  app?.unmount()
  container?.remove()
})

describe('FormField validation timing', () => {
  it('keeps invalid initial values quiet until submission', async () => {
    mountForm({ initialName: 'J' })
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(container.querySelector('[role="alert"]').textContent).toBe('')
    container
      .querySelector('form')
      .dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    await vi.waitFor(() =>
      expect(container.querySelector('[role="alert"]').textContent).toBe(
        'Name needs two characters'
      )
    )
  })

  it('applies the same validation timing to native input bindings', async () => {
    mountForm({ native: true })
    await typeName('J')
    container.querySelector('input').dispatchEvent(new Event('change', { bubbles: true }))
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(container.querySelector('[role="alert"]').textContent).toBe('')
    container
      .querySelector('form')
      .dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    await vi.waitFor(() =>
      expect(container.querySelector('[role="alert"]').textContent).toBe(
        'Name needs two characters'
      )
    )
    await typeName('Jo')
    await vi.waitFor(() => expect(container.querySelector('[role="alert"]').textContent).toBe(''))
  })

  it('keeps errors hidden while typing and after leaving an unsubmitted field', async () => {
    mountForm()
    await typeName('J')
    container.querySelector('input').dispatchEvent(new Event('blur'))
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(container.querySelector('[role="alert"]').textContent).toBe('')
  })

  it('validates on submit and clears errors while correcting the field', async () => {
    mountForm()
    await typeName('J')
    container
      .querySelector('form')
      .dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    await vi.waitFor(() =>
      expect(container.querySelector('[role="alert"]').textContent).toBe(
        'Name needs two characters'
      )
    )
    await typeName('Jo')
    await vi.waitFor(() => expect(container.querySelector('[role="alert"]').textContent).toBe(''))
  })
})
