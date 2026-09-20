<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { Spinner } from '@shared-ui/components/ui/spinner'

const props = defineProps({
  article: { type: Object, required: true },
  baseUrl: { type: String, required: true }
})

const height = ref('100%')
const ready = ref(false)

const src = computed(
  () =>
    `${props.baseUrl}/${encodeURIComponent(props.article.locale)}/articles/${encodeURIComponent(props.article.slug)}?embed=1`
)

const frameOrigin = computed(() => {
  try {
    return new URL(props.baseUrl, window.location.origin).origin
  } catch {
    return ''
  }
})

const frame = ref(null)

watch(src, () => {
  ready.value = false
})

const onMessage = (event) => {
  if (event.origin !== frameOrigin.value || event.data?.source !== 'libredesk-hc-embed') return
  if (event.data.type === 'loaded') {
    ready.value = true
    height.value = '100%'
    frame.value?.parentElement?.scrollTo({ top: 0 })
  } else if (event.data.type === 'height' && Number.isFinite(event.data.height)) {
    height.value = `${event.data.height}px`
  } else if (event.data.type === 'scroll' && Number.isFinite(event.data.top)) {
    const scroller = frame.value?.parentElement
    if (scroller) scroller.scrollTo({ top: frame.value.offsetTop + event.data.top, behavior: 'smooth' })
  }
}

onMounted(() => window.addEventListener('message', onMessage))
onUnmounted(() => window.removeEventListener('message', onMessage))
</script>

<template>
  <div class="relative min-h-full">
    <div
      v-if="!ready"
      class="absolute inset-0 flex items-center justify-center bg-background"
      role="status"
    >
      <Spinner size="md" :absolute="false" :center="false" />
    </div>
    <iframe
      ref="frame"
      :key="src"
      :src="src"
      :title="article.title"
      class="block w-full border-0"
      :style="{ height }"
    />
  </div>
</template>
