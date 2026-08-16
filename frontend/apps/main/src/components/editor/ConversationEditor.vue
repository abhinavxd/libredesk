<template>
  <div
    class="editor-wrapper flex flex-col h-full overflow-y-auto"
    :class="{ 'pointer-events-none': disabled }"
  >
    <BubbleMenu
      v-if="editor"
      :editor="editor"
      :tippy-options="{ duration: 100, maxWidth: 'none' }"
      :should-show="shouldShowBubble"
      class="bg-background p-1 box will-change-transform"
    >
      <EditorToolbar
        :editor="editor"
        :ai-prompts="aiPrompts"
        @open-link="linkDialog?.open()"
        @ai-prompt="emitPrompt"
      />
    </BubbleMenu>
    <EditorContent :editor="editor" class="native-html" />

    <EditorLinkDialog ref="linkDialog" :editor="editor" :allow-button="false" />
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { EditorContent, BubbleMenu } from '@tiptap/vue-3'
import { useTypingIndicator } from '@shared-ui/composables'
import { useConversationStore } from '@main/stores/conversation'
import EditorToolbar from './EditorToolbar.vue'
import EditorLinkDialog from './EditorLinkDialog.vue'
import { buildConversationExtensions } from './editorExtensions'
import { useTextEditor } from './useTextEditor'

const textContent = defineModel('textContent', { default: '' })
const htmlContent = defineModel('htmlContent', { default: '' })

const props = defineProps({
  placeholder: String,
  insertContent: String,
  messageType: String,
  autoFocus: { type: Boolean, default: true },
  aiPrompts: { type: Array, default: () => [] },
  disabled: { type: Boolean, default: false },
  enableMentions: { type: Boolean, default: false },
  getSuggestions: { type: Function, default: null },
  enableInlineImages: { type: Boolean, default: false },
  linkedModel: { type: String, default: 'messages' }
})

const emit = defineEmits(['send', 'aiPromptSelected', 'mentionsChanged', 'filesDropped'])

const linkDialog = ref(null)

const emitPrompt = (key) => emit('aiPromptSelected', key)

// Suppress the formatting bubble when an image node is selected so it
// doesn't fight with the image's own size/remove toolbar.
const shouldShowBubble = ({ editor: e, state }) => {
  const { selection } = state
  if (selection.node?.type?.name === 'image') return false
  if (!e.view.hasFocus()) return false
  // Keep the menu open while editing a table so its row/column controls stay reachable with a collapsed cursor.
  if (e.isActive('table')) return true
  if (selection.empty) return false
  return true
}

const conversationStore = useConversationStore()
const { startTyping, stopTyping } = useTypingIndicator(conversationStore.sendTyping, {
  get isPrivateMessage() {
    return props.messageType === 'private_note'
  }
})

const { editor, extractMentions, focus } = useTextEditor({
  extensions: buildConversationExtensions({ getPlaceholder: () => props.placeholder }),
  htmlContent,
  textContent,
  autoFocus: props.autoFocus,
  editable: !props.disabled,
  insertContent: () => props.insertContent,
  isInlineEnabled: () => props.enableInlineImages,
  linkedModel: props.linkedModel,
  getSuggestions: props.getSuggestions,
  onSend: () => {
    emit('send')
    stopTyping()
  },
  onUpdate: () => {
    startTyping()
    if (props.enableMentions) emit('mentionsChanged', extractMentions())
  },
  onBlur: stopTyping,
  onOtherFiles: (files) => emit('filesDropped', files)
})

// Pointer-events alone still lets an already-focused editor take keystrokes.
watch(
  () => props.disabled,
  (disabled) => editor.value?.setEditable(!disabled, false)
)

defineExpose({ focus, extractMentions })
</script>

<style lang="scss" src="./editorStyles.scss"></style>
