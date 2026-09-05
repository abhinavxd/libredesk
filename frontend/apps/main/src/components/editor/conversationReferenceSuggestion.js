import ConversationReferenceList from './ConversationReferenceList.vue'
import { createSuggestionRenderer } from './suggestionRenderer'

export default {
  char: '#',
  allowSpaces: false,
  allow: ({ editor }) => editor.options.editorProps?.conversationReferencesEnabled?.() ?? false,
  items: ({ query, editor }) => editor.options.editorProps?.getConversationSuggestions?.(query) ?? [],
  render: () => createSuggestionRenderer(ConversationReferenceList)
}
