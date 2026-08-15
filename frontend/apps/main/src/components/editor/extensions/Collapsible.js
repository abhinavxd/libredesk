import { Node, mergeAttributes } from '@tiptap/vue-3'
import { Plugin, PluginKey, TextSelection } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'
import { exitOnEmptyTrailingLine } from './exitBlock'

// Expandable section using native <details>. A closed one is not clickable, so a decoration
// marks it open in the editor without storing that in the saved HTML.
export const Details = Node.create({
  name: 'details',
  group: 'block',
  content: 'detailsSummary detailsContent',
  defining: true,
  isolating: true,

  addOptions() {
    return { defaultSummary: 'Summary' }
  },

  parseHTML() {
    return [{ tag: 'details' }]
  },

  renderHTML({ HTMLAttributes }) {
    return ['details', mergeAttributes(HTMLAttributes, { class: 'hc-details' }), 0]
  },

  addProseMirrorPlugins() {
    const typeName = this.name
    return [
      new Plugin({
        key: new PluginKey('detailsOpenInEditor'),
        props: {
          decorations: (state) => {
            const decorations = []
            state.doc.descendants((node, pos) => {
              if (node.type.name === typeName) {
                decorations.push(Decoration.node(pos, pos + node.nodeSize, { open: 'true' }))
              }
            })
            return DecorationSet.create(state.doc, decorations)
          }
        }
      })
    ]
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
                {
                  type: 'detailsSummary',
                  content: [{ type: 'text', text: this.options.defaultSummary }]
                },
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
  },

  // The summary is isolating, so a plain Enter would be swallowed; jump into the body instead.
  addKeyboardShortcuts() {
    return {
      Enter: () => {
        const { state, view } = this.editor
        const { $from, empty } = state.selection
        if (!empty || $from.parent.type.name !== this.name) return false
        const $body = state.doc.resolve($from.after() + 1)
        view.dispatch(state.tr.setSelection(TextSelection.near($body, 1)).scrollIntoView())
        return true
      }
    }
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
  },

  addKeyboardShortcuts() {
    return {
      Enter: () => exitOnEmptyTrailingLine(this.editor, 'details')
    }
  }
})
