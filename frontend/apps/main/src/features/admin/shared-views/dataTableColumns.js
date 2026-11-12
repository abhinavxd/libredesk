

import { h } from 'vue'
import { RouterLink } from 'vue-router'
import dropdown from './dataTableDropdown.vue'
import TableDate from '@main/components/table/TableDate.vue'

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
            to: { name: 'edit-shared-view', params: { id: row.original.id } },
            class: 'text-foreground font-medium hover:underline'
          },
          () => row.getValue('name')
        )
      )
    }
  },
  {
    accessorKey: 'visibility',
    meta: { compact: true },
    header: function () {
      return h('div', null, i18n._('{count, plural, one {Visibility} other {Visibilities}}'))
    },
    cell: function ({ row }) {
      const visibility = row.getValue('visibility')
      const label = visibility === 'all' ? i18n._('All agents') : i18n._('{count, plural, one {Team} other {Teams}}')
      return h('div', { class: 'capitalize' }, label)
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
      const sharedView = row.original
      return h(
        'div',
        { class: 'relative' },
        h(dropdown, {
          sharedView
        })
      )
    }
  }
]
