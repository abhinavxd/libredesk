<script setup>
import { computed, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { sanitize, allowedCssProperties } from 'lettersanitizer'
import { BookOpen } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { HoverCardRoot, HoverCardContent, HoverCardPortal, HoverCardTrigger } from 'reka-ui'

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
    const number = node.textContent.replace(/^\((\d+)\)$/, '$1')
    return h(
      HoverCardRoot,
      { openDelay: 150, closeDelay: 150 },
      {
        default: () => [
          h(
            HoverCardTrigger,
            { asChild: true },
            {
              default: () =>
                h(
                  Button,
                  {
                    as: 'button',
                    type: 'button',
                    variant: 'secondary',
                    'aria-label': `${number} ${title}`,
                    class:
                      'mx-0.5 h-auto rounded-full bg-foreground/10 px-1.5 py-0.5 text-sm leading-none !text-foreground hover:bg-foreground/15'
                  },
                  { default: () => number }
                )
            }
          ),
          h(
            HoverCardPortal,
            {},
            {
              default: () =>
                h(
                  HoverCardContent,
                  { asChild: true, side: 'top', sideOffset: 6, class: 'z-50' },
                  {
                    default: () =>
                      h(
                        Button,
                        {
                          as: 'a',
                          variant: 'secondary',
                          href: attributes.href,
                          target: '_blank',
                          rel: 'noopener noreferrer',
                          'aria-label': title,
                          class:
                            'h-auto max-w-60 flex-col items-start gap-1 whitespace-normal break-words p-3 text-left text-xs shadow-md hover:bg-accent'
                        },
                        {
                          default: () => [
                            h('span', { class: 'flex items-center gap-1 opacity-70' }, [
                              h(BookOpen, { class: 'h-3 w-3', 'aria-hidden': true }),
                              t('globals.terms.helpCenterArticle')
                            ]),
                            h('span', { class: 'font-medium' }, title)
                          ]
                        }
                      )
                  }
                )
            }
          )
        ]
      }
    )
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
