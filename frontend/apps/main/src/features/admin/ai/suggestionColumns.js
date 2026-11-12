const viewSourceMessage = msg({ message: "View conversation", context: "AI suggestion action" })
import { defineMessage as msg } from '@lingui/core/macro'


import { h } from 'vue'
import TableDate from '@main/components/table/TableDate.vue'

const conversationHref = (uuid) => `/inboxes/all/conversation/${uuid}`

export const createSuggestionColumns = (i18n, { onReview } = {}) => [
  {
    accessorKey: 'question',
    header: () => i18n._('Question'),
    cell: ({ row }) =>
      h(
        'span',
        {
          class: 'text-foreground font-medium hover:underline cursor-pointer',
          onClick: () => onReview && onReview(row.original)
        },
        row.getValue('question')
      )
  },
  {
    accessorKey: 'answer',
    enableSorting: false,
    header: () => i18n._('Answer'),
    cell: ({ row }) =>
      h('div', { class: 'text-muted-foreground line-clamp-2 max-w-md' }, row.getValue('answer'))
  },
  {
    id: 'source',
    enableGlobalFilter: false,
    enableSorting: false,
    header: () => i18n._('Source'),
    cell: ({ row }) => {
      const uuid = row.original.conversation_uuid
      const ref = row.original.conversation_reference_number
      if (!uuid) return h('span', { class: 'text-muted-foreground' }, '-')
      return h(
        'a',
        {
          href: conversationHref(uuid),
          target: '_blank',
          class: 'text-foreground font-medium hover:underline'
        },
        ref ? `#${ref}` : i18n._(viewSourceMessage)
      )
    }
  },
  {
    accessorKey: 'created_at',
    meta: { compact: true },
    enableGlobalFilter: false,
    header: () => i18n._('Created at'),
    cell: ({ row }) => h(TableDate, { value: row.getValue('created_at') })
  }
]
