import { VueRenderer } from '@tiptap/vue-3'

export function createSuggestionRenderer(ListComponent) {
  let component
  let popup
  let dismissed = false

  return {
    onStart: (props) => {
      dismissed = false
      component = new VueRenderer(ListComponent, {
        props: { ...props, query: props.query },
        editor: props.editor
      })
      if (!props.clientRect) return

      popup = document.createElement('div')
      popup.style.position = 'fixed'
      popup.style.zIndex = '9999'
      if (component.element) popup.appendChild(component.element)
      document.body.appendChild(popup)
      updatePosition(popup, props.clientRect)
    },

    onUpdate: (props) => {
      component.updateProps({ ...props, query: props.query })
      if (!props.clientRect || !popup) return
      updatePosition(popup, props.clientRect)
    },

    onKeyDown: (props) => {
      if (dismissed) return false
      if (props.event.key === 'Escape') {
        dismissed = true
        if (popup) popup.style.display = 'none'
        return true
      }
      return component.ref?.onKeyDown(props)
    },

    onExit: () => {
      popup?.remove()
      component.destroy()
    }
  }
}

function updatePosition(popup, clientRect) {
  const rect = clientRect()
  if (!rect) return

  popup.style.left = `${rect.left}px`
  popup.style.top = `${rect.bottom + 4}px`

  requestAnimationFrame(() => {
    const popupRect = popup.getBoundingClientRect()
    if (popupRect.right > window.innerWidth) {
      popup.style.left = `${window.innerWidth - popupRect.width - 8}px`
    }
    if (popupRect.bottom > window.innerHeight) {
      popup.style.top = `${rect.top - popupRect.height - 4}px`
    }
  })
}
