import { h } from 'vue'
import TableDate from '@main/components/table/TableDate.vue'
import dropdown from './editorPromptDropdown.vue'

export const createEditorPromptColumns = (t, { onEdit } = {}) => [
  {
    accessorKey: 'title',
    header: () => h('div', null, t('globals.terms.title')),
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
    header: () => h('div', null, t('globals.terms.updatedAt')),
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
