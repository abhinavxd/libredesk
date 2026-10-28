import { h } from 'vue'
import { RouterLink } from 'vue-router'
import UserDataTableDropDown from '@/features/admin/agents/dataTableDropdown.vue'
import { Badge } from '@shared-ui/components/ui/badge/index.js'
import TableDate from '@main/components/table/TableDate.vue'

export const createColumns = (t) => [
  {
    accessorKey: 'first_name',
    header: function () {
      return h('div', null, t('globals.terms.firstName'))
    },
    cell: function ({ row }) {
      return h('div', null,
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
      return h('div', null, t('globals.terms.lastName'))
    },
    cell: function ({ row }) {
      return h('div', null, row.getValue('last_name'))
    }
  },
  {
    accessorKey: 'enabled',
    meta: { compact: true },
    enableGlobalFilter: false,
    header: function () {
      return h('div', null, t('globals.terms.status'))
    },
    cell: function ({ row }) {
      return h(
        'div',
        null,
        h(Badge, { variant: row.getValue('enabled') ? 'success' : 'secondary' }, () =>
          row.getValue('enabled') ? t('globals.terms.enabled') : t('globals.terms.disabled')
        )
      )
    }
  },
  {
    accessorKey: 'email',
    header: function () {
      return h('div', null, t('globals.terms.email'))
    },
    cell: function ({ row }) {
      return h('div', null, row.getValue('email'))
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
    accessorKey: 'updated_at',
    meta: { compact: true },
    enableGlobalFilter: false,
    header: function () {
      return h('div', null, t('globals.terms.updatedAt'))
    },
    cell: function ({ row }) {
      return h(TableDate, { value: row.getValue('updated_at') })
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
