<script setup>
import { computed, h } from 'vue'
import { sanitize, allowedCssProperties } from 'lettersanitizer'
import ArticleCitation from '@widget/components/ArticleCitation.vue'

const props = defineProps({
  html: { type: String, required: true }
})

const content = computed(() => {
  const parser = new DOMParser()
  const original = parser.parseFromString(props.html, 'text/html')
  const titles = new Map(
    [...original.querySelectorAll('a.ld-article-citation')].map((link) => [
      link.getAttribute('href'),
      link.getAttribute('title')
    ])
  )
  const safe = sanitize(props.html, '', {
    allowedSchemas: ['cid', 'https', 'http', 'mailto'],
    allowedCssProperties: [...allowedCssProperties, 'transform', 'transform-origin']
  })
  return { nodes: parser.parseFromString(safe, 'text/html').body.childNodes, titles }
})

const renderNode = (node) => {
  if (node.nodeType === Node.TEXT_NODE) return node.textContent
  if (node.nodeType !== Node.ELEMENT_NODE) return null

  const attributes = Object.fromEntries(
    [...node.attributes].map((attribute) => [attribute.name, attribute.value])
  )
  const title = content.value.titles.get(attributes.href)
  if (
    node.localName === 'a' &&
    [...node.classList].some((name) => name.endsWith('_ld-article-citation')) &&
    title
  ) {
    return h(ArticleCitation, {
      number: node.textContent.replace(/^\((\d+)\)$/, '$1'),
      title,
      href: attributes.href
    })
  }
  if (node.localName === 'sup' && node.querySelector('a[class$="_ld-article-citation"]')) {
    return h('span', attributes, [...node.childNodes].map(renderNode))
  }
  if (attributes.target === '_blank') attributes.rel = 'noopener noreferrer'
  return h(node.localName, attributes, [...node.childNodes].map(renderNode))
}

const ContentNodes = () =>
  h(
    'div',
    { class: 'native-html [&_pre]:whitespace-pre-wrap [&_code]:whitespace-pre-wrap' },
    [...content.value.nodes].map(renderNode)
  )
</script>

<template>
  <ContentNodes />
</template>
