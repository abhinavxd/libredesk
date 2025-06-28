import { h } from 'vue'
import dropdown from './dataTableDropdown.vue'

export const createColumns = (t) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.name'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center font-medium' }, row.getValue('name'))
    }
  },
  {
    accessorKey: 'description',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.description'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center font-medium' }, row.getValue('description'))
    }
  },
  {
    id: 'actions',
    enableHiding: false,
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
