import Mention from '@tiptap/extension-mention'

export const TicketReference = Mention.extend({
  name: 'ticketReference',
  addAttributes() {
    return {
      ...this.parent?.(),
      href: { default: null, parseHTML: (element) => element.getAttribute('href') },
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
