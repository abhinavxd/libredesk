import { Extension } from '@tiptap/vue-3'
import { Plugin, PluginKey } from '@tiptap/pm/state'

// A block at the very end of the doc leaves nowhere to put the cursor.
export const TrailingNode = Extension.create({
  name: 'trailingNode',

  addProseMirrorPlugins() {
    return [
      new Plugin({
        key: new PluginKey('trailingNode'),
        appendTransaction: (_transactions, _oldState, state) => {
          const { doc, tr, schema } = state
          if (doc.lastChild?.type.name === 'paragraph') return
          return tr.insert(doc.content.size, schema.nodes.paragraph.create())
        }
      })
    ]
  }
})
