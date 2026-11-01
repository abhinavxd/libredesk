

import { h } from 'vue'
import TableDate from '@main/components/table/TableDate.vue'
import dropdown from './editorPromptDropdown.vue'

export const createEditorPromptColumns = (i18n, { onEdit } = {}) => [
  {
    accessorKey: 'title',
    header: () => h('div', null, i18n._('{count, plural, one {Title} other {Titles}}')),
    cell: ({ row }) =>
      h(
        'div',
        null,
        onEdit
          ? h(
              'span',
              {
                class: 'text-foreground font-medium hover:underline cursor-pointer',
                onClick: () => onEdit(row.original)
              },
              row.getValue('title')
            )
          : row.getValue('title')
      )
  },
  {
    accessorKey: 'updated_at',
    meta: { compact: true },
    enableGlobalFilter: false,
    header: () => h('div', null, i18n._('Updated at')),
    cell: ({ row }) =>
      h(TableDate, { value: row.getValue('updated_at') })
  },
  {
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => h('div', { class: 'relative' }, h(dropdown, { prompt: row.original }))
  }
]
