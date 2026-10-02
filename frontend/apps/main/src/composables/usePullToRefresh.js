import { computed, ref } from 'vue'
import { useIsMobile } from '@shared-ui/composables'

const INDICATOR_HEIGHT = 56
const REFRESH_THRESHOLD = 56
const MAX_PULL_DISTANCE = 88

// Gives a dragged list the same resistance users expect from native refresh
// controls, while leaving regular vertical scrolling untouched.
const applyResistance = (distance) => Math.min(MAX_PULL_DISTANCE, Math.round(distance * 0.55))

export function usePullToRefresh(refresh) {
  const isMobile = useIsMobile()
  const scrollElement = ref(null)
  const pullDistance = ref(0)
  const refreshing = ref(false)
  let touchStart = null

  const isPulling = computed(() => pullDistance.value > 0 && !refreshing.value)
  const isReadyToRefresh = computed(() => pullDistance.value >= REFRESH_THRESHOLD)
  const indicatorText = computed(() => {
    if (refreshing.value) return 'Refreshing messages…'
    return isReadyToRefresh.value ? 'Release to refresh' : 'Pull to refresh'
  })
  const indicatorStyle = computed(() => ({
    transform: `translateY(${pullDistance.value - INDICATOR_HEIGHT}px)`
  }))
  const contentStyle = computed(() =>
    pullDistance.value ? { transform: `translateY(${pullDistance.value}px)` } : undefined
  )

  function resetPull() {
    touchStart = null
    if (!refreshing.value) pullDistance.value = 0
  }

  function startPull(event) {
    const element = scrollElement.value
    if (!isMobile.value || refreshing.value || !element || element.scrollTop > 0 || event.touches.length !== 1) {
      resetPull()
      return
    }

    const touch = event.touches[0]
    touchStart = { x: touch.clientX, y: touch.clientY }
  }

  function movePull(event) {
    const element = scrollElement.value
    if (!touchStart || !isMobile.value || refreshing.value || !element) return

    const touch = event.touches[0]
    const verticalDistance = touch.clientY - touchStart.y
    const horizontalDistance = Math.abs(touch.clientX - touchStart.x)
    if (verticalDistance <= 0 || horizontalDistance > verticalDistance || element.scrollTop > 0) {
      resetPull()
      return
    }

    if (event.cancelable) event.preventDefault()
    pullDistance.value = applyResistance(verticalDistance)
  }

  async function finishPull() {
    const shouldRefresh = isReadyToRefresh.value && !refreshing.value
    touchStart = null
    if (!shouldRefresh) {
      pullDistance.value = 0
      return
    }

    refreshing.value = true
    pullDistance.value = REFRESH_THRESHOLD
    try {
      await refresh()
    } finally {
      refreshing.value = false
      pullDistance.value = 0
    }
  }

  return {
    contentStyle,
    finishPull,
    indicatorStyle,
    indicatorText,
    isPulling,
    isReadyToRefresh,
    movePull,
    pullDistance,
    refreshing,
    resetPull,
    scrollElement,
    startPull
  }
}
