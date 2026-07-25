import { Node, mergeAttributes } from '@tiptap/vue-3'

// Expandable section rendered as native <details>/<summary>. Editor CSS force-shows
// the collapsed body to keep it editable.
export const Details = Node.create({
  name: 'details',
  group: 'block',
  content: 'detailsSummary detailsContent',
  defining: true,
  isolating: true,

  parseHTML() {
    return [{ tag: 'details' }]
  },

  renderHTML({ HTMLAttributes }) {
    return ['details', mergeAttributes(HTMLAttributes, { class: 'hc-details' }), 0]
  },

  addCommands() {
    return {
      setDetails:
        () =>
        ({ chain }) =>
          chain()
            .insertContent({
              type: this.name,
              content: [
                { type: 'detailsSummary', content: [{ type: 'text', text: 'Summary' }] },
                { type: 'detailsContent', content: [{ type: 'paragraph' }] }
              ]
            })
            .run()
    }
  }
})

export const DetailsSummary = Node.create({
  name: 'detailsSummary',
  content: 'inline*',
  defining: true,
  isolating: true,

  parseHTML() {
    return [{ tag: 'summary' }]
  },

  renderHTML({ HTMLAttributes }) {
    return ['summary', mergeAttributes(HTMLAttributes, { class: 'hc-details-summary' }), 0]
  }
})

export const DetailsContent = Node.create({
  name: 'detailsContent',
  content: 'block+',
  defining: true,
  isolating: true,

  parseHTML() {
    return [{ tag: 'div.hc-details-content' }]
  },

  renderHTML({ HTMLAttributes }) {
    return ['div', mergeAttributes(HTMLAttributes, { class: 'hc-details-content' }), 0]
  }
})
