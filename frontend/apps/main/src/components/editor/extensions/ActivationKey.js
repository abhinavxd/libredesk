import { Node, mergeAttributes } from '@tiptap/vue-3'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { KEY_PLACEHOLDER_TYPE, keyPlaceholderLabel, newKeyPlaceholderID } from '@main/utils/activation-keys'

// An activation key placeholder: an atom, so it is selected, moved and
// deleted as one piece and its label can't be edited. Styles for
// `.ld-activation-key` are in editorStyles.scss.
export const ActivationKey = Node.create({
  name: 'activationKey',
  group: 'inline',
  inline: true,
  atom: true,
  selectable: true,
  draggable: false,

  addAttributes() {
    return {
      id: {
        default: null,
        parseHTML: (el) => el.getAttribute('data-id'),
        renderHTML: (attrs) => ({ 'data-id': attrs.id })
      },
      appId: {
        default: null,
        parseHTML: (el) => el.getAttribute('data-app-id'),
        renderHTML: (attrs) => ({ 'data-app-id': attrs.appId })
      }
    }
  },

  parseHTML() {
    return [{ tag: `span[data-type="${KEY_PLACEHOLDER_TYPE}"]`, priority: 100 }]
  },

  renderHTML({ node, HTMLAttributes }) {
    return [
      'span',
      mergeAttributes({ 'data-type': KEY_PLACEHOLDER_TYPE, class: 'ld-activation-key' }, HTMLAttributes),
      keyPlaceholderLabel(node.attrs.id)
    ]
  },

  renderText({ node }) {
    return keyPlaceholderLabel(node.attrs.id)
  },

  addCommands() {
    return {
      insertActivationKey:
        (appId) =>
        ({ commands }) =>
          commands.insertContent({ type: this.name, attrs: { id: newKeyPlaceholderID(), appId: String(appId) } }),
      // One game per email: switching the game re-targets every chip.
      setActivationKeyApp:
        (appId) =>
        ({ state, tr, dispatch }) => {
          state.doc.descendants((node, pos) => {
            if (node.type.name === this.name && node.attrs.appId !== String(appId)) {
              tr.setNodeMarkup(pos, undefined, { ...node.attrs, appId: String(appId) })
            }
          })
          if (dispatch) dispatch(tr)
          return true
        }
    }
  },

  addProseMirrorPlugins() {
    const name = this.name
    return [
      new Plugin({
        key: new PluginKey('activationKeyUnique'),
        // Each chip is one key: drop any copy that repeats an ID or has none.
        appendTransaction(transactions, _oldState, newState) {
          if (!transactions.some((tr) => tr.docChanged)) return null
          const seen = new Set()
          const extra = []
          newState.doc.descendants((node, pos) => {
            if (node.type.name !== name) return
            if (!node.attrs.id || seen.has(node.attrs.id)) extra.push(pos)
            else seen.add(node.attrs.id)
          })
          if (!extra.length) return null
          const tr = newState.tr
          extra.reverse().forEach((pos) => tr.delete(pos, pos + 1))
          return tr
        }
      })
    ]
  }
})

export default ActivationKey
