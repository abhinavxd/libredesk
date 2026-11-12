

import { h } from 'vue'
import { RouterLink } from 'vue-router'
import UserDataTableDropDown from '@/features/admin/agents/dataTableDropdown.vue'
import { Badge } from '@shared-ui/components/ui/badge/index.js'
import TableDate from '@main/components/table/TableDate.vue'

export const createColumns = (i18n) => [
  {
    accessorKey: 'first_name',
    header: function () {
      return h('div', null, i18n._('{count, plural, one {First name} other {First names}}'))
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
      return h('div', null, i18n._('{count, plural, one {Last name} other {Last names}}'))
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
      return h('div', null, i18n._('{count, plural, one {Status} other {Statuses}}'))
    },
    cell: function ({ row }) {
      return h(
        'div',
        null,
        h(Badge, { variant: row.getValue('enabled') ? 'success' : 'secondary' }, () =>
          row.getValue('enabled') ? i18n._('Enabled') : i18n._('Disabled')
        )
      )
    }
  },
  {
    accessorKey: 'email',
    header: function () {
      return h('div', null, i18n._('{count, plural, one {Email} other {Emails}}'))
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
      return h('div', null, i18n._('Created at'))
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
      return h('div', null, i18n._('Updated at'))
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
