import { describe, expect, test } from 'vitest'
import { resolvePreChatForm } from './preChatForm.js'

const legacy = {
  enabled: true,
  title: 'Before we start',
  fields: [{ key: 'email' }]
}

describe('resolvePreChatForm', () => {
  test('keeps legacy settings when audience settings are absent', () => {
    expect(resolvePreChatForm(legacy, true)).toEqual(legacy)
    expect(resolvePreChatForm(legacy, false)).toEqual(legacy)
  })

  test('selects the visitor form', () => {
    const config = {
      ...legacy,
      visitors: { enabled: true, title: 'Choose a plan', fields: [{ key: 'plan' }] },
      users: { enabled: true, title: 'What is the issue?', fields: [{ key: 'issue' }] }
    }

    expect(resolvePreChatForm(config, true)).toMatchObject({
      enabled: true,
      title: 'Choose a plan',
      fields: [{ key: 'plan' }]
    })
  })

  test('selects the logged-in user form', () => {
    const config = {
      ...legacy,
      visitors: { enabled: true, title: 'Choose a plan', fields: [{ key: 'plan' }] },
      users: { enabled: true, title: 'What is the issue?', fields: [{ key: 'issue' }] }
    }

    expect(resolvePreChatForm(config, false)).toMatchObject({
      enabled: true,
      title: 'What is the issue?',
      fields: [{ key: 'issue' }]
    })
  })

  test('keeps explicit disabled and empty audience values', () => {
    const config = {
      ...legacy,
      users: { enabled: false, title: '', fields: [] }
    }

    expect(resolvePreChatForm(config, false)).toMatchObject({ enabled: false, title: '', fields: [] })
  })

  test('global setting cannot be overridden by an audience', () => {
    const config = {
      ...legacy,
      enabled: false,
      visitors: { enabled: true, title: 'Choose a plan', fields: [{ key: 'plan' }] }
    }

    expect(resolvePreChatForm(config, true).enabled).toBe(false)
  })
})
