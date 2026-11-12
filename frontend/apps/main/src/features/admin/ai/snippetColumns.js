

import { h } from 'vue'
import TableDate from '@main/components/table/TableDate.vue'
import { Badge } from '@shared-ui/components/ui/badge/index.js'
import dropdown from './snippetDropdown.vue'

export const createSnippetColumns = (i18n, { onEdit } = {}) => [
  {
    accessorKey: 'title',
    header: () => h('div', null, i18n._('{count, plural, one {Title} other {Titles}}')),
    cell: ({ row }) => {
      const title = onEdit
        ? h(
            'span',
            {
              class: 'text-foreground font-medium hover:underline cursor-pointer',
              onClick: () => onEdit(row.original)
            },
            row.getValue('title')
          )
        : row.getValue('title')
      const children = [title]
      if (row.original.source === 'conversation') {
        children.push(
          h(Badge, { variant: 'secondary', class: 'ml-2 align-middle' }, () =>
            i18n._('Auto-created')
          )
        )
      }
      if (row.original.source === 'url') {
        children.push(
          h(
            Badge,
            { variant: 'secondary', class: 'ml-2 align-middle', title: row.original.source_url },
            () => i18n._('Imported')
          )
        )
      }
      return h('div', null, children)
    }
  },
  {
    accessorKey: 'enabled',
    meta: { compact: true },
    enableGlobalFilter: false,
    header: () => h('div', null, i18n._('{count, plural, one {Status} other {Statuses}}')),
    cell: ({ row }) =>
      h(
        'div',
        null,
        h(Badge, { variant: row.getValue('enabled') ? 'success' : 'secondary' }, () =>
          row.getValue('enabled') ? i18n._('Enabled') : i18n._('Disabled')
        )
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
    cell: ({ row }) => h('div', { class: 'relative' }, h(dropdown, { snippet: row.original }))
  }
]
