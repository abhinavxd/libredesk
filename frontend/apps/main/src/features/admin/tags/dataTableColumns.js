

import { h } from 'vue'
import dropdown from './dataTableDropdown.vue'
import TableDate from '@main/components/table/TableDate.vue'

export const createColumns = (i18n, { onEdit } = {}) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', null, i18n._('{count, plural, one {Name} other {Names}}'))
    },
    cell: function ({ row }) {
      return h('div', null,
        onEdit
          ? h('span', {
              class: 'text-foreground font-medium hover:underline cursor-pointer',
              onClick: () => onEdit(row.original)
            }, row.getValue('name'))
          : row.getValue('name')
      )
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
      const tag = row.original
      return h(
        'div',
        { class: 'relative' },
        h(dropdown, {
          tag
        })
      )
    }
  }
]
