<script setup>
import { computed, ref } from 'vue'
import articleCSS from '@public-static/article-content.css?raw'

const props = defineProps({ article: { type: Object, required: true }, helpSlug: { type: String, required: true }, dark: Boolean })
const emit = defineEmits(['article'])
const frame = ref(null)
const documentHTML = computed(() => {
  const doc = document.implementation.createHTMLDocument(props.article.title)
  const policy = doc.createElement('meta')
  policy.httpEquiv = 'Content-Security-Policy'
  policy.content = "default-src 'none'; img-src https: http: data:; style-src 'unsafe-inline'; font-src 'none'; frame-src 'none'; form-action 'none'; base-uri 'none'"
  doc.head.append(policy)
  const style = doc.createElement('style')
  style.textContent = `${articleCSS}\nbody{margin:0;padding:16px;font:14px/1.6 system-ui;overflow-wrap:anywhere}img{max-width:100%}a{color:inherit}table{display:block;overflow:auto}.hc-prose pre{background:var(--hc-accent-tint);color:inherit}.hc-callout{background:var(--hc-accent-tint);color:inherit}`
  doc.head.append(style)
  doc.body.className = 'hc-prose'
  doc.body.innerHTML = props.article.content
  for (const element of doc.querySelectorAll('script,iframe,object,embed,form,base,meta,link,style')) {
    if (!doc.head.contains(element)) element.remove()
  }
  for (const element of doc.body.querySelectorAll('*')) {
    for (const attr of [...element.attributes]) {
      if (attr.name.startsWith('on') || attr.name === 'srcdoc') element.removeAttribute(attr.name)
    }
    for (const attr of ['href', 'src']) {
      const value = element.getAttribute(attr)
      if (!value || value.startsWith('#')) continue
      let url
      try { url = new URL(value, window.location.origin) } catch { element.removeAttribute(attr); continue }
      if (!['https:', 'http:', 'mailto:'].includes(url.protocol)) element.removeAttribute(attr)
      else element.setAttribute(attr, url.href)
    }
  }
  return '<!doctype html>' + doc.documentElement.outerHTML
})
const loaded = () => {
  const doc = frame.value.contentDocument
  if (!doc) return
  const theme = getComputedStyle(frame.value)
  doc.body.style.color = theme.color
  doc.body.style.background = theme.backgroundColor
  for (const [target, source] of Object.entries({ '--hc-border': '--border', '--hc-accent': '--primary', '--hc-accent-ink': '--primary', '--hc-accent-tint': '--muted', '--hc-muted': '--muted-foreground' })) {
    doc.documentElement.style.setProperty(target, `hsl(${theme.getPropertyValue(source)})`)
  }
  doc.addEventListener('click', event => {
    const link = event.target.closest('a[href]')
    if (!link || link.getAttribute('href').startsWith('#')) return
    event.preventDefault()
    const url = new URL(link.href)
    const match = url.pathname.match(/^\/hc\/([^/]+)\/([^/]+)\/articles\/([^/]+)$/)
    if (url.origin === window.location.origin && match?.[1] === props.helpSlug) emit('article', { slug: decodeURIComponent(match[3]), locale: decodeURIComponent(match[2]) })
    else window.open(url.href, '_blank', 'noopener,noreferrer')
  })
}
</script>

<template>
  <iframe ref="frame" :key="`${article.id}-${dark}`" :title="article.title" :srcdoc="documentHTML" sandbox="allow-same-origin" class="w-full flex-1 min-h-0 border-0 bg-background text-foreground" @load="loaded" />
</template>
