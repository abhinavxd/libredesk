<script setup>
import { onMounted, ref, watch } from 'vue'
import { useResizeObserver } from '@vueuse/core'
import { DEFAULT_SITE_TITLE } from '@main/stores/appSettings'

// logo is the admin's site logo; blank falls back to the Fernmail fern.
const props = defineProps({
  name: { type: String, default: DEFAULT_SITE_TITLE },
  logo: { type: String, default: '' },
  // fit shrinks a long name toward minFontSize (px) before it truncates.
  fit: { type: Boolean, default: false },
  minFontSize: { type: Number, default: 14 }
})

const root = ref(null)
const label = ref(null)

// Measure at the inherited size, then scale down by how much the name overflows.
function fitName() {
  const el = label.value
  if (!props.fit || !el) return
  el.style.fontSize = ''
  const overflow = el.clientWidth / el.scrollWidth
  if (!(overflow < 1)) return
  const base = parseFloat(getComputedStyle(el).fontSize)
  el.style.fontSize = `${Math.max(props.minFontSize, Math.floor(base * overflow * 2) / 2)}px`
}

useResizeObserver(root, fitName)
watch(() => [props.name, props.logo, props.fit], fitName, { flush: 'post' })
// The web font changes the name's width once it loads.
onMounted(() => document.fonts?.ready.then(fitName))
</script>

<template>
  <span
    ref="root"
    class="min-w-0 max-w-full items-center gap-2 font-semibold tracking-tight"
    :class="fit ? 'flex w-full' : 'inline-flex'"
  >
    <img
      v-if="logo"
      :src="logo"
      alt=""
      class="h-7 w-auto max-w-[8rem] shrink-0 object-contain"
    />
    <img v-else src="/images/fern.svg" alt="" class="h-7 w-7 shrink-0" />
    <span ref="label" class="truncate" :title="name">{{ name }}</span>
  </span>
</template>
