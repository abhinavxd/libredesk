<script setup>
import { computed, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { sanitize, allowedCssProperties } from 'lettersanitizer'
import { BookOpen } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger
} from '@shared-ui/components/ui/tooltip'

const props = defineProps({
  html: { type: String, required: true }
})
const { t } = useI18n()

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
    return h(
      Tooltip,
      {},
      {
        default: () => [
          h(
            TooltipTrigger,
            { asChild: true },
            {
              default: () =>
                h(
                  Button,
                  {
                    as: 'a',
                    variant: 'ghost',
                    href: attributes.href,
                    target: '_blank',
                    rel: 'noopener noreferrer',
                    'aria-label': `${node.textContent} ${title}`,
                    class:
                      'h-auto px-0.5 py-0 text-[10px] leading-none !text-muted-foreground hover:!text-foreground'
                  },
                  { default: () => node.textContent }
                )
            }
          ),
          h(
            TooltipContent,
            { class: 'max-w-60 whitespace-normal break-words' },
            {
              default: () => [
                h('span', { class: 'mb-1 flex items-center gap-1 opacity-70' }, [
                  h(BookOpen, { class: 'h-3 w-3', 'aria-hidden': true }),
                  t('globals.terms.article')
                ]),
                h('span', { class: 'font-medium' }, title)
              ]
            }
          )
        ]
      }
    )
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
  <TooltipProvider :delay-duration="150">
    <ContentNodes />
  </TooltipProvider>
</template>
