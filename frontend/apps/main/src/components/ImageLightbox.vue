<template>
  <VueEasyLightbox
    :visible="modelValue"
    :imgs="imgs"
    :index="index"
    :loop="images.length > 1"
    teleport="body"
    @hide="close"
    @on-index-change="onIndexChange"
  />
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import VueEasyLightbox from 'vue-easy-lightbox'

const props = defineProps({
  modelValue: { type: Boolean, required: true },
  images: { type: Array, required: true },
  startIndex: { type: Number, default: 0 }
})
const emit = defineEmits(['update:modelValue'])

const index = ref(0)

const imgs = computed(() =>
  props.images.map((img) => ({ src: img.url, title: img.name || '' }))
)

function clamp(n, min, max) {
  return Math.min(Math.max(n, min), max)
}

function close() {
  emit('update:modelValue', false)
}

function onIndexChange(_prev, next) {
  index.value = next
}

function step(delta) {
  const total = props.images.length
  if (total <= 1) return
  index.value = (index.value + delta + total) % total
}

const KEY_ACTIONS = {
  ArrowLeft: () => step(-1),
  ArrowRight: () => step(1),
  Escape: close,
  ArrowUp: () => {},
  ArrowDown: () => {},
  PageUp: () => {},
  PageDown: () => {}
}

// Capture phase so we run before the lib's bubble listener AND before sibling
// listeners (message-list virtualizer, etc.) see the key.
function onDocKeydown(e) {
  if (!props.modelValue) return
  const action = KEY_ACTIONS[e.key]
  if (!action) return
  e.preventDefault()
  e.stopPropagation()
  action()
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      index.value = clamp(props.startIndex, 0, props.images.length - 1)
      document.addEventListener('keydown', onDocKeydown, true)
    } else {
      document.removeEventListener('keydown', onDocKeydown, true)
    }
  }
)
</script>

<style>
.vel-img-wrapper,
.vel-img,
.vel-fade-enter-active,
.vel-fade-leave-active {
  transition: none !important;
}
</style>
