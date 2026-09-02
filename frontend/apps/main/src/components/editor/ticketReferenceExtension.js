import Mention from '@tiptap/extension-mention'

export const conversationReferenceHref = (uuid) =>
  `/inboxes/all/conversation/${encodeURIComponent(uuid)}`

export const TicketReference = Mention.extend({
  name: 'ticketReference',
  // Link's a[href] mark rule is registered first at the default priority and would swallow the anchor.
  parseHTML: () => [{ tag: 'a.ld-ticket-reference', priority: 51 }],
  renderText: ({ node }) => `#${node.attrs.label}`,
  renderHTML: ({ node, HTMLAttributes }) => [
    'a',
    {
      ...HTMLAttributes,
      href: conversationReferenceHref(node.attrs.id),
      class: [HTMLAttributes.class, 'ld-ticket-reference'].filter(Boolean).join(' ')
    },
    `#${node.attrs.label}`
  ]
})
