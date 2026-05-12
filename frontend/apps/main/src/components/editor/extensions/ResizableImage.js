import Image from '@tiptap/extension-image'

// Custom Image extension with drag-handle resizing and Gmail-style size presets
// (Small / Best fit / Original / Remove). Styles for .image-resizer,
// .image-resize-handle, and .image-size-toolbar live in TextEditor.vue's
// global <style> block.
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
      }
    }
  },
  addNodeView () {
    return ({ node, getPos, editor: nodeEditor }) => {
      const wrapper = document.createElement('div')
      wrapper.classList.add('image-resizer')
      wrapper.style.display = 'inline-block'
      wrapper.style.position = 'relative'
      wrapper.style.lineHeight = '0'

      const img = document.createElement('img')
      img.src = node.attrs.src
      img.alt = node.attrs.alt || ''
      img.title = node.attrs.title || ''
      img.classList.add('inline-image')
      img.style.maxWidth = '100%'
      img.style.height = 'auto'
      if (node.attrs.width) img.style.width = node.attrs.width + 'px'
      wrapper.appendChild(img)

      const toolbar = document.createElement('div')
      toolbar.classList.add('image-size-toolbar')

      let naturalWidth = 0
      img.addEventListener('load', () => { naturalWidth = img.naturalWidth })

      const commitWidth = (newWidth) => {
        const pos = getPos()
        if (typeof pos !== 'number') return
        nodeEditor.chain().focus().command(({ tr }) => {
          tr.setNodeMarkup(pos, undefined, { ...node.attrs, width: newWidth || null })
          return true
        }).run()
      }

      const sizes = [
        { label: 'Small', value: 400 },
        { label: 'Best fit', value: 'fit' },
        { label: 'Original', value: 'original' }
      ]
      // Toolbar buttons use pointerdown so touch + pen + mouse all work.
      // preventDefault avoids stealing focus from the editor.
      sizes.forEach(({ label, value }) => {
        const btn = document.createElement('button')
        btn.textContent = label
        btn.type = 'button'
        btn.addEventListener('pointerdown', (e) => {
          e.preventDefault()
          e.stopPropagation()
          if (value === 'original') {
            img.style.width = naturalWidth ? naturalWidth + 'px' : 'auto'
            commitWidth(naturalWidth || null)
          } else if (value === 'fit') {
            img.style.width = ''
            commitWidth(null)
          } else {
            img.style.width = value + 'px'
            commitWidth(value)
          }
        })
        toolbar.appendChild(btn)
      })

      const sep = document.createElement('span')
      sep.classList.add('image-toolbar-sep')
      toolbar.appendChild(sep)

      const removeBtn = document.createElement('button')
      removeBtn.textContent = 'Remove'
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

      // CSS keys off ProseMirror's `.ProseMirror-selectednode` class which
      // ProseMirror toggles automatically when the image node is selected.
      // Avoids a global document click listener per image (which would leak
      // closures across the entire page for every embedded image).
      const handle = document.createElement('div')
      handle.classList.add('image-resize-handle')
      wrapper.appendChild(handle)

      let startX = 0
      let startWidth = 0
      const onPointerMove = (e) => {
        const newWidth = Math.max(50, startWidth + (e.clientX - startX))
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
      const onPointerDown = (e) => {
        e.preventDefault()
        e.stopPropagation()
        startX = e.clientX
        startWidth = img.offsetWidth
        window.addEventListener('pointermove', onPointerMove)
        window.addEventListener('pointerup', onPointerUp)
        wrapper.classList.add('resizing')
      }
      handle.addEventListener('pointerdown', onPointerDown)

      return {
        dom: wrapper,
        update: (updatedNode) => {
          if (updatedNode.type.name !== 'image') return false
          img.src = updatedNode.attrs.src
          img.style.width = updatedNode.attrs.width ? updatedNode.attrs.width + 'px' : ''
          return true
        },
        destroy: () => {
          handle.removeEventListener('pointerdown', onPointerDown)
          window.removeEventListener('pointermove', onPointerMove)
          window.removeEventListener('pointerup', onPointerUp)
        }
      }
    }
  }
})

export default ResizableImage
