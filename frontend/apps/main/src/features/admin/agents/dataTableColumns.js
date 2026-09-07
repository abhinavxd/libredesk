import { h } from 'vue'
import { RouterLink } from 'vue-router'
import UserDataTableDropDown from '@/features/admin/agents/dataTableDropdown.vue'
import { Badge } from '@shared-ui/components/ui/badge/index.js'
import { format } from 'date-fns'

export const createColumns = (t) => [
  {
    accessorKey: 'first_name',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.firstName'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' },
        h(RouterLink,
          {
            to: { name: 'edit-agent', params: { id: row.original.id } },
            class: 'text-foreground font-medium hover:underline'
          },
          () => row.getValue('first_name')
        )
      )
    }
  },
  {
    accessorKey: 'last_name',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.lastName'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, row.getValue('last_name'))
    }
  },
  {
    accessorKey: 'enabled',
    enableGlobalFilter: false,
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.status'))
    },
    cell: function ({ row }) {
      return h(
        'div',
        { class: 'text-center' },
        h(Badge, { variant: row.getValue('enabled') ? 'success' : 'secondary' }, () =>
          row.getValue('enabled') ? t('globals.terms.enabled') : t('globals.terms.disabled')
        )
      )
    }
  },
  {
    accessorKey: 'email',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.email'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, row.getValue('email'))
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
        { class: 'text-center' },
        format(row.getValue('created_at'), 'PPpp')
      )
    }
  },
  {
    accessorKey: 'updated_at',
    enableGlobalFilter: false,
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.updatedAt'))
    },
    cell: function ({ row }) {
      return h(
        'div',
        { class: 'text-center' },
        format(row.getValue('updated_at'), 'PPpp')
      )
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => {
      const user = row.original
      return h(
        'div',
        { class: 'relative' },
        h(UserDataTableDropDown, {
          user
        })
      )
    }
  }
]
