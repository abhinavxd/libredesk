import Mention from '@tiptap/extension-mention'

const conversationPath = /^\/inboxes\/all\/conversation\/[^/?#]+$/

export const isSafeTicketReferenceHref = (href) => {
  if (!href || href.startsWith('//')) return false

  try {
    const url = new URL(href, window.location.origin)
    return (
      (url.protocol === 'http:' || url.protocol === 'https:') &&
      url.origin === window.location.origin &&
      conversationPath.test(url.pathname)
    )
  } catch {
    return false
  }
}

export const TicketReference = Mention.extend({
  name: 'ticketReference',
  addAttributes() {
    return {
      ...this.parent?.(),
      href: {
        default: null,
        parseHTML: (element) => {
          const href = element.getAttribute('href')
          return isSafeTicketReferenceHref(href) ? href : null
        }
      },
      type: {
        default: 'ticket-reference',
        parseHTML: (element) => element.getAttribute('data-type') || 'ticket-reference',
        renderHTML: (attributes) => ({ 'data-type': attributes.type })
      }
    }
  },
  parseHTML: () => [{ tag: 'a.ld-ticket-reference' }],
  renderText: ({ node }) => `#${node.attrs.label}`,
  renderHTML: ({ node, HTMLAttributes }) => [
    'a',
    {
      ...HTMLAttributes,
      href: node.attrs.href,
      class: [HTMLAttributes.class, 'ld-ticket-reference'].filter(Boolean).join(' ')
    },
    `#${node.attrs.label}`
  ]
})
