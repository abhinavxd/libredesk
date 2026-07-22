import { h } from 'vue'
import { format } from 'date-fns'
import { Badge } from '@shared-ui/components/ui/badge/index.js'
import dropdown from './snippetDropdown.vue'

export const createSnippetColumns = (t, { onEdit } = {}) => [
  {
    accessorKey: 'title',
    header: () => h('div', { class: 'text-center' }, t('globals.terms.title')),
    cell: ({ row }) => {
      const title = onEdit
        ? h(
            'span',
            {
              class: 'text-primary hover:underline cursor-pointer',
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
      return h('div', { class: 'text-center' }, children)
    }
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
    cell: ({ row }) => h('div', { class: 'relative' }, h(dropdown, { snippet: row.original }))
  }
]
