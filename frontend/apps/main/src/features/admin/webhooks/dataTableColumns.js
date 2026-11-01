

import { h } from 'vue'
import { RouterLink } from 'vue-router'
import dropdown from './dataTableDropdown.vue'
import TableDate from '@main/components/table/TableDate.vue'
import { Badge } from '@shared-ui/components/ui/badge'

export const createColumns = (i18n) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', null, i18n._('{count, plural, one {Name} other {Names}}'))
    },
    cell: function ({ row }) {
      return h('div', null,
        h(RouterLink,
          {
            to: { name: 'edit-webhook', params: { id: row.original.id } },
            class: 'text-foreground font-medium hover:underline'
          },
          () => row.getValue('name')
        )
      )
    }
  },
  {
    accessorKey: 'url',
    header: function () {
      return h('div', null, i18n._('{count, plural, one {URL} other {URLs}}'))
    },
    cell: function ({ row }) {
      const url = row.getValue('url')
      return h('div', { class: 'font-mono text-xs' }, url)
    }
  },
  {
    accessorKey: 'events',
    meta: { compact: true },
    enableGlobalFilter: false,
    header: function () {
      return h('div', null, i18n._('{count, plural, one {Event} other {Events}}', 2))
    },
    cell: function ({ row }) {
      const events = row.getValue('events')
      return h('div', null, [
        h(
          Badge,
          { variant: 'secondary', class: 'text-xs' },
          () => `${events.length} ${i18n._('{count, plural, one {Event} other {Events}}', 2)}`
        )
      ])
    },
    sortingFn: (rowA, rowB) => {
      const a = rowA.original.events?.length || 0
      const b = rowB.original.events?.length || 0
      return a - b
    }
  },
  {
    accessorKey: 'is_active',
    meta: { compact: true },
    enableGlobalFilter: false,
    header: () => h('div', null, i18n._('{count, plural, one {Status} other {Statuses}}')),
    cell: ({ row }) => {
      const isActive = row.getValue('is_active')
      return h('div', null, [
        h(
          Badge,
          {
            variant: isActive ? 'success' : 'secondary',
            class: 'text-xs'
          },
          () => isActive ? i18n._('Active') : i18n._('{count, plural, one {Inactive} other {Inactives}}')
        )
      ])
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
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => {
      const webhook = row.original
      return h(
        'div',
        { class: 'relative' },
        h(dropdown, {
          webhook
        })
      )
    }
  }
]
