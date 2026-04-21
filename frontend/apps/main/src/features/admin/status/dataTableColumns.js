import { h } from 'vue'
import dropdown from './dataTableDropdown.vue'
import { format } from 'date-fns'
import { CONVERSATION_DEFAULT_STATUSES_LIST } from '@/constants/conversation.js'

const DEFAULT_STATUS_KEY = {
  Open: 'globals.terms.open',
  Snoozed: 'globals.terms.snoozed',
  Resolved: 'globals.terms.resolved',
  Closed: 'globals.terms.closed'
}

export const createColumns = (t, { onEdit } = {}) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.name'))
    },
    cell: function ({ row }) {
      const name = row.getValue('name')
      const isDefault = CONVERSATION_DEFAULT_STATUSES_LIST.includes(name)
      const label = isDefault ? t(DEFAULT_STATUS_KEY[name]) : name
      return h('div', { class: 'text-center' },
        onEdit && !isDefault
          ? h('span', {
              class: 'text-primary hover:underline cursor-pointer',
              onClick: () => onEdit(row.original)
            }, label)
          : label
      )
    }
  },
  {
    accessorKey: 'category',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.category'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, t(`globals.terms.${row.getValue('category')}`))
    }
  },
  {
    accessorKey: 'created_at',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.createdAt'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, format(row.getValue('created_at'), 'PPpp'))
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => {
      const status = row.original
      return h(
        'div',
        { class: 'relative' },
        h(dropdown, {
          status
        })
      )
    }
  }
]
