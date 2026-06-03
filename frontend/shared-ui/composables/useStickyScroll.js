import { ref, onMounted, onUnmounted } from 'vue'

// Sticks the scroll container to the bottom while the user hasn't scrolled away.
// Programmatic scroll events are distinguished from user scrolls via a one-shot flag.
// Apply `overflow-anchor: none` to the scroll container to prevent browser scroll anchoring
// from firing spurious scroll events during content growth.
export function useStickyScroll (scrollEl, contentEl, options = {}) {
  const {
    tolerance = 100,
    skipAutoScroll = () => false,
    onArriveBottom = () => { }
  } = options

  const hasUserScrolled = ref(false)
  let isProgrammaticScroll = false
  let resizeObserver = null

  const scrollToBottom = () => {
    const el = scrollEl.value
    if (!el) return
    isProgrammaticScroll = true
    el.scrollTop = el.scrollHeight
    requestAnimationFrame(() => { isProgrammaticScroll = false })
  }

  const scrollToOffset = (top) => {
    const el = scrollEl.value
    if (!el) return
    isProgrammaticScroll = true
    el.scrollTop = top
    requestAnimationFrame(() => { isProgrammaticScroll = false })
  }

  const handleScroll = () => {
    if (isProgrammaticScroll) return
    const el = scrollEl.value
    if (!el) return
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight <= tolerance
    hasUserScrolled.value = !atBottom
    if (atBottom) onArriveBottom()
  }

  const onContentResize = () => {
    if (skipAutoScroll() || hasUserScrolled.value) return
    scrollToBottom()
  }

  onMounted(() => {
    if (contentEl.value && typeof ResizeObserver !== 'undefined') {
      resizeObserver = new ResizeObserver(onContentResize)
      resizeObserver.observe(contentEl.value)
    }
  })

  onUnmounted(() => {
    if (resizeObserver) resizeObserver.disconnect()
  })

  return { hasUserScrolled, scrollToBottom, scrollToOffset, handleScroll }
}
