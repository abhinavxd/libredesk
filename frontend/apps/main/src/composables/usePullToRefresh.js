import { computed, ref } from 'vue'
import { tryOnScopeDispose, usePreferredReducedMotion } from '@vueuse/core'
import { useIsMobile } from '@shared-ui/composables'

// Modelled on iOS's UIRefreshControl: the list rubber-bands past its top, the
// spinner fills in with the pull, and the refresh starts the moment the pull is
// deep enough rather than on release.
const HOLD_OFFSET = 60 // gap the list rests in while refreshing
const TRIGGER_OFFSET = 80 // pull that starts a refresh, after resistance
const RUBBER_BAND = 0.55 // UIScrollView's rubber-band coefficient
const SPRING_RESPONSE = 0.5 // seconds; critically damped like a scroll bounce-back
const VELOCITY_WINDOW = 100 // ms of movement that count toward release velocity

// UIScrollView's rubber band: resistance grows with distance, and the offset
// approaches but never reaches the height of the view.
export const rubberBand = (distance, dimension) =>
  (1 - 1 / ((distance * RUBBER_BAND) / dimension + 1)) * dimension

// The finger travel that produces an offset, so a grab mid-animation continues
// from where the list is instead of jumping.
const fingerTravel = (offset, dimension) => {
  const clamped = Math.min(offset, dimension - 1)
  return (dimension / RUBBER_BAND) * (clamped / (dimension - clamped))
}

export function usePullToRefresh(refresh) {
  const isMobile = useIsMobile()
  const reducedMotion = usePreferredReducedMotion()
  const scrollElement = ref(null)
  const offset = ref(0)
  const refreshing = ref(false)
  // True while the spinner fades out after a refresh, so it keeps spinning
  // instead of rewinding its spokes.
  const dismissing = ref(false)
  let gesture = null
  let samples = []
  let frame = 0

  const progress = computed(() => Math.min(1, offset.value / TRIGGER_OFFSET))
  const spinning = computed(() => refreshing.value || dismissing.value)
  const indicatorVisible = computed(() => offset.value > 0 || refreshing.value)
  // The spinner rides down centred in the gap, then stays where it will rest.
  const indicatorStyle = computed(() => ({
    transform: `translateY(${Math.min(offset.value, HOLD_OFFSET) / 2}px) translateY(-50%)`
  }))
  const contentStyle = computed(() =>
    offset.value ? { transform: `translateY(${offset.value}px)` } : undefined
  )

  const restOffset = () => (refreshing.value ? HOLD_OFFSET : 0)

  function stopSettling() {
    if (frame) cancelAnimationFrame(frame)
    frame = 0
  }

  // A critically damped spring that starts at the finger's release velocity,
  // so the hand-off from drag to animation has no jolt.
  function settle(target, velocity = 0) {
    stopSettling()
    const from = offset.value - target
    if (reducedMotion.value === 'reduce' || typeof requestAnimationFrame !== 'function') {
      offset.value = target
      dismissing.value = false
      return
    }
    const omega = (2 * Math.PI) / SPRING_RESPONSE
    const start = performance.now()
    const step = (now) => {
      const t = (now - start) / 1000
      const decay = Math.exp(-omega * t)
      const x = (from + (velocity + omega * from) * t) * decay
      const v = (velocity - omega * (velocity + omega * from) * t) * decay
      if (Math.abs(x) < 0.5 && Math.abs(v) < 20) {
        offset.value = target
        dismissing.value = false
        frame = 0
        return
      }
      offset.value = Math.max(0, target + x)
      frame = requestAnimationFrame(step)
    }
    frame = requestAnimationFrame(step)
  }

  function releaseVelocity() {
    const now = performance.now()
    const recent = samples.filter((sample) => now - sample.time <= VELOCITY_WINDOW)
    if (recent.length < 2) return 0
    const first = recent[0]
    const last = recent[recent.length - 1]
    return last.time > first.time ? ((last.offset - first.offset) / (last.time - first.time)) * 1000 : 0
  }

  async function startRefresh() {
    refreshing.value = true
    // Native refresh controls tick when they fire. Android browsers can echo
    // that; iOS Safari has no web haptics, so this quietly does nothing there.
    if (typeof navigator !== 'undefined' && typeof navigator.vibrate === 'function') {
      navigator.vibrate(10)
    }
    try {
      await refresh()
    } finally {
      refreshing.value = false
      if (!gesture?.pulling) {
        dismissing.value = true
        settle(0)
      }
    }
  }

  function startPull(event) {
    const element = scrollElement.value
    if (!isMobile.value || !element || element.scrollTop > 0 || event.touches.length !== 1) {
      gesture = null
      return
    }
    // Catching the list mid-bounce takes it over from where it is.
    stopSettling()
    dismissing.value = false
    const touch = event.touches[0]
    const dimension = element.clientHeight || window.innerHeight
    gesture = {
      x: touch.clientX,
      y: touch.clientY,
      dimension,
      base: fingerTravel(offset.value, dimension),
      pulling: offset.value > 0
    }
    samples = []
  }

  function movePull(event) {
    const element = scrollElement.value
    if (!gesture || !element) return

    const touch = event.touches[0]
    const verticalDistance = touch.clientY - gesture.y
    const horizontalDistance = Math.abs(touch.clientX - gesture.x)
    if (!gesture.pulling) {
      // Leave ordinary scrolling and sideways swipes to the browser.
      if (verticalDistance <= 0 || horizontalDistance > verticalDistance || element.scrollTop > 0) {
        gesture = null
        return
      }
      gesture.pulling = true
    }

    if (event.cancelable) event.preventDefault()
    const travel = gesture.base + verticalDistance
    offset.value = travel > 0 ? rubberBand(travel, gesture.dimension) : 0
    samples.push({ time: performance.now(), offset: offset.value })
    if (samples.length > 8) samples.shift()

    if (!refreshing.value && offset.value >= TRIGGER_OFFSET) startRefresh()
  }

  function finishPull() {
    if (!gesture) return
    const velocity = gesture.pulling ? releaseVelocity() : 0
    gesture = null
    samples = []
    settle(restOffset(), velocity)
  }

  tryOnScopeDispose(stopSettling)

  return {
    contentStyle,
    finishPull,
    indicatorStyle,
    indicatorVisible,
    movePull,
    offset,
    progress,
    refreshing,
    resetPull: finishPull,
    scrollElement,
    spinning,
    startPull
  }
}
