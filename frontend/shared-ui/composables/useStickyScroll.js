import { ref, onMounted, onUnmounted } from 'vue'

export function useStickyScroll (scrollEl, contentEl, options = {}) {
  const {
    tolerance = 100,
    skipAutoScroll = () => false,
    onArriveBottom = () => { }
  } = options

  const hasUserScrolled = ref(false)
  let lastScrollTop = 0
  let isProgrammaticScroll = false
  let resizeObserver = null

  const scrollToBottom = () => {
    const el = scrollEl.value
    if (!el) return
    isProgrammaticScroll = true
    el.scrollTop = el.scrollHeight
    lastScrollTop = el.scrollTop
    requestAnimationFrame(() => { isProgrammaticScroll = false })
  }

  const scrollToOffset = (top) => {
    const el = scrollEl.value
    if (!el) return
    isProgrammaticScroll = true
    el.scrollTop = top
    lastScrollTop = el.scrollTop
    requestAnimationFrame(() => { isProgrammaticScroll = false })
  }

  const handleScroll = () => {
    const el = scrollEl.value
    if (!el) return
    const scrolledUp = el.scrollTop < lastScrollTop
    lastScrollTop = el.scrollTop
    if (isProgrammaticScroll) return
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight <= tolerance
    if (atBottom) {
      hasUserScrolled.value = false
      onArriveBottom()
    } else if (scrolledUp) {
      hasUserScrolled.value = true
    }
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
