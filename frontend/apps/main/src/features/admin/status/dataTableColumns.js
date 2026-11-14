import { h } from 'vue'
import dropdown from './dataTableDropdown.vue'
import TableDate from '@main/components/table/TableDate.vue'
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
      return h('div', null, t('globals.terms.name'))
    },
    cell: function ({ row }) {
      const name = row.getValue('name')
      const isDefault = CONVERSATION_DEFAULT_STATUSES_LIST.includes(name)
      const label = isDefault ? t(DEFAULT_STATUS_KEY[name]) : name
      return h('div', null,
        onEdit && !isDefault
          ? h('span', {
              class: 'text-foreground font-medium hover:underline cursor-pointer',
              onClick: () => onEdit(row.original)
            }, label)
          : label
      )
    }
  },
  {
    accessorKey: 'category',
    meta: { compact: true },
    header: function () {
      return h('div', null, t('globals.terms.category'))
    },
    cell: function ({ row }) {
      return h('div', null, t(`globals.terms.${row.getValue('category')}`))
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
