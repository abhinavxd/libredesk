import { h } from 'vue'
import dropdown from './dataTableDropdown.vue'
import { format } from 'date-fns'

export const createColumns = (t, { onEdit } = {}) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.name'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' },
        onEdit
          ? h('span', {
              class: 'text-primary hover:underline cursor-pointer',
              onClick: () => onEdit(row.original)
            }, row.getValue('name'))
          : row.getValue('name')
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
