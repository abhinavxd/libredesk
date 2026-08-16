import { ref, watch, onUnmounted } from 'vue'
import { useEditor } from '@tiptap/vue-3'
import { useInlineImageUpload } from '@main/composables/useInlineImageUpload'

export function useTextEditor({
  extensions,
  htmlContent,
  textContent,
  autoFocus = true,
  editable = true,
  insertContent = () => '',
  isInlineEnabled = () => false,
  linkedModel = 'messages',
  getSuggestions = null,
  onSend = () => {},
  onUpdate = () => {},
  onBlur = () => {},
  onOtherFiles = () => {}
}) {
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
    content: htmlContent.value,
    editorProps: {
      attributes: { class: 'outline-none' },
      getSuggestions,
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
      }
    },
    onUpdate: ({ editor }) => {
      isInternalUpdate.value = true
      htmlContent.value = editor.getHTML()
      textContent.value = editor.getText()
      isInternalUpdate.value = false
      onUpdate()
    },
    onBlur
  })

  watch(
    htmlContent,
    (newContent) => {
      if (!isInternalUpdate.value && editor.value && newContent !== editor.value.getHTML()) {
        editor.value.commands.setContent(newContent || '', false)
        textContent.value = editor.value.getText()
      }
    },
    { immediate: true }
  )

  watch(insertContent, (val) => {
    if (val) editor.value?.commands.insertContent(val)
  })

  onUnmounted(() => {
    editor.value?.destroy()
  })

  const focus = (position) => {
    editor.value?.commands.focus(position)
  }

  return { editor, insertImages, extractMentions, focus }
}
