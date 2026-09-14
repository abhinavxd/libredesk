import MentionList from './MentionList.vue'
import { createSuggestionRenderer } from './suggestionRenderer'

export default {
  char: '@',
  allowSpaces: true,
  allow: ({ editor }) => editor.options.editorProps?.enableMentions?.() ?? false,
  items: ({ query, editor }) => editor.options.editorProps?.getSuggestions?.(query) ?? [],
  render: () => createSuggestionRenderer(MentionList)
}
