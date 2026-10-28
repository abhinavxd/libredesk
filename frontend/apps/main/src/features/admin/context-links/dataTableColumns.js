import { h } from 'vue'
import { RouterLink } from 'vue-router'
import dropdown from './dataTableDropdown.vue'
import TableDate from '@main/components/table/TableDate.vue'
import { Badge } from '@shared-ui/components/ui/badge'

export const createColumns = (t) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', null, t('globals.terms.name'))
    },
    cell: function ({ row }) {
      return h(
        'div',
        null,
        h(
          RouterLink,
          {
            to: { name: 'edit-context-link', params: { id: row.original.id } },
            class: 'text-foreground font-medium hover:underline'
          },
          () => row.getValue('name')
        )
      )
    }
  },
  {
    accessorKey: 'url_template',
    header: function () {
      return h('div', null, t('contextLink.urlTemplate'))
    },
    cell: function ({ row }) {
      const url = row.getValue('url_template')
      return h('div', { class: 'font-mono text-xs' }, url)
    }
  },
  {
    accessorKey: 'is_active',
    meta: { compact: true },
    enableGlobalFilter: false,
    header: () => h('div', null, t('globals.terms.status')),
    cell: ({ row }) => {
      const isActive = row.getValue('is_active')
      return h('div', null, [
        h(
          Badge,
          {
            variant: isActive ? 'success' : 'secondary',
            class: 'text-xs'
          },
          () => (isActive ? t('globals.terms.active') : t('globals.terms.inactive'))
        )
      ])
    }
  },
  {
    accessorKey: 'created_at',
    meta: { compact: true },
    enableGlobalFilter: false,
    header: function () {
      return h('div', null, t('globals.terms.createdAt'))
    },
    cell: function ({ row }) {
      return h(TableDate, { value: row.getValue('created_at') })
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
