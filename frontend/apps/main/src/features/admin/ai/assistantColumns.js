import { h } from 'vue'
import TableDate from '@main/components/table/TableDate.vue'
import { Badge } from '@shared-ui/components/ui/badge/index.js'
import dropdown from './assistantDropdown.vue'

export const createAssistantColumns = (t, { onEdit } = {}) => [
  {
    accessorKey: 'name',
    header: () => h('div', null, t('globals.terms.name')),
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
              row.getValue('name')
            )
          : row.getValue('name')
      )
  },
  {
    accessorKey: 'description',
    header: () => h('div', null, t('globals.terms.description')),
    cell: ({ row }) =>
      h(
        'div',
        { class: 'text-muted-foreground' },
        row.getValue('description')
      )
  },
  {
    accessorKey: 'enabled',
    meta: { compact: true },
    enableGlobalFilter: false,
    header: () => h('div', null, t('globals.terms.status')),
    cell: ({ row }) =>
      h(
        'div',
        null,
        h(Badge, { variant: row.getValue('enabled') ? 'success' : 'secondary' }, () =>
          row.getValue('enabled') ? t('globals.terms.enabled') : t('globals.terms.disabled')
        )
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
    cell: ({ row }) => h('div', { class: 'relative' }, h(dropdown, { assistant: row.original }))
  }
]
