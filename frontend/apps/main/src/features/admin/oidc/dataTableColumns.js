import { h } from 'vue'
import { RouterLink } from 'vue-router'
import dropdown from './dataTableDropdown.vue'
import { Badge } from '@shared-ui/components/ui/badge/index.js'
import { format } from 'date-fns'

export const createColumns = (t) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.name'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' },
        h(RouterLink,
          {
            to: { name: 'edit-sso', params: { id: row.original.id } },
            class: 'text-foreground font-medium hover:underline'
          },
          () => row.getValue('name')
        )
      )
    }
  },
  {
    accessorKey: 'provider',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.provider'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, row.getValue('provider'))
    }
  },
  {
    accessorKey: 'enabled',
    enableGlobalFilter: false,
    header: () => h('div', { class: 'text-center' }, t('globals.terms.status')),
    cell: ({ row }) => {
      const enabled = row.getValue('enabled')
      return h(
        'div',
        { class: 'text-center' },
        h(Badge, { variant: enabled ? 'success' : 'secondary' }, () =>
          enabled ? t('globals.terms.enabled') : t('globals.terms.disabled')
        )
      )
    }
  },
  {
    accessorKey: 'created_at',
    enableGlobalFilter: false,
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.createdAt'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, format(row.getValue('created_at'), 'PPpp'))
    }
  },
  {
    accessorKey: 'updated_at',
    enableGlobalFilter: false,
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.updatedAt'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, format(row.getValue('updated_at'), 'PPpp'))
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => {
      const role = row.original
      return h(
        'div',
        { class: 'relative' },
        h(dropdown, {
          role
        })
      )
    }
  }
]
