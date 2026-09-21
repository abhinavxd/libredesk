<template>
  <div ref="container">
    <iframe
      v-if="displayHTML !== null"
      ref="frame"
      :srcdoc="frameDocument"
      :title="t('globals.terms.message')"
      sandbox="allow-same-origin"
      referrerpolicy="no-referrer"
      class="block w-full border-0"
      :style="{ height: frameHeight + 'px' }"
      @load="onFrameLoad"
    />
    <div v-else class="whitespace-pre-wrap">{{ textContent }}</div>
    <p v-if="showBlockedNotice && displayHTML !== null && message.display.blocked_images > 0" class="mt-2 text-xs text-muted-foreground">
      {{ t('conversation.imagesBlocked', { count: message.display.blocked_images }) }}
      <span v-if="blockedDomains"> ({{ blockedDomains }})</span>
    </p>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { messageDocument } from '../utils/messageDocument.js'
import { localMediaURL } from '../utils/resourceURL.js'

const props = defineProps({
  message: { type: Object, required: true },
  showBlockedNotice: { type: Boolean, default: true },
  showQuotedText: { type: Boolean, default: true }
})
const emit = defineEmits(['image-click', 'resize'])
const { t } = useI18n()
const container = ref(null)
const frame = ref(null)
const frameHeight = ref(1)
let resizeObserver
let themeObserver

const displayHTML = computed(() =>
  props.message.content_type !== 'text' && typeof props.message.display?.html === 'string'
    ? props.message.display.html
    : null
)
const frameDocument = computed(() => displayHTML.value === null ? '' : messageDocument(displayHTML.value))

function resizeFrame() {
  const body = frame.value?.contentDocument?.body
  if (!body) return
  frameHeight.value = Math.max(1, Math.ceil(body.getBoundingClientRect().height))
  nextTick(() => emit('resize'))
}

function updateAppearance() {
  const body = frame.value?.contentDocument?.body
  if (!body || !container.value) return
  const style = getComputedStyle(container.value)
  body.style.color = style.color
  body.style.fontSize = style.fontSize
  body.style.lineHeight = style.lineHeight
  body.style.setProperty('--link-color', `hsl(${style.getPropertyValue('--link')})`)
  body.classList.toggle('hide-quotes', !props.showQuotedText)
  resizeFrame()
}

function onContentClick(event) {
  if (event.button > 1) return
  const image = event.target.closest?.('img')
  const anchor = event.target.closest?.('a')
  if (image) {
    event.preventDefault()
    const images = Array.from(frame.value.contentDocument.querySelectorAll('img'))
      .map(el => ({ url: localMediaURL(el.getAttribute('src')), name: el.getAttribute('alt') || '' }))
      .filter(el => el.url)
    const index = images.findIndex(el => el.url === localMediaURL(image.getAttribute('src')))
    if (index >= 0) emit('image-click', { images, index })
  } else if (anchor) {
    event.preventDefault()
    try {
      const url = new URL(anchor.getAttribute('href'), window.location.origin)
      if (['https:', 'http:', 'mailto:'].includes(url.protocol)) {
        window.open(url.href, '_blank', 'noopener,noreferrer')
      }
    } catch {
      return
    }
  }
}

function disconnectObservers() {
  resizeObserver?.disconnect()
  themeObserver?.disconnect()
}

function onFrameLoad() {
  disconnectObservers()
  const body = frame.value?.contentDocument?.body
  if (!body) return
  body.addEventListener('click', onContentClick)
  body.addEventListener('auxclick', onContentClick)
  resizeObserver = new ResizeObserver(resizeFrame)
  resizeObserver.observe(body)
  themeObserver = new MutationObserver(updateAppearance)
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class', 'style'] })
  updateAppearance()
}

watch(() => props.showQuotedText, updateAppearance)
watch(displayHTML, disconnectObservers)
onBeforeUnmount(disconnectObservers)
const textContent = computed(() =>
  props.message.content_type === 'text'
    ? props.message.content || ''
    : props.message.text_content || props.message.content || ''
)
const blockedDomains = computed(() =>
  Array.isArray(props.message.display?.blocked_domains)
    ? props.message.display.blocked_domains.filter(domain => typeof domain === 'string').join(', ')
    : ''
)
</script>
