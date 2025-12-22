<template>
  <div class="editor-wrapper h-full overflow-y-auto">
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
          :class="{ 'bg-gray-200 dark:bg-secondary': editor?.isActive('bold') }"
        >
          <Bold size="14" />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          @click.prevent="editor?.chain().focus().toggleItalic().run()"
          :class="{ 'bg-gray-200 dark:bg-secondary': editor?.isActive('italic') }"
        >
          <Italic size="14" />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          @click.prevent="editor?.chain().focus().toggleBulletList().run()"
          :class="{ 'bg-gray-200 dark:bg-secondary': editor?.isActive('bulletList') }"
        >
          <List size="14" />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          @click.prevent="editor?.chain().focus().toggleOrderedList().run()"
          :class="{ 'bg-gray-200 dark:bg-secondary': editor?.isActive('orderedList') }"
        >
          <ListOrdered size="14" />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          @click.prevent="openLinkModal"
          :class="{ 'bg-gray-200 dark:bg-secondary': editor?.isActive('link') }"
        >
          <LinkIcon size="14" />
        </Button>
        <!-- Image upload button -->
        <Button
          size="sm"
          variant="ghost"
          @click.prevent="triggerImageUpload"
          :disabled="isUploadingImage"
        >
          <ImageIcon size="14" />
        </Button>
      </div>
    </BubbleMenu>
    <EditorContent :editor="editor" class="native-html" />

    <!-- Hidden file input for image upload -->
    <input
      ref="imageInput"
      type="file"
      accept="image/*"
      class="hidden"
      @change="handleImageSelect"
    />

    <!-- Upload indicator -->
    <div v-if="isUploadingImage" class="text-xs text-muted-foreground mt-1 flex items-center gap-1">
      <Loader2 size="12" class="animate-spin" />
      Uploading image...
    </div>

    <Dialog v-model:open="showLinkDialog">
      <DialogContent class="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>
            {{ editor?.isActive('link')
              ? $t('globals.messages.edit', { name: $t('globals.terms.link', 1).toLowerCase() + ' ' + $t('globals.terms.url', 1).toLowerCase() })
              : $t('globals.messages.add', { name: $t('globals.terms.link', 1).toLowerCase() + ' ' + $t('globals.terms.url', 1).toLowerCase() })
            }}
          </DialogTitle>
          <DialogDescription></DialogDescription>
        </DialogHeader>
        <form @submit.stop.prevent="setLink">
          <div class="grid gap-4 py-4">
            <Input
              v-model="linkUrl"
              type="text"
              :placeholder="$t('globals.messages.enter', { name: $t('globals.terms.url', 1) })"
              @keydown.enter.prevent="setLink"
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" @click="unsetLink" v-if="editor?.isActive('link')">
              {{ $t('globals.messages.remove', { name: $t('globals.terms.link', 1) }) }}
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
  Link as LinkIcon,
  Image as ImageIcon,
  Loader2,
} from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription
} from '@/components/ui/dialog'
import Placeholder from '@tiptap/extension-placeholder'
import Image from '@tiptap/extension-image'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import Table from '@tiptap/extension-table'
import TableRow from '@tiptap/extension-table-row'
import TableCell from '@tiptap/extension-table-cell'
import TableHeader from '@tiptap/extension-table-header'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'

const textContent = defineModel('textContent', { default: '' })
const htmlContent = defineModel('htmlContent', { default: '' })
const showLinkDialog = ref(false)
const linkUrl = ref('')
const imageInput = ref(null)
const isUploadingImage = ref(false)
const emitter = useEmitter()

const props = defineProps({
  placeholder: String,
  insertContent: String,
  autoFocus: {
    type: Boolean,
    default: true
  },
  aiPrompts: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['send', 'aiPromptSelected'])

const emitPrompt = (key) => emit('aiPromptSelected', key)

/**
 * Upload an image file to the server and return the URL
 */
const uploadImage = async (file) => {
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

/**
 * Insert an image into the editor at the current cursor position
 */
const insertImage = (url) => {
  if (url && editor.value) {
    editor.value.chain().focus().setImage({ src: url }).run()
  }
}

/**
 * Handle paste events to capture images from clipboard
 */
const handlePaste = async (view, event) => {
  const items = event.clipboardData?.items
  if (!items) return false

  for (const item of items) {
    if (item.type.startsWith('image/')) {
      event.preventDefault()
      const file = item.getAsFile()
      if (file) {
        const url = await uploadImage(file)
        if (url) {
          insertImage(url)
        }
      }
      return true
    }
  }
  return false
}

/**
 * Handle drop events for drag & drop images
 */
const handleDrop = async (view, event) => {
  const files = event.dataTransfer?.files
  if (!files || files.length === 0) return false

  for (const file of files) {
    if (file.type.startsWith('image/')) {
      event.preventDefault()
      const url = await uploadImage(file)
      if (url) {
        insertImage(url)
      }
      return true
    }
  }
  return false
}

/**
 * Trigger the hidden file input for image selection
 */
const triggerImageUpload = () => {
  imageInput.value?.click()
}

/**
 * Handle image selection from file input
 */
const handleImageSelect = async (event) => {
  const file = event.target.files?.[0]
  if (file && file.type.startsWith('image/')) {
    const url = await uploadImage(file)
    if (url) {
      insertImage(url)
    }
  }
  // Reset the input so the same file can be selected again
  event.target.value = ''
}

// Custom table extensions with inline styles for email compatibility
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

// Custom Image extension with resizing support
const ResizableImage = Image.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      width: {
        default: null,
        parseHTML: element => element.getAttribute('width'),
        renderHTML: attributes => {
          if (!attributes.width) return {}
          return { width: attributes.width }
        }
      },
      height: {
        default: null,
        parseHTML: element => element.getAttribute('height'),
        renderHTML: attributes => {
          if (!attributes.height) return {}
          return { height: attributes.height }
        }
      }
    }
  }
})

const isInternalUpdate = ref(false)

const editor = useEditor({
  extensions: [
    StarterKit.configure(),
    ResizableImage.configure({
      HTMLAttributes: { class: 'inline-image', style: 'max-width: 100%; height: auto;' },
      allowBase64: false,
    }),
    Placeholder.configure({ placeholder: () => props.placeholder }),
    Link,
    CustomTable.configure({ resizable: false }),
    TableRow,
    CustomTableCell,
    CustomTableHeader
  ],
  autofocus: props.autoFocus,
  content: htmlContent.value,
  editorProps: {
    attributes: { class: 'outline-none' },
    handlePaste,
    handleDrop,
    handleKeyDown: (view, event) => {
      if (event.ctrlKey && event.key.toLowerCase() === 'b') {
        event.stopPropagation()
        return false
      }
      if (event.ctrlKey && event.key === 'Enter') {
        emit('send')
        return true
      }
    }
  },
  onUpdate: ({ editor }) => {
    isInternalUpdate.value = true
    htmlContent.value = editor.getHTML()
    textContent.value = editor.getText()
    isInternalUpdate.value = false
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
</script>

<style lang="scss">
.tiptap p.is-editor-empty:first-child::before {
  content: attr(data-placeholder);
  float: left;
  color: #adb5bd;
  pointer-events: none;
  height: 0;
}

.editor-wrapper div[aria-expanded='false'] {
  display: flex;
  flex-direction: column;
  height: 100%;
}

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
  .tableWrapper {
    margin: 1.5rem 0;
    overflow-x: auto;
  }

  a {
    color: #0066cc;
    cursor: pointer;

    &:hover {
      color: #003d7a;
    }
  }

  // Inline image styling
  .inline-image {
    max-width: 100%;
    height: auto;
    border-radius: 4px;
    margin: 8px 0;
    cursor: pointer;

    &:hover {
      outline: 2px solid #0066cc;
    }
  }

  // Image selected state
  .ProseMirror-selectednode .inline-image {
    outline: 2px solid #0066cc;
  }
}
</style>
