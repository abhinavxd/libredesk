import { h } from 'vue'
import { RouterLink } from 'vue-router'
import dropdown from './dataTableDropdown.vue'
import { format } from 'date-fns'
import { Badge } from '@shared-ui/components/ui/badge'

export const createColumns = (t) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.name'))
    },
    cell: function ({ row }) {
      return h(
        'div',
        { class: 'text-center' },
        h(
          RouterLink,
          {
            to: { name: 'edit-context-link', params: { id: row.original.id } },
            class: 'text-primary hover:underline'
          },
          () => row.getValue('name')
        )
      )
    }
  },
  {
    accessorKey: 'url_template',
    header: function () {
      return h('div', { class: 'text-center' }, t('contextLink.urlTemplate'))
    },
    cell: function ({ row }) {
      const url = row.getValue('url_template')
      return h('div', { class: 'text-center font-mono mt-1 max-w-sm truncate' }, url)
    }
  },
  {
    accessorKey: 'is_active',
    enableGlobalFilter: false,
    header: () => h('div', { class: 'text-center' }, t('globals.terms.status')),
    cell: ({ row }) => {
      const isActive = row.getValue('is_active')
      return h('div', { class: 'text-center' }, [
        h(
          Badge,
          {
            variant: isActive ? 'default' : 'secondary',
            class: 'text-xs'
          },
          () => (isActive ? t('globals.terms.active') : t('globals.terms.inactive'))
        )
      ])
    }
  },
  {
    accessorKey: 'created_at',
    enableGlobalFilter: false,
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.createdAt'))
    },
    cell: function ({ row }) {
      return h(
        'div',
        { class: 'text-center text-sm' },
        format(row.getValue('created_at'), 'PPpp')
      )
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => {
      const contextLink = row.original
      return h('div', { class: 'relative' }, h(dropdown, { contextLink }))
    }
  }
]
