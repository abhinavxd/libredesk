import { ref, watch, onUnmounted } from 'vue'
import { prepareEditorContent, serializeEditorContent } from './prepareEditorContent'
import { useEditor } from '@tiptap/vue-3'
import { useInlineImageUpload } from '@main/composables/useInlineImageUpload'

export function useTextEditor({
  extensions,
  restrictResources = false,
  htmlContent,
  textContent,
  autoFocus = true,
  editable = true,
  insertContent = () => '',
  isInlineEnabled = () => false,
  linkedModel = 'messages',
  getSuggestions = null,
  enableMentions = () => false,
  getConversationSuggestions = null,
  conversationReferencesEnabled = () => false,
  onSend = () => {},
  onToggleMessageType = null,
  onUpdate = () => {},
  onBlur = () => {},
  onOtherFiles = () => {}
}) {
  const prepare = (content) => restrictResources ? prepareEditorContent(content) : content
  const serialize = (content) => restrictResources ? serializeEditorContent(content) : content
  const isInternalUpdate = ref(false)

  const { handlePaste, handleDrop, insertImages } = useInlineImageUpload({
    getEditor: () => editor.value,
    isInlineEnabled,
    linkedModel,
    onOtherFiles
  })

  const extractMentions = () => {
    if (!editor.value) return []
    const mentions = []
    const json = editor.value.getJSON()

    const traverse = (node) => {
      if (node.type === 'mention' && node.attrs) {
        mentions.push({ id: node.attrs.id, type: node.attrs.type })
      }
      if (node.content) node.content.forEach(traverse)
    }

    if (json.content) json.content.forEach(traverse)
    return mentions
  }

  const editor = useEditor({
    extensions,
    autofocus: autoFocus,
    editable,
    content: prepare(htmlContent.value),
    editorProps: {
      attributes: { class: 'outline-none' },
      getSuggestions,
      enableMentions,
      getConversationSuggestions,
      conversationReferencesEnabled,
      transformPastedHTML: prepare,
      handlePaste,
      handleDrop,
      handleKeyDown: (view, event) => {
        if (event.ctrlKey && event.key.toLowerCase() === 'b') {
          event.stopPropagation()
          return false
        }
        if (event.ctrlKey && event.key === 'Enter') {
          onSend()
          return true
        }
        if (
          onToggleMessageType &&
          (event.ctrlKey || event.metaKey) &&
          !event.shiftKey &&
          !event.altKey &&
          event.key.toLowerCase() === 'p'
        ) {
          event.preventDefault()
          onToggleMessageType()
          return true
        }
      }
    },
    onUpdate: ({ editor }) => {
      isInternalUpdate.value = true
      htmlContent.value = serialize(editor.getHTML())
      textContent.value = editor.getText()
      isInternalUpdate.value = false
      onUpdate()
    },
    onBlur
  })

  watch(
    htmlContent,
    (newContent) => {
      if (!isInternalUpdate.value && editor.value && newContent !== serialize(editor.value.getHTML())) {
        editor.value.commands.setContent(prepare(newContent || ''), false)
        textContent.value = editor.value.getText()
      }
    },
    { immediate: true }
  )

  watch(insertContent, (val) => {
    if (val) editor.value?.commands.insertContent(prepare(val))
  })

  onUnmounted(() => {
    editor.value?.destroy()
  })

  const focus = (position) => {
    editor.value?.commands.focus(position)
  }

  return { editor, insertImages, extractMentions, focus }
}
