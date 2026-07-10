import { h } from 'vue'
import { format } from 'date-fns'
import { Badge } from '@shared-ui/components/ui/badge/index.js'
import dropdown from './toolDropdown.vue'

export const createToolColumns = (t, { onEdit } = {}) => [
  {
    accessorKey: 'name',
    header: () => h('div', { class: 'text-center' }, t('globals.terms.name')),
    cell: ({ row }) =>
      h(
        'div',
        { class: 'text-center' },
        onEdit
          ? h(
              'span',
              {
                class: 'text-primary hover:underline cursor-pointer',
                onClick: () => onEdit(row.original)
              },
              row.getValue('name')
            )
          : row.getValue('name')
      )
  },
  {
    accessorKey: 'url',
    header: () => h('div', { class: 'text-center' }, t('globals.terms.url')),
    cell: ({ row }) =>
      h('div', { class: 'text-center break-all text-muted-foreground' }, row.getValue('url'))
  },
  {
    accessorKey: 'enabled',
    enableGlobalFilter: false,
    header: () => h('div', { class: 'text-center' }, t('globals.terms.enabled')),
    cell: ({ row }) =>
      h(
        'div',
        { class: 'text-center' },
        h(Badge, { variant: row.getValue('enabled') ? 'default' : 'secondary' }, () =>
          row.getValue('enabled') ? t('globals.terms.enabled') : t('globals.terms.disabled')
        )
      )
  },
  {
    accessorKey: 'updated_at',
    enableGlobalFilter: false,
    header: () => h('div', { class: 'text-center' }, t('globals.terms.updatedAt')),
    cell: ({ row }) =>
      h('div', { class: 'text-center' }, format(row.getValue('updated_at'), 'PPpp'))
  },
  {
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => h('div', { class: 'relative' }, h(dropdown, { tool: row.original }))
  }
]
