import { h } from 'vue'
import { RouterLink } from 'vue-router'
import dropdown from './dataTableDropdown.vue'
import TableDate from '@main/components/table/TableDate.vue'

export const createColumns = (t) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', null, t('globals.terms.name'))
    },
    cell: function ({ row }) {
      return h('div', null,
        h(RouterLink,
          {
            to: { name: 'edit-role', params: { id: row.original.id } },
            class: 'text-foreground font-medium hover:underline'
          },
          () => row.getValue('name')
        )
      )
    }
  },
  {
    accessorKey: 'description',
    header: function () {
      return h('div', null, t('globals.terms.description'))
    },
    cell: function ({ row }) {
      return h('div', null, row.getValue('description'))
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
