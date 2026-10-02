<script setup>
import { useSidebar } from '@shared-ui/components/ui/sidebar'

const EDGE_WIDTH = 32
const MIN_SWIPE_DISTANCE = 56

const { isMobile, openMobile, setOpenMobile } = useSidebar()

let gestureStart = null

function resetGesture() {
  gestureStart = null
}

function startGesture(event) {
  if (!isMobile.value || openMobile.value || event.touches.length !== 1) {
    resetGesture()
    return
  }

  const touch = event.touches[0]
  if (touch.clientX > EDGE_WIDTH) {
    resetGesture()
    return
  }

  gestureStart = { x: touch.clientX, y: touch.clientY }
}

function finishGesture(event) {
  if (!gestureStart || !isMobile.value || openMobile.value) {
    resetGesture()
    return
  }

  const touch = event.changedTouches[0]
  const horizontalDistance = touch.clientX - gestureStart.x
  const verticalDistance = Math.abs(touch.clientY - gestureStart.y)
  resetGesture()

  if (horizontalDistance >= MIN_SWIPE_DISTANCE && horizontalDistance > verticalDistance) {
    setOpenMobile(true)
  }
}
</script>

<template>
  <div
    class="flex min-h-0 min-w-0 flex-1"
    @touchstart.passive="startGesture"
    @touchend="finishGesture"
    @touchcancel="resetGesture"
  >
    <slot />
  </div>
</template>
