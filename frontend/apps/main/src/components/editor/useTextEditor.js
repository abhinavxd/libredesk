import { ref, watch, onUnmounted } from 'vue'
import { useEditor } from '@tiptap/vue-3'
import { useTypingIndicator } from '@shared-ui/composables'
import { useConversationStore } from '@main/stores/conversation'
import { useInlineImageUpload } from '@main/composables/useInlineImageUpload'

export function useTextEditor({ props, extensions, htmlContent, textContent, emit }) {
  const isInternalUpdate = ref(false)

  const { handlePaste, handleDrop, insertImages } = useInlineImageUpload({
    getEditor: () => editor.value,
    isInlineEnabled: () => props.enableInlineImages,
    linkedModel: props.linkedModel,
    onOtherFiles: (files) => emit('filesDropped', files)
  })

  const conversationStore = useConversationStore()
  const { startTyping, stopTyping } = useTypingIndicator(conversationStore.sendTyping, {
    get isPrivateMessage() {
      return props.messageType === 'private_note'
    }
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
          stopTyping()
          return true
        }
      }
    },
    onUpdate: ({ editor }) => {
      isInternalUpdate.value = true
      htmlContent.value = editor.getHTML()
      textContent.value = editor.getText()
      isInternalUpdate.value = false

      startTyping()

      if (props.enableMentions) {
        emit('mentionsChanged', extractMentions())
      }
    },
    onBlur: () => {
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

  watch(
    () => props.insertContent,
    (val) => {
      if (val) editor.value?.commands.insertContent(val)
    }
  )

  onUnmounted(() => {
    editor.value?.destroy()
  })

  const focus = () => {
    editor.value?.commands.focus()
  }

  return { editor, handlePaste, handleDrop, insertImages, extractMentions, focus }
}
