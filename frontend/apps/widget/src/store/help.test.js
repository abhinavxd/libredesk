import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import api from '@widget/api/index.js'
import { useHelpStore } from './help.js'

vi.mock('@widget/api/index.js', () => ({ default: { sendHelpFeedback: vi.fn() } }))

beforeEach(() => {
  setActivePinia(createPinia())
  vi.resetAllMocks()
})

const setup = () => {
  const help = useHelpStore()
  help.data = { slug: 'support' }
  help.article = { id: 1, slug: 'first', locale: 'en-US' }
  return help
}

describe('article feedback', () => {
  it.each([true, false])('retains a %s vote when returning to an article', async (helpful) => {
    const help = setup()
    await help.vote(helpful)
    help.article = { id: 2, slug: 'second', locale: 'en-US' }
    expect(help.feedback[2]).toBeUndefined()
    help.article = { id: 1, slug: 'first', locale: 'en-US' }
    expect(useHelpStore().feedback[1]).toBe(helpful)
    await help.vote(!helpful)
    expect(api.sendHelpFeedback).toHaveBeenCalledTimes(1)
  })

  it('records a pending vote against its original article and prevents duplicate submission', async () => {
    let complete
    api.sendHelpFeedback.mockImplementation(
      () =>
        new Promise((resolve) => {
          complete = resolve
        })
    )
    const help = setup()
    const pending = help.vote(true)
    await help.vote(false)
    help.article = { id: 2, slug: 'second', locale: 'en-US' }
    complete()
    await pending
    expect(help.feedback).toEqual({ 1: true })
    expect(help.feedbackPending).toEqual({})
    expect(api.sendHelpFeedback).toHaveBeenCalledTimes(1)
  })

  it('discards a pending vote after the widget identity resets', async () => {
    let complete
    api.sendHelpFeedback.mockImplementation(
      () =>
        new Promise((resolve) => {
          complete = resolve
        })
    )
    const help = setup()
    const pending = help.vote(true)
    help.reset()
    complete()
    await pending
    expect(help.feedback).toEqual({})
    expect(help.feedbackPending).toEqual({})
  })

  it('allows retry after a failed request', async () => {
    api.sendHelpFeedback.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce({})
    const help = setup()
    await expect(help.vote(false)).rejects.toThrow('offline')
    expect(help.feedback).toEqual({})
    expect(help.feedbackPending).toEqual({})
    await help.vote(false)
    expect(help.feedback[1]).toBe(false)
  })
})
