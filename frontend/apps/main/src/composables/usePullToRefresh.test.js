// @vitest-environment jsdom

import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'

vi.mock('@shared-ui/composables', () => ({ useIsMobile: () => ref(true) }))

import { usePullToRefresh } from './usePullToRefresh'

const touch = (x, y) => ({ clientX: x, clientY: y })
const event = (touches) => ({
  touches,
  cancelable: true,
  preventDefault: vi.fn()
})

describe('usePullToRefresh', () => {
  it('shows pull progress and refreshes after a downward swipe from the top', async () => {
    const refresh = vi.fn().mockResolvedValue()
    const control = usePullToRefresh(refresh)
    control.scrollElement.value = document.createElement('div')

    control.startPull(event([touch(20, 100)]))
    const move = event([touch(20, 220)])
    control.movePull(move)

    expect(move.preventDefault).toHaveBeenCalledOnce()
    expect(control.pullDistance.value).toBeGreaterThanOrEqual(56)
    expect(control.indicatorText.value).toBe('Release to refresh')

    await control.finishPull()

    expect(refresh).toHaveBeenCalledOnce()
    expect(control.refreshing.value).toBe(false)
    expect(control.pullDistance.value).toBe(0)
  })

  it('does not hijack a normal list scroll', async () => {
    const refresh = vi.fn()
    const control = usePullToRefresh(refresh)
    control.scrollElement.value = document.createElement('div')
    Object.defineProperty(control.scrollElement.value, 'scrollTop', { value: 40 })

    control.startPull(event([touch(20, 100)]))
    control.movePull(event([touch(20, 220)]))
    await control.finishPull()

    expect(control.pullDistance.value).toBe(0)
    expect(refresh).not.toHaveBeenCalled()
  })
})
