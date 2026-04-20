<template>
  <Teleport to="body">
    <div
      v-if="modelValue"
      class="fixed inset-0 z-[9999] flex items-center justify-center bg-black/80"
      tabindex="0"
      ref="rootEl"
      @click.self="zoomScale === 1 ? close() : resetZoom()"
      @keydown.escape="close"
      @keydown.left="prev"
      @keydown.right="next"
      @wheel.prevent="onWheel"
    >
      <!-- Top toolbar -->
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
          download
          class="text-white/70 hover:text-white"
          :title="t('imageLightbox.download')"
          :aria-label="t('imageLightbox.download')"
          @click.stop
        >
          <Download :size="20" />
        </a>
        <button
          class="text-white hover:text-gray-300"
          :title="t('imageLightbox.close')"
          :aria-label="t('imageLightbox.close')"
          @click="close"
        >
          <X :size="24" />
        </button>
      </div>

      <!-- Counter -->
      <div
        v-if="images.length > 1"
        class="absolute top-4 left-4 text-white/70 text-sm z-10"
      >
        {{ index + 1 }} / {{ images.length }}
      </div>

      <!-- Prev / Next -->
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

      <!-- Loading spinner -->
      <div
        v-if="imageLoading"
        class="absolute inset-0 flex items-center justify-center pointer-events-none"
      >
        <div class="w-8 h-8 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
      </div>

      <!-- Zoomable image -->
      <div
        :class="zoomScale > 1 ? 'overflow-visible' : 'overflow-hidden'"
        style="max-width: 90vw; max-height: 90vh;"
        @pointerdown.prevent="startPan"
        @touchstart.prevent="onTouchStart"
        @touchmove.prevent="onTouchMove"
        @touchend="onTouchEnd"
      >
        <img
          v-if="currentImage"
          :key="currentImage.url"
          :src="currentImage.url"
          :alt="currentImage.name || ''"
          class="max-w-[90vw] max-h-[90vh] object-contain rounded shadow-2xl select-none transition-opacity duration-150"
          :class="imageLoading ? 'opacity-0' : 'opacity-100'"
          :style="imageStyle"
          draggable="false"
          @load="imageLoading = false"
          @click.stop="zoomScale === 1 ? zoomIn() : null"
          @dblclick.stop="resetZoom"
        />
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { Download, X, ChevronLeft, ChevronRight, ZoomIn, ZoomOut } from 'lucide-vue-next'

const props = defineProps({
  // v-model:open style controls visibility
  modelValue: { type: Boolean, required: true },
  // Array of { url, name? } objects to flip through
  images: { type: Array, required: true },
  // Which image to show first when opened
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

const zoomIn = () => {
  zoomScale.value = Math.min(zoomScale.value * 1.4, 8)
}

const zoomOut = () => {
  zoomScale.value = Math.max(zoomScale.value / 1.4, 1)
  if (zoomScale.value === 1) resetZoom()
}

const onWheel = (e) => {
  if (e.deltaY < 0) {
    zoomScale.value = Math.min(zoomScale.value * 1.15, 8)
  } else {
    zoomScale.value = Math.max(zoomScale.value / 1.15, 1)
    if (zoomScale.value === 1) resetZoom()
  }
}

const startPan = (e) => {
  if (zoomScale.value <= 1) return
  isPanning.value = true
  panStart = { x: e.clientX, y: e.clientY, panX: panX.value, panY: panY.value }

  const onMove = (ev) => {
    panX.value = panStart.panX + (ev.clientX - panStart.x)
    panY.value = panStart.panY + (ev.clientY - panStart.y)
  }
  const onUp = () => {
    isPanning.value = false
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
  }
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
}

const onTouchStart = (e) => {
  if (e.touches.length === 2) {
    lastTouchDist = Math.hypot(
      e.touches[0].clientX - e.touches[1].clientX,
      e.touches[0].clientY - e.touches[1].clientY
    )
  } else if (e.touches.length === 1 && zoomScale.value > 1) {
    isPanning.value = true
    panStart = { x: e.touches[0].clientX, y: e.touches[0].clientY, panX: panX.value, panY: panY.value }
  }
}

const onTouchMove = (e) => {
  if (e.touches.length === 2) {
    const dist = Math.hypot(
      e.touches[0].clientX - e.touches[1].clientX,
      e.touches[0].clientY - e.touches[1].clientY
    )
    if (lastTouchDist > 0) {
      const delta = dist / lastTouchDist
      zoomScale.value = Math.max(1, Math.min(8, zoomScale.value * delta))
      if (zoomScale.value === 1) resetZoom()
    }
    lastTouchDist = dist
  } else if (e.touches.length === 1 && isPanning.value) {
    panX.value = panStart.panX + (e.touches[0].clientX - panStart.x)
    panY.value = panStart.panY + (e.touches[0].clientY - panStart.y)
  }
}

const onTouchEnd = () => {
  isPanning.value = false
  lastTouchDist = 0
}

const prev = () => {
  if (props.images.length <= 1) return
  imageLoading.value = true
  resetZoom()
  index.value = (index.value - 1 + props.images.length) % props.images.length
}

const next = () => {
  if (props.images.length <= 1) return
  imageLoading.value = true
  resetZoom()
  index.value = (index.value + 1) % props.images.length
}

// Preload neighbouring images so prev/next feels instant.
watch(index, () => {
  const imgs = props.images
  if (imgs.length <= 1) return
  const p = (index.value - 1 + imgs.length) % imgs.length
  const n = (index.value + 1) % imgs.length
  if (imgs[p]?.url) new Image().src = imgs[p].url
  if (imgs[n]?.url) new Image().src = imgs[n].url
})

// On open: jump to startIndex, focus the root for keyboard nav, preload all.
watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    index.value = Math.max(0, Math.min(props.startIndex, props.images.length - 1))
    imageLoading.value = false
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
