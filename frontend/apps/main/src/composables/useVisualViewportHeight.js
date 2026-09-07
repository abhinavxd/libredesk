import { onMounted, onUnmounted } from 'vue'

// iOS Safari and Chrome Android keep dvh at full height while the soft keyboard is up; visualViewport tracks the space actually left.
export function useVisualViewportHeight () {
  const viewport = window.visualViewport
  const update = () =>
    document.documentElement.style.setProperty(
      '--visual-viewport-height',
      `${viewport?.height || window.innerHeight}px`
    )

  onMounted(() => {
    update()
    ;(viewport || window).addEventListener('resize', update)
  })

  onUnmounted(() => (viewport || window).removeEventListener('resize', update))
}
