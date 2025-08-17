<template>
  <div class="editor-wrapper h-full overflow-y-auto">
    <BubbleMenu
      :editor="editor"
      :tippy-options="{ duration: 100 }"
      v-if="editor"
      class="bg-background p-2 box will-change-transform max-w-fit"
    >
      <div class="flex gap-1 items-center justify-start whitespace-nowrap">
        <DropdownMenu v-if="aiPrompts.length > 0">
          <DropdownMenuTrigger>
            <Button size="sm" variant="ghost" class="flex items-center justify-center" title="AI Prompts">
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

        <!-- Heading Dropdown for Article Mode -->
        <DropdownMenu v-if="editorType === 'article'">
          <DropdownMenuTrigger>
            <Button size="sm" variant="ghost" class="flex items-center justify-center" title="Heading Options">
              <span class="flex items-center">
                <Type size="14" />
                <span class="ml-1 text-xs font-medium">{{ getCurrentHeadingText() }}</span>
                <ChevronDown class="w-3 h-3 ml-1" />
              </span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem @select="setParagraph" title="Set Paragraph">
              <span class="font-normal">Paragraph</span>
            </DropdownMenuItem>
            <DropdownMenuItem @select="() => setHeading(1)" title="Set Heading 1">
              <span class="text-xl font-bold">Heading 1</span>
            </DropdownMenuItem>
            <DropdownMenuItem @select="() => setHeading(2)" title="Set Heading 2">
              <span class="text-lg font-bold">Heading 2</span>
            </DropdownMenuItem>
            <DropdownMenuItem @select="() => setHeading(3)" title="Set Heading 3">
              <span class="text-base font-semibold">Heading 3</span>
            </DropdownMenuItem>
            <DropdownMenuItem @select="() => setHeading(4)" title="Set Heading 4">
              <span class="text-sm font-semibold">Heading 4</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <Button
          size="sm"
          variant="ghost"
          @click.prevent="editor?.chain().focus().toggleBold().run()"
          :class="{ 'bg-gray-200 dark:bg-secondary': editor?.isActive('bold') }"
          title="Bold"
        >
          <Bold size="14" />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          @click.prevent="editor?.chain().focus().toggleItalic().run()"
          :class="{ 'bg-gray-200 dark:bg-secondary': editor?.isActive('italic') }"
          title="Italic"
        >
          <Italic size="14" />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          @click.prevent="editor?.chain().focus().toggleBulletList().run()"
          :class="{ 'bg-gray-200 dark:bg-secondary': editor?.isActive('bulletList') }"
          title="Bullet List"
        >
          <List size="14" />
        </Button>

        <Button
          size="sm"
          variant="ghost"
          @click.prevent="editor?.chain().focus().toggleOrderedList().run()"
          :class="{ 'bg-gray-200 dark:bg-secondary': editor?.isActive('orderedList') }"
          title="Ordered List"
        >
          <ListOrdered size="14" />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          @click.prevent="openLinkModal"
          :class="{ 'bg-gray-200 dark:bg-secondary': editor?.isActive('link') }"
          title="Insert Link"
        >
          <LinkIcon size="14" />
        </Button>

        <!-- Additional tools for Article Mode -->
        <template v-if="editorType === 'article'">
          <Button
            size="sm"
            variant="ghost"
            @click.prevent="editor?.chain().focus().toggleCodeBlock().run()"
            :class="{ 'bg-gray-200 dark:bg-secondary': editor?.isActive('codeBlock') }"
            title="Code Block"
          >
            <Code size="14" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            @click.prevent="editor?.chain().focus().toggleBlockquote().run()"
            :class="{ 'bg-gray-200 dark:bg-secondary': editor?.isActive('blockquote') }"
            title="Blockquote"
          >
            <Quote size="14" />
          </Button>
        </template>
        <div v-if="showLinkInput" class="flex space-x-2 p-2 bg-background border rounded">
          <Input
            v-model="linkUrl"
            type="text"
            placeholder="Enter link URL"
            class="border p-1 text-sm w-[200px]"
          />
          <Button size="sm" @click="setLink" title="Set Link">
            <Check size="14" />
          </Button>
          <Button size="sm" @click="unsetLink" title="Unset Link">
            <X size="14" />
          </Button>
        </div>
      </div>
    </BubbleMenu>
    <EditorContent :editor="editor" class="native-html" />
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
  Check,
  X,
  Type,
  Code,
  Quote
} from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { Input } from '@shared-ui/components/ui/input'
import Placeholder from '@tiptap/extension-placeholder'
import Image from '@tiptap/extension-image'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import Table from '@tiptap/extension-table'
import TableRow from '@tiptap/extension-table-row'
import TableCell from '@tiptap/extension-table-cell'
import TableHeader from '@tiptap/extension-table-header'
import { useTypingIndicator } from '@shared-ui/composables'
import { useConversationStore } from '@main/stores/conversation'

const textContent = defineModel('textContent', { default: '' })
const htmlContent = defineModel('htmlContent', { default: '' })
const showLinkInput = ref(false)
const linkUrl = ref('')

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
  },
  editorType: {
    type: String,
    default: 'conversation',
    validator: (value) => ['conversation', 'article'].includes(value)
  }
})

const emit = defineEmits(['send', 'aiPromptSelected'])

const emitPrompt = (key) => emit('aiPromptSelected', key)

// Set up typing indicator
const conversationStore = useConversationStore()
const { startTyping, stopTyping } = useTypingIndicator(conversationStore.sendTyping)

// To preseve the table styling in emails, need to set the table style inline.
// Created these custom extensions to set the table style inline.
const CustomTable = Table.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        parseHTML: (element) =>
          (element.getAttribute('style') || '') + '; border: 1px solid #dee2e6 !important; width: 100%; margin:0; table-layout: fixed; border-collapse: collapse; position:relative; border-radius: 0.25rem;'
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

const isInternalUpdate = ref(false)

// Configure extensions based on editor type
const getExtensions = () => {
  const baseExtensions = [
    StarterKit.configure({
      heading: props.editorType === 'article' ? { levels: [1, 2, 3, 4] } : false
    }),
    Image.configure({ HTMLAttributes: { class: 'inline-image' } }),
    Placeholder.configure({ placeholder: () => props.placeholder }),
    Link
  ]

  // Add table extensions
  if (props.editorType === 'article') {
    baseExtensions.push(
      CustomTable.configure({ resizable: true }),
      TableRow,
      CustomTableCell,
      CustomTableHeader
    )
  } else {
    baseExtensions.push(
      CustomTable.configure({ resizable: false }),
      TableRow,
      CustomTableCell,
      CustomTableHeader
    )
  }

  return baseExtensions
}

const editor = useEditor({
  extensions: getExtensions(),
  autofocus: props.autoFocus,
  content: htmlContent.value,
  editorProps: {
    attributes: { class: 'outline-none' },
    handleKeyDown: (view, event) => {
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
  showLinkInput.value = true
}

const setLink = () => {
  if (linkUrl.value) {
    editor.value?.chain().focus().extendMarkRange('link').setLink({ href: linkUrl.value }).run()
  }
  showLinkInput.value = false
}

const unsetLink = () => {
  editor.value?.chain().focus().unsetLink().run()
  showLinkInput.value = false
}

// Heading functions for article mode
const setHeading = (level) => {
  editor.value?.chain().focus().toggleHeading({ level }).run()
}

const setParagraph = () => {
  editor.value?.chain().focus().setParagraph().run()
}

const getCurrentHeadingLevel = () => {
  if (!editor.value) return null
  for (let level = 1; level <= 4; level++) {
    if (editor.value.isActive('heading', { level })) {
      return level
    }
  }
  return null
}

const getCurrentHeadingText = () => {
  const level = getCurrentHeadingLevel()
  if (level) return `H${level}`
  if (editor.value?.isActive('paragraph')) return 'P'
  return 'T'
}
</script>

<style lang="scss">
// Moving placeholder to the top.
.tiptap p.is-editor-empty:first-child::before {
  content: attr(data-placeholder);
  float: left;
  color: #adb5bd;
  pointer-events: none;
  height: 0;
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
}
</style>
