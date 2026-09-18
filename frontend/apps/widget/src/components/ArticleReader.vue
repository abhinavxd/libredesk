<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'

const props = defineProps({
  article: { type: Object, required: true },
  baseUrl: { type: String, required: true }
})

const height = ref('100%')

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

const onMessage = (event) => {
  if (event.origin !== frameOrigin.value || event.data?.source !== 'libredesk-hc-embed') return
  if (event.data.type === 'loaded') {
    height.value = '100%'
    frame.value?.parentElement?.scrollTo({ top: 0 })
  } else if (event.data.type === 'height' && Number.isFinite(event.data.height)) {
    height.value = `${event.data.height}px`
  }
}

onMounted(() => window.addEventListener('message', onMessage))
onUnmounted(() => window.removeEventListener('message', onMessage))
</script>

<template>
  <iframe
    ref="frame"
    :key="src"
    :src="src"
    :title="article.title"
    class="block w-full border-0"
    :style="{ height }"
  />
</template>
