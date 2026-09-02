import { VueRenderer } from '@tiptap/vue-3'
import TicketReferenceList from './TicketReferenceList.vue'

export default {
  char: '#',
  allowSpaces: false,
  allow: ({ editor }) => editor.options.editorProps?.ticketReferencesEnabled?.() ?? false,
  items: async ({ query, editor }) =>
    editor.options.editorProps?.getTicketSuggestions?.(query) || [],
  render: () => {
    let component
    let popup
    const position = (rect) => {
      if (!rect || !popup) return
      const bounds = rect()
      if (!bounds) return
      popup.style.left = `${bounds.left}px`
      popup.style.top = `${bounds.bottom + 4}px`
    }
    return {
      onStart: (props) => {
        component = new VueRenderer(TicketReferenceList, {
          props: { ...props, query: props.query },
          editor: props.editor
        })
        if (!props.clientRect) return
        popup = document.createElement('div')
        popup.style.position = 'fixed'
        popup.style.zIndex = '9999'
        popup.appendChild(component.element)
        document.body.appendChild(popup)
        position(props.clientRect)
      },
      onUpdate: (props) => {
        component?.updateProps({ ...props, query: props.query })
        position(props.clientRect)
      },
      onKeyDown: (props) => {
        if (props.event.key === 'Escape') {
          if (popup) popup.style.display = 'none'
          return true
        }
        return component?.ref?.onKeyDown(props)
      },
      onExit: () => {
        popup?.remove()
        component?.destroy()
      }
    }
  }
}
