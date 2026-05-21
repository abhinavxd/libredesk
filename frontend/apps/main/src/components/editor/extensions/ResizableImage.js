import Image from '@tiptap/extension-image'
import { getI18n } from '@main/i18n'

// Styles for `.image-resizer`, `.image-resize-handle*`, `.image-size-toolbar`,
// and `.image-upload-placeholder*` are in TextEditor.vue's global <style>
// block because they need to apply inside the tiptap-rendered DOM.
export const ResizableImage = Image.extend({
  addAttributes () {
    return {
      ...this.parent?.(),
      width: {
        default: null,
        parseHTML: (el) => el.getAttribute('width') || el.style.width?.replace('px', '') || null,
        renderHTML: (attrs) => {
          if (!attrs.width) return {}
          return { width: attrs.width, style: `width: ${attrs.width}px` }
        }
      },
      height: {
        default: null,
        parseHTML: (el) => el.getAttribute('height') || null,
        renderHTML: (attrs) => (attrs.height ? { height: attrs.height } : {})
      },
      // Transient placeholder state - never persisted in HTML.
      uploading: {
        default: false,
        parseHTML: () => false,
        renderHTML: () => ({})
      },
      uploadId: {
        default: null,
        parseHTML: () => null,
        renderHTML: () => ({})
      },
      uploadName: {
        default: null,
        parseHTML: () => null,
        renderHTML: () => ({})
      }
    }
  },
  renderHTML (props) {
    // Don't serialize uploading placeholders - they shouldn't end up in
    // saved drafts or sent messages.
    if (props.node.attrs.uploading) {
      return ['span', { 'data-upload-placeholder': '' }]
    }
    return this.parent?.(props) ?? ['img', props.HTMLAttributes]
  },
  addNodeView () {
    return ({ node, getPos, editor: nodeEditor }) => {
      const t = getI18n().global.t

      const wrapper = document.createElement('div')
      wrapper.classList.add('image-resizer')
      wrapper.style.display = 'inline-block'
      wrapper.style.position = 'relative'
      wrapper.style.lineHeight = '0'

      const placeholder = document.createElement('div')
      placeholder.classList.add('image-upload-placeholder')
      const placeholderRow = document.createElement('div')
      placeholderRow.classList.add('image-upload-placeholder-row')
      const spinner = document.createElement('div')
      spinner.className = 'w-7 h-7 border-2 border-muted-foreground border-t-primary rounded-full animate-spin'
      const nameEl = document.createElement('span')
      nameEl.classList.add('image-upload-placeholder-name')
      placeholderRow.appendChild(spinner)
      placeholderRow.appendChild(nameEl)
      placeholder.appendChild(placeholderRow)
      wrapper.appendChild(placeholder)

      const img = document.createElement('img')
      img.classList.add('inline-image')
      img.style.maxWidth = '100%'
      img.style.height = 'auto'
      wrapper.appendChild(img)

      const toolbar = document.createElement('div')
      toolbar.classList.add('image-size-toolbar')

      let naturalWidth = 0
      img.addEventListener('load', () => { naturalWidth = img.naturalWidth })

      const commitWidth = (newWidth) => {
        const pos = getPos()
        if (typeof pos !== 'number') return
        const current = nodeEditor.state.doc.nodeAt(pos)
        if (!current) return
        nodeEditor.chain().focus().command(({ tr }) => {
          tr.setNodeMarkup(pos, undefined, { ...current.attrs, width: newWidth || null })
          return true
        }).run()
      }

      const clampToNatural = (w) => (naturalWidth ? Math.min(w, naturalWidth) : w)
      const sizes = [
        { label: t('globals.terms.small'), getWidth: () => clampToNatural(400) },
        { label: t('globals.messages.bestFit'), getWidth: () => clampToNatural(nodeEditor.view.dom.clientWidth) }
      ]
      // Toolbar buttons use pointerdown so touch + pen + mouse all work.
      // preventDefault avoids stealing focus from the editor.
      sizes.forEach(({ label, getWidth }) => {
        const btn = document.createElement('button')
        btn.textContent = label
        btn.type = 'button'
        btn.addEventListener('pointerdown', (e) => {
          e.preventDefault()
          e.stopPropagation()
          const w = getWidth()
          img.style.width = w + 'px'
          commitWidth(w)
        })
        toolbar.appendChild(btn)
      })

      const sep = document.createElement('span')
      sep.classList.add('image-toolbar-sep')
      toolbar.appendChild(sep)

      const removeBtn = document.createElement('button')
      removeBtn.textContent = t('globals.terms.remove')
      removeBtn.type = 'button'
      removeBtn.classList.add('image-toolbar-remove')
      removeBtn.addEventListener('pointerdown', (e) => {
        e.preventDefault()
        e.stopPropagation()
        const pos = getPos()
        if (typeof pos === 'number') {
          nodeEditor.chain().focus().deleteRange({ from: pos, to: pos + 1 }).run()
        }
      })
      toolbar.appendChild(removeBtn)
      wrapper.appendChild(toolbar)

      // Inline images grow rightward in text flow, so width is the only
      // axis we can actually change. Left-side handles flip the sign so
      // dragging outward in either direction reads as "grow."
      const corners = [
        { className: 'image-resize-handle-tl', direction: -1 },
        { className: 'image-resize-handle-tr', direction: 1 },
        { className: 'image-resize-handle-bl', direction: -1 },
        { className: 'image-resize-handle-br', direction: 1 }
      ]

      let startX = 0
      let startWidth = 0
      let activeDirection = 1
      const onPointerMove = (e) => {
        const newWidth = Math.max(50, startWidth + activeDirection * (e.clientX - startX))
        img.style.width = newWidth + 'px'
      }
      const onPointerUp = () => {
        window.removeEventListener('pointermove', onPointerMove)
        window.removeEventListener('pointerup', onPointerUp)
        wrapper.classList.remove('resizing')
        try {
          commitWidth(Math.round(img.offsetWidth))
        } catch (err) {
          // Node may have been removed/replaced mid-drag (autosave
          // re-render, paste over selection, etc.). Drop the commit.
        }
      }

      corners.forEach(({ className, direction }) => {
        const handle = document.createElement('div')
        handle.classList.add('image-resize-handle', className)
        handle.addEventListener('pointerdown', (e) => {
          e.preventDefault()
          e.stopPropagation()
          startX = e.clientX
          startWidth = img.offsetWidth
          activeDirection = direction
          window.addEventListener('pointermove', onPointerMove)
          window.addEventListener('pointerup', onPointerUp)
          wrapper.classList.add('resizing')
        })
        wrapper.appendChild(handle)
      })

      const applyState = (n) => {
        if (n.attrs.uploading) {
          wrapper.classList.add('uploading')
          nameEl.textContent = n.attrs.uploadName || ''
        } else {
          wrapper.classList.remove('uploading')
          img.src = n.attrs.src
          img.alt = n.attrs.alt || ''
          img.title = n.attrs.title || ''
          img.style.width = n.attrs.width ? n.attrs.width + 'px' : ''
        }
      }
      applyState(node)

      return {
        dom: wrapper,
        update: (updatedNode) => {
          if (updatedNode.type.name !== 'image') return false
          applyState(updatedNode)
          return true
        },
        // Drag listeners are added on pointerdown and torn down on pointerup,
        // but if the nodeView is destroyed mid-drag those window listeners
        // would leak.
        destroy: () => {
          window.removeEventListener('pointermove', onPointerMove)
          window.removeEventListener('pointerup', onPointerUp)
        }
      }
    }
  }
})

export default ResizableImage
