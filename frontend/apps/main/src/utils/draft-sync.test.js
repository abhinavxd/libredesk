import { afterEach, describe, expect, it, vi } from 'vitest'
import { createDraftSync } from './draft-sync'
const settle = async () => {
  for (let i = 0; i < 5; i++) await Promise.resolve()
}
afterEach(() => vi.useRealTimers())

describe('draft persistence retries', () => {
  it('surfaces failed writes and retries without another keystroke', async () => {
    vi.useFakeTimers()
    const write = vi.fn().mockRejectedValueOnce(new Error('offline')).mockResolvedValue(undefined)
    const onState = vi.fn()
    const sync = createDraftSync({ write, onState })
    sync.save('A', 'reply', { content: 'hello' })
    await settle()
    expect(onState).toHaveBeenLastCalledWith('A::reply', 'error')
    await vi.advanceTimersByTimeAsync(5000)
    expect(write).toHaveBeenCalledTimes(2)
    expect(onState).toHaveBeenLastCalledWith('A::reply', 'saved')
    sync.dispose()
  })

  it('serializes deletion after an in-flight save and retries only the latest content', async () => {
    vi.useFakeTimers()
    let finish
    const write = vi
      .fn()
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            finish = resolve
          })
      )
      .mockResolvedValue(undefined)
    const sync = createDraftSync({ write, onState: vi.fn() })
    sync.save('A', 'reply', { content: 'old' })
    sync.save('A', 'reply', { content: 'new' })
    sync.save('A', 'reply', null)
    expect(write).toHaveBeenCalledTimes(1)
    finish()
    await settle()
    expect(write.mock.calls).toEqual([
      ['A', 'reply', { content: 'old' }],
      ['A', 'reply', null]
    ])
    sync.dispose()
  })
})
