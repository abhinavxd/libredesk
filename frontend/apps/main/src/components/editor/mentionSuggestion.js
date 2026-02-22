import { VueRenderer } from '@tiptap/vue-3'
import MentionList from './MentionList.vue'

export default {
  char: '@',
  allowSpaces: true,

  items: async ({ query, editor }) => {
    // Get the suggestion handler from editor options
    const getSuggestions = editor.options.editorProps?.getSuggestions
    if (!getSuggestions) return []

    const items = await getSuggestions(query)
    return items
  },

  render: () => {
    let component
    let popup

    return {
      onStart: (props) => {
        component = new VueRenderer(MentionList, {
          props: {
            ...props,
            query: props.query
          },
          editor: props.editor
        })

        if (!props.clientRect) {
          return
        }

        // Create popup container with CSS positioning
        popup = document.createElement('div')
        popup.style.position = 'fixed'
        popup.style.zIndex = '9999'
        if (component.element) {
          popup.appendChild(component.element)
        }
        document.body.appendChild(popup)

        // Position the popup
        updatePosition(popup, props.clientRect)
      },

      onUpdate: (props) => {
        component.updateProps({
          ...props,
          query: props.query
        })

        if (!props.clientRect || !popup) {
          return
        }

        updatePosition(popup, props.clientRect)
      },

      onKeyDown: (props) => {
        if (props.event.key === 'Escape') {
          if (popup) {
            popup.style.display = 'none'
          }
          return true
        }

        return component.ref?.onKeyDown(props)
      },

      onExit: () => {
        if (popup && popup.parentNode) {
          popup.parentNode.removeChild(popup)
        }
        component.destroy()
      }
    }
  }
}

function updatePosition(popup, clientRect) {
  const rect = clientRect()
  if (!rect) return

  // Position below the cursor
  popup.style.left = `${rect.left}px`
  popup.style.top = `${rect.bottom + 4}px`

  // Ensure popup doesn't go off-screen to the right
  requestAnimationFrame(() => {
    const popupRect = popup.getBoundingClientRect()
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight

    // Adjust horizontal position if needed
    if (popupRect.right > viewportWidth) {
      popup.style.left = `${viewportWidth - popupRect.width - 8}px`
    }

    // If popup goes below viewport, show above cursor
    if (popupRect.bottom > viewportHeight) {
      popup.style.top = `${rect.top - popupRect.height - 4}px`
    }
  })
}
