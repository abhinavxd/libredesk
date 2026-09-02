import TicketReferenceList from './TicketReferenceList.vue'
import { createSuggestionRenderer } from './suggestionRenderer'

export default {
  char: '#',
  allowSpaces: false,
  allow: ({ editor }) => editor.options.editorProps?.ticketReferencesEnabled?.() ?? false,
  items: ({ query, editor }) => editor.options.editorProps?.getTicketSuggestions?.(query) ?? [],
  render: () => createSuggestionRenderer(TicketReferenceList)
}
