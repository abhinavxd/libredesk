import { Node, mergeAttributes } from '@tiptap/vue-3'

export const CALLOUT_VARIANTS = ['info', 'success', 'warning', 'danger']

export const Callout = Node.create({
  name: 'callout',
  group: 'block',
  content: 'block+',
  defining: true,

  addAttributes() {
    return {
      variant: {
        default: 'info',
        parseHTML: (el) => {
          const m = (el.className || '').match(/hc-callout-([a-z]+)/)
          return m && CALLOUT_VARIANTS.includes(m[1]) ? m[1] : 'info'
        },
        renderHTML: () => ({})
      }
    }
  },

  parseHTML() {
    return [{ tag: 'div.hc-callout' }]
  },

  renderHTML({ node, HTMLAttributes }) {
    const variant = node.attrs.variant
    return ['div', mergeAttributes(HTMLAttributes, { class: `hc-callout hc-callout-${variant}` }), 0]
  },

  addCommands() {
    return {
      toggleCallout:
        (variant = 'info') =>
        ({ editor, commands }) => {
          if (editor.isActive(this.name)) {
            return commands.updateAttributes(this.name, { variant })
          }
          return commands.wrapIn(this.name, { variant })
        },
      unsetCallout:
        () =>
        ({ commands }) =>
          commands.lift(this.name)
    }
  }
})
