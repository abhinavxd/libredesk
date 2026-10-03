// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'

vi.mock('@shared-ui/composables', () => ({ useIsMobile: () => ref(true) }))

import { rubberBand, usePullToRefresh } from './usePullToRefresh'

const touch = (x, y) => ({ clientX: x, clientY: y })
const event = (touches) => ({
  touches,
  cancelable: true,
  preventDefault: vi.fn()
})

// Drags the finger down from y=100 in small steps, like a real touchmove stream.
function drag(control, to, { from = 100, x = 20 } = {}) {
  control.startPull(event([touch(x, from)]))
  const moves = []
  for (let y = from + 10; y <= to; y += 10) {
    vi.advanceTimersByTime(16)
    const move = event([touch(x, y)])
    control.movePull(move)
    moves.push(move)
  }
  return moves
}

function setup(refresh = vi.fn().mockResolvedValue()) {
  const control = usePullToRefresh(refresh)
  const element = document.createElement('div')
  Object.defineProperty(element, 'clientHeight', { value: 700 })
  control.scrollElement.value = element
  return { control, element, refresh }
}

describe('rubberBand', () => {
  it('resists more the further it is pulled and never reaches the view height', () => {
    expect(rubberBand(0, 700)).toBe(0)
    expect(rubberBand(10, 700)).toBeCloseTo(5.5, 0)
    const half = rubberBand(350, 700)
    const full = rubberBand(700, 700)
    expect(full - half).toBeLessThan(half)
    expect(rubberBand(100000, 700)).toBeLessThan(700)
  })
})

describe('usePullToRefresh', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'requestAnimationFrame', 'cancelAnimationFrame', 'performance'] })
  })
  afterEach(() => vi.useRealTimers())

  it('refreshes as soon as the pull is deep enough, without waiting for release', async () => {
    let finish
    const { control, refresh } = setup(vi.fn(() => new Promise((resolve) => (finish = resolve))))

    const moves = drag(control, 300)

    expect(moves.every((move) => move.preventDefault.mock.calls.length === 1)).toBe(true)
    expect(refresh).toHaveBeenCalledOnce()
    expect(control.refreshing.value).toBe(true)
    expect(control.spinning.value).toBe(true)

    // Released, the list settles into the gap the spinner rests in.
    control.finishPull()
    vi.advanceTimersByTime(1000)
    expect(control.offset.value).toBe(60)

    finish()
    await Promise.resolve()
    await Promise.resolve()
    expect(control.refreshing.value).toBe(false)
    vi.advanceTimersByTime(1000)
    expect(control.offset.value).toBe(0)
    expect(control.indicatorVisible.value).toBe(false)
  })

  it('springs back without refreshing after a short pull', () => {
    const { control, refresh } = setup()

    drag(control, 180)
    expect(control.offset.value).toBeGreaterThan(0)
    expect(control.progress.value).toBeLessThan(1)

    control.finishPull()
    vi.advanceTimersByTime(1000)

    expect(refresh).not.toHaveBeenCalled()
    expect(control.offset.value).toBe(0)
  })

  it('picks the list up mid-bounce from where it is', () => {
    const { control } = setup()

    drag(control, 180)
    control.finishPull()
    vi.advanceTimersByTime(60)
    const midBounce = control.offset.value
    expect(midBounce).toBeGreaterThan(0)

    control.startPull(event([touch(20, 300)]))
    control.movePull(event([touch(20, 300)]))

    expect(control.offset.value).toBeCloseTo(midBounce, 5)
  })

  it('does not hijack a normal list scroll', () => {
    const { control, element, refresh } = setup()
    Object.defineProperty(element, 'scrollTop', { value: 40 })

    const moves = drag(control, 300)
    control.finishPull()

    expect(moves[0].preventDefault).not.toHaveBeenCalled()
    expect(control.offset.value).toBe(0)
    expect(refresh).not.toHaveBeenCalled()
  })

  it('drops the spring when Reduce Motion is on', () => {
    window.matchMedia = vi.fn((query) => ({
      matches: query.includes('reduce'),
      addEventListener: () => {},
      removeEventListener: () => {}
    }))
    try {
      const { control } = setup()

      drag(control, 180)
      control.finishPull()

      expect(control.offset.value).toBe(0)
    } finally {
      delete window.matchMedia
    }
  })

  it('leaves sideways swipes alone', () => {
    const { control } = setup()

    control.startPull(event([touch(20, 100)]))
    const move = event([touch(80, 110)])
    control.movePull(move)

    expect(move.preventDefault).not.toHaveBeenCalled()
    expect(control.offset.value).toBe(0)
  })
})
