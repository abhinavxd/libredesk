import { ref, watch, onUnmounted } from 'vue'
import { createNodeFromContent, useEditor } from '@tiptap/vue-3'
import { Slice } from '@tiptap/pm/model'
import { Selection } from '@tiptap/pm/state'
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
  enableMentions = () => false,
  getConversationSuggestions = null,
  conversationReferencesEnabled = () => false,
  onSend = () => {},
  onToggleMessageType = null,
  onUpdate = () => {},
  onBlur = () => {},
  onOtherFiles = () => {}
}) {
  const isInternalUpdate = ref(false)
  let hasBeenFocused = false

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
      enableMentions,
      getConversationSuggestions,
      conversationReferencesEnabled,
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
      htmlContent.value = editor.getHTML()
      textContent.value = editor.getText()
      isInternalUpdate.value = false
      onUpdate()
    },
    onFocus: () => {
      hasBeenFocused = true
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

  const insertHtml = (html) => {
    if (!editor.value) return
    const { state, view } = editor.value
    const tr = state.tr
    // A never-focused editor parks its selection at the start, which would prepend to a draft.
    if (!hasBeenFocused) tr.setSelection(Selection.atEnd(tr.doc))
    // replaceSelection fits the closed blocks around the cursor without splitting off empty paragraphs.
    const slice = new Slice(createNodeFromContent(html, state.schema), 0, 0)
    tr.replaceSelection(slice)
    view.dispatch(tr.scrollIntoView())
  }

  return { editor, insertImages, extractMentions, focus, insertHtml }
}
