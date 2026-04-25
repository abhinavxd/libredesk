<template>
  <div class="editor-wrapper h-full overflow-y-auto" :class="{ 'pointer-events-none': disabled }">
    <BubbleMenu
      :editor="editor"
      :tippy-options="{ duration: 100 }"
      v-if="editor"
      class="bg-background p-1 box will-change-transform"
    >
      <div class="flex space-x-1 items-center">
        <DropdownMenu v-if="aiPrompts.length > 0">
          <DropdownMenuTrigger>
            <Button size="sm" variant="ghost" class="flex items-center justify-center">
              <span class="flex items-center">
                <span class="text-medium">AI</span>
                <Bot size="14" class="ml-1" />
                <ChevronDown class="w-4 h-4 ml-2" />
              </span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem
              v-for="prompt in aiPrompts"
              :key="prompt.key"
              @select="emitPrompt(prompt.key)"
            >
              {{ prompt.title }}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button
          size="sm"
          variant="ghost"
          @click.prevent="editor?.chain().focus().toggleBold().run()"
          :class="{ 'bg-secondary': editor?.isActive('bold') }"
        >
          <Bold size="14" />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          @click.prevent="editor?.chain().focus().toggleItalic().run()"
          :class="{ 'bg-secondary': editor?.isActive('italic') }"
        >
          <Italic size="14" />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          @click.prevent="editor?.chain().focus().toggleBulletList().run()"
          :class="{ 'bg-secondary': editor?.isActive('bulletList') }"
        >
          <List size="14" />
        </Button>

        <Button
          size="sm"
          variant="ghost"
          @click.prevent="editor?.chain().focus().toggleOrderedList().run()"
          :class="{ 'bg-secondary': editor?.isActive('orderedList') }"
        >
          <ListOrdered size="14" />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          @click.prevent="openLinkModal"
          :class="{ 'bg-secondary': editor?.isActive('link') }"
        >
          <LinkIcon size="14" />
        </Button>
      </div>
    </BubbleMenu>
    <EditorContent :editor="editor" class="native-html" />

    <Dialog v-model:open="showLinkDialog">
      <DialogContent class="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>
            {{
              editor?.isActive('link')
                ? $t('editor.editLinkUrl')
                : $t('editor.addLinkUrl')
            }}
          </DialogTitle>
          <DialogDescription></DialogDescription>
        </DialogHeader>
        <form @submit.stop.prevent="setLink">
          <div class="grid gap-4 py-4">
            <Input
              v-model="linkUrl"
              type="text"
              :placeholder="$t('placeholders.enterUrl')"
              @keydown.enter.prevent="setLink"
            />
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              @click="unsetLink"
              v-if="editor?.isActive('link')"
            >
              {{ $t('actions.removeLink') }}
            </Button>
            <Button type="submit">
              {{ $t('globals.messages.save') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, watch, onUnmounted } from 'vue'
import { useEditor, EditorContent, BubbleMenu } from '@tiptap/vue-3'
import {
  ChevronDown,
  Bold,
  Italic,
  Bot,
  List,
  ListOrdered,
  Link as LinkIcon
} from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { Input } from '@shared-ui/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription
} from '@shared-ui/components/ui/dialog'
import Placeholder from '@tiptap/extension-placeholder'
import Image from '@tiptap/extension-image'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import Mention from '@tiptap/extension-mention'
import Table from '@tiptap/extension-table'
import TableRow from '@tiptap/extension-table-row'
import TableCell from '@tiptap/extension-table-cell'
import TableHeader from '@tiptap/extension-table-header'
import { useTypingIndicator } from '@shared-ui/composables'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useConversationStore } from '@main/stores/conversation'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import api from '@main/api'
import mentionSuggestion from './mentionSuggestion'

const textContent = defineModel('textContent', { default: '' })
const htmlContent = defineModel('htmlContent', { default: '' })
const showLinkDialog = ref(false)
const linkUrl = ref('')

const props = defineProps({
  placeholder: String,
  insertContent: String,
  messageType: String,
  autoFocus: {
    type: Boolean,
    default: true
  },
  aiPrompts: {
    type: Array,
    default: () => []
  },
  disabled: {
    type: Boolean,
    default: false
  },
  enableMentions: {
    type: Boolean,
    default: false
  },
  getSuggestions: {
    type: Function,
    default: null
  }
})

const emit = defineEmits(['send', 'aiPromptSelected', 'mentionsChanged', 'filesDropped'])

const emitPrompt = (key) => emit('aiPromptSelected', key)
const emitter = useEmitter()
const isUploadingImage = ref(false)

// Downscale images larger than MAX_UPLOAD_DIM before upload. Display size is
// controlled separately by the editor's image toolbar, so there's no point
// uploading multi-megapixel screenshots in full resolution.
const MAX_UPLOAD_DIM = 2000
const resizeImage = (file) => {
  return new Promise((resolve) => {
    if (!file.type.startsWith('image/') || file.type === 'image/gif') {
      resolve(file)
      return
    }
    const img = new window.Image()
    const url = URL.createObjectURL(file)
    img.onload = () => {
      URL.revokeObjectURL(url)
      if (img.width <= MAX_UPLOAD_DIM && img.height <= MAX_UPLOAD_DIM) {
        resolve(file)
        return
      }
      let w = img.width
      let h = img.height
      if (w > MAX_UPLOAD_DIM) {
        h = Math.round(h * (MAX_UPLOAD_DIM / w))
        w = MAX_UPLOAD_DIM
      }
      if (h > MAX_UPLOAD_DIM) {
        w = Math.round(w * (MAX_UPLOAD_DIM / h))
        h = MAX_UPLOAD_DIM
      }
      const canvas = document.createElement('canvas')
      canvas.width = w
      canvas.height = h
      canvas.getContext('2d').drawImage(img, 0, 0, w, h)
      canvas.toBlob(
        (blob) => resolve(blob ? new File([blob], file.name, { type: file.type }) : file),
        file.type,
        0.92
      )
    }
    img.onerror = () => {
      URL.revokeObjectURL(url)
      resolve(file)
    }
    img.src = url
  })
}

const uploadImage = async (file) => {
  file = await resizeImage(file)
  isUploadingImage.value = true
  try {
    const response = await api.uploadMedia({
      files: file,
      inline: true,
      linked_model: 'messages'
    })
    return response.data.data.url
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message || 'Failed to upload image'
    })
    return null
  } finally {
    isUploadingImage.value = false
  }
}

const insertImage = (url) => {
  if (url && editor.value) {
    editor.value.chain().focus().setImage({ src: url }).run()
  }
}

// Paste handler: catch image content from the clipboard, upload, then insert.
const handlePaste = (view, event) => {
  const items = event.clipboardData?.items
  if (!items) return false
  for (const item of items) {
    if (item.type.startsWith('image/')) {
      event.preventDefault()
      const file = item.getAsFile()
      if (file) {
        uploadImage(file).then((url) => {
          if (url) insertImage(url)
        })
      }
      return true
    }
  }
  return false
}

// Drop handler: image files go inline, everything else is emitted as
// `filesDropped` so the parent can attach them as regular attachments.
const handleDrop = (view, event) => {
  const files = event.dataTransfer?.files
  if (!files || files.length === 0) return false

  const imageFiles = []
  const otherFiles = []
  for (const file of files) {
    if (file.type.startsWith('image/')) imageFiles.push(file)
    else otherFiles.push(file)
  }
  if (imageFiles.length === 0 && otherFiles.length === 0) return false

  event.preventDefault()
  for (const file of imageFiles) {
    uploadImage(file).then((url) => {
      if (url) insertImage(url)
    })
  }
  if (otherFiles.length > 0) emit('filesDropped', otherFiles)
  return true
}

// Set up typing indicator
const conversationStore = useConversationStore()
const { startTyping, stopTyping } = useTypingIndicator(conversationStore.sendTyping, {
  get isPrivateMessage() { return props.messageType === 'private_note' }
}) 

// To preseve the table styling in emails, need to set the table style inline.
// Created these custom extensions to set the table style inline.
const CustomTable = Table.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        parseHTML: (element) =>
          (element.getAttribute('style') || '') +
          '; border: 1px solid #dee2e6 !important; width: 100%; margin:0; table-layout: fixed; border-collapse: collapse; position:relative; border-radius: 0.25rem;'
      }
    }
  }
})

const CustomTableCell = TableCell.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        parseHTML: (element) =>
          (element.getAttribute('style') || '') +
          '; border: 1px solid #dee2e6 !important; box-sizing: border-box !important; min-width: 1em !important; padding: 6px 8px !important; vertical-align: top !important;'
      }
    }
  }
})

const CustomTableHeader = TableHeader.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        parseHTML: (element) =>
          (element.getAttribute('style') || '') +
          '; background-color: #f8f9fa !important; color: #212529 !important; font-weight: bold !important; text-align: left !important; border: 1px solid #dee2e6 !important; padding: 6px 8px !important;'
      }
    }
  }
})

// Extend Mention to include 'type' attribute for agent/team distinction
const CustomMention = Mention.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      type: {
        default: null,
        parseHTML: (element) => element.getAttribute('data-type'),
        renderHTML: (attributes) => {
          if (!attributes.type) return {}
          return { 'data-type': attributes.type }
        }
      }
    }
  }
})

// Custom Image extension with drag-handle resizing and Gmail-style size presets
// (Small / Best fit / Original / Remove). Renders a node-view that wraps the
// <img> with a corner resize handle and a hover toolbar.
const ResizableImage = Image.extend({
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

      // Toolbar (visible when wrapper is selected)
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

      // Bottom-right resize handle. We don't manage selected state ourselves;
      // CSS keys off ProseMirror's `.ProseMirror-selectednode` class which
      // ProseMirror toggles automatically when the image node is selected.
      // That avoids a global document click listener per image (which leaks
      // closures across the entire page for every embedded image).
      const handle = document.createElement('div')
      handle.classList.add('image-resize-handle')
      wrapper.appendChild(handle)

      // Drag the corner handle to resize. Pointer events for touch + pen.
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

const isInternalUpdate = ref(false)

const buildExtensions = () => {
  const extensions = [
    StarterKit.configure(),
    ResizableImage.configure({
      HTMLAttributes: { class: 'inline-image', style: 'max-width: 100%; height: auto;' },
      allowBase64: false
    }),
    Placeholder.configure({ placeholder: () => props.placeholder }),
    Link,
    CustomTable.configure({ resizable: false }),
    TableRow,
    CustomTableCell,
    CustomTableHeader,
    // Always include mention extension - it gracefully handles missing getSuggestions
    CustomMention.configure({
      HTMLAttributes: {
        class: 'ld-mention'
      },
      suggestion: mentionSuggestion
    })
  ]

  return extensions
}

// Extract mentions from editor content
const extractMentions = () => {
  if (!editor.value) return []
  const mentions = []
  const json = editor.value.getJSON()

  const traverse = (node) => {
    if (node.type === 'mention' && node.attrs) {
      mentions.push({
        id: node.attrs.id,
        type: node.attrs.type
      })
    }
    if (node.content) {
      node.content.forEach(traverse)
    }
  }

  if (json.content) {
    json.content.forEach(traverse)
  }

  return mentions
}


const editor = useEditor({
  extensions: buildExtensions(),
  autofocus: props.autoFocus,
  content: htmlContent.value,
  editorProps: {
    attributes: { class: 'outline-none' },
    getSuggestions: props.getSuggestions,
    handlePaste,
    handleDrop,
    handleKeyDown: (view, event) => {
      if (event.ctrlKey && event.key.toLowerCase() === 'b') {
        event.stopPropagation()
        return false
      }
      if (event.ctrlKey && event.key === 'Enter') {
        emit('send')
        // Stop typing when sending
        stopTyping()
        return true
      }
    }
  },
  // To update state when user types.
  onUpdate: ({ editor }) => {
    isInternalUpdate.value = true
    htmlContent.value = editor.getHTML()
    textContent.value = editor.getText()
    isInternalUpdate.value = false

    // Trigger typing indicator when user types
    startTyping()

    // Emit mentions if enabled
    if (props.enableMentions) {
      emit('mentionsChanged', extractMentions())
    }
  },
  onBlur: () => {
    // Stop typing when editor loses focus
    stopTyping()
  }
})

watch(
  htmlContent,
  (newContent) => {
    if (!isInternalUpdate.value && editor.value && newContent !== editor.value.getHTML()) {
      editor.value.commands.setContent(newContent || '', false)
      textContent.value = editor.value.getText()
      editor.value.commands.focus()
    }
  },
  { immediate: true }
)

// Insert content at cursor position when insertContent prop changes.
watch(
  () => props.insertContent,
  (val) => {
    if (val) editor.value?.commands.insertContent(val)
  }
)

onUnmounted(() => {
  editor.value?.destroy()
})

const openLinkModal = () => {
  if (editor.value?.isActive('link')) {
    linkUrl.value = editor.value.getAttributes('link').href
  } else {
    linkUrl.value = ''
  }
  showLinkDialog.value = true
}

const setLink = () => {
  if (linkUrl.value) {
    editor.value?.chain().focus().extendMarkRange('link').setLink({ href: linkUrl.value }).run()
  }
  showLinkDialog.value = false
}

const unsetLink = () => {
  editor.value?.chain().focus().unsetLink().run()
  showLinkDialog.value = false
}

// Expose focus method for parent components
const focus = () => {
  editor.value?.commands.focus()
}

defineExpose({ focus, extractMentions })
</script>

<style lang="scss">
// Moving placeholder to the top.
.tiptap p.is-editor-empty:first-child::before {
  content: attr(data-placeholder);
  float: left;
  color: #adb5bd;
  pointer-events: none;
  height: 0;
  font-size: 0.875rem;
}

// Ensure the parent div has a proper height
.editor-wrapper div[aria-expanded='false'] {
  display: flex;
  flex-direction: column;
  height: 100%;
}

// Ensure the editor content has a proper height and breaks words
.tiptap.ProseMirror {
  flex: 1;
  min-height: 70px;
  overflow-y: auto;
  word-wrap: break-word !important;
  overflow-wrap: break-word !important;
  word-break: break-word;
  white-space: pre-wrap;
  max-width: 100%;
}

.tiptap {
  // Table styling
  .tableWrapper {
    margin: 1.5rem 0;
    overflow-x: auto;
  }

  // Anchor tag styling
  a {
    color: #0066cc;
    cursor: pointer;

    &:hover {
      color: #003d7a;
    }
  }

  // Mention styling
  .ld-mention {
    background-color: hsl(var(--primary) / 0.1);
    border-radius: 0.25rem;
    padding: 0 0.25rem;
    color: hsl(var(--primary));
    font-weight: 500;
  }

  // Selected image gets an outline so the user knows what's focused.
  // Hardcoded brand blue rather than a theme token so it stays visible
  // against arbitrary email content (light backgrounds, dark images, etc.).
  .ProseMirror-selectednode .inline-image {
    outline: 2px solid #0066cc;
  }

  // Wrapper added by ResizableImage's nodeView.
  .image-resizer {
    display: inline-block;
    position: relative;
    margin: 4px 0;

    .image-resize-handle {
      display: none;
      position: absolute;
      bottom: 4px;
      right: 4px;
      width: 12px;
      height: 12px;
      background: #0066cc;
      border: 2px solid white;
      border-radius: 2px;
      cursor: nwse-resize;
      z-index: 10;
      box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.15);
    }

    // Floating size toolbar — sits above image to avoid BubbleMenu overlap.
    .image-size-toolbar {
      display: none;
      position: absolute;
      top: 4px;
      left: 50%;
      transform: translateX(-50%);
      background: hsl(var(--background) / 0.95);
      border: 1px solid hsl(var(--border));
      border-radius: 6px;
      padding: 2px;
      z-index: 10000;
      white-space: nowrap;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
      backdrop-filter: blur(4px);

      button {
        padding: 2px 8px;
        font-size: 11px;
        color: hsl(var(--muted-foreground));
        background: none;
        border: none;
        border-radius: 4px;
        cursor: pointer;
        line-height: 1.6;

        &:hover {
          background: hsl(var(--accent));
          color: hsl(var(--accent-foreground));
        }
      }

      .image-toolbar-sep {
        width: 1px;
        height: 14px;
        background: hsl(var(--border));
        margin: 0 2px;
        align-self: center;
      }

      .image-toolbar-remove {
        color: hsl(var(--destructive)) !important;

        &:hover {
          background: hsl(var(--destructive) / 0.1) !important;
          color: hsl(var(--destructive)) !important;
        }
      }
    }

    // ProseMirror toggles `.ProseMirror-selectednode` on the wrapper for us
    // when the image node is selected, so we don't need to manage selected
    // state with a document-level click listener.
    &.ProseMirror-selectednode .image-resize-handle,
    &.resizing .image-resize-handle {
      display: block;
    }

    &.ProseMirror-selectednode .image-size-toolbar {
      display: flex;
    }

    &.ProseMirror-selectednode .inline-image,
    &.resizing .inline-image {
      outline: 2px solid #0066cc;
    }

    &.resizing .inline-image {
      opacity: 0.8;
    }
  }
}
</style>