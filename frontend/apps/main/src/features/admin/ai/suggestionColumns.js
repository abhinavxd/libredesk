import { h } from 'vue'
import { format } from 'date-fns'

const conversationHref = (uuid) => `/inboxes/all/conversation/${uuid}`

export const createSuggestionColumns = (t, { onReview } = {}) => [
  {
    accessorKey: 'question',
    header: () => t('globals.terms.question'),
    cell: ({ row }) =>
      h(
        'span',
        {
          class: 'text-primary hover:underline cursor-pointer',
          onClick: () => onReview && onReview(row.original)
        },
        row.getValue('question')
      )
  },
  {
    accessorKey: 'answer',
    enableSorting: false,
    header: () => t('globals.terms.answer'),
    cell: ({ row }) =>
      h('div', { class: 'text-muted-foreground line-clamp-2 max-w-md' }, row.getValue('answer'))
  },
  {
    id: 'source',
    enableGlobalFilter: false,
    enableSorting: false,
    header: () => t('admin.ai.suggestion.source'),
    cell: ({ row }) => {
      const uuid = row.original.conversation_uuid
      const ref = row.original.conversation_reference_number
      if (!uuid) return h('span', { class: 'text-muted-foreground' }, '-')
      return h(
        'a',
        {
          href: conversationHref(uuid),
          target: '_blank',
          class: 'text-primary hover:underline'
        },
        ref ? `#${ref}` : t('admin.ai.suggestion.viewSource')
      )
    }
  },
  {
    accessorKey: 'created_at',
    enableGlobalFilter: false,
    header: () => t('globals.terms.createdAt'),
    cell: ({ row }) => format(row.getValue('created_at'), 'PPp')
  }
]
