import { h } from 'vue'
import TableDate from '@main/components/table/TableDate.vue'
import { Badge } from '@shared-ui/components/ui/badge/index.js'
import dropdown from './snippetDropdown.vue'

export const createSnippetColumns = (t, { onEdit } = {}) => [
  {
    accessorKey: 'title',
    header: () => h('div', null, t('globals.terms.title')),
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
            t('admin.ai.snippet.autoCreated')
          )
        )
      }
      if (row.original.source === 'url') {
        children.push(
          h(
            Badge,
            { variant: 'secondary', class: 'ml-2 align-middle', title: row.original.source_url },
            () => t('admin.ai.snippet.imported')
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
    cell: ({ row }) => h('div', { class: 'relative' }, h(dropdown, { snippet: row.original }))
  }
]
