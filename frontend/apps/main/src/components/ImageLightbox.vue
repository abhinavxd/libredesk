<template>
  <Teleport to="body">
    <div
      v-if="modelValue"
      class="fixed inset-0 z-[9999] flex items-center justify-center bg-black/80"
      tabindex="0"
      role="dialog"
      aria-modal="true"
      aria-label="Image preview"
      ref="rootEl"
      @click.self="onBackdropClick"
      @keydown.escape="close"
      @keydown.left="prev"
      @keydown.right="next"
      @keydown.tab="trapFocus"
      @wheel.prevent="onWheel"
    >
      <div class="absolute top-4 right-4 flex items-center gap-3 z-10">
        <button
          class="text-white/70 hover:text-white flex items-center gap-1 text-sm"
          :title="t('imageLightbox.zoomIn')"
          :aria-label="t('imageLightbox.zoomIn')"
          @click.stop="zoomIn"
        >
          <ZoomIn :size="20" />
        </button>
        <button
          class="text-white/70 hover:text-white text-xs font-mono min-w-[3rem] text-center"
          :title="t('imageLightbox.resetZoom')"
          :aria-label="t('imageLightbox.resetZoom')"
          @click.stop="resetZoom"
        >
          {{ Math.round(zoomScale * 100) }}%
        </button>
        <button
          class="text-white/70 hover:text-white flex items-center gap-1 text-sm"
          :title="t('imageLightbox.zoomOut')"
          :aria-label="t('imageLightbox.zoomOut')"
          @click.stop="zoomOut"
        >
          <ZoomOut :size="20" />
        </button>
        <a
          v-if="currentImage?.url"
          :href="currentImage.url"
          target="_blank"
          rel="noopener"
          class="text-white/70 hover:text-white"
          :title="t('globals.terms.download')"
          :aria-label="t('globals.terms.download')"
          @click.stop
        >
          <Download :size="20" />
        </a>
        <button
          class="text-white hover:text-gray-300"
          :title="t('globals.messages.close')"
          :aria-label="t('globals.messages.close')"
          @click="close"
        >
          <X :size="24" />
        </button>
      </div>

      <div
        v-if="images.length > 1"
        class="absolute top-4 left-4 text-white/70 text-sm z-10"
      >
        {{ index + 1 }} / {{ images.length }}
      </div>

      <button
        v-if="images.length > 1"
        class="absolute left-4 top-1/2 -translate-y-1/2 text-white hover:text-gray-300 z-10 p-2"
        :aria-label="t('imageLightbox.previous')"
        @click.stop="prev"
      >
        <ChevronLeft :size="32" />
      </button>
      <button
        v-if="images.length > 1"
        class="absolute right-4 top-1/2 -translate-y-1/2 text-white hover:text-gray-300 z-10 p-2"
        :aria-label="t('imageLightbox.next')"
        @click.stop="next"
      >
        <ChevronRight :size="32" />
      </button>

      <div
        v-if="imageLoading"
        class="absolute inset-0 flex items-center justify-center pointer-events-none"
      >
        <div class="w-8 h-8 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
      </div>

      <img
        v-if="currentImage"
        :key="currentImage.url"
        :ref="onImgRef"
        :src="currentImage.url"
        :alt="currentImage.name || ''"
        class="max-w-[90vw] max-h-[90vh] object-contain rounded shadow-2xl select-none transition-opacity duration-150"
        :class="imageLoading ? 'opacity-0' : 'opacity-100'"
        :style="imageStyle"
        draggable="false"
        @pointerdown.prevent="startPan"
        @touchstart.prevent="onTouchStart"
        @touchmove.prevent="onTouchMove"
        @touchend="onTouchEnd"
        @load="imageLoading = false"
        @click.stop="zoomScale === 1 && zoomIn()"
        @dblclick.stop="resetZoom"
      />
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { Download, X, ChevronLeft, ChevronRight, ZoomIn, ZoomOut } from 'lucide-vue-next'

const ZOOM_MIN = 1
const ZOOM_MAX = 8
const ZOOM_STEP = 1.4
const WHEEL_STEP = 1.15

const props = defineProps({
  modelValue: { type: Boolean, required: true },
  images: { type: Array, required: true },
  startIndex: { type: Number, default: 0 }
})
const emit = defineEmits(['update:modelValue'])

const { t } = useI18n()

const rootEl = ref(null)
const index = ref(0)
const imageLoading = ref(false)

const zoomScale = ref(1)
const panX = ref(0)
const panY = ref(0)
const isPanning = ref(false)
let panStart = { x: 0, y: 0, panX: 0, panY: 0 }
let lastTouchDist = 0
let previouslyFocused = null
// When a pan ends over the backdrop, browsers synthesize a click event
// in the same gesture. Suppress that one click so the pan doesn't
// accidentally trigger resetZoom or close.
let suppressNextClick = false

const currentImage = computed(() => props.images[index.value])

const imageStyle = computed(() => ({
  transform: `scale(${zoomScale.value}) translate(${panX.value / zoomScale.value}px, ${panY.value / zoomScale.value}px)`,
  cursor: zoomScale.value > 1 ? 'grab' : 'zoom-in',
  transition: isPanning.value ? 'none' : 'transform 0.15s ease'
}))

const close = () => emit('update:modelValue', false)

const resetZoom = () => {
  zoomScale.value = 1
  panX.value = 0
  panY.value = 0
}

const applyZoom = (factor) => {
  const next = Math.max(ZOOM_MIN, Math.min(ZOOM_MAX, zoomScale.value * factor))
  zoomScale.value = next
  if (next === ZOOM_MIN) resetZoom()
}

const zoomIn = () => applyZoom(ZOOM_STEP)
const zoomOut = () => applyZoom(1 / ZOOM_STEP)
const onWheel = (e) => applyZoom(e.deltaY < 0 ? WHEEL_STEP : 1 / WHEEL_STEP)

const beginPan = (clientX, clientY) => {
  isPanning.value = true
  panStart = { x: clientX, y: clientY, panX: panX.value, panY: panY.value }
}

const updatePan = (clientX, clientY) => {
  panX.value = panStart.panX + (clientX - panStart.x)
  panY.value = panStart.panY + (clientY - panStart.y)
}

const touchDistance = (touches) =>
  Math.hypot(touches[0].clientX - touches[1].clientX, touches[0].clientY - touches[1].clientY)

const startPan = (e) => {
  if (zoomScale.value <= 1) return
  beginPan(e.clientX, e.clientY)

  const onMove = (ev) => updatePan(ev.clientX, ev.clientY)
  const onUp = () => {
    isPanning.value = false
    suppressNextClick = true
    setTimeout(() => { suppressNextClick = false }, 0)
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
  }
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
}

const onBackdropClick = () => {
  if (suppressNextClick) return
  if (zoomScale.value === 1) close()
  else resetZoom()
}

// `<img>` `@load` can race with Vue's listener attachment when the image
// is already in cache (preloaded neighbour). Sync state from the element
// itself once the ref is set.
const onImgRef = (el) => {
  if (el?.complete && el.naturalHeight > 0) imageLoading.value = false
}

const trapFocus = (e) => {
  const focusables = rootEl.value?.querySelectorAll(
    'button, [href], [tabindex]:not([tabindex="-1"])'
  )
  if (!focusables || focusables.length === 0) return
  const first = focusables[0]
  const last = focusables[focusables.length - 1]
  const active = document.activeElement
  if (e.shiftKey && (active === first || active === rootEl.value)) {
    e.preventDefault()
    last.focus()
  } else if (!e.shiftKey && active === last) {
    e.preventDefault()
    first.focus()
  }
}

const onTouchStart = (e) => {
  if (e.touches.length === 2) {
    lastTouchDist = touchDistance(e.touches)
    return
  }
  if (e.touches.length === 1 && zoomScale.value > 1) {
    beginPan(e.touches[0].clientX, e.touches[0].clientY)
  }
}

const onTouchMove = (e) => {
  if (e.touches.length === 2) {
    const dist = touchDistance(e.touches)
    if (lastTouchDist > 0) applyZoom(dist / lastTouchDist)
    lastTouchDist = dist
    return
  }
  if (e.touches.length === 1 && isPanning.value) {
    updatePan(e.touches[0].clientX, e.touches[0].clientY)
  }
}

const onTouchEnd = () => {
  isPanning.value = false
  lastTouchDist = 0
}

const step = (delta) => {
  const total = props.images.length
  if (total <= 1) return
  imageLoading.value = true
  resetZoom()
  index.value = (index.value + delta + total) % total
}

const prev = () => step(-1)
const next = () => step(1)

// Preload neighbouring images so prev/next feels instant.
watch(index, () => {
  const imgs = props.images
  if (imgs.length <= 1) return
  const p = (index.value - 1 + imgs.length) % imgs.length
  const n = (index.value + 1) % imgs.length
  if (imgs[p]?.url) new Image().src = imgs[p].url
  if (imgs[n]?.url) new Image().src = imgs[n].url
})

watch(
  () => props.modelValue,
  (open) => {
    if (!open) {
      previouslyFocused?.focus?.()
      previouslyFocused = null
      return
    }
    previouslyFocused = document.activeElement
    index.value = Math.max(0, Math.min(props.startIndex, props.images.length - 1))
    imageLoading.value = true
    resetZoom()
    nextTick(() => {
      rootEl.value?.focus()
      props.images.forEach((img) => {
        if (img?.url) new Image().src = img.url
      })
    })
  }
)
</script>
