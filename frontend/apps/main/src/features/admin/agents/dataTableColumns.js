import { h } from 'vue'
import UserDataTableDropDown from '@/features/admin/agents/dataTableDropdown.vue'
import { format } from 'date-fns'

export const createColumns = (t) => [
  {
    accessorKey: 'first_name',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.firstName'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center font-medium' }, row.getValue('first_name'))
    }
  },
  {
    accessorKey: 'last_name',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.lastName'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center font-medium' }, row.getValue('last_name'))
    }
  },
  {
    accessorKey: 'enabled',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.enabled'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center font-medium' }, row.getValue('enabled') ? t('globals.messages.yes') : t('globals.messages.no'))
    }
  },
  {
    accessorKey: 'email',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.email'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center font-medium' }, row.getValue('email'))
    }
  },
  {
    accessorKey: 'created_at',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.createdAt'))
    },
    cell: function ({ row }) {
      return h(
        'div',
        { class: 'text-center font-medium' },
        format(row.getValue('created_at'), 'PPpp')
      )
    }
  },
  {
    accessorKey: 'updated_at',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.updatedAt'))
    },
    cell: function ({ row }) {
      return h(
        'div',
        { class: 'text-center font-medium' },
        format(row.getValue('updated_at'), 'PPpp')
      )
    }
  },
  {
    id: 'actions',
    enableHiding: false,
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
